package handler

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestSubmitUsageRecordTaskCopiesRequestContext(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-request-123")
	parent = context.WithValue(parent, ctxkey.RequestID, "request-456")

	var gotClientRequestID string
	var gotRequestID string
	h := &GatewayHandler{}
	h.submitUsageRecordTask(parent, func(ctx context.Context) {
		gotClientRequestID, _ = ctx.Value(ctxkey.ClientRequestID).(string)
		gotRequestID, _ = ctx.Value(ctxkey.RequestID).(string)
	})

	require.Equal(t, "client-request-123", gotClientRequestID)
	require.Equal(t, "request-456", gotRequestID)
}

func TestOpenAISubmitUsageRecordTaskCopiesRequestContext(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.ClientRequestID, "openai-client-request-123")
	parent = context.WithValue(parent, ctxkey.RequestID, "openai-request-456")

	var gotClientRequestID string
	var gotRequestID string
	h := &OpenAIGatewayHandler{}
	h.submitUsageRecordTask(parent, func(ctx context.Context) {
		gotClientRequestID, _ = ctx.Value(ctxkey.ClientRequestID).(string)
		gotRequestID, _ = ctx.Value(ctxkey.RequestID).(string)
	})

	require.Equal(t, "openai-client-request-123", gotClientRequestID)
	require.Equal(t, "openai-request-456", gotRequestID)
}

func TestUsageWorkerPreservesEgressProxyID(t *testing.T) {
	pool := service.NewUsageRecordWorkerPoolWithOptions(service.UsageRecordWorkerPoolOptions{WorkerCount: 1, QueueSize: 4})
	defer pool.Stop()
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), ctxkey.EgressProxyID, int64(42)))
	cancel()
	for _, submit := range []func(context.Context, service.UsageRecordTask){
		(&GatewayHandler{usageRecordWorkerPool: pool}).submitUsageRecordTask,
		(&OpenAIGatewayHandler{usageRecordWorkerPool: pool}).submitUsageRecordTask,
	} {
		result := make(chan int64, 1)
		submit(parent, func(ctx context.Context) { result <- service.EgressProxyIDFrom(ctx, nil) })
		select {
		case id := <-result:
			require.Equal(t, int64(42), id)
		case <-time.After(5 * time.Second):
			t.Fatal("usage worker did not run")
		}
	}
}
