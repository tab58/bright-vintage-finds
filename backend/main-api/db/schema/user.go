package schema

import (
	"main-api/db/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

const (
	AccountStatusActive    string = "active"
	AccountStatusInactive  string = "inactive"
	AccountStatusSuspended string = "suspended"
)

type User struct {
	ent.Schema
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.WithSortableID{},
		mixin.WithCreationTracking{},
		mixin.WithUpdateTracking{},
		mixin.WithSoftDelete{},
	}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("idp_id").
			Unique().
			Immutable(),
		field.String("email").
			Unique(),
		field.String("full_name").
			NotEmpty(),
		field.Enum("account_status").
			Values(AccountStatusActive, AccountStatusInactive, AccountStatusSuspended).
			Default(AccountStatusInactive),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("items", Item.Type),
	}
}
