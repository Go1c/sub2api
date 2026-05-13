package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	InvoiceStatusProcessing = "processing"
	InvoiceStatusCompleted  = "completed"
	InvoiceStatusFailed     = "failed"

	defaultInvoiceTaxRate = 0.01
)

var (
	ErrInvoiceNotFound        = infraerrors.NotFound("INVOICE_NOT_FOUND", "invoice request not found")
	ErrInvoiceFeatureDisabled = infraerrors.Forbidden("INVOICE_FEATURE_DISABLED", "invoice feature is not enabled for this user")
	ErrInvoiceAmountExceeded  = infraerrors.BadRequest("INVOICE_AMOUNT_EXCEEDED", "invoice amount exceeds remaining rechargeable amount")
	ErrInvoiceInvalidInput    = infraerrors.BadRequest("INVOICE_INVALID_INPUT", "invalid invoice request")
	ErrInvoiceInvalidStatus   = infraerrors.Conflict("INVOICE_INVALID_STATUS", "invoice request status does not allow this operation")
)

type InvoiceRequest struct {
	ID                         int64
	OrderNo                    string
	UserID                     int64
	UserEmail                  string
	Title                      string
	TaxNumber                  string
	Amount                     float64
	RecipientEmail             string
	Status                     string
	FileName                   string
	FilePath                   string
	FileSize                   int64
	ContentType                string
	TaxRate                    float64
	TaxAmount                  float64
	UserCompletedInvoiceAmount float64
	UserTotalRecharged         float64
	FailureReason              string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	CompletedAt                *time.Time
	User                       *User
}

type InvoiceOverview struct {
	TotalRecharged    float64 `json:"total_recharged"`
	UsedInvoiceAmount float64 `json:"used_invoice_amount"`
	RemainingAmount   float64 `json:"remaining_amount"`
	Enabled           bool    `json:"enabled"`
}

type CreateInvoiceRequestInput struct {
	Title          string
	TaxNumber      string
	Amount         float64
	RecipientEmail string
}

type CompleteInvoiceInput struct {
	FileName    string
	FilePath    string
	FileSize    int64
	ContentType string
	Bytes       []byte
	TaxRate     *float64
}

type CompleteInvoicePersistInput struct {
	FileName    string
	FilePath    string
	FileSize    int64
	ContentType string
	TaxRate     float64
	TaxAmount   float64
}

type InvoiceListFilter struct {
	Search string
	Status string
	UserID int64
}

type InvoiceEmailAttachment struct {
	FileName    string
	ContentType string
	Bytes       []byte
}

type InvoiceRepository interface {
	Create(ctx context.Context, req *InvoiceRequest) error
	GetByID(ctx context.Context, id int64) (*InvoiceRequest, error)
	ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]InvoiceRequest, *pagination.PaginationResult, error)
	ListAdmin(ctx context.Context, params pagination.PaginationParams, filter InvoiceListFilter) ([]InvoiceRequest, *pagination.PaginationResult, error)
	SumActiveAmountByUser(ctx context.Context, userID int64) (float64, error)
	SumCompletedAmountByUser(ctx context.Context, userID int64) (float64, error)
	MarkCompletedAndDeduct(ctx context.Context, id int64, input CompleteInvoicePersistInput) (*InvoiceRequest, error)
	MarkFailed(ctx context.Context, id int64, reason string) (*InvoiceRequest, error)
}

type InvoiceUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

type InvoiceEmailSender interface {
	SendInvoiceAttachment(ctx context.Context, to string, invoice *InvoiceRequest, attachment InvoiceEmailAttachment) error
}

type InvoiceService struct {
	repo        InvoiceRepository
	userRepo    InvoiceUserReader
	emailSender InvoiceEmailSender
}

func NewInvoiceService(repo InvoiceRepository, userRepo InvoiceUserReader, emailSender InvoiceEmailSender) *InvoiceService {
	return &InvoiceService{repo: repo, userRepo: userRepo, emailSender: emailSender}
}

func FormatInvoiceOrderNo(id int64) string {
	return fmt.Sprintf("INV%08d", id)
}

func (s *InvoiceService) Overview(ctx context.Context, userID int64) (*InvoiceOverview, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	used, err := s.repo.SumActiveAmountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	remaining := roundMoney(user.TotalRecharged - used)
	if remaining < 0 {
		remaining = 0
	}
	return &InvoiceOverview{
		TotalRecharged:    roundMoney(user.TotalRecharged),
		UsedInvoiceAmount: roundMoney(used),
		RemainingAmount:   remaining,
		Enabled:           user.InvoiceEnabled,
	}, nil
}

func (s *InvoiceService) Create(ctx context.Context, userID int64, input CreateInvoiceRequestInput) (*InvoiceRequest, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.InvoiceEnabled {
		return nil, ErrInvoiceFeatureDisabled
	}

	title := strings.TrimSpace(input.Title)
	taxNumber := strings.TrimSpace(input.TaxNumber)
	recipientEmail := strings.TrimSpace(input.RecipientEmail)
	amount := roundMoney(input.Amount)
	if title == "" || taxNumber == "" || recipientEmail == "" || amount <= 0 {
		return nil, ErrInvoiceInvalidInput
	}

	activeAmount, err := s.repo.SumActiveAmountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if roundMoney(user.TotalRecharged-activeAmount) < amount {
		return nil, ErrInvoiceAmountExceeded
	}

	req := &InvoiceRequest{
		UserID:         user.ID,
		UserEmail:      user.Email,
		Title:          title,
		TaxNumber:      taxNumber,
		Amount:         amount,
		RecipientEmail: recipientEmail,
		Status:         InvoiceStatusProcessing,
	}
	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *InvoiceService) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]InvoiceRequest, *pagination.PaginationResult, error) {
	return s.repo.ListByUser(ctx, userID, params)
}

func (s *InvoiceService) ListAdmin(ctx context.Context, params pagination.PaginationParams, filter InvoiceListFilter) ([]InvoiceRequest, *pagination.PaginationResult, error) {
	return s.repo.ListAdmin(ctx, params, filter)
}

func (s *InvoiceService) Get(ctx context.Context, id int64) (*InvoiceRequest, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *InvoiceService) Complete(ctx context.Context, id int64, input CompleteInvoiceInput) (*InvoiceRequest, error) {
	invoice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if invoice.Status != InvoiceStatusProcessing {
		return nil, ErrInvoiceInvalidStatus
	}

	fileName := strings.TrimSpace(input.FileName)
	filePath := strings.TrimSpace(input.FilePath)
	if fileName == "" || filePath == "" || len(input.Bytes) == 0 {
		return nil, ErrInvoiceInvalidInput
	}

	taxRate := defaultInvoiceTaxRate
	if input.TaxRate != nil {
		taxRate = *input.TaxRate
	}
	if taxRate < 0 {
		return nil, ErrInvoiceInvalidInput
	}
	taxAmount := roundMoney(invoice.Amount * taxRate)

	if s.emailSender == nil {
		return nil, ErrEmailNotConfigured
	}
	attachment := InvoiceEmailAttachment{
		FileName:    fileName,
		ContentType: strings.TrimSpace(input.ContentType),
		Bytes:       input.Bytes,
	}
	if err := s.emailSender.SendInvoiceAttachment(ctx, invoice.RecipientEmail, invoice, attachment); err != nil {
		_, _ = s.repo.MarkFailed(ctx, id, fmt.Sprintf("send invoice email: %v", err))
		return nil, fmt.Errorf("send invoice email: %w", err)
	}

	return s.repo.MarkCompletedAndDeduct(ctx, id, CompleteInvoicePersistInput{
		FileName:    fileName,
		FilePath:    filePath,
		FileSize:    input.FileSize,
		ContentType: strings.TrimSpace(input.ContentType),
		TaxRate:     taxRate,
		TaxAmount:   taxAmount,
	})
}

func (s *InvoiceService) MarkFailed(ctx context.Context, id int64, reason string) (*InvoiceRequest, error) {
	return s.repo.MarkFailed(ctx, id, strings.TrimSpace(reason))
}

func cloneInvoiceRequest(src *InvoiceRequest) *InvoiceRequest {
	if src == nil {
		return nil
	}
	copy := *src
	if src.CompletedAt != nil {
		completedAt := *src.CompletedAt
		copy.CompletedAt = &completedAt
	}
	if src.User != nil {
		user := *src.User
		copy.User = &user
	}
	return &copy
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
