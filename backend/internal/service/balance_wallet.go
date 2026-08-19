package service

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const BalanceCurrencyCNY = "CNY"

var (
	ErrInvalidBalanceDebitRequest = infraerrors.BadRequest("INVALID_BALANCE_DEBIT_REQUEST", "invalid balance debit request")
	ErrInvalidBalanceClient       = infraerrors.Unauthorized("INVALID_BALANCE_CLIENT", "invalid balance client")
	ErrBalanceUserInactive        = infraerrors.Forbidden("USER_INACTIVE", "user account is not active")
	ErrBalancePurposeNotAllowed   = infraerrors.Forbidden("PURPOSE_NOT_ALLOWED", "purpose is not allowed for this balance client")
	ErrBalanceInsufficient        = infraerrors.Forbidden("INSUFFICIENT_BALANCE", "insufficient balance")
	ErrBalanceDebitBusy           = infraerrors.Conflict("BALANCE_DEBIT_BUSY", "balance debit is busy").WithMetadata(map[string]string{"retry_after": "1"})
	ErrBalanceStoreUnavailable    = infraerrors.ServiceUnavailable("BALANCE_STORE_UNAVAILABLE", "balance store is unavailable")
	ErrBalanceTransactionQuery    = infraerrors.BadRequest("INVALID_BALANCE_TRANSACTION_QUERY", "invalid balance transaction query")
	ErrBalanceTransactionNotFound = infraerrors.NotFound("BALANCE_TRANSACTION_NOT_FOUND", "balance transaction not found")
	ErrBalanceClientNotFound      = infraerrors.NotFound("BALANCE_CLIENT_NOT_FOUND", "balance client not found")
	ErrBalanceClientInvalid       = infraerrors.BadRequest("INVALID_BALANCE_CLIENT_REQUEST", "invalid balance client request")
	ErrBalanceClientConflict      = infraerrors.Conflict("BALANCE_CLIENT_CONFLICT", "balance client already exists")
)

var (
	balanceAmountPattern  = regexp.MustCompile(`^(?:0|[1-9][0-9]*)(?:\.[0-9]{1,2})?$`)
	balancePurposePattern = regexp.MustCompile(`^[a-z0-9]+(?:[_-][a-z0-9]+)*$`)
	balanceAmountMax      = decimal.RequireFromString("999999999999999999.99")
)

// NormalizedBalanceDebitRequest is the canonical, validated debit command.
// Monetary values remain decimal strings until PostgreSQL numeric arithmetic.
type NormalizedBalanceDebitRequest struct {
	Amount             string
	Currency           string
	Purpose            string
	Ref                string
	IdempotencyKeyHash string
	Fingerprint        string
}

// NormalizeBalanceDebitRequest validates the frozen wallet contract and builds
// a canonical fingerprint. It deliberately never converts money through float64.
func NormalizeBalanceDebitRequest(amountRaw, currency, purpose, ref, idempotencyKey string) (NormalizedBalanceDebitRequest, error) {
	key, err := NormalizeIdempotencyKey(idempotencyKey)
	if err != nil {
		return NormalizedBalanceDebitRequest{}, err
	}
	if key == "" {
		return NormalizedBalanceDebitRequest{}, ErrIdempotencyKeyRequired
	}

	amountRaw = strings.TrimSpace(amountRaw)
	amount, err := decimal.NewFromString(amountRaw)
	if err != nil || !balanceAmountPattern.MatchString(amountRaw) || !amount.IsPositive() || amount.GreaterThan(balanceAmountMax) {
		return NormalizedBalanceDebitRequest{}, ErrInvalidBalanceDebitRequest
	}
	if currency != BalanceCurrencyCNY || !IsValidBalancePurpose(purpose) {
		return NormalizedBalanceDebitRequest{}, ErrInvalidBalanceDebitRequest
	}

	ref = strings.TrimSpace(ref)
	if !isValidBalanceRef(ref) {
		return NormalizedBalanceDebitRequest{}, ErrInvalidBalanceDebitRequest
	}

	canonicalAmount := amount.StringFixed(2)
	return NormalizedBalanceDebitRequest{
		Amount:             canonicalAmount,
		Currency:           currency,
		Purpose:            purpose,
		Ref:                ref,
		IdempotencyKeyHash: HashIdempotencyKey(key),
		Fingerprint:        HashBalanceDebitFingerprint(canonicalAmount, currency, purpose, ref),
	}, nil
}

// HashBalanceDebitFingerprint hashes the canonical request fields stored with the ledger row.
func HashBalanceDebitFingerprint(amount, currency, purpose, ref string) string {
	fingerprintInput := amount + "\n" + currency + "\n" + purpose + "\n" + ref
	fingerprintSum := sha256.Sum256([]byte(fingerprintInput))
	return hex.EncodeToString(fingerprintSum[:])
}

// IsValidBalancePurpose reports whether purpose is a 1-64 character lowercase slug.
func IsValidBalancePurpose(purpose string) bool {
	return len(purpose) >= 1 && len(purpose) <= 64 && balancePurposePattern.MatchString(purpose)
}

func isValidBalanceRef(ref string) bool {
	if ref == "" || !utf8.ValidString(ref) || utf8.RuneCountInString(ref) > 128 {
		return false
	}
	for _, r := range ref {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
