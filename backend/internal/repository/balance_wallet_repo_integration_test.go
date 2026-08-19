//go:build integration

package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBalanceWalletRepositoryDebitReplayAndRetry(t *testing.T) {
	ctx := context.Background()
	repo := NewBalanceWalletRepository(integrationDB)

	client := &service.BalanceDebitClient{
		ClientID:        uuid.NewString(),
		Name:            "wallet-test",
		SecretHash:      service.HashBalanceClientSecret("bcs_test"),
		SecretPrefix:    "bcs_test",
		AllowedPurposes: []string{"cchaven_monthly"},
		Status:          service.BalanceClientStatusActive,
	}
	require.NoError(t, repo.CreateClient(ctx, client))

	user := mustCreateUser(t, testEntClient(t), &service.User{
		Email:       "wallet-" + uuid.NewString() + "@example.com",
		Balance:     25.50,
		Status:      service.StatusActive,
		Concurrency: 1,
	})

	normalized, err := service.NormalizeBalanceDebitRequest("19.9", "CNY", "cchaven_monthly", "order-1", "order-1")
	require.NoError(t, err)
	result, err := repo.Debit(ctx, service.BalanceDebitCommand{
		UserID: user.ID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: normalized,
	})
	require.NoError(t, err)
	require.False(t, result.Replayed)
	require.Equal(t, "19.90", result.Amount)
	require.Equal(t, "5.60000000", result.BalanceAfter)
	require.NotEmpty(t, result.TxnID)

	replayed, err := repo.Debit(ctx, service.BalanceDebitCommand{
		UserID: user.ID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: normalized,
	})
	require.NoError(t, err)
	require.True(t, replayed.Replayed)
	require.Equal(t, result.TxnID, replayed.TxnID)
	require.Equal(t, result.BalanceAfter, replayed.BalanceAfter)

	conflict := normalized
	conflict.Ref = "other"
	conflict.Fingerprint = service.HashBalanceDebitFingerprint(conflict.Amount, conflict.Currency, conflict.Purpose, conflict.Ref)
	_, err = repo.Debit(ctx, service.BalanceDebitCommand{UserID: user.ID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: conflict})
	require.Error(t, err)
	require.Equal(t, "IDEMPOTENCY_KEY_CONFLICT", errors.Reason(err))

	insufficient, err := service.NormalizeBalanceDebitRequest("10", "CNY", "cchaven_monthly", "order-2", "order-2")
	require.NoError(t, err)
	_, err = repo.Debit(ctx, service.BalanceDebitCommand{UserID: user.ID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: insufficient})
	require.Error(t, err)
	require.Equal(t, "INSUFFICIENT_BALANCE", errors.Reason(err))

	_, err = integrationDB.ExecContext(ctx, `UPDATE users SET balance = 20 WHERE id = $1`, user.ID)
	require.NoError(t, err)
	retried, err := repo.Debit(ctx, service.BalanceDebitCommand{UserID: user.ID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: insufficient})
	require.NoError(t, err)
	require.False(t, retried.Replayed)

	var txCount, outboxCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM balance_debit_transactions WHERE user_id = $1`, user.ID).Scan(&txCount))
	require.Equal(t, 2, txCount)
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM balance_cache_invalidation_outbox WHERE user_id = $1`, user.ID).Scan(&outboxCount))
	require.Equal(t, 1, outboxCount)

	page, err := repo.ListTransactions(ctx, user.ID, service.BalanceTransactionFilter{
		Page: 1, PageSize: 20, Purpose: "cchaven_monthly", ClientID: client.ClientID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), page.Total)
	require.Len(t, page.Items, 2)
	require.True(t, page.Items[0].CreatedAt.After(page.Items[1].CreatedAt) || page.Items[0].CreatedAt.Equal(page.Items[1].CreatedAt))

	got, err := repo.GetTransaction(ctx, user.ID, result.TxnID)
	require.NoError(t, err)
	require.Equal(t, "wallet-test", got.ClientName)
	require.Equal(t, "order-1", got.Ref)

	other := mustCreateUser(t, testEntClient(t), &service.User{
		Email: "wallet-other-" + uuid.NewString() + "@example.com", Status: service.StatusActive, Concurrency: 1,
	})
	_, err = repo.GetTransaction(ctx, other.ID, result.TxnID)
	require.Error(t, err)
	require.Equal(t, "BALANCE_TRANSACTION_NOT_FOUND", errors.Reason(err))

	// Avoid a timestamp tie hiding an ORDER BY id regression on unusually coarse clocks.
	time.Sleep(time.Millisecond)
}

func TestBalanceWalletRepositorySerializesConcurrentDebits(t *testing.T) {
	ctx := context.Background()
	repo := NewBalanceWalletRepository(integrationDB)
	client := createWalletTestClient(t, ctx, repo)

	t.Run("different keys cannot overspend", func(t *testing.T) {
		user := mustCreateUser(t, testEntClient(t), &service.User{
			Email: "wallet-concurrent-" + uuid.NewString() + "@example.com", Balance: 15,
			Status: service.StatusActive, Concurrency: 1,
		})
		requests := make([]service.NormalizedBalanceDebitRequest, 2)
		var err error
		requests[0], err = service.NormalizeBalanceDebitRequest("10", "CNY", "cchaven_monthly", "a", "a")
		require.NoError(t, err)
		requests[1], err = service.NormalizeBalanceDebitRequest("10", "CNY", "cchaven_monthly", "b", "b")
		require.NoError(t, err)

		errs := runConcurrentDebits(ctx, repo, user.ID, client.ID, client.SecretHash, requests)
		success, insufficient := 0, 0
		for _, err := range errs {
			switch errors.Reason(err) {
			case "":
				success++
			case "INSUFFICIENT_BALANCE":
				insufficient++
			default:
				require.NoError(t, err)
			}
		}
		require.Equal(t, 1, success)
		require.Equal(t, 1, insufficient)
		var balance string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance::text FROM users WHERE id = $1`, user.ID).Scan(&balance))
		require.Equal(t, "5.00000000", balance)
	})

	t.Run("same key creates one ledger row", func(t *testing.T) {
		user := mustCreateUser(t, testEntClient(t), &service.User{
			Email: "wallet-idem-" + uuid.NewString() + "@example.com", Balance: 30,
			Status: service.StatusActive, Concurrency: 1,
		})
		request, err := service.NormalizeBalanceDebitRequest("10", "CNY", "cchaven_monthly", "same", "same")
		require.NoError(t, err)
		results := make([]*service.BalanceDebitResult, 2)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i := range results {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				results[index], errs[index] = repo.Debit(ctx, service.BalanceDebitCommand{UserID: user.ID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: request})
			}(i)
		}
		wg.Wait()
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])
		require.Equal(t, results[0].TxnID, results[1].TxnID)
		require.NotEqual(t, results[0].Replayed, results[1].Replayed)
		var count int
		require.NoError(t, integrationDB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM balance_debit_transactions WHERE user_id = $1`, user.ID).Scan(&count))
		require.Equal(t, 1, count)
	})
}

func TestBalanceCacheOutboxClaimDoesNotDeleteNewlyMergedEvent(t *testing.T) {
	ctx := context.Background()
	repo := NewBalanceWalletRepository(integrationDB)
	outbox := NewBalanceCacheInvalidationOutboxRepository(integrationDB)
	client := createWalletTestClient(t, ctx, repo)
	user := mustCreateUser(t, testEntClient(t), &service.User{
		Email: "wallet-outbox-" + uuid.NewString() + "@example.com", Balance: 30,
		Status: service.StatusActive, Concurrency: 1,
	})
	first, err := service.NormalizeBalanceDebitRequest("1", "CNY", "cchaven_monthly", "first", "first")
	require.NoError(t, err)
	_, err = repo.Debit(ctx, service.BalanceDebitCommand{UserID: user.ID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: first})
	require.NoError(t, err)
	events, err := outbox.Claim(ctx, 100, time.Minute)
	require.NoError(t, err)
	var claimed *service.BalanceCacheInvalidationEvent
	for i := range events {
		if events[i].UserID == user.ID {
			claimed = &events[i]
			break
		}
	}
	require.NotNil(t, claimed)

	second, err := service.NormalizeBalanceDebitRequest("1", "CNY", "cchaven_monthly", "second", "second")
	require.NoError(t, err)
	_, err = repo.Debit(ctx, service.BalanceDebitCommand{UserID: user.ID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: second})
	require.NoError(t, err)
	require.NoError(t, outbox.Complete(ctx, claimed.ID, claimed.ClaimToken))

	var count int
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM balance_cache_invalidation_outbox WHERE user_id = $1`, user.ID).Scan(&count))
	require.Equal(t, 1, count)
}

func TestBalanceClientRotationRejectsPreviouslyAuthenticatedOldSecret(t *testing.T) {
	ctx := context.Background()
	repo := NewBalanceWalletRepository(integrationDB)
	wallet := service.NewBalanceWalletService(repo)
	created, err := wallet.CreateClient(ctx, service.CreateBalanceClientInput{
		Name: "rotate-" + uuid.NewString(), AllowedPurposes: []string{"cchaven_monthly"},
	})
	require.NoError(t, err)
	oldSecret := created.Secret
	authenticated, err := wallet.AuthenticateClient(ctx, oldSecret)
	require.NoError(t, err)
	require.NotEmpty(t, authenticated.SecretHash)
	rotated, err := wallet.RotateClientSecret(ctx, created.ClientID)
	require.NoError(t, err)
	require.NotEqual(t, oldSecret, rotated.Secret)

	user := mustCreateUser(t, testEntClient(t), &service.User{
		Email: "wallet-rotate-" + uuid.NewString() + "@example.com", Balance: 10,
		Status: service.StatusActive, Concurrency: 1,
	})
	_, err = wallet.Debit(ctx, user.ID, authenticated, service.BalanceDebitInput{
		Amount: "1", Currency: "CNY", Purpose: "cchaven_monthly", Ref: "rotated", IdempotencyKey: "rotated",
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_BALANCE_CLIENT", errors.Reason(err))
	_, err = wallet.AuthenticateClient(ctx, oldSecret)
	require.Error(t, err)
	require.Equal(t, "INVALID_BALANCE_CLIENT", errors.Reason(err))
	_, err = wallet.AuthenticateClient(ctx, rotated.Secret)
	require.NoError(t, err)
}

func createWalletTestClient(t *testing.T, ctx context.Context, repo service.BalanceWalletRepository) *service.BalanceDebitClient {
	t.Helper()
	client := &service.BalanceDebitClient{
		ClientID: uuid.NewString(), Name: "wallet-" + uuid.NewString(), SecretHash: service.HashBalanceClientSecret(uuid.NewString()),
		SecretPrefix: "bcs_test", AllowedPurposes: []string{"cchaven_monthly"}, Status: service.BalanceClientStatusActive,
	}
	require.NoError(t, repo.CreateClient(ctx, client))
	return client
}

func runConcurrentDebits(
	ctx context.Context,
	repo service.BalanceWalletRepository,
	userID, clientID int64,
	clientSecretHash string,
	requests []service.NormalizedBalanceDebitRequest,
) []error {
	errs := make([]error, len(requests))
	var wg sync.WaitGroup
	for i := range requests {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, errs[index] = repo.Debit(ctx, service.BalanceDebitCommand{UserID: userID, ClientID: clientID, ClientSecretHash: clientSecretHash, Request: requests[index]})
		}(i)
	}
	wg.Wait()
	return errs
}
