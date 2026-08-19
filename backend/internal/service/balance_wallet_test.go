//go:build unit

package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type balanceWalletCaptureRepo struct {
	command       BalanceDebitCommand
	created       *BalanceDebitClient
	rotatedHash   string
	rotatedPrefix string
	debitCalls    int
	listCalls     int
}

func (r *balanceWalletCaptureRepo) CreateClient(_ context.Context, client *BalanceDebitClient) error {
	copy := *client
	r.created = &copy
	client.ID = 1
	client.CreatedAt = time.Unix(100, 0).UTC()
	client.UpdatedAt = client.CreatedAt
	return nil
}
func (r *balanceWalletCaptureRepo) ListClients(context.Context) ([]BalanceDebitClient, error) {
	return nil, nil
}
func (r *balanceWalletCaptureRepo) GetClient(context.Context, string) (*BalanceDebitClient, error) {
	return nil, ErrBalanceClientNotFound
}
func (r *balanceWalletCaptureRepo) GetActiveClientBySecretHash(context.Context, string) (*BalanceDebitClient, error) {
	return nil, ErrBalanceClientNotFound
}
func (r *balanceWalletCaptureRepo) UpdateClient(context.Context, string, UpdateBalanceClientInput) (*BalanceDebitClient, error) {
	return nil, nil
}
func (r *balanceWalletCaptureRepo) RotateClientSecret(_ context.Context, clientID, secretHash, secretPrefix string) (*BalanceDebitClient, error) {
	r.rotatedHash = secretHash
	r.rotatedPrefix = secretPrefix
	return &BalanceDebitClient{
		ID: 1, ClientID: clientID, Name: "client", SecretHash: secretHash, SecretPrefix: secretPrefix,
		AllowedPurposes: []string{"valid"}, Status: BalanceClientStatusActive,
	}, nil
}
func (r *balanceWalletCaptureRepo) DeactivateClient(context.Context, string) error { return nil }
func (r *balanceWalletCaptureRepo) Debit(_ context.Context, command BalanceDebitCommand) (*BalanceDebitResult, error) {
	r.debitCalls++
	r.command = command
	return &BalanceDebitResult{TxnID: "txn", Amount: command.Request.Amount, BalanceAfter: "1.00000000", Currency: "CNY"}, nil
}
func (r *balanceWalletCaptureRepo) ListTransactions(context.Context, int64, BalanceTransactionFilter) (*BalanceTransactionPage, error) {
	r.listCalls++
	return &BalanceTransactionPage{}, nil
}
func (r *balanceWalletCaptureRepo) GetTransaction(context.Context, int64, string) (*BalanceDebitTransaction, error) {
	return nil, ErrBalanceTransactionNotFound
}

func TestNormalizeBalanceDebitRequestCanonicalizesMoneyFingerprint(t *testing.T) {
	one, err := NormalizeBalanceDebitRequest("19.9", "CNY", "cchaven_monthly", " CC20260819-100001 ", "order-1")
	require.NoError(t, err)
	two, err := NormalizeBalanceDebitRequest("19.90", "CNY", "cchaven_monthly", "CC20260819-100001", "order-1")
	require.NoError(t, err)

	require.Equal(t, "19.90", one.Amount)
	require.Equal(t, "CC20260819-100001", one.Ref)
	require.Equal(t, one.Fingerprint, two.Fingerprint)
	require.Len(t, one.Fingerprint, 64)
	require.Len(t, one.IdempotencyKeyHash, 64)
}

func TestNormalizeBalanceDebitRequestRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name       string
		amount     string
		currency   string
		purpose    string
		ref        string
		key        string
		wantReason string
	}{
		{name: "missing key", amount: "1", currency: "CNY", purpose: "valid", ref: "r", wantReason: "IDEMPOTENCY_KEY_REQUIRED"},
		{name: "invalid key", amount: "1", currency: "CNY", purpose: "valid", ref: "r", key: "has space", wantReason: "IDEMPOTENCY_KEY_INVALID"},
		{name: "zero", amount: "0", currency: "CNY", purpose: "valid", ref: "r", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "negative", amount: "-1", currency: "CNY", purpose: "valid", ref: "r", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "three decimals", amount: "1.001", currency: "CNY", purpose: "valid", ref: "r", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "exponent", amount: "1e2", currency: "CNY", purpose: "valid", ref: "r", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "too large", amount: "1000000000000000000", currency: "CNY", purpose: "valid", ref: "r", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "currency", amount: "1", currency: "USD", purpose: "valid", ref: "r", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "purpose upper", amount: "1", currency: "CNY", purpose: "Invalid", ref: "r", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "purpose long", amount: "1", currency: "CNY", purpose: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ref: "r", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "empty ref", amount: "1", currency: "CNY", purpose: "valid", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
		{name: "control ref", amount: "1", currency: "CNY", purpose: "valid", ref: "line\nbreak", key: "k", wantReason: "INVALID_BALANCE_DEBIT_REQUEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeBalanceDebitRequest(tt.amount, tt.currency, tt.purpose, tt.ref, tt.key)
			require.Error(t, err)
			require.Equal(t, tt.wantReason, infraerrors.Reason(err))
		})
	}
}

func TestValidateBalancePurposeSlug(t *testing.T) {
	for _, purpose := range []string{"a", "cchaven_monthly", "client-2"} {
		require.True(t, IsValidBalancePurpose(purpose), purpose)
	}
	for _, purpose := range []string{"", "_leading", "trailing_", "two__parts", "UPPER", "has space"} {
		require.False(t, IsValidBalancePurpose(purpose), purpose)
	}
}

func TestBalanceWalletDebitCarriesAuthenticatedSecretHashIntoTransaction(t *testing.T) {
	repo := &balanceWalletCaptureRepo{}
	wallet := NewBalanceWalletService(repo)
	_, err := wallet.Debit(context.Background(), 9, &BalanceDebitClient{ID: 3, SecretHash: "authenticated-hash"}, BalanceDebitInput{
		Amount: "1", Currency: "CNY", Purpose: "valid", Ref: "ref", IdempotencyKey: "key",
	})
	require.NoError(t, err)
	require.Equal(t, "authenticated-hash", repo.command.ClientSecretHash)
}

func TestBalanceWalletDebitRejectsMissingAuthenticatedSecretHash(t *testing.T) {
	repo := &balanceWalletCaptureRepo{}
	wallet := NewBalanceWalletService(repo)

	_, err := wallet.Debit(context.Background(), 9, &BalanceDebitClient{ID: 3}, BalanceDebitInput{
		Amount: "1", Currency: "CNY", Purpose: "valid", Ref: "ref", IdempotencyKey: "key",
	})

	require.Error(t, err)
	require.Equal(t, "INVALID_BALANCE_CLIENT", infraerrors.Reason(err))
	require.Zero(t, repo.debitCalls)
}

func TestBalanceWalletListTransactionsRejectsInvalidClientID(t *testing.T) {
	repo := &balanceWalletCaptureRepo{}
	wallet := NewBalanceWalletService(repo)

	_, err := wallet.ListTransactions(context.Background(), 9, BalanceTransactionFilter{ClientID: "not-a-uuid"})

	require.Error(t, err)
	require.Equal(t, "INVALID_BALANCE_TRANSACTION_QUERY", infraerrors.Reason(err))
	require.Zero(t, repo.listCalls)
}

func TestBalanceClientSecretsAreOneTimePlaintextAndHashOnlyAtRest(t *testing.T) {
	repo := &balanceWalletCaptureRepo{}
	wallet := NewBalanceWalletService(repo)
	created, err := wallet.CreateClient(context.Background(), CreateBalanceClientInput{
		Name: " client ", AllowedPurposes: []string{"valid"},
	})
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`^bcs_[0-9a-f]{64}$`), created.Secret)
	require.Empty(t, created.SecretHash)
	require.NotNil(t, repo.created)
	require.Empty(t, repo.created.Secret)
	require.Equal(t, HashBalanceClientSecret(created.Secret), repo.created.SecretHash)
	require.Equal(t, "client", repo.created.Name)

	rotated, err := wallet.RotateClientSecret(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`^bcs_[0-9a-f]{64}$`), rotated.Secret)
	require.NotEqual(t, created.Secret, rotated.Secret)
	require.Empty(t, rotated.SecretHash)
	require.Equal(t, HashBalanceClientSecret(rotated.Secret), repo.rotatedHash)
	require.Equal(t, repo.rotatedPrefix, rotated.SecretPrefix)
}
