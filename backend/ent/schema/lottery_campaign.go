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

type LotteryCampaign struct {
	ent.Schema
}

func (LotteryCampaign) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_campaigns"},
	}
}

func (LotteryCampaign) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(120).
			NotEmpty(),
		field.String("subtitle").
			MaxLen(240).
			NotEmpty(),
		field.String("status").
			MaxLen(20).
			Default(domain.LotteryStatusActive),
		field.Int("prize_count").
			Positive(),
		field.Int("max_participants").
			Positive(),
		field.Int("joined_count").
			Default(0).
			NonNegative(),
		field.Int("winner_count").
			Default(0).
			NonNegative(),
		field.Int("early_boost_participant_percent").
			Default(25).
			NonNegative(),
		field.Int("recharge_boost_cap_percent").
			Default(0).
			NonNegative(),
		field.String("promo_text").
			MaxLen(240).
			Default(""),
		field.String("promo_image_url").
			MaxLen(2048).
			Default(""),
		field.Int64("created_by"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("finished_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (LotteryCampaign) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("created_by"),
	}
}
