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

// AccountErrorHistory holds the schema definition for the AccountErrorHistory entity.
// 账号错误历史：记录每个账号来自不同来源的错误，供管理后台按需排查。
// 这是 best-effort 监控数据，写入异步非阻塞；字段可空性按来源不同而不同（只有
// gateway 真实请求来源才会填满 user/model/upstream_status_code）。
//
// 注意：建表以 migrations/152_add_account_error_histories.sql 为准，本 schema 仅用于
// ORM 读写，字段与类型必须与该 SQL 严格一致。
type AccountErrorHistory struct {
	ent.Schema
}

func (AccountErrorHistory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_error_histories"},
	}
}

func (AccountErrorHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		// user_id / user_email / model / upstream_status_code: 仅真实请求（gateway 源）齐全，
		// 其余来源留空。
		field.Int64("user_id").
			Optional().
			Nillable(),
		field.String("user_email").
			Optional().
			Nillable().
			MaxLen(255),
		field.String("model").
			Optional().
			Nillable().
			MaxLen(200),
		field.Int("upstream_status_code").
			Optional().
			Nillable(),
		field.Enum("source").
			Values("gateway", "test", "ratelimit", "refresh", "schedule", "admin"),
		field.String("message").
			Optional().
			Default("").
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("fingerprint").
			Optional().
			Default("").
			MaxLen(64),
		field.Int("dup_count").
			Default(1),
		field.Time("created_at").
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (AccountErrorHistory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("error_histories").
			Field("account_id").
			Unique().
			Required(),
	}
}

func (AccountErrorHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "created_at"),
		index.Fields("account_id", "source", "upstream_status_code", "fingerprint", "created_at"),
	}
}
