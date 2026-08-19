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

// BalanceDebitClient is a server-side consumer allowed to debit user wallets.
type BalanceDebitClient struct {
	ent.Schema
}

func (BalanceDebitClient) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "balance_debit_clients"}}
}

func (BalanceDebitClient) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (BalanceDebitClient) Fields() []ent.Field {
	return []ent.Field{
		field.String("client_id").
			SchemaType(map[string]string{dialect.Postgres: "uuid"}).
			Unique(),
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("secret_hash").MaxLen(64).NotEmpty().Unique(),
		field.String("secret_prefix").MaxLen(32).NotEmpty(),
		field.JSON("allowed_purposes", []string{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("status").MaxLen(16).Default("active"),
		field.Time("last_used_at").Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (BalanceDebitClient) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("transactions", BalanceDebitTransaction.Type),
	}
}

func (BalanceDebitClient) Indexes() []ent.Index {
	return []ent.Index{index.Fields("status")}
}
