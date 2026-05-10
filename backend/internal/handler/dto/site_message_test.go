package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSiteMessageFromServiceMarksAdminSender(t *testing.T) {
	message := &service.SiteMessage{
		ID:          1,
		SenderID:    10,
		RecipientID: 20,
		Subject:     "notice",
		Content:     "body",
		Sender: &service.User{
			ID:       10,
			Email:    "admin@example.com",
			Username: "admin",
			Role:     service.RoleAdmin,
		},
		Recipient: &service.User{
			ID:       20,
			Email:    "user@example.com",
			Username: "user",
			Role:     service.RoleUser,
		},
	}

	got := SiteMessageFromService(message)

	require.NotNil(t, got.Sender)
	require.True(t, got.Sender.IsAdmin)
	require.NotNil(t, got.Recipient)
	require.False(t, got.Recipient.IsAdmin)
}

func TestAdminSendSiteMessageRequestDefaultsEmailCopyOn(t *testing.T) {
	req := AdminSendSiteMessageRequest{}

	require.True(t, req.ShouldSendEmail())
}

func TestAdminSendSiteMessageRequestCanDisableEmailCopy(t *testing.T) {
	sendEmail := false
	req := AdminSendSiteMessageRequest{SendEmail: &sendEmail}

	require.False(t, req.ShouldSendEmail())
}
