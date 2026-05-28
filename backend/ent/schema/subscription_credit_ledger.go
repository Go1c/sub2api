package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SubscriptionCreditLedger 订阅额度流水。
//
// 所有订阅额度变化（购买、消费、过期、限额事件、管理员调整）都写入此表，
// 提供完整审计能力。详见 doc/plan/2026-05-28-subscription-credit-pool-plan.md。
//
// type 取值：purchase / consume / limit_reached / expire / admin_adjust
// event_key 用于幂等事件（limit_reached / expire），格式：
//
//	total:{subscription_id}
//	daily:{subscription_id}:{daily_window_start_rfc3339}
//	weekly:{subscription_id}:{weekly_window_start_rfc3339}
//	expire:{subscription_id}:{expires_at_rfc3339}
//
// 唯一索引 subscription_credit_ledger_event_key_unique 保证同一窗口/事件只写一次。
//
// delta_usd 符号：
//   - 增加额度（purchase / admin_adjust 正向）：> 0
//   - 减少额度（consume / expire / admin_adjust 负向）：< 0
//   - 幂等事件（limit_reached）：固定 0
type SubscriptionCreditLedger struct {
	ent.Schema
}

func (SubscriptionCreditLedger) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscription_credit_ledger"},
	}
}

func (SubscriptionCreditLedger) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("subscription_id"),
		field.Int64("group_id").
			Optional().
			Nillable(),
		field.Int64("api_key_id").
			Optional().
			Nillable(),
		field.Int64("usage_log_id").
			Optional().
			Nillable(),
		field.Int64("order_id").
			Optional().
			Nillable(),

		field.String("type").
			MaxLen(32),
		field.Float("delta_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("balance_delta_usd").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("remaining_after_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),

		field.Text("reason").
			Default(""),
		field.String("event_key").
			Optional().
			Nillable().
			MaxLen(128),

		field.JSON("metadata", map[string]any{}).
			Default(map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),

		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (SubscriptionCreditLedger) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("subscription_credit_ledgers").
			Field("user_id").
			Unique().
			Required(),
		edge.From("subscription", UserSubscription.Type).
			Ref("credit_ledger").
			Field("subscription_id").
			Unique().
			Required(),
		edge.From("group", Group.Type).
			Ref("subscription_credit_ledgers").
			Field("group_id").
			Unique(),
		edge.From("api_key", APIKey.Type).
			Ref("subscription_credit_ledgers").
			Field("api_key_id").
			Unique(),
		edge.From("usage_log", UsageLog.Type).
			Ref("subscription_credit_ledgers").
			Field("usage_log_id").
			Unique(),
		edge.From("order", PaymentOrder.Type).
			Ref("subscription_credit_ledgers").
			Field("order_id").
			Unique(),
	}
}

func (SubscriptionCreditLedger) Indexes() []ent.Index {
	return []ent.Index{
		// 主要查询路径：按用户/订阅倒序时间
		index.Fields("user_id", "created_at"),
		index.Fields("subscription_id", "created_at"),
		// 幂等事件唯一索引由 SQL migration 创建（部分索引，Ent 不支持带 WHERE 的唯一索引）
		index.Fields("event_key"),
	}
}
