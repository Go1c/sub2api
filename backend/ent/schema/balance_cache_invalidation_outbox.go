package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// BalanceCacheInvalidationOutbox coalesces cache invalidation work per user.
type BalanceCacheInvalidationOutbox struct {
	ent.Schema
}

func (BalanceCacheInvalidationOutbox) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "balance_cache_invalidation_outbox"}}
}

func (BalanceCacheInvalidationOutbox) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (BalanceCacheInvalidationOutbox) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Unique(),
		field.Int("attempts").Default(0),
		field.Time("next_attempt_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("claimed_at").Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("claim_token").MaxLen(36).Optional().Nillable(),
		field.Text("last_error").Default(""),
	}
}

func (BalanceCacheInvalidationOutbox) Indexes() []ent.Index {
	return []ent.Index{index.Fields("next_attempt_at", "id")}
}
