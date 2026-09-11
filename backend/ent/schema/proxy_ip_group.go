package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProxyIPGroup is an admin-defined set of proxies with a uniform per-IP concurrency cap.
type ProxyIPGroup struct {
	ent.Schema
}

func (ProxyIPGroup) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "proxy_ip_groups"},
	}
}

func (ProxyIPGroup) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (ProxyIPGroup) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.Int("per_ip_concurrency").
			Default(10).
			Range(1, 1000),
	}
}

func (ProxyIPGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("accounts", Account.Type).
			Ref("proxy_ip_group"),
	}
}

func (ProxyIPGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
		index.Fields("deleted_at"),
	}
}
