package service

import (
	"context"
	"fmt"
	"time"
)

const (
	UsageExportDefaultLimit = 100
	UsageExportMaxLimit     = 500
	UsageExportMaxLookback  = 24 * time.Hour
)

// UsageExportRow is the slim incremental dump used by downstream resellers.
type UsageExportRow struct {
	ID                  int64
	CorrelationID       *string
	RequestID           string
	CreatedAt           time.Time
	APIKeyID            int64
	Model               string
	RequestedModel      string
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	TotalCost           float64
	ActualCost          float64
	BillingType         int8
}

type UsageExportResult struct {
	Items       []UsageExportRow
	NextAfterID int64
	HasMore     bool
}

type usageLogExporter interface {
	ListExport(ctx context.Context, userID, afterID int64, since *time.Time, limit int) ([]UsageExportRow, error)
}

func (s *UsageService) Export(ctx context.Context, userID, afterID int64, since *time.Time, limit int) (*UsageExportResult, error) {
	if s == nil || s.usageRepo == nil {
		return nil, fmt.Errorf("usage export unavailable")
	}
	exporter, ok := s.usageRepo.(usageLogExporter)
	if !ok {
		return nil, fmt.Errorf("usage export unavailable")
	}
	if limit <= 0 {
		limit = UsageExportDefaultLimit
	}
	if limit > UsageExportMaxLimit {
		limit = UsageExportMaxLimit
	}
	rows, err := exporter.ListExport(ctx, userID, afterID, since, limit+1)
	if err != nil {
		return nil, fmt.Errorf("list usage export: %w", err)
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	result := &UsageExportResult{Items: rows, HasMore: hasMore}
	if len(rows) > 0 {
		result.NextAfterID = rows[len(rows)-1].ID
	} else if afterID > 0 {
		result.NextAfterID = afterID
	}
	return result, nil
}
