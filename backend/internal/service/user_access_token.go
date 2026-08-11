package service

import "time"

// UserAccessToken is the domain model for a long-lived opaque user access token.
// Token (plaintext) is only populated on create response; never persisted.
// TokenHash is internal (repo/service); list/API responses must leave it empty.
type UserAccessToken struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Name        string     `json:"name"`
	Token       string     `json:"token,omitempty"` // plaintext, create-only
	TokenHash   string     `json:"-"`
	TokenPrefix string     `json:"token_prefix"`
	ExpiresAt   time.Time  `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateUserAccessTokenInput is the create payload.
type CreateUserAccessTokenInput struct {
	Name          string
	ExpiresInDays *int // nil → default 7; valid range 1–30 inclusive
}
