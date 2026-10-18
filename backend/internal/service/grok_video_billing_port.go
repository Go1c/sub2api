package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// GrokVideoPendingBilling is the create-time snapshot used when status polling
// first observes a completed video URL.
type GrokVideoPendingBilling struct {
	Model                string `json:"model"`
	BillingModel         string `json:"billing_model,omitempty"`
	UpstreamModel        string `json:"upstream_model,omitempty"`
	VideoResolution      string `json:"video_resolution,omitempty"`
	VideoDurationSeconds int    `json:"video_duration_seconds,omitempty"`
	OriginalModel        string `json:"original_model,omitempty"`
	CreatedAt            string `json:"created_at,omitempty"`
}

func GrokVideoPendingCreatedAtNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// GrokVideoE2EDuration returns wall time from create accept to discovery of completion.
func GrokVideoE2EDuration(createdAt string, discoveredAt time.Time) time.Duration {
	createdAt = strings.TrimSpace(createdAt)
	if createdAt == "" {
		return 0
	}
	if discoveredAt.IsZero() {
		discoveredAt = time.Now()
	}
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		created, err = time.Parse(time.RFC3339, createdAt)
		if err != nil || created.IsZero() {
			return 0
		}
	}
	d := discoveredAt.Sub(created)
	if d < 0 {
		return 0
	}
	return d
}

// StableGrokVideoBillingRequestID is the durable usage_logs key for one async video task.
func StableGrokVideoBillingRequestID(taskRequestID string) string {
	taskRequestID = strings.TrimSpace(taskRequestID)
	if taskRequestID == "" {
		return ""
	}
	if strings.HasPrefix(taskRequestID, "grok-video:") {
		return taskRequestID
	}
	return "grok-video:" + taskRequestID
}

func (s *OpenAIGatewayService) LoadGrokVideoPendingBilling(ctx context.Context, requestID string, userID, apiKeyID int64) (*GrokVideoPendingBilling, error) {
	_ = ctx
	if s == nil || strings.TrimSpace(requestID) == "" || userID <= 0 || apiKeyID <= 0 {
		return nil, fmt.Errorf("grok video pending billing is unavailable")
	}
	return nil, fmt.Errorf("grok video pending billing is unavailable")
}

func (s *OpenAIGatewayService) ClaimGrokVideoBilling(ctx context.Context, requestID string, userID, apiKeyID int64) (bool, error) {
	_ = ctx
	if s == nil || strings.TrimSpace(requestID) == "" || userID <= 0 || apiKeyID <= 0 {
		return false, fmt.Errorf("grok video billing claim is unavailable")
	}
	return false, fmt.Errorf("grok video billing claim is unavailable")
}
