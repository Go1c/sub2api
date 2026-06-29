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

type AdminSendCompensationBatchRequest struct {
	RecipientMode       string   `json:"recipient_mode" binding:"required"`
	RecipientEmails     []string `json:"recipient_emails"`
	Subject             string   `json:"subject" binding:"required"`
	Content             string   `json:"content" binding:"required"`
	CompensationEnabled bool     `json:"compensation_enabled"`
	CompensationAmount  float64  `json:"compensation_amount"`
	CompensationCodes   []string `json:"compensation_codes"`
	CompensationFormat  string   `json:"compensation_format"`
	SendEmail           *bool    `json:"send_email"`
}

func (r AdminSendCompensationBatchRequest) ShouldSendEmail() bool {
	return r.SendEmail != nil && *r.SendEmail
}

type SiteMessageCompensationCodeAssignment struct {
	Recipient string `json:"recipient"`
	Code      string `json:"code"`
	Status    string `json:"status"`
}

type SiteMessageCompensationBatch struct {
	ID             string                                  `json:"id"`
	Subject        string                                  `json:"subject"`
	Content        string                                  `json:"content"`
	Mode           string                                  `json:"mode"`
	Audience       string                                  `json:"audience"`
	RecipientCount int                                     `json:"recipient_count"`
	Amount         float64                                 `json:"amount"`
	CodeCount      int                                     `json:"code_count"`
	Operator       string                                  `json:"operator"`
	SentAt         time.Time                               `json:"sent_at"`
	Codes          []SiteMessageCompensationCodeAssignment `json:"codes"`
	MessageIDs     []int64                                 `json:"message_ids"`
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

func SiteMessageCompensationBatchFromService(batch *service.SiteMessageCompensationBatch) *SiteMessageCompensationBatch {
	if batch == nil {
		return nil
	}
	return &SiteMessageCompensationBatch{
		ID:             batch.ID,
		Subject:        batch.Subject,
		Content:        batch.Content,
		Mode:           batch.Mode,
		Audience:       batch.Audience,
		RecipientCount: batch.RecipientCount,
		Amount:         batch.Amount,
		CodeCount:      batch.CodeCount,
		Operator:       batch.Operator,
		SentAt:         batch.SentAt,
		Codes:          SiteMessageCompensationCodeAssignmentsFromService(batch.Codes),
		MessageIDs:     append([]int64(nil), batch.MessageIDs...),
	}
}

func SiteMessageCompensationCodeAssignmentsFromService(items []service.SiteMessageCompensationCodeAssignment) []SiteMessageCompensationCodeAssignment {
	out := make([]SiteMessageCompensationCodeAssignment, 0, len(items))
	for i := range items {
		out = append(out, SiteMessageCompensationCodeAssignment{
			Recipient: items[i].Recipient,
			Code:      items[i].Code,
			Status:    items[i].Status,
		})
	}
	return out
}
