package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestShouldTryGroupFallbackOnAccountError(t *testing.T) {
	fallbackID := int64(20)
	currentKey := &service.APIKey{
		Group: &service.Group{
			ID:                         10,
			FallbackGroupIDOnExhausted: &fallbackID,
		},
	}

	require.True(t, shouldTryGroupFallbackOnAccountError(currentKey, false, false))
	require.False(t, shouldTryGroupFallbackOnAccountError(currentKey, true, false))
	require.False(t, shouldTryGroupFallbackOnAccountError(currentKey, false, true))
	require.False(t, shouldTryGroupFallbackOnAccountError(&service.APIKey{Group: &service.Group{ID: 10}}, false, false))
}

func TestTryGroupFallbackAllowsExclusiveTarget(t *testing.T) {
	fallbackID := int64(20)
	user := &service.User{ID: 1, AllowedGroups: nil}
	currentGroup := &service.Group{
		ID:                         10,
		Platform:                   service.PlatformAnthropic,
		Status:                     service.StatusActive,
		FallbackGroupIDOnExhausted: &fallbackID,
	}
	fallbackGroup := &service.Group{
		ID:          fallbackID,
		Platform:    service.PlatformAnthropic,
		Status:      service.StatusActive,
		IsExclusive: true,
	}
	currentKey := &service.APIKey{
		ID:      100,
		UserID:  user.ID,
		User:    user,
		GroupID: &currentGroup.ID,
		Group:   currentGroup,
	}

	result := tryGroupFallback(
		context.Background(),
		zap.NewNop(),
		currentKey,
		false,
		false,
		func(context.Context, int64) (*service.Group, error) {
			return fallbackGroup, nil
		},
		func(context.Context, *service.APIKey, *service.Group) *service.UserSubscription {
			return nil
		},
		func(context.Context, *service.User, *service.APIKey, *service.Group, *service.UserSubscription) error {
			return nil
		},
		func(int, string, string, int) {
			t.Fatal("billing error writer should not be called")
		},
	)

	require.False(t, result.ErrorWritten)
	require.NotNil(t, result.APIKey)
	require.Equal(t, fallbackID, *result.APIKey.GroupID)
	require.Same(t, fallbackGroup, result.APIKey.Group)
	require.Same(t, fallbackGroup, result.Group)
}
