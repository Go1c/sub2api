package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const (
	affiliateCodeLength      = 12
	affiliateCodeMaxAttempts = 12
)

var affiliateCodeCharset = []byte("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

type affiliateQueryExecer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type affiliateRepository struct {
	client *dbent.Client
}

func NewAffiliateRepository(client *dbent.Client, _ *sql.DB) service.AffiliateRepository {
	return &affiliateRepository{client: client}
}

func (r *affiliateRepository) EnsureUserAffiliate(ctx context.Context, userID int64) (*service.AffiliateSummary, error) {
	if userID <= 0 {
		return nil, service.ErrUserNotFound
	}
	client := clientFromContext(ctx, r.client)
	return ensureUserAffiliateWithClient(ctx, client, userID)
}

func (r *affiliateRepository) GetAffiliateByCode(ctx context.Context, code string) (*service.AffiliateSummary, error) {
	client := clientFromContext(ctx, r.client)
	return queryAffiliateByCode(ctx, client, code)
}

func (r *affiliateRepository) BindInviter(ctx context.Context, userID, inviterID int64) (bool, error) {
	var bound bool
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, inviterID); err != nil {
			return err
		}

		res, err := txClient.ExecContext(txCtx,
			"UPDATE user_affiliates SET inviter_id = $1, updated_at = NOW() WHERE user_id = $2 AND inviter_id IS NULL",
			inviterID, userID,
		)
		if err != nil {
			return fmt.Errorf("bind inviter: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			bound = false
			return nil
		}

		if _, err = txClient.ExecContext(txCtx,
			"UPDATE user_affiliates SET aff_count = aff_count + 1, updated_at = NOW() WHERE user_id = $1",
			inviterID,
		); err != nil {
			return fmt.Errorf("increment inviter aff_count: %w", err)
		}
		bound = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return bound, nil
}

func (r *affiliateRepository) BindInviterWithSignupBonus(ctx context.Context, req service.AffiliateSignupBonusRequest) (*service.AffiliateSignupBonusResult, error) {
	result := &service.AffiliateSignupBonusResult{}
	if req.UserID <= 0 || req.InviterID <= 0 {
		return result, nil
	}

	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, req.UserID); err != nil {
			return err
		}
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, req.InviterID); err != nil {
			return err
		}

		res, err := txClient.ExecContext(txCtx,
			"UPDATE user_affiliates SET inviter_id = $1, updated_at = NOW() WHERE user_id = $2 AND inviter_id IS NULL",
			req.InviterID, req.UserID,
		)
		if err != nil {
			return fmt.Errorf("bind inviter: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			result.Bound = false
			result.FailureReason = "already_bound"
			return insertAffiliateInviteLog(txCtx, txClient, service.AffiliateInviteLogEntry{
				InviterID:       &req.InviterID,
				InviteeID:       &req.UserID,
				AffiliateCode:   req.AffiliateCode,
				Success:         false,
				FailureReason:   result.FailureReason,
				FingerprintHash: req.FingerprintHash,
				IPAddress:       req.IPAddress,
				UserAgent:       req.UserAgent,
			})
		}

		if _, err = txClient.ExecContext(txCtx,
			"UPDATE user_affiliates SET aff_count = aff_count + 1, updated_at = NOW() WHERE user_id = $1",
			req.InviterID,
		); err != nil {
			return fmt.Errorf("increment inviter aff_count: %w", err)
		}

		result.Bound = true
		if req.Amount > 0 {
			if err := lockAffiliateSignupBonusScopes(txCtx, txClient, req); err != nil {
				return err
			}
		}
		award, reason, err := r.resolveSignupBonusAward(txCtx, txClient, req)
		if err != nil {
			return err
		}
		result.AwardedAmount = award
		result.FailureReason = reason

		if award > 0 {
			affected, err := txClient.User.Update().
				Where(user.IDEQ(req.InviterID)).
				AddBalance(award).
				Save(txCtx)
			if err != nil {
				return fmt.Errorf("credit affiliate signup bonus: %w", err)
			}
			if affected == 0 {
				return service.ErrUserNotFound
			}
			if _, err = txClient.ExecContext(txCtx, `
INSERT INTO user_affiliate_ledger (user_id, action, amount, source_user_id, created_at, updated_at)
VALUES ($1, 'signup_bonus', $2, $3, NOW(), NOW())`, req.InviterID, award, req.UserID); err != nil {
				return fmt.Errorf("insert affiliate signup bonus ledger: %w", err)
			}
		}

		return insertAffiliateInviteLog(txCtx, txClient, service.AffiliateInviteLogEntry{
			InviterID:       &req.InviterID,
			InviteeID:       &req.UserID,
			AffiliateCode:   req.AffiliateCode,
			Success:         true,
			FailureReason:   reason,
			BonusAmount:     award,
			FingerprintHash: req.FingerprintHash,
			IPAddress:       req.IPAddress,
			UserAgent:       req.UserAgent,
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func lockAffiliateSignupBonusScopes(ctx context.Context, client affiliateQueryExecer, req service.AffiliateSignupBonusRequest) error {
	for _, key := range affiliateSignupBonusLockKeys(req) {
		if _, err := client.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", key); err != nil {
			return fmt.Errorf("lock affiliate signup bonus scope: %w", err)
		}
	}
	return nil
}

func affiliateSignupBonusLockKeys(req service.AffiliateSignupBonusRequest) []string {
	keys := make([]string, 0, 3)
	if req.FingerprintHash != "" {
		keys = append(keys, "affiliate_signup_bonus:fingerprint:"+req.FingerprintHash)
	}
	if req.InviterTotalCap > 0 && req.InviterID > 0 {
		keys = append(keys, fmt.Sprintf("affiliate_signup_bonus:inviter:%d", req.InviterID))
	}
	if req.DailyTotalCap > 0 {
		keys = append(keys, "affiliate_signup_bonus:daily")
	}
	return keys
}

func (r *affiliateRepository) resolveSignupBonusAward(ctx context.Context, client *dbent.Client, req service.AffiliateSignupBonusRequest) (float64, string, error) {
	amount := req.Amount
	if amount <= 0 {
		return 0, "", nil
	}

	if req.FingerprintHash != "" {
		var reused int
		rows, err := client.QueryContext(ctx, `
SELECT COUNT(*)
FROM affiliate_invite_logs
WHERE fingerprint_hash = $1
  AND bonus_amount > 0`, req.FingerprintHash)
		if err != nil {
			return 0, "", fmt.Errorf("query affiliate signup fingerprint: %w", err)
		}
		if rows.Next() {
			if err := rows.Scan(&reused); err != nil {
				_ = rows.Close()
				return 0, "", err
			}
		}
		if err := rows.Close(); err != nil {
			return 0, "", err
		}
		if reused > 0 {
			return 0, "fingerprint_reused", nil
		}
	}

	if req.InviterTotalCap > 0 {
		existing, err := queryAffiliateLedgerSum(ctx, client, `
SELECT COALESCE(SUM(amount), 0)::double precision
FROM user_affiliate_ledger
WHERE user_id = $1 AND action = 'signup_bonus'`, req.InviterID)
		if err != nil {
			return 0, "", err
		}
		award, reason := computeAffiliateSignupBonusAward(amount, req.InviterTotalCap, existing, 0, 0)
		if award <= 0 {
			return 0, reason, nil
		}
	}

	if req.DailyTotalCap > 0 {
		existing, err := queryAffiliateLedgerSum(ctx, client, `
SELECT COALESCE(SUM(amount), 0)::double precision
FROM user_affiliate_ledger
WHERE action = 'signup_bonus'
  AND created_at >= date_trunc('day', NOW())
  AND created_at < date_trunc('day', NOW()) + interval '1 day'`)
		if err != nil {
			return 0, "", err
		}
		award, reason := computeAffiliateSignupBonusAward(amount, 0, 0, req.DailyTotalCap, existing)
		if award <= 0 {
			return 0, reason, nil
		}
	}

	if amount <= 0 {
		return 0, "cap_reached", nil
	}
	return amount, "", nil
}

func computeAffiliateSignupBonusAward(amount, inviterTotalCap, inviterAwarded, dailyTotalCap, dailyAwarded float64) (float64, string) {
	if amount <= 0 {
		return 0, ""
	}
	if inviterTotalCap > 0 && inviterAwarded+amount > inviterTotalCap {
		return 0, "inviter_total_cap_reached"
	}
	if dailyTotalCap > 0 && dailyAwarded+amount > dailyTotalCap {
		return 0, "daily_total_cap_reached"
	}
	return amount, ""
}

func queryAffiliateLedgerSum(ctx context.Context, client affiliateQueryExecer, query string, args ...any) (float64, error) {
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("query affiliate ledger sum: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var total float64
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *affiliateRepository) AccrueQuota(ctx context.Context, inviterID, inviteeUserID int64, amount float64, freezeHours int) (bool, error) {
	if amount <= 0 {
		return false, nil
	}

	var applied bool
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		// freezeHours > 0: add to frozen quota; == 0: add to available quota directly
		var updateSQL string
		if freezeHours > 0 {
			updateSQL = "UPDATE user_affiliates SET aff_frozen_quota = aff_frozen_quota + $1, aff_history_quota = aff_history_quota + $1, updated_at = NOW() WHERE user_id = $2"
		} else {
			updateSQL = "UPDATE user_affiliates SET aff_quota = aff_quota + $1, aff_history_quota = aff_history_quota + $1, updated_at = NOW() WHERE user_id = $2"
		}
		res, err := txClient.ExecContext(txCtx, updateSQL, amount, inviterID)
		if err != nil {
			return err
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			applied = false
			return nil
		}

		if freezeHours > 0 {
			if _, err = txClient.ExecContext(txCtx, `
INSERT INTO user_affiliate_ledger (user_id, action, amount, source_user_id, frozen_until, created_at, updated_at)
VALUES ($1, 'accrue', $2, $3, NOW() + make_interval(hours => $4), NOW(), NOW())`,
				inviterID, amount, inviteeUserID, freezeHours); err != nil {
				return fmt.Errorf("insert affiliate accrue ledger: %w", err)
			}
		} else {
			if _, err = txClient.ExecContext(txCtx, `
INSERT INTO user_affiliate_ledger (user_id, action, amount, source_user_id, created_at, updated_at)
VALUES ($1, 'accrue', $2, $3, NOW(), NOW())`, inviterID, amount, inviteeUserID); err != nil {
				return fmt.Errorf("insert affiliate accrue ledger: %w", err)
			}
		}

		applied = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return applied, nil
}

func (r *affiliateRepository) GetAccruedRebateFromInvitee(ctx context.Context, inviterID, inviteeUserID int64) (float64, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx,
		`SELECT COALESCE(SUM(amount), 0)::double precision FROM user_affiliate_ledger WHERE user_id = $1 AND source_user_id = $2 AND action = 'accrue'`,
		inviterID, inviteeUserID)
	if err != nil {
		return 0, fmt.Errorf("query accrued rebate from invitee: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var total float64
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, rows.Close()
}

func (r *affiliateRepository) ThawFrozenQuota(ctx context.Context, userID int64) (float64, error) {
	var thawed float64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		var err error
		thawed, err = thawFrozenQuotaTx(txCtx, txClient, userID)
		return err
	})
	return thawed, err
}

// thawFrozenQuotaTx moves matured frozen quota to available quota within an existing tx.
func thawFrozenQuotaTx(txCtx context.Context, txClient *dbent.Client, userID int64) (float64, error) {
	rows, err := txClient.QueryContext(txCtx, `
WITH matured AS (
    UPDATE user_affiliate_ledger
    SET frozen_until = NULL, updated_at = NOW()
    WHERE user_id = $1
      AND frozen_until IS NOT NULL
      AND frozen_until <= NOW()
    RETURNING amount
)
SELECT COALESCE(SUM(amount), 0) FROM matured`, userID)
	if err != nil {
		return 0, fmt.Errorf("thaw frozen quota: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var thawed float64
	if rows.Next() {
		if err := rows.Scan(&thawed); err != nil {
			return 0, err
		}
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if thawed <= 0 {
		return 0, nil
	}

	_, err = txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_quota = aff_quota + $1,
    aff_frozen_quota = GREATEST(aff_frozen_quota - $1, 0),
    updated_at = NOW()
WHERE user_id = $2`, thawed, userID)
	if err != nil {
		return 0, fmt.Errorf("move thawed quota: %w", err)
	}
	return thawed, nil
}

func (r *affiliateRepository) TransferQuotaToBalance(ctx context.Context, userID int64) (float64, float64, error) {
	var transferred float64
	var newBalance float64

	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}

		// Thaw any matured frozen quota before transfer.
		if _, err := thawFrozenQuotaTx(txCtx, txClient, userID); err != nil {
			return fmt.Errorf("thaw before transfer: %w", err)
		}

		rows, err := txClient.QueryContext(txCtx, `
WITH claimed AS (
	SELECT aff_quota::double precision AS amount
	FROM user_affiliates
	WHERE user_id = $1
	  AND aff_quota > 0
	FOR UPDATE
),
cleared AS (
	UPDATE user_affiliates ua
	SET aff_quota = 0,
	    updated_at = NOW()
	FROM claimed c
	WHERE ua.user_id = $1
	RETURNING c.amount
)
SELECT amount
FROM cleared`, userID)
		if err != nil {
			return fmt.Errorf("claim affiliate quota: %w", err)
		}

		if !rows.Next() {
			_ = rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
			return service.ErrAffiliateQuotaEmpty
		}
		if err := rows.Scan(&transferred); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if transferred <= 0 {
			return service.ErrAffiliateQuotaEmpty
		}

		affected, err := txClient.User.Update().
			Where(user.IDEQ(userID)).
			AddBalance(transferred).
			AddTotalRecharged(transferred).
			Save(txCtx)
		if err != nil {
			return fmt.Errorf("credit user balance by affiliate quota: %w", err)
		}
		if affected == 0 {
			return service.ErrUserNotFound
		}

		newBalance, err = queryUserBalance(txCtx, txClient, userID)
		if err != nil {
			return err
		}

		if _, err = txClient.ExecContext(txCtx, `
INSERT INTO user_affiliate_ledger (user_id, action, amount, source_user_id, created_at, updated_at)
VALUES ($1, 'transfer', $2, NULL, NOW(), NOW())`, userID, transferred); err != nil {
			return fmt.Errorf("insert affiliate transfer ledger: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	return transferred, newBalance, nil
}

func (r *affiliateRepository) ListInvitees(ctx context.Context, inviterID int64, limit int) ([]service.AffiliateInvitee, error) {
	if limit <= 0 {
		limit = 100
	}
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
SELECT ua.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       ua.created_at,
       COALESCE(SUM(ual.amount), 0)::double precision AS total_rebate
FROM user_affiliates ua
LEFT JOIN users u ON u.id = ua.user_id
LEFT JOIN user_affiliate_ledger ual
       ON ual.user_id = $1
      AND ual.source_user_id = ua.user_id
      AND ual.action = 'accrue'
WHERE ua.inviter_id = $1
GROUP BY ua.user_id, u.email, u.username, ua.created_at
ORDER BY ua.created_at DESC
LIMIT $2`, inviterID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	invitees := make([]service.AffiliateInvitee, 0)
	for rows.Next() {
		var item service.AffiliateInvitee
		var createdAt time.Time
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &createdAt, &item.TotalRebate); err != nil {
			return nil, err
		}
		item.CreatedAt = &createdAt
		invitees = append(invitees, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invitees, nil
}

func (r *affiliateRepository) RecordInviteLog(ctx context.Context, entry service.AffiliateInviteLogEntry) error {
	client := clientFromContext(ctx, r.client)
	return insertAffiliateInviteLog(ctx, client, entry)
}

func insertAffiliateInviteLog(ctx context.Context, client affiliateQueryExecer, entry service.AffiliateInviteLogEntry) error {
	_, err := client.ExecContext(ctx, `
INSERT INTO affiliate_invite_logs (
    inviter_id,
    invitee_id,
    affiliate_code,
    success,
    failure_reason,
    bonus_amount,
    fingerprint_hash,
    ip_address,
    user_agent,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`,
		entry.InviterID,
		entry.InviteeID,
		strings.ToUpper(strings.TrimSpace(entry.AffiliateCode)),
		entry.Success,
		strings.TrimSpace(entry.FailureReason),
		entry.BonusAmount,
		strings.TrimSpace(entry.FingerprintHash),
		strings.TrimSpace(entry.IPAddress),
		strings.TrimSpace(entry.UserAgent),
	)
	if err != nil {
		return fmt.Errorf("insert affiliate invite log: %w", err)
	}
	return nil
}

func (r *affiliateRepository) ListInviteLogs(ctx context.Context, filter service.AffiliateInviteLogFilter) ([]service.AffiliateInviteLog, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	client := clientFromContext(ctx, r.client)
	where := []string{"1=1"}
	args := make([]any, 0, 3)
	if filter.AccountID > 0 {
		args = append(args, filter.AccountID)
		where = append(where, fmt.Sprintf("(ail.inviter_id = $%d OR ail.invitee_id = $%d)", len(args), len(args)))
	}
	if filter.InviterID > 0 {
		args = append(args, filter.InviterID)
		where = append(where, fmt.Sprintf("ail.inviter_id = $%d", len(args)))
	}
	if filter.InviteeID > 0 {
		args = append(args, filter.InviteeID)
		where = append(where, fmt.Sprintf("ail.invitee_id = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	countQuery := "SELECT COUNT(*) FROM affiliate_invite_logs ail WHERE " + whereSQL
	rows, err := client.QueryContext(ctx, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count affiliate invite logs: %w", err)
	}
	var total int64
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			_ = rows.Close()
			return nil, 0, err
		}
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	query := fmt.Sprintf(`
SELECT ail.id,
       ail.inviter_id,
       COALESCE(inviter.email, ''),
       COALESCE(inviter.username, ''),
       ail.invitee_id,
       COALESCE(invitee.email, ''),
       COALESCE(invitee.username, ''),
       ail.affiliate_code,
       ail.success,
       ail.failure_reason,
       ail.bonus_amount::double precision,
       ail.fingerprint_hash,
       ail.ip_address,
       ail.user_agent,
       ail.created_at
FROM affiliate_invite_logs ail
LEFT JOIN users inviter ON inviter.id = ail.inviter_id
LEFT JOIN users invitee ON invitee.id = ail.invitee_id
WHERE %s
ORDER BY ail.created_at DESC, ail.id DESC
LIMIT $%d OFFSET $%d`, whereSQL, len(args)-1, len(args))

	rows, err = client.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list affiliate invite logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateInviteLog, 0)
	for rows.Next() {
		var item service.AffiliateInviteLog
		var inviterID, inviteeID sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&inviterID,
			&item.InviterEmail,
			&item.InviterUsername,
			&inviteeID,
			&item.InviteeEmail,
			&item.InviteeUsername,
			&item.AffiliateCode,
			&item.Success,
			&item.FailureReason,
			&item.BonusAmount,
			&item.FingerprintHash,
			&item.IPAddress,
			&item.UserAgent,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if inviterID.Valid {
			id := inviterID.Int64
			item.InviterID = &id
		}
		if inviteeID.Valid {
			id := inviteeID.Int64
			item.InviteeID = &id
		}
		if !filter.IncludeSensitive {
			item.FingerprintHash = ""
			item.IPAddress = ""
			item.UserAgent = ""
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *affiliateRepository) withTx(ctx context.Context, fn func(txCtx context.Context, txClient *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin affiliate transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit affiliate transaction: %w", err)
	}
	return nil
}

func ensureUserAffiliateWithClient(ctx context.Context, client affiliateQueryExecer, userID int64) (*service.AffiliateSummary, error) {
	summary, err := queryAffiliateByUserID(ctx, client, userID)
	if err == nil {
		return summary, nil
	}
	if !errors.Is(err, service.ErrAffiliateProfileNotFound) {
		return nil, err
	}

	for i := 0; i < affiliateCodeMaxAttempts; i++ {
		code, codeErr := generateAffiliateCode()
		if codeErr != nil {
			return nil, codeErr
		}
		_, insertErr := client.ExecContext(ctx, `
INSERT INTO user_affiliates (user_id, aff_code, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING`, userID, code)
		if insertErr == nil {
			break
		}
		if isAffiliateUniqueViolation(insertErr) {
			continue
		}
		return nil, insertErr
	}

	return queryAffiliateByUserID(ctx, client, userID)
}

func queryAffiliateByUserID(ctx context.Context, client affiliateQueryExecer, userID int64) (*service.AffiliateSummary, error) {
	rows, err := client.QueryContext(ctx, `
SELECT user_id,
       aff_code,
       aff_code_custom,
       aff_rebate_rate_percent,
       inviter_id,
       aff_count,
       aff_quota::double precision,
       aff_frozen_quota::double precision,
       aff_history_quota::double precision,
       created_at,
       updated_at
FROM user_affiliates
WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateProfileNotFound
	}

	var out service.AffiliateSummary
	var inviterID sql.NullInt64
	var rebateRate sql.NullFloat64
	if err := rows.Scan(
		&out.UserID,
		&out.AffCode,
		&out.AffCodeCustom,
		&rebateRate,
		&inviterID,
		&out.AffCount,
		&out.AffQuota,
		&out.AffFrozenQuota,
		&out.AffHistoryQuota,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if inviterID.Valid {
		out.InviterID = &inviterID.Int64
	}
	if rebateRate.Valid {
		v := rebateRate.Float64
		out.AffRebateRatePercent = &v
	}
	return &out, nil
}

func queryAffiliateByCode(ctx context.Context, client affiliateQueryExecer, code string) (*service.AffiliateSummary, error) {
	rows, err := client.QueryContext(ctx, `
SELECT user_id,
       aff_code,
       aff_code_custom,
       aff_rebate_rate_percent,
       inviter_id,
       aff_count,
       aff_quota::double precision,
       aff_frozen_quota::double precision,
       aff_history_quota::double precision,
       created_at,
       updated_at
FROM user_affiliates
WHERE aff_code = $1
LIMIT 1`, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateProfileNotFound
	}

	var out service.AffiliateSummary
	var inviterID sql.NullInt64
	var rebateRate sql.NullFloat64
	if err := rows.Scan(
		&out.UserID,
		&out.AffCode,
		&out.AffCodeCustom,
		&rebateRate,
		&inviterID,
		&out.AffCount,
		&out.AffQuota,
		&out.AffFrozenQuota,
		&out.AffHistoryQuota,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if inviterID.Valid {
		out.InviterID = &inviterID.Int64
	}
	if rebateRate.Valid {
		v := rebateRate.Float64
		out.AffRebateRatePercent = &v
	}
	return &out, nil
}

func queryUserBalance(ctx context.Context, client affiliateQueryExecer, userID int64) (float64, error) {
	rows, err := client.QueryContext(ctx,
		"SELECT balance::double precision FROM users WHERE id = $1 LIMIT 1",
		userID,
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, service.ErrUserNotFound
	}
	var balance float64
	if err := rows.Scan(&balance); err != nil {
		return 0, err
	}
	return balance, nil
}

func generateAffiliateCode() (string, error) {
	buf := make([]byte, affiliateCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate affiliate code: %w", err)
	}
	for i := range buf {
		buf[i] = affiliateCodeCharset[int(buf[i])%len(affiliateCodeCharset)]
	}
	return string(buf), nil
}

func isAffiliateUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == "23505"
	}
	return false
}

// UpdateUserAffCode 改写用户的邀请码（自定义专属邀请码）。
// 唯一性冲突返回 ErrAffiliateCodeTaken。
func (r *affiliateRepository) UpdateUserAffCode(ctx context.Context, userID int64, newCode string) error {
	if userID <= 0 {
		return service.ErrUserNotFound
	}
	code := strings.ToUpper(strings.TrimSpace(newCode))
	if code == "" {
		return service.ErrAffiliateCodeInvalid
	}

	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		res, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_code = $1,
    aff_code_custom = true,
    updated_at = NOW()
WHERE user_id = $2`, code, userID)
		if err != nil {
			if isAffiliateUniqueViolation(err) {
				return service.ErrAffiliateCodeTaken
			}
			return fmt.Errorf("update aff_code: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return service.ErrUserNotFound
		}
		return nil
	})
}

// ResetUserAffCode 把 aff_code 还原为系统随机码，并清除 aff_code_custom 标记。
func (r *affiliateRepository) ResetUserAffCode(ctx context.Context, userID int64) (string, error) {
	if userID <= 0 {
		return "", service.ErrUserNotFound
	}
	var newCode string
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		for i := 0; i < affiliateCodeMaxAttempts; i++ {
			candidate, codeErr := generateAffiliateCode()
			if codeErr != nil {
				return codeErr
			}
			res, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_code = $1,
    aff_code_custom = false,
    updated_at = NOW()
WHERE user_id = $2`, candidate, userID)
			if err != nil {
				if isAffiliateUniqueViolation(err) {
					continue
				}
				return fmt.Errorf("reset aff_code: %w", err)
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return service.ErrUserNotFound
			}
			newCode = candidate
			return nil
		}
		return fmt.Errorf("reset aff_code: exhausted attempts")
	})
	if err != nil {
		return "", err
	}
	return newCode, nil
}

// SetUserRebateRate 设置或清除用户专属返利比例。ratePercent==nil 表示清除（沿用全局）。
func (r *affiliateRepository) SetUserRebateRate(ctx context.Context, userID int64, ratePercent *float64) error {
	if userID <= 0 {
		return service.ErrUserNotFound
	}
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		// nullableArg lets us use a single UPDATE for both "set value" and
		// "clear" cases — database/sql converts nil interface{} to SQL NULL.
		res, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_rebate_rate_percent = $1,
    updated_at = NOW()
WHERE user_id = $2`, nullableArg(ratePercent), userID)
		if err != nil {
			return fmt.Errorf("set aff_rebate_rate_percent: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return service.ErrUserNotFound
		}
		return nil
	})
}

// BatchSetUserRebateRate 批量为多个用户设置专属比例（nil 清除）。
func (r *affiliateRepository) BatchSetUserRebateRate(ctx context.Context, userIDs []int64, ratePercent *float64) error {
	if len(userIDs) == 0 {
		return nil
	}
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		for _, uid := range userIDs {
			if uid <= 0 {
				continue
			}
			if _, err := ensureUserAffiliateWithClient(txCtx, txClient, uid); err != nil {
				return err
			}
		}
		_, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_rebate_rate_percent = $1,
    updated_at = NOW()
WHERE user_id = ANY($2)`, nullableArg(ratePercent), pq.Array(userIDs))
		if err != nil {
			return fmt.Errorf("batch set aff_rebate_rate_percent: %w", err)
		}
		return nil
	})
}

// nullableArg unwraps a *float64 into an interface{} suitable for SQL parameter
// binding: nil pointer → SQL NULL, non-nil → the float value.
func nullableArg(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

// ListUsersWithCustomSettings 列出有专属配置（自定义码或专属比例）的用户。
//
// 单一查询同时处理"无搜索"与"按邮箱/用户名模糊搜索"：
// 空 search 时拼接出的 LIKE 模式为 "%%"，匹配所有行；非空时按 ILIKE 子串匹配。
// 这避免了为两种情况维护两份 SQL 模板。
func (r *affiliateRepository) ListUsersWithCustomSettings(ctx context.Context, filter service.AffiliateAdminFilter) ([]service.AffiliateAdminEntry, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	likePattern := "%" + strings.TrimSpace(filter.Search) + "%"

	const baseFrom = `
FROM user_affiliates ua
JOIN users u ON u.id = ua.user_id
WHERE (ua.aff_code_custom = true OR ua.aff_rebate_rate_percent IS NOT NULL)
  AND (u.email ILIKE $1 OR u.username ILIKE $1)`

	client := clientFromContext(ctx, r.client)

	total, err := scanInt64(ctx, client, "SELECT COUNT(*)"+baseFrom, likePattern)
	if err != nil {
		return nil, 0, fmt.Errorf("count affiliate admin entries: %w", err)
	}

	listQuery := `
SELECT ua.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       ua.aff_code,
       ua.aff_code_custom,
       ua.aff_rebate_rate_percent,
       ua.aff_count` + baseFrom + `
ORDER BY ua.updated_at DESC
LIMIT $2 OFFSET $3`

	rows, err := client.QueryContext(ctx, listQuery, likePattern, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list affiliate admin entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	entries := make([]service.AffiliateAdminEntry, 0)
	for rows.Next() {
		var e service.AffiliateAdminEntry
		var rebate sql.NullFloat64
		if err := rows.Scan(&e.UserID, &e.Email, &e.Username, &e.AffCode,
			&e.AffCodeCustom, &rebate, &e.AffCount); err != nil {
			return nil, 0, err
		}
		if rebate.Valid {
			v := rebate.Float64
			e.AffRebateRatePercent = &v
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

// scanInt64 runs a query expected to return a single int64 column (e.g. COUNT).
func scanInt64(ctx context.Context, client affiliateQueryExecer, query string, args ...any) (int64, error) {
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, nil
	}
	var v int64
	if err := rows.Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}
