package mixin

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// WithSoftDelete is a mixin that adds a deleted_at timestamp for soft deletion
type WithSoftDelete struct {
	mixin.Schema
}

func (WithSoftDelete) Fields() []ent.Field {
	return []ent.Field{
		field.Time("deleted_at").
			Optional().
			Nillable().
			Default(nil),
	}
}
