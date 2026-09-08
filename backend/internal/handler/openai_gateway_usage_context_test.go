package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestSubmitUsageRecordTaskCopiesRequestContext(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-request-123")
	parent = context.WithValue(parent, ctxkey.RequestID, "request-456")
	parent = context.WithValue(parent, ctxkey.Sub2RequestID, "3fa85f64-5717-4562-b3fc-2c963f66afa6")

	var gotClientRequestID string
	var gotRequestID string
	var gotSub2RequestID string
	h := &GatewayHandler{}
	h.submitUsageRecordTask(parent, func(ctx context.Context) {
		gotClientRequestID, _ = ctx.Value(ctxkey.ClientRequestID).(string)
		gotRequestID, _ = ctx.Value(ctxkey.RequestID).(string)
		gotSub2RequestID, _ = ctx.Value(ctxkey.Sub2RequestID).(string)
	})

	require.Equal(t, "client-request-123", gotClientRequestID)
	require.Equal(t, "request-456", gotRequestID)
	require.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", gotSub2RequestID)
}

func TestOpenAISubmitUsageRecordTaskCopiesRequestContext(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.ClientRequestID, "openai-client-request-123")
	parent = context.WithValue(parent, ctxkey.RequestID, "openai-request-456")
	parent = context.WithValue(parent, ctxkey.Sub2RequestID, "3fa85f64-5717-4562-b3fc-2c963f66afa6")

	var gotClientRequestID string
	var gotRequestID string
	var gotSub2RequestID string
	h := &OpenAIGatewayHandler{}
	h.submitUsageRecordTask(parent, func(ctx context.Context) {
		gotClientRequestID, _ = ctx.Value(ctxkey.ClientRequestID).(string)
		gotRequestID, _ = ctx.Value(ctxkey.RequestID).(string)
		gotSub2RequestID, _ = ctx.Value(ctxkey.Sub2RequestID).(string)
	})

	require.Equal(t, "openai-client-request-123", gotClientRequestID)
	require.Equal(t, "openai-request-456", gotRequestID)
	require.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", gotSub2RequestID)
}
