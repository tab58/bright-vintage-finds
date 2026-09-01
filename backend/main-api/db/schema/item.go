package schema

import (
	"main-api/db/schema/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

const (
	ItemStatusDraft    string = "draft"
	ItemStatusListed   string = "listed"
	ItemStatusSold     string = "sold"
	ItemStatusArchived string = "archived"
)

// Item is a vintage good in the owner's inventory. Prices are in USD cents.
type Item struct {
	ent.Schema
}

func (Item) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.WithSortableID{},
		mixin.WithCreationTracking{},
		mixin.WithUpdateTracking{},
		mixin.WithSoftDelete{},
	}
}

func (Item) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable(),
		field.String("category").
			Optional().
			Nillable(),
		field.String("condition").
			Optional().
			Nillable(),
		field.Enum("status").
			Values(ItemStatusDraft, ItemStatusListed, ItemStatusSold, ItemStatusArchived).
			Default(ItemStatusDraft),
		field.Int64("acquisition_cost_cents").
			Optional().
			Nillable(),
		field.Int64("listing_price_cents").
			Optional().
			Nillable(),

		// Set together when the item sells; sales insight is computed from these.
		field.Int64("sold_price_cents").
			Optional().
			Nillable(),
		field.Time("sold_at").
			Optional().
			Nillable(),
	}
}

func (Item) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("items").
			Unique().
			Required(),

		edge.To("images", ItemImage.Type),
	}
}
