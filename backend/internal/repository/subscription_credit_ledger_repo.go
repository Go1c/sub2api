package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// subscriptionCreditLedgerRepository 实现 service.SubscriptionCreditLedgerRepository。
//
// 写入路径使用 service.SQLExecer，允许 caller 在已有事务内调用（参考 enqueueSchedulerOutbox）。
// 查询路径直接走 *sql.DB（不需要事务隔离）。
type subscriptionCreditLedgerRepository struct {
	db *sql.DB
}

// NewSubscriptionCreditLedgerRepository 构造订阅额度流水仓储。
// sqlDB 必须非 nil；查询/写入均直接走 SQL。
func NewSubscriptionCreditLedgerRepository(sqlDB *sql.DB) service.SubscriptionCreditLedgerRepository {
	return &subscriptionCreditLedgerRepository{db: sqlDB}
}

const subscriptionCreditLedgerInsertSQL = `
INSERT INTO subscription_credit_ledger (
    user_id, subscription_id, group_id, api_key_id, usage_log_id, order_id,
    type, delta_usd, balance_delta_usd, remaining_after_usd, reason, event_key, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
`

// Create 写入一条流水。可在已有事务内调用。
func (r *subscriptionCreditLedgerRepository) Create(ctx context.Context, exec service.SQLExecer, entry *service.SubscriptionCreditLedgerEntry) error {
	if entry == nil {
		return errors.New("nil ledger entry")
	}
	if exec == nil {
		exec = r.db
	}
	metaJSON, err := encodeLedgerMetadata(entry.Metadata)
	if err != nil {
		return fmt.Errorf("encode ledger metadata: %w", err)
	}
	eventKey := normalizeEventKey(entry.EventKey)
	_, err = exec.ExecContext(ctx, subscriptionCreditLedgerInsertSQL,
		entry.UserID,
		entry.SubscriptionID,
		entry.GroupID,
		entry.APIKeyID,
		entry.UsageLogID,
		entry.OrderID,
		entry.Type,
		entry.DeltaUSD,
		entry.BalanceDeltaUSD,
		entry.RemainingAfterUSD,
		entry.Reason,
		eventKey,
		metaJSON,
	)
	return err
}

// CreateLimitReachedEvent 写入幂等事件。
//
// 通过 ON CONFLICT (subscription_id, type, event_key) DO NOTHING 实现：
//   - 首次写入返回 created=true
//   - event_key 已存在返回 created=false（说明事件已被记录）
//
// 必须保证 entry.EventKey != nil 且非空字符串，否则视为参数错误。
func (r *subscriptionCreditLedgerRepository) CreateLimitReachedEvent(ctx context.Context, exec service.SQLExecer, entry *service.SubscriptionCreditLedgerEntry) (bool, error) {
	if entry == nil {
		return false, errors.New("nil ledger entry")
	}
	if entry.EventKey == nil || strings.TrimSpace(*entry.EventKey) == "" {
		return false, errors.New("event_key is required for idempotent event")
	}
	if exec == nil {
		exec = r.db
	}
	metaJSON, err := encodeLedgerMetadata(entry.Metadata)
	if err != nil {
		return false, fmt.Errorf("encode ledger metadata: %w", err)
	}
	// 注意：唯一索引是 (subscription_id, type, event_key) WHERE event_key IS NOT NULL AND event_key <> ''
	// 因此 ON CONFLICT 必须明确列出这三列以匹配该部分索引。
	const q = `
INSERT INTO subscription_credit_ledger (
    user_id, subscription_id, group_id, api_key_id, usage_log_id, order_id,
    type, delta_usd, balance_delta_usd, remaining_after_usd, reason, event_key, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (subscription_id, type, event_key)
WHERE event_key IS NOT NULL AND event_key <> ''
DO NOTHING
`
	res, err := exec.ExecContext(ctx, q,
		entry.UserID,
		entry.SubscriptionID,
		entry.GroupID,
		entry.APIKeyID,
		entry.UsageLogID,
		entry.OrderID,
		entry.Type,
		entry.DeltaUSD,
		entry.BalanceDeltaUSD,
		entry.RemainingAfterUSD,
		entry.Reason,
		*entry.EventKey,
		metaJSON,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// ListByUserID 按用户列出 ledger，按 created_at DESC 分页。
func (r *subscriptionCreditLedgerRepository) ListByUserID(ctx context.Context, userID int64, ledgerType string, params pagination.PaginationParams) ([]service.SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error) {
	whereClause := "user_id = $1"
	args := []any{userID}
	if strings.TrimSpace(ledgerType) != "" {
		whereClause += " AND type = $2"
		args = append(args, strings.TrimSpace(ledgerType))
	}
	return r.list(ctx, whereClause, args, params)
}

// ListBySubscriptionID 按订阅列出 ledger，按 created_at DESC 分页。
func (r *subscriptionCreditLedgerRepository) ListBySubscriptionID(ctx context.Context, subscriptionID int64, ledgerType string, params pagination.PaginationParams) ([]service.SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error) {
	whereClause := "subscription_id = $1"
	args := []any{subscriptionID}
	if strings.TrimSpace(ledgerType) != "" {
		whereClause += " AND type = $2"
		args = append(args, strings.TrimSpace(ledgerType))
	}
	return r.list(ctx, whereClause, args, params)
}

func (r *subscriptionCreditLedgerRepository) list(ctx context.Context, whereClause string, whereArgs []any, params pagination.PaginationParams) ([]service.SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	// 总数
	var total int64
	countSQL := "SELECT COUNT(*) FROM subscription_credit_ledger WHERE " + whereClause
	if err := r.db.QueryRowContext(ctx, countSQL, whereArgs...).Scan(&total); err != nil {
		return nil, nil, err
	}

	// 列表
	limit := pageSize
	offset := (page - 1) * pageSize
	listSQL := fmt.Sprintf(`
SELECT id, user_id, subscription_id, group_id, api_key_id, usage_log_id, order_id,
       type, delta_usd, balance_delta_usd, remaining_after_usd, reason, event_key,
       metadata, created_at
FROM subscription_credit_ledger
WHERE %s
ORDER BY created_at DESC, id DESC
LIMIT $%d OFFSET $%d
`, whereClause, len(whereArgs)+1, len(whereArgs)+2)
	args := append(append([]any{}, whereArgs...), limit, offset)
	rows, err := r.db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	entries := make([]service.SubscriptionCreditLedgerEntry, 0, pageSize)
	for rows.Next() {
		var (
			e           service.SubscriptionCreditLedgerEntry
			groupID     sql.NullInt64
			apiKeyID    sql.NullInt64
			usageLogID  sql.NullInt64
			orderID     sql.NullInt64
			eventKey    sql.NullString
			metadataRaw []byte
		)
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.SubscriptionID,
			&groupID, &apiKeyID, &usageLogID, &orderID,
			&e.Type, &e.DeltaUSD, &e.BalanceDeltaUSD, &e.RemainingAfterUSD,
			&e.Reason, &eventKey, &metadataRaw, &e.CreatedAt,
		); err != nil {
			return nil, nil, err
		}
		if groupID.Valid {
			v := groupID.Int64
			e.GroupID = &v
		}
		if apiKeyID.Valid {
			v := apiKeyID.Int64
			e.APIKeyID = &v
		}
		if usageLogID.Valid {
			v := usageLogID.Int64
			e.UsageLogID = &v
		}
		if orderID.Valid {
			v := orderID.Int64
			e.OrderID = &v
		}
		if eventKey.Valid && eventKey.String != "" {
			s := eventKey.String
			e.EventKey = &s
		}
		if len(metadataRaw) > 0 {
			var m map[string]any
			if err := json.Unmarshal(metadataRaw, &m); err != nil {
				return nil, nil, fmt.Errorf("decode ledger metadata: %w", err)
			}
			e.Metadata = m
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	if totalPages < 1 {
		totalPages = 1
	}
	result := &pagination.PaginationResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    int(totalPages),
	}
	return entries, result, nil
}

// encodeLedgerMetadata 把 map metadata 编码为 JSONB。nil → "{}"::jsonb。
func encodeLedgerMetadata(m map[string]any) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// normalizeEventKey 处理 event_key 的 NULL 语义。
// EventKey 为 nil 或空字符串均存为 NULL。
func normalizeEventKey(p *string) any {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return s
}

// 编译期断言：subscriptionCreditLedgerRepository 实现接口
var _ service.SubscriptionCreditLedgerRepository = (*subscriptionCreditLedgerRepository)(nil)
var _ time.Time // 防止 time import 被 lint 删（list 函数会 Scan time）
