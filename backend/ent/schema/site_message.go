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

// SiteMessage holds the schema definition for the SiteMessage entity.
//
// 站内信使用硬删除；保留期由业务配置控制，过期消息由清理任务删除。
type SiteMessage struct {
	ent.Schema
}

func (SiteMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "site_messages"},
	}
}

func (SiteMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("sender_id").
			Comment("发件人用户ID"),
		field.Int64("recipient_id").
			Comment("收件人用户ID"),
		field.Int64("parent_id").
			Optional().
			Nillable().
			Comment("回复的父站内信ID"),
		field.String("subject").
			MaxLen(200).
			NotEmpty().
			Comment("站内信标题"),
		field.String("content").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty().
			Comment("站内信内容"),
		field.Time("read_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("收件人首次读取时间，空表示未读"),
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

func (SiteMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sender", User.Type).
			Ref("sent_site_messages").
			Field("sender_id").
			Unique().
			Required(),
		edge.From("recipient", User.Type).
			Ref("received_site_messages").
			Field("recipient_id").
			Unique().
			Required(),
		edge.To("replies", SiteMessage.Type),
		edge.From("parent", SiteMessage.Type).
			Ref("replies").
			Field("parent_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}

func (SiteMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("recipient_id", "created_at"),
		index.Fields("sender_id", "created_at"),
		index.Fields("recipient_id", "read_at"),
		index.Fields("parent_id", "created_at"),
		index.Fields("created_at"),
	}
}
