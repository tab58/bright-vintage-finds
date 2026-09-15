package schema

import (
	"main-api/db/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SellingPlace is a platform or venue an item can be sold on (e.g. Whatnot,
// a local store). Built-ins are seeded; the owner can add custom places.
type SellingPlace struct {
	ent.Schema
}

func (SellingPlace) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.WithSortableID{},
		mixin.WithCreationTracking{},
		mixin.WithUpdateTracking{},
		mixin.WithSoftDelete{},
	}
}

func (SellingPlace) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Unique(),
		field.Bool("is_builtin").
			Default(false),
	}
}

func (SellingPlace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("items", Item.Type).
			Ref("selling_places"),
	}
}