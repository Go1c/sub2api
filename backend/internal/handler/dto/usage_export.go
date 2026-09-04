package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type UsageExportItem struct {
	ID                  int64     `json:"id"`
	CorrelationID       *string   `json:"correlation_id"`
	RequestID           string    `json:"request_id"`
	CreatedAt           time.Time `json:"created_at"`
	APIKeyID            int64     `json:"api_key_id"`
	Model               string    `json:"model"`
	RequestedModel      string    `json:"requested_model"`
	InputTokens         int       `json:"input_tokens"`
	OutputTokens        int       `json:"output_tokens"`
	CacheCreationTokens int       `json:"cache_creation_tokens"`
	CacheReadTokens     int       `json:"cache_read_tokens"`
	TotalCost           float64   `json:"total_cost"`
	ActualCost          float64   `json:"actual_cost"`
	BillingType         int8      `json:"billing_type"`
}

type UsageExportData struct {
	Items       []UsageExportItem `json:"items"`
	NextAfterID int64             `json:"next_after_id,omitempty"`
	HasMore     bool              `json:"has_more"`
}

func UsageExportFromService(result *service.UsageExportResult) UsageExportData {
	out := UsageExportData{Items: []UsageExportItem{}, HasMore: false}
	if result == nil {
		return out
	}
	out.HasMore = result.HasMore
	out.NextAfterID = result.NextAfterID
	out.Items = make([]UsageExportItem, 0, len(result.Items))
	for i := range result.Items {
		row := result.Items[i]
		out.Items = append(out.Items, UsageExportItem{
			ID:                  row.ID,
			CorrelationID:       row.CorrelationID,
			RequestID:           row.RequestID,
			CreatedAt:           row.CreatedAt,
			APIKeyID:            row.APIKeyID,
			Model:               row.Model,
			RequestedModel:      row.RequestedModel,
			InputTokens:         row.InputTokens,
			OutputTokens:        row.OutputTokens,
			CacheCreationTokens: row.CacheCreationTokens,
			CacheReadTokens:     row.CacheReadTokens,
			TotalCost:           row.TotalCost,
			ActualCost:          row.ActualCost,
			BillingType:         row.BillingType,
		})
	}
	return out
}
