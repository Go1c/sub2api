package service

// fork 侧 reseller usage correlation（自 dev 的 gateway_usage_billing.go 抽出）。

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

func usageCorrelationIDFromCtx(ctx context.Context) *string {
	if ctx == nil {
		return nil
	}
	value, _ := ctx.Value(ctxkey.Sub2RequestID).(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
