package mixin

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// WithCreationTracking is a mixin that adds a creation timestamp
type WithCreationTracking struct {
	mixin.Schema
}

func (WithCreationTracking) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}
