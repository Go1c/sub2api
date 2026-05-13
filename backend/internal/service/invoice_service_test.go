package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type invoiceRepoStub struct {
	items        map[int64]*InvoiceRequest
	activeTotal  float64
	completed    *InvoiceRequest
	failedReason string
}

func (r *invoiceRepoStub) Create(_ context.Context, req *InvoiceRequest) error {
	if r.items == nil {
		r.items = make(map[int64]*InvoiceRequest)
	}
	req.ID = int64(len(r.items) + 1)
	req.OrderNo = FormatInvoiceOrderNo(req.ID)
	req.Status = InvoiceStatusProcessing
	r.items[req.ID] = cloneInvoiceRequest(req)
	return nil
}

func (r *invoiceRepoStub) GetByID(_ context.Context, id int64) (*InvoiceRequest, error) {
	if item := r.items[id]; item != nil {
		return cloneInvoiceRequest(item), nil
	}
	return nil, ErrInvoiceNotFound
}

func (r *invoiceRepoStub) ListByUser(_ context.Context, _ int64, params pagination.PaginationParams) ([]InvoiceRequest, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{Page: params.Page, PageSize: params.PageSize}, nil
}

func (r *invoiceRepoStub) ListAdmin(_ context.Context, params pagination.PaginationParams, _ InvoiceListFilter) ([]InvoiceRequest, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{Page: params.Page, PageSize: params.PageSize}, nil
}

func (r *invoiceRepoStub) SumActiveAmountByUser(context.Context, int64) (float64, error) {
	return r.activeTotal, nil
}

func (r *invoiceRepoStub) SumCompletedAmountByUser(context.Context, int64) (float64, error) {
	return 0, nil
}

func (r *invoiceRepoStub) MarkCompletedAndDeduct(_ context.Context, id int64, input CompleteInvoicePersistInput) (*InvoiceRequest, error) {
	item := r.items[id]
	if item == nil {
		return nil, ErrInvoiceNotFound
	}
	item.Status = InvoiceStatusCompleted
	item.FileName = input.FileName
	item.FilePath = input.FilePath
	item.FileSize = input.FileSize
	item.TaxRate = input.TaxRate
	item.TaxAmount = input.TaxAmount
	now := time.Unix(1778000000, 0)
	item.CompletedAt = &now
	r.completed = cloneInvoiceRequest(item)
	return cloneInvoiceRequest(item), nil
}

func (r *invoiceRepoStub) MarkFailed(_ context.Context, id int64, reason string) (*InvoiceRequest, error) {
	item := r.items[id]
	if item == nil {
		return nil, ErrInvoiceNotFound
	}
	item.Status = InvoiceStatusFailed
	item.FailureReason = reason
	r.failedReason = reason
	return cloneInvoiceRequest(item), nil
}

type invoiceUserReaderStub struct {
	user *User
}

func (r invoiceUserReaderStub) GetByID(context.Context, int64) (*User, error) {
	if r.user == nil {
		return nil, ErrUserNotFound
	}
	copy := *r.user
	return &copy, nil
}

type invoiceEmailSenderStub struct {
	err       error
	recipient string
	fileName  string
}

func (s *invoiceEmailSenderStub) SendInvoiceAttachment(_ context.Context, to string, invoice *InvoiceRequest, attachment InvoiceEmailAttachment) error {
	s.recipient = to
	s.fileName = attachment.FileName
	if invoice == nil {
		return errors.New("missing invoice")
	}
	return s.err
}

func TestInvoiceServiceCreateRejectsDisabledUser(t *testing.T) {
	svc := NewInvoiceService(&invoiceRepoStub{}, invoiceUserReaderStub{user: &User{
		ID:             7,
		Email:          "user@example.com",
		TotalRecharged: 100,
		InvoiceEnabled: false,
	}}, nil)

	_, err := svc.Create(context.Background(), 7, CreateInvoiceRequestInput{
		Title:          "Acme Inc.",
		TaxNumber:      "TAX123",
		Amount:         10,
		RecipientEmail: "billing@example.com",
	})

	require.ErrorIs(t, err, ErrInvoiceFeatureDisabled)
}

func TestInvoiceServiceCreateChecksRemainingInvoiceAmount(t *testing.T) {
	repo := &invoiceRepoStub{activeTotal: 80}
	svc := NewInvoiceService(repo, invoiceUserReaderStub{user: &User{
		ID:             7,
		Email:          "user@example.com",
		TotalRecharged: 100,
		InvoiceEnabled: true,
	}}, nil)

	_, err := svc.Create(context.Background(), 7, CreateInvoiceRequestInput{
		Title:          "Acme Inc.",
		TaxNumber:      "TAX123",
		Amount:         25,
		RecipientEmail: "billing@example.com",
	})

	require.ErrorIs(t, err, ErrInvoiceAmountExceeded)
}

func TestInvoiceServiceCompleteSendsEmailThenDeductsDefaultTaxRate(t *testing.T) {
	repo := &invoiceRepoStub{items: map[int64]*InvoiceRequest{
		1: {
			ID:             1,
			OrderNo:        "INV00000001",
			UserID:         7,
			UserEmail:      "user@example.com",
			Title:          "Acme Inc.",
			TaxNumber:      "TAX123",
			Amount:         200,
			RecipientEmail: "billing@example.com",
			Status:         InvoiceStatusProcessing,
		},
	}}
	mailer := &invoiceEmailSenderStub{}
	svc := NewInvoiceService(repo, invoiceUserReaderStub{}, mailer)

	completed, err := svc.Complete(context.Background(), 1, CompleteInvoiceInput{
		FileName:    "invoice.pdf",
		FilePath:    "data/invoices/INV00000001/invoice.pdf",
		FileSize:    1234,
		ContentType: "application/pdf",
		Bytes:       []byte("%PDF"),
	})

	require.NoError(t, err)
	require.Equal(t, InvoiceStatusCompleted, completed.Status)
	require.Equal(t, "billing@example.com", mailer.recipient)
	require.Equal(t, "invoice.pdf", mailer.fileName)
	require.NotNil(t, repo.completed)
	require.Equal(t, 0.01, repo.completed.TaxRate)
	require.Equal(t, 2.0, repo.completed.TaxAmount)
}

func TestInvoiceServiceCompleteMarksFailedWhenEmailFailsWithoutTaxDeduction(t *testing.T) {
	repo := &invoiceRepoStub{items: map[int64]*InvoiceRequest{
		1: {
			ID:             1,
			OrderNo:        "INV00000001",
			UserID:         7,
			UserEmail:      "user@example.com",
			Title:          "Acme Inc.",
			TaxNumber:      "TAX123",
			Amount:         200,
			RecipientEmail: "billing@example.com",
			Status:         InvoiceStatusProcessing,
		},
	}}
	svc := NewInvoiceService(repo, invoiceUserReaderStub{}, &invoiceEmailSenderStub{err: errors.New("smtp down")})

	_, err := svc.Complete(context.Background(), 1, CompleteInvoiceInput{
		FileName:    "invoice.pdf",
		FilePath:    "data/invoices/INV00000001/invoice.pdf",
		FileSize:    1234,
		ContentType: "application/pdf",
		Bytes:       []byte("%PDF"),
	})

	require.Error(t, err)
	require.Nil(t, repo.completed)
	require.Contains(t, repo.failedReason, "smtp down")
}
