package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

const (
	balanceClientSecretPrefix       = "bcs_"
	balanceClientSecretBytes        = 32
	balanceClientDisplaySecretChars = 8
)

var balanceClientSecretPattern = regexp.MustCompile(`^bcs_[0-9a-f]{64}$`)

type BalanceWalletService struct {
	repo BalanceWalletRepository
}

func NewBalanceWalletService(repo BalanceWalletRepository) *BalanceWalletService {
	return &BalanceWalletService{repo: repo}
}

func (s *BalanceWalletService) AuthenticateClient(ctx context.Context, rawSecret string) (*BalanceDebitClient, error) {
	secret := strings.TrimSpace(rawSecret)
	if !balanceClientSecretPattern.MatchString(secret) || s == nil || s.repo == nil {
		return nil, ErrInvalidBalanceClient
	}
	client, err := s.repo.GetActiveClientBySecretHash(ctx, HashBalanceClientSecret(secret))
	if err != nil {
		if errors.Is(err, ErrBalanceClientNotFound) || infraerrors.IsNotFound(err) {
			return nil, ErrInvalidBalanceClient
		}
		return nil, ErrBalanceStoreUnavailable.WithCause(err)
	}
	return client, nil
}

func (s *BalanceWalletService) Debit(ctx context.Context, userID int64, client *BalanceDebitClient, input BalanceDebitInput) (*BalanceDebitResult, error) {
	if userID <= 0 || s == nil || s.repo == nil {
		return nil, ErrInvalidBalanceDebitRequest
	}
	if client == nil || client.ID <= 0 || client.SecretHash == "" {
		return nil, ErrInvalidBalanceClient
	}
	normalized, err := NormalizeBalanceDebitRequest(input.Amount, input.Currency, input.Purpose, input.Ref, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	return s.repo.Debit(ctx, BalanceDebitCommand{
		UserID: userID, ClientID: client.ID, ClientSecretHash: client.SecretHash, Request: normalized,
	})
}

func (s *BalanceWalletService) ListTransactions(ctx context.Context, userID int64, filter BalanceTransactionFilter) (*BalanceTransactionPage, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	filter.Purpose = strings.TrimSpace(filter.Purpose)
	filter.Ref = strings.TrimSpace(filter.Ref)
	filter.ClientID = strings.TrimSpace(filter.ClientID)
	if filter.ClientID != "" {
		if _, err := uuid.Parse(filter.ClientID); err != nil {
			return nil, ErrBalanceTransactionQuery
		}
	}
	return s.repo.ListTransactions(ctx, userID, filter)
}

func (s *BalanceWalletService) GetTransaction(ctx context.Context, userID int64, txnID string) (*BalanceDebitTransaction, error) {
	if _, err := uuid.Parse(strings.TrimSpace(txnID)); err != nil {
		return nil, ErrBalanceTransactionNotFound
	}
	return s.repo.GetTransaction(ctx, userID, strings.TrimSpace(txnID))
}

func (s *BalanceWalletService) CreateClient(ctx context.Context, input CreateBalanceClientInput) (*BalanceDebitClient, error) {
	name, purposes, err := normalizeBalanceClientInput(input.Name, input.AllowedPurposes)
	if err != nil {
		return nil, err
	}
	secret, prefix, hash, err := generateBalanceClientSecret()
	if err != nil {
		return nil, ErrBalanceStoreUnavailable.WithCause(err)
	}
	client := &BalanceDebitClient{
		ClientID: uuid.NewString(), Name: name, SecretHash: hash,
		SecretPrefix: prefix, AllowedPurposes: purposes, Status: BalanceClientStatusActive,
	}
	if err := s.repo.CreateClient(ctx, client); err != nil {
		return nil, err
	}
	client.Secret = secret
	client.SecretHash = ""
	return client, nil
}

func (s *BalanceWalletService) ListClients(ctx context.Context) ([]BalanceDebitClient, error) {
	return s.repo.ListClients(ctx)
}

func (s *BalanceWalletService) GetClient(ctx context.Context, clientID string) (*BalanceDebitClient, error) {
	if _, err := uuid.Parse(strings.TrimSpace(clientID)); err != nil {
		return nil, ErrBalanceClientNotFound
	}
	return s.repo.GetClient(ctx, strings.TrimSpace(clientID))
}

func (s *BalanceWalletService) UpdateClient(ctx context.Context, clientID string, input UpdateBalanceClientInput) (*BalanceDebitClient, error) {
	if _, err := uuid.Parse(strings.TrimSpace(clientID)); err != nil {
		return nil, ErrBalanceClientNotFound
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return nil, ErrBalanceClientInvalid
		}
		input.Name = &name
	}
	if input.AllowedPurposes != nil {
		purposes, err := normalizeAllowedBalancePurposes(*input.AllowedPurposes)
		if err != nil {
			return nil, err
		}
		input.AllowedPurposes = &purposes
	}
	if input.Status != nil && *input.Status != BalanceClientStatusActive && *input.Status != BalanceClientStatusInactive {
		return nil, ErrBalanceClientInvalid
	}
	return s.repo.UpdateClient(ctx, strings.TrimSpace(clientID), input)
}

func (s *BalanceWalletService) RotateClientSecret(ctx context.Context, clientID string) (*BalanceDebitClient, error) {
	if _, err := uuid.Parse(strings.TrimSpace(clientID)); err != nil {
		return nil, ErrBalanceClientNotFound
	}
	secret, prefix, hash, err := generateBalanceClientSecret()
	if err != nil {
		return nil, ErrBalanceStoreUnavailable.WithCause(err)
	}
	client, err := s.repo.RotateClientSecret(ctx, strings.TrimSpace(clientID), hash, prefix)
	if err != nil {
		return nil, err
	}
	client.Secret = secret
	client.SecretHash = ""
	return client, nil
}

func (s *BalanceWalletService) DeactivateClient(ctx context.Context, clientID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(clientID)); err != nil {
		return ErrBalanceClientNotFound
	}
	return s.repo.DeactivateClient(ctx, strings.TrimSpace(clientID))
}

func HashBalanceClientSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func generateBalanceClientSecret() (secret, prefix, hash string, err error) {
	random := make([]byte, balanceClientSecretBytes)
	if _, err = rand.Read(random); err != nil {
		return "", "", "", err
	}
	hexSecret := hex.EncodeToString(random)
	secret = balanceClientSecretPrefix + hexSecret
	prefix = balanceClientSecretPrefix + hexSecret[:balanceClientDisplaySecretChars]
	return secret, prefix, HashBalanceClientSecret(secret), nil
}

func normalizeBalanceClientInput(name string, purposes []string) (string, []string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return "", nil, ErrBalanceClientInvalid
	}
	normalized, err := normalizeAllowedBalancePurposes(purposes)
	return name, normalized, err
}

func normalizeAllowedBalancePurposes(purposes []string) ([]string, error) {
	if len(purposes) == 0 {
		return nil, ErrBalanceClientInvalid
	}
	seen := make(map[string]struct{}, len(purposes))
	result := make([]string, 0, len(purposes))
	for _, purpose := range purposes {
		purpose = strings.TrimSpace(purpose)
		if !IsValidBalancePurpose(purpose) {
			return nil, ErrBalanceClientInvalid
		}
		if _, exists := seen[purpose]; exists {
			continue
		}
		seen[purpose] = struct{}{}
		result = append(result, purpose)
	}
	if len(result) == 0 {
		return nil, ErrBalanceClientInvalid
	}
	sort.Strings(result)
	return result, nil
}
