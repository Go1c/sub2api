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

func TestSiteMessageFromServiceIncludesReplies(t *testing.T) {
	parentID := int64(1)
	message := &service.SiteMessage{
		ID:          parentID,
		SenderID:    10,
		RecipientID: 20,
		Subject:     "notice",
		Content:     "body",
		Replies: []service.SiteMessage{
			{
				ID:          2,
				SenderID:    20,
				RecipientID: 10,
				ParentID:    &parentID,
				Subject:     "Re: notice",
				Content:     "reply body",
			},
		},
	}

	got := SiteMessageFromService(message)

	require.Len(t, got.Replies, 1)
	require.Equal(t, int64(2), got.Replies[0].ID)
	require.Equal(t, "reply body", got.Replies[0].Content)
	require.NotNil(t, got.Replies[0].ParentID)
	require.Equal(t, parentID, *got.Replies[0].ParentID)
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
