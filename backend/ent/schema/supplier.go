package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

type Supplier struct{ ent.Schema }

func (Supplier) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "suppliers"}}
}

func (Supplier) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}, mixins.SoftDeleteMixin{}}
}

func (Supplier) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("base_url").MaxLen(2048).NotEmpty(),
		field.String("status").MaxLen(20).Default("active"),
		field.String("notes").Default("").SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Int64("balance_account_id").Optional().Nillable(),
		field.String("balance_api_key").Default("").Sensitive().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Float("balance").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.String("balance_status").MaxLen(20).Default("unknown"),
		field.String("balance_error").Default("").SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("balance_updated_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("balance_checked_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int64("revision").Default(1),
		field.Int64("balance_probe_sequence").Default(0),
	}
}

func (Supplier) Edges() []ent.Edge {
	return []ent.Edge{edge.From("accounts", Account.Type).Ref("supplier")}
}

func (Supplier) Indexes() []ent.Index {
	return []ent.Index{index.Fields("status"), index.Fields("deleted_at"), index.Fields("balance_account_id")}
}
