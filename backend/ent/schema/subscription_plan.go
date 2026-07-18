package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SubscriptionPlan holds the schema definition for the SubscriptionPlan entity.
//
// 删除策略：硬删除
// SubscriptionPlan 使用硬删除而非软删除，原因如下：
//   - 套餐为管理员维护的商品配置，删除即表示下架移除
//   - 通过 for_sale 字段控制是否在售，删除仅用于彻底移除
//   - 已购买的订阅记录保存在 UserSubscription 中，不受套餐删除影响
//     （user_subscriptions.plan_id FK 设置为 ON DELETE SET NULL）
//
// 额度池字段（详见 doc/plan/2026-05-28-subscription-credit-pool-plan.md）：
//   - quota_usd: 套餐总额度，购买时快照到 user_subscriptions.quota_limit_usd
//   - daily_limit_usd / weekly_limit_usd: 窗口节流配置
//   - scope_type / scope_config: 覆盖范围描述
//   - validity_days: 有效期（1-30 天）
type SubscriptionPlan struct {
	ent.Schema
}

func (SubscriptionPlan) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscription_plans"},
	}
}

func (SubscriptionPlan) Fields() []ent.Field {
	return []ent.Field{
		// group_id 改为可空：额度池套餐不绑定具体分组（由 scope_* 描述覆盖范围）
		field.Int64("group_id").
			Optional().
			Nillable(),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.String("description").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.Float("price").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}),
		field.Float("original_price").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Optional().
			Nillable(),
		// currency is a display-only ISO 4217 label shown next to the price.
		// Empty means no label; it does not affect checkout or settlement currency.
		field.String("currency").
			MaxLen(3).
			Default(""),
		field.Int("validity_days").
			Default(30),
		field.String("validity_unit").
			MaxLen(10).
			Default("day"),
		field.String("features").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("purchase_notice").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("product_name").
			MaxLen(100).
			Default(""),
		field.Bool("for_sale").
			Default(true),
		field.Int("sort_order").
			Default(0),

		// 额度套餐字段（订阅额度池模型）
		field.Float("quota_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}).
			Default(0),
		field.Float("daily_limit_usd").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("weekly_limit_usd").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.String("scope_type").
			MaxLen(32).
			Default(domain.SubscriptionScopeAllAvailableGroups),
		field.JSON("scope_config", map[string]any{}).
			Default(map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),

		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (SubscriptionPlan) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id"),
		index.Fields("for_sale"),
	}
}
