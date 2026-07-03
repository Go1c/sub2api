package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmailQueueDefaultBufferHandlesBulkSiteMessageCopies(t *testing.T) {
	queue := NewEmailQueueService(nil, 1)
	defer queue.Stop()

	require.GreaterOrEqual(t, cap(queue.taskChan), 1000)
}

func TestEmailQueueSiteMessageSendSlotIsRateLimited(t *testing.T) {
	queue := NewEmailQueueService(nil, 1)
	defer queue.Stop()

	start := time.Now()
	release, ok := queue.acquireSiteMessageSendSlot()
	require.True(t, ok)
	release()
	release, ok = queue.acquireSiteMessageSendSlot()
	require.True(t, ok)
	release()

	require.GreaterOrEqual(t, time.Since(start), 190*time.Millisecond)
	require.Equal(t, 100*time.Millisecond, defaultSiteMessageEmailSendInterval)
}
