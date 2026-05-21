package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type LotteryCode struct {
	ent.Schema
}

func (LotteryCode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_codes"},
	}
}

func (LotteryCode) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("campaign_id"),
		field.String("code").
			MaxLen(128).
			NotEmpty(),
		field.Int64("assigned_user_id").
			Optional().
			Nillable(),
		field.Int64("assigned_draw_id").
			Optional().
			Nillable(),
		field.Time("assigned_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
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

func (LotteryCode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("campaign_id", "code").
			Unique(),
		index.Fields("campaign_id", "assigned_at"),
		index.Fields("assigned_user_id"),
		index.Fields("assigned_draw_id"),
	}
}
