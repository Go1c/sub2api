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

type LotteryDraw struct {
	ent.Schema
}

func (LotteryDraw) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_draws"},
	}
}

func (LotteryDraw) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("campaign_id"),
		field.Int64("user_id"),
		field.Bool("won").
			Default(false),
		field.Int64("lottery_code_id").
			Optional().
			Nillable(),
		field.Int64("site_message_id").
			Optional().
			Nillable(),
		field.String("result_label").
			MaxLen(80).
			NotEmpty(),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (LotteryDraw) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("campaign_id", "user_id").
			Unique(),
		index.Fields("campaign_id", "created_at"),
		index.Fields("user_id", "created_at"),
		index.Fields("lottery_code_id"),
		index.Fields("site_message_id"),
	}
}
