package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type InvoiceRequest struct {
	ID                         int64      `json:"id"`
	OrderNo                    string     `json:"order_no"`
	UserID                     int64      `json:"user_id"`
	UserEmail                  string     `json:"user_email"`
	Title                      string     `json:"title"`
	TaxNumber                  string     `json:"tax_number"`
	Amount                     float64    `json:"amount"`
	RecipientEmail             string     `json:"recipient_email"`
	Status                     string     `json:"status"`
	FileName                   string     `json:"file_name"`
	FileSize                   int64      `json:"file_size"`
	HasFile                    bool       `json:"has_file"`
	TaxRate                    float64    `json:"tax_rate"`
	TaxAmount                  float64    `json:"tax_amount"`
	UserCompletedInvoiceAmount float64    `json:"user_completed_invoice_amount"`
	UserTotalRecharged         float64    `json:"user_total_recharged"`
	FailureReason              string     `json:"failure_reason,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
	CompletedAt                *time.Time `json:"completed_at,omitempty"`
	User                       *User      `json:"user,omitempty"`
}

type CreateInvoiceRequest struct {
	Title          string  `json:"title" binding:"required"`
	TaxNumber      string  `json:"tax_number" binding:"required"`
	Amount         float64 `json:"amount" binding:"required,gt=0"`
	RecipientEmail string  `json:"recipient_email" binding:"required,email"`
}

type FailInvoiceRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func InvoiceRequestFromService(item *service.InvoiceRequest) *InvoiceRequest {
	if item == nil {
		return nil
	}
	return &InvoiceRequest{
		ID:                         item.ID,
		OrderNo:                    item.OrderNo,
		UserID:                     item.UserID,
		UserEmail:                  item.UserEmail,
		Title:                      item.Title,
		TaxNumber:                  item.TaxNumber,
		Amount:                     item.Amount,
		RecipientEmail:             item.RecipientEmail,
		Status:                     item.Status,
		FileName:                   item.FileName,
		FileSize:                   item.FileSize,
		HasFile:                    item.FilePath != "",
		TaxRate:                    item.TaxRate,
		TaxAmount:                  item.TaxAmount,
		UserCompletedInvoiceAmount: item.UserCompletedInvoiceAmount,
		UserTotalRecharged:         item.UserTotalRecharged,
		FailureReason:              item.FailureReason,
		CreatedAt:                  item.CreatedAt,
		UpdatedAt:                  item.UpdatedAt,
		CompletedAt:                item.CompletedAt,
		User:                       UserFromServiceShallow(item.User),
	}
}

func InvoiceRequestsFromService(items []service.InvoiceRequest) []InvoiceRequest {
	out := make([]InvoiceRequest, 0, len(items))
	for i := range items {
		if converted := InvoiceRequestFromService(&items[i]); converted != nil {
			out = append(out, *converted)
		}
	}
	return out
}
