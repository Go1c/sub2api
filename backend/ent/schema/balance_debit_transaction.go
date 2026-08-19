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

// BalanceDebitTransaction is the immutable successful debit ledger.
type BalanceDebitTransaction struct {
	ent.Schema
}

func (BalanceDebitTransaction) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "balance_debit_transactions"}}
}

func (BalanceDebitTransaction) Fields() []ent.Field {
	return []ent.Field{
		field.String("txn_id").SchemaType(map[string]string{dialect.Postgres: "uuid"}).Unique(),
		// Intentionally no users edge/FK: physical user deletion must not erase financial audit rows.
		field.Int64("user_id"),
		field.Int64("balance_client_id"),
		field.String("idempotency_key_hash").MaxLen(64),
		field.String("request_fingerprint").MaxLen(64),
		field.Float("amount").SchemaType(map[string]string{dialect.Postgres: "numeric(20,2)"}),
		field.String("currency").MaxLen(3),
		field.String("purpose").MaxLen(64),
		field.String("ref").MaxLen(128),
		field.Float("balance_before").SchemaType(map[string]string{dialect.Postgres: "numeric(20,8)"}),
		field.Float("balance_after").SchemaType(map[string]string{dialect.Postgres: "numeric(20,8)"}),
		field.Time("created_at").Default(time.Now).Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (BalanceDebitTransaction) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("client", BalanceDebitClient.Type).
			Ref("transactions").
			Field("balance_client_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (BalanceDebitTransaction) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("balance_client_id", "user_id", "idempotency_key_hash").Unique(),
		index.Fields("user_id", "created_at", "id"),
		index.Fields("user_id", "ref"),
		index.Fields("balance_client_id", "created_at"),
	}
}
