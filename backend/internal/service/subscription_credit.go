package service

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// 订阅 scope 类型常量（由 domain 包统一定义，service 层重新导出方便引用）
const (
	SubscriptionScopeAllAvailableGroups = domain.SubscriptionScopeAllAvailableGroups
	SubscriptionScopeSelectedGroups     = domain.SubscriptionScopeSelectedGroups
	SubscriptionScopePlatforms          = domain.SubscriptionScopePlatforms
)

// 订阅额度流水类型常量
const (
	SubscriptionCreditLedgerPurchase     = domain.SubscriptionCreditLedgerPurchase
	SubscriptionCreditLedgerConsume      = domain.SubscriptionCreditLedgerConsume
	SubscriptionCreditLedgerLimitReached = domain.SubscriptionCreditLedgerLimitReached
	SubscriptionCreditLedgerExpire       = domain.SubscriptionCreditLedgerExpire
	SubscriptionCreditLedgerWindowReset  = domain.SubscriptionCreditLedgerWindowReset
	SubscriptionCreditLedgerAdminAdjust  = domain.SubscriptionCreditLedgerAdminAdjust
)

// LimitReached 事件维度（用于 event_key 前缀和 outbox payload kind 后缀）
const (
	SubscriptionLimitReachedTotal  = domain.SubscriptionLimitReachedTotal
	SubscriptionLimitReachedDaily  = domain.SubscriptionLimitReachedDaily
	SubscriptionLimitReachedWeekly = domain.SubscriptionLimitReachedWeekly
)

// SchedulerOutboxEventSubscriptionNotify 是订阅通知专用的 outbox event_type。
const SchedulerOutboxEventSubscriptionNotify = domain.SchedulerOutboxEventSubscriptionNotify

// SubscriptionScopeConfig 描述订阅覆盖范围的细节。
// 取决于 UserSubscription.ScopeType：
//
//	SubscriptionScopeAllAvailableGroups: 两个字段都忽略
//	SubscriptionScopeSelectedGroups    : 使用 GroupIDs
//	SubscriptionScopePlatforms         : 使用 Platforms
//
// 存储为 user_subscriptions.scope_config / subscription_plans.scope_config 的 JSONB。
type SubscriptionScopeConfig struct {
	GroupIDs  []int64  `json:"group_ids,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
}

// SubscriptionCreditLedgerEntry 订阅额度流水的领域对象。
type SubscriptionCreditLedgerEntry struct {
	ID                int64
	UserID            int64
	SubscriptionID    int64
	GroupID           *int64
	APIKeyID          *int64
	UsageLogID        *int64
	OrderID           *int64
	Type              string
	DeltaUSD          float64
	BalanceDeltaUSD   float64
	RemainingAfterUSD float64
	Reason            string
	EventKey          *string
	Metadata          map[string]any
	CreatedAt         time.Time
}
