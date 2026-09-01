package mixin

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// WithUpdateTracking is a mixin that tracks updates via an updated_at timestamp and a version number
type WithUpdateTracking struct {
	mixin.Schema
}

func (WithUpdateTracking) Fields() []ent.Field {
	return []ent.Field{
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Int("version").
			Default(1),
	}
}
