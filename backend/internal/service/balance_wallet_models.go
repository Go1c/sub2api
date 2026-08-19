package service

import (
	"context"
	"time"
)

const (
	BalanceClientStatusActive   = "active"
	BalanceClientStatusInactive = "inactive"
)

// BalanceDebitClient is an external server-side wallet consumer. Secret is
// populated only for create and rotate responses; SecretHash is never exposed.
type BalanceDebitClient struct {
	ID              int64
	ClientID        string
	Name            string
	Secret          string
	SecretHash      string
	SecretPrefix    string
	AllowedPurposes []string
	Status          string
	LastUsedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateBalanceClientInput struct {
	Name            string
	AllowedPurposes []string
}

type UpdateBalanceClientInput struct {
	Name            *string
	AllowedPurposes *[]string
	Status          *string
}

type BalanceDebitInput struct {
	Amount         string
	Currency       string
	Purpose        string
	Ref            string
	IdempotencyKey string
}

type BalanceDebitCommand struct {
	UserID           int64
	ClientID         int64
	ClientSecretHash string
	Request          NormalizedBalanceDebitRequest
}

type BalanceDebitResult struct {
	TxnID        string
	Amount       string
	BalanceAfter string
	Currency     string
	Replayed     bool
}

type BalanceDebitTransaction struct {
	ID              int64
	TxnID           string
	UserID          int64
	BalanceClientID int64
	ClientID        string
	ClientName      string
	Amount          string
	BalanceBefore   string
	BalanceAfter    string
	Currency        string
	Purpose         string
	Ref             string
	CreatedAt       time.Time
}

type BalanceTransactionFilter struct {
	Page     int
	PageSize int
	Purpose  string
	Ref      string
	ClientID string
}

type BalanceTransactionPage struct {
	Items    []BalanceDebitTransaction
	Total    int64
	Page     int
	PageSize int
}

// BalanceWalletRepository owns the atomic wallet transaction and its read models.
type BalanceWalletRepository interface {
	CreateClient(ctx context.Context, client *BalanceDebitClient) error
	ListClients(ctx context.Context) ([]BalanceDebitClient, error)
	GetClient(ctx context.Context, clientID string) (*BalanceDebitClient, error)
	GetActiveClientBySecretHash(ctx context.Context, secretHash string) (*BalanceDebitClient, error)
	UpdateClient(ctx context.Context, clientID string, input UpdateBalanceClientInput) (*BalanceDebitClient, error)
	RotateClientSecret(ctx context.Context, clientID, secretHash, secretPrefix string) (*BalanceDebitClient, error)
	DeactivateClient(ctx context.Context, clientID string) error

	Debit(ctx context.Context, command BalanceDebitCommand) (*BalanceDebitResult, error)
	ListTransactions(ctx context.Context, userID int64, filter BalanceTransactionFilter) (*BalanceTransactionPage, error)
	GetTransaction(ctx context.Context, userID int64, txnID string) (*BalanceDebitTransaction, error)
}

type BalanceCacheInvalidationEvent struct {
	ID         int64
	UserID     int64
	Attempts   int
	ClaimToken string
}

type BalanceCacheInvalidationOutboxRepository interface {
	Claim(ctx context.Context, limit int, lease time.Duration) ([]BalanceCacheInvalidationEvent, error)
	Complete(ctx context.Context, id int64, claimToken string) error
	Fail(ctx context.Context, id int64, claimToken, lastError string, retryAt time.Time) error
}
