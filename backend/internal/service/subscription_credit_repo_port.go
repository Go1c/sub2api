package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// RenewalReason 描述用户是否允许购买新订阅的原因。
type RenewalReason string

const (
	// RenewalReasonNoSubscription 用户没有任何 active 订阅 → 允许购买。
	RenewalReasonNoSubscription RenewalReason = "no_subscription"
	// RenewalReasonExhausted 当前订阅总额度已耗尽（exhausted_at != nil）→ 允许购买。
	RenewalReasonExhausted RenewalReason = "exhausted"
	// RenewalReasonExpired 当前订阅已过期（expires_at <= now）→ 允许购买。
	RenewalReasonExpired RenewalReason = "expired"
	// RenewalReasonNotExhausted 当前订阅仍可使用 → 拒绝购买（对应 SUBSCRIPTION_RENEWAL_NOT_ALLOWED）。
	RenewalReasonNotExhausted RenewalReason = "not_exhausted"
)

// RenewalEligibility 描述用户当前是否可以购买新订阅。
// Handler 层基于 Reason 映射成对应的 API 错误码与 HTTP 响应。
type RenewalEligibility struct {
	Allowed      bool
	Reason       RenewalReason
	Subscription *UserSubscription // 当前 active 订阅；nil 表示用户无任何 active 订阅
}

// SQLTxBeginner 提供 SQL 事务能力。在订阅履约等流程中由 service 层显式控制事务边界。
type SQLTxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// SubscriptionCreditExtension 是 UserSubscriptionRepository 的额度池扩展接口。
//
// 拆为独立接口的原因：
//   - 现有 UserSubscriptionRepository 是 legacy 订阅接口，向后兼容
//   - 新增方法集中在订阅额度池语义（GetUsableCreditSubscription 等）
//   - service 层可以单独 mock 这部分能力
type SubscriptionCreditExtension interface {
	// GetUsableCreditSubscription 返回用户当前的"可消费"订阅。
	//
	//   可消费 = status='active' AND exhausted_at IS NULL
	//          AND expires_at > now AND deleted_at IS NULL
	//
	// 通过部分唯一索引 user_subscriptions_user_active_usable 保证最多 1 条。
	// 不存在时返回 (nil, ErrSubscriptionNotFound)。
	GetUsableCreditSubscription(ctx context.Context, userID int64) (*UserSubscription, error)

	// HasUsableCreditSubscription 用户是否当前有可消费订阅。
	// 等价于 GetUsableCreditSubscription 返回非 nil；提供单独方法避免不必要的字段读取。
	HasUsableCreditSubscription(ctx context.Context, userID int64) (bool, error)

	// GetRenewalEligibility 用户是否允许购买新订阅。
	//   - 无 active 订阅 → Allowed=true, Reason=no_subscription
	//   - 当前订阅 exhausted_at != nil → Allowed=true, Reason=exhausted
	//   - 当前订阅 expires_at <= now → Allowed=true, Reason=expired
	//   - 当前订阅仍可使用 → Allowed=false, Reason=not_exhausted
	GetRenewalEligibility(ctx context.Context, userID int64) (RenewalEligibility, error)

	// LockUserForSubscriptionWrite 在事务内对 users 行加 FOR UPDATE 行锁。
	// 用于订阅履约 / 余额扣费等关键路径，防止并发购买写入两条订阅。
	LockUserForSubscriptionWrite(ctx context.Context, tx *sql.Tx, userID int64) error

	// InsertCreditSubscription 在调用方事务内插入一条额度池订阅。
	// 仅用于购买履约路径；调用方负责先锁用户行并做二次可消费订阅校验。
	InsertCreditSubscription(ctx context.Context, tx *sql.Tx, sub *UserSubscription) (*UserSubscription, error)

	// ExpireCreditSubscriptions 推进已到期 active 订阅，并在同一事务内记录剩余额度销毁流水与通知 outbox。
	ExpireCreditSubscriptions(ctx context.Context) (int64, error)

	// MarkExpiredCreditLogged 标记过期销毁 ledger 已写入，避免重复写。
	MarkExpiredCreditLogged(ctx context.Context, id int64, loggedAt time.Time) error
}

// SubscriptionCreditLedgerRepository 订阅额度流水仓储。
type SubscriptionCreditLedgerRepository interface {
	// Create 写入一条流水。可在已有事务内调用（传 *sql.Tx）。
	Create(ctx context.Context, exec SQLExecer, entry *SubscriptionCreditLedgerEntry) error

	// CreateLimitReachedEvent 写入幂等事件（limit_reached / expire / window_reset）。
	// 使用 ON CONFLICT (subscription_id, type, event_key) DO NOTHING 实现幂等。
	// 返回 created：本次是否首次写入。
	CreateLimitReachedEvent(ctx context.Context, exec SQLExecer, entry *SubscriptionCreditLedgerEntry) (created bool, err error)

	ListByUserID(ctx context.Context, userID int64, ledgerType string, params pagination.PaginationParams) ([]SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error)
	ListBySubscriptionID(ctx context.Context, subscriptionID int64, ledgerType string, params pagination.PaginationParams) ([]SubscriptionCreditLedgerEntry, *pagination.PaginationResult, error)
}

// SQLExecer 抽象 *sql.DB / *sql.Tx 共有的接口。
// 让 repository 方法可以在事务内或非事务内调用。
type SQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
