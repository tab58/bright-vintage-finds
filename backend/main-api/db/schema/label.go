package schema

import (
	"main-api/db/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Label is a user-defined item-type tag (e.g. "Glassware"); items can have
// several. New labels can be created from the intake page.
type Label struct {
	ent.Schema
}

func (Label) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.WithSortableID{},
		mixin.WithCreationTracking{},
		mixin.WithUpdateTracking{},
		mixin.WithSoftDelete{},
	}
}

func (Label) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Unique(),
	}
}

func (Label) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("items", Item.Type).
			Ref("labels"),
	}
}