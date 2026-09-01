package mixin

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/segmentio/ksuid"
)

// WithSortableID is a mixin that adds a KSUID as an identifier to the schema (alphabetical order = creation order)
type WithSortableID struct {
	mixin.Schema
}

func (WithSortableID) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			DefaultFunc(func() string {
				return ksuid.New().String()
			}).
			Immutable(),
	}
}
