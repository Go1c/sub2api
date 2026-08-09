package dto

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserFromServiceShallow_MapsDeletedAt(t *testing.T) {
	ts := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)

	deleted := UserFromServiceShallow(&service.User{ID: 1, Email: "d@test.com", DeletedAt: &ts})
	require.NotNil(t, deleted.DeletedAt)
	require.Equal(t, ts, *deleted.DeletedAt)

	active := UserFromServiceShallow(&service.User{ID: 2, Email: "a@test.com"})
	require.Nil(t, active.DeletedAt, "active user must have nil DeletedAt")
}

func TestUserFromServiceShallow_MapsInvoiceEnabled(t *testing.T) {
	enabled := UserFromServiceShallow(&service.User{ID: 1, Email: "a@test.com", InvoiceEnabled: true})
	require.True(t, enabled.InvoiceEnabled)

	disabled := UserFromServiceShallow(&service.User{ID: 2, Email: "b@test.com", InvoiceEnabled: false})
	require.False(t, disabled.InvoiceEnabled)
}

func TestUserFromServiceShallow_MapsSubscriptionPurchaseDisabled(t *testing.T) {
	banned := UserFromServiceShallow(&service.User{ID: 1, Email: "a@test.com", SubscriptionPurchaseDisabled: true})
	require.True(t, banned.SubscriptionPurchaseDisabled)

	allowed := UserFromServiceShallow(&service.User{ID: 2, Email: "b@test.com", SubscriptionPurchaseDisabled: false})
	require.False(t, allowed.SubscriptionPurchaseDisabled)
}
