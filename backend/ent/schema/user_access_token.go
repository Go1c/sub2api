package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserAccessToken holds long-lived opaque tokens for user-side key management APIs.
// Plaintext is never stored; only SHA-256 hash + display prefix.
type UserAccessToken struct {
	ent.Schema
}

func (UserAccessToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_access_tokens"},
	}
}

func (UserAccessToken) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (UserAccessToken) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		// Full token SHA-256 hex (64 chars); unique for O(1) auth lookup.
		field.String("token_hash").
			MaxLen(64).
			NotEmpty().
			Unique(),
		// Display prefix only (e.g. "uat_" + first 8 chars of secret); never the full secret.
		field.String("token_prefix").
			MaxLen(32).
			NotEmpty(),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("last_used_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		// Non-null means revoked; auth must reject.
		field.Time("revoked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (UserAccessToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("access_tokens").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (UserAccessToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("user_id", "created_at"),
	}
}
