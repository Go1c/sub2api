package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserSubscription holds the schema definition for the UserSubscription entity.
//
// 订阅额度池模型（详见 doc/plan/2026-05-28-subscription-credit-pool-plan.md）：
//   - 用户级订阅，group_id 可空（额度池订阅不绑定具体分组）
//   - quota_limit_usd / quota_used_usd 表达总额度
//   - daily_limit_usd / weekly_limit_usd 表达窗口节流（NULL 表示无限制）
//   - exhausted_at 时间戳标记总额度耗尽时刻；NULL 表示订阅当前仍可消费
//   - 月限字段已删除（订阅最长 30 天，月限≡总额度）
type UserSubscription struct {
	ent.Schema
}

func (UserSubscription) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_subscriptions"},
	}
}

func (UserSubscription) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (UserSubscription) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("group_id").
			Optional().
			Nillable(),

		// 套餐与覆盖范围
		field.Int64("plan_id").
			Optional().
			Nillable(),
		field.String("scope_type").
			MaxLen(32).
			Default(domain.SubscriptionScopeAllAvailableGroups),
		field.JSON("scope_config", map[string]any{}).
			Default(map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),

		// 总额度（DB CHECK 约束 >= 0；业务层创建额度池订阅时强制 > 0）
		field.Float("quota_limit_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0),
		field.Float("quota_used_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0),

		// 窗口节流（NULL 表示该窗口无限制）
		field.Float("daily_limit_usd").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("weekly_limit_usd").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),

		// 生命周期
		field.Time("starts_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("status").
			MaxLen(20).
			Default(domain.SubscriptionStatusActive),

		// 耗尽时间戳（总额度耗尽时写入；NULL = 仍可消费）
		field.Time("exhausted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		// 过期销毁 ledger 已写入标记（避免重复写）
		field.Time("expired_credit_logged_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		// 窗口状态（按 UTC date_trunc 对齐）
		field.Time("daily_window_start").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("weekly_window_start").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		field.Float("daily_usage_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0),
		field.Float("weekly_usage_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0),

		// 用户手动重置周限时间戳（NULL = 本订阅周期内尚未手动重置）
		field.Time("weekly_limit_user_reset_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		// 审计信息
		field.Int64("assigned_by").
			Optional().
			Nillable(),
		field.Time("assigned_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("notes").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
	}
}

func (UserSubscription) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("subscriptions").
			Field("user_id").
			Unique().
			Required(),
		edge.From("group", Group.Type).
			Ref("subscriptions").
			Field("group_id").
			Unique(),
		edge.From("assigned_by_user", User.Type).
			Ref("assigned_subscriptions").
			Field("assigned_by").
			Unique(),
		edge.To("usage_logs", UsageLog.Type),
		edge.To("credit_ledger", SubscriptionCreditLedger.Type),
	}
}

func (UserSubscription) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("group_id"),
		index.Fields("status"),
		index.Fields("expires_at"),
		index.Fields("user_id", "status", "expires_at"),
		index.Fields("assigned_by"),
		// 多订阅购买由业务设置控制；历史的 user_active_usable 部分唯一索引
		// 已由迁移移除，避免阻断允许多订阅时的履约写入。
		index.Fields("deleted_at"),
	}
}
