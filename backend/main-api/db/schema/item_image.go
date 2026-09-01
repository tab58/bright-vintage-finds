package schema

import (
	"main-api/db/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ItemImage is an uploaded picture of an Item, stored in S3.
type ItemImage struct {
	ent.Schema
}

func (ItemImage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.WithSortableID{},
		mixin.WithCreationTracking{},
		mixin.WithSoftDelete{},
	}
}

func (ItemImage) Fields() []ent.Field {
	return []ent.Field{
		field.String("upload_bucket").
			NotEmpty().
			Immutable(),
		field.String("upload_key").
			NotEmpty().
			Immutable(),
		field.String("filename").
			Optional().
			Nillable(),
		field.String("content_type").
			Optional().
			Nillable(),
		field.Int64("size_bytes").
			Optional().
			Nillable(),
		field.Int("display_order").
			Default(0),
	}
}

func (ItemImage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("item", Item.Type).
			Ref("images").
			Unique().
			Required().
			Immutable(),
	}
}
