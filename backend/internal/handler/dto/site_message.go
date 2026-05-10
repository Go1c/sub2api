package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type SiteMessageRecipient struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
}

type SiteMessage struct {
	ID          int64                 `json:"id"`
	SenderID    int64                 `json:"sender_id"`
	RecipientID int64                 `json:"recipient_id"`
	ParentID    *int64                `json:"parent_id,omitempty"`
	Subject     string                `json:"subject"`
	Content     string                `json:"content"`
	ReadAt      *time.Time            `json:"read_at,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	Sender      *SiteMessageRecipient `json:"sender,omitempty"`
	Recipient   *SiteMessageRecipient `json:"recipient,omitempty"`
	Replies     []SiteMessage         `json:"replies,omitempty"`
}

type CreateSiteMessageRequest struct {
	Recipient string `json:"recipient" binding:"required"`
	Subject   string `json:"subject" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

type ReplySiteMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type AdminSendSiteMessageRequest struct {
	Subject   string `json:"subject" binding:"required"`
	Content   string `json:"content" binding:"required"`
	SendEmail *bool  `json:"send_email"`
}

func (r AdminSendSiteMessageRequest) ShouldSendEmail() bool {
	return r.SendEmail == nil || *r.SendEmail
}

func SiteMessageFromService(message *service.SiteMessage) *SiteMessage {
	if message == nil {
		return nil
	}
	return &SiteMessage{
		ID:          message.ID,
		SenderID:    message.SenderID,
		RecipientID: message.RecipientID,
		ParentID:    message.ParentID,
		Subject:     message.Subject,
		Content:     message.Content,
		ReadAt:      message.ReadAt,
		CreatedAt:   message.CreatedAt,
		UpdatedAt:   message.UpdatedAt,
		Sender:      SiteMessageUserFromService(message.Sender),
		Recipient:   SiteMessageUserFromService(message.Recipient),
		Replies:     SiteMessagesFromService(message.Replies),
	}
}

func SiteMessagesFromService(items []service.SiteMessage) []SiteMessage {
	out := make([]SiteMessage, 0, len(items))
	for i := range items {
		if converted := SiteMessageFromService(&items[i]); converted != nil {
			out = append(out, *converted)
		}
	}
	return out
}

func SiteMessageUserFromService(user *service.User) *SiteMessageRecipient {
	if user == nil {
		return nil
	}
	return &SiteMessageRecipient{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		IsAdmin:  user.IsAdmin(),
	}
}

func SiteMessageRecipientFromService(recipient *service.SiteMessageRecipient) *SiteMessageRecipient {
	if recipient == nil {
		return nil
	}
	return &SiteMessageRecipient{
		ID:       recipient.ID,
		Email:    recipient.Email,
		Username: recipient.Username,
	}
}

func SiteMessageRecipientsFromService(items []service.SiteMessageRecipient) []SiteMessageRecipient {
	out := make([]SiteMessageRecipient, 0, len(items))
	for i := range items {
		if converted := SiteMessageRecipientFromService(&items[i]); converted != nil {
			out = append(out, *converted)
		}
	}
	return out
}
