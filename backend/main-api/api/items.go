package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	db_platform "main-api/db"
	"main-api/db/generated"
	"main-api/db/generated/item"
	"main-api/db/generated/itemimage"
	"main-api/db/generated/label"
	"main-api/db/generated/sellingplace"

	"github.com/danielgtaylor/huma/v2"
)

// ItemOutput is an inventory item as returned by the API.
type ItemOutput struct {
	ID string `json:"id"`

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	Condition   *string `json:"condition,omitempty"`
	Status      string  `json:"status"`

	AcquisitionCostCents *int64     `json:"acquisition_cost_cents,omitempty"`
	PurchasedAt          *time.Time `json:"purchased_at,omitempty"`
	ListingPriceCents    *int64     `json:"listing_price_cents,omitempty"`

	Length            *float64 `json:"length,omitempty"`
	Width             *float64 `json:"width,omitempty"`
	Height            *float64 `json:"height,omitempty"`
	MeasurementUnit   string   `json:"measurement_unit"`
	ExtraMeasurements *string  `json:"extra_measurements,omitempty"`
	WeightLbs         *int     `json:"weight_lbs,omitempty"`
	WeightOz          *float64 `json:"weight_oz,omitempty"`
	Notes             *string  `json:"notes,omitempty"`

	WhatnotNumber *string `json:"whatnot_number,omitempty"`

	SellingPlaces []string `json:"selling_places"` // names
	Labels        []string `json:"labels"`         // names
	ImageCount    int      `json:"image_count"`

	SoldPriceCents *int64     `json:"sold_price_cents,omitempty"`
	SoldAt         *time.Time `json:"sold_at,omitempty"`
	SoldPlace      *string    `json:"sold_place,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// itemOutput maps a loaded item (with edges eager-loaded by itemLoaded) to
// its API representation.
func itemOutput(ctx context.Context, db *db_platform.Client, it *generated.Item) (*ItemOutput, error) {
	out := &ItemOutput{
		ID:                   it.ID,
		Name:                 it.Name,
		Description:          it.Description,
		Category:             it.Category,
		Condition:            it.Condition,
		Status:               string(it.Status),
		AcquisitionCostCents: it.AcquisitionCostCents,
		PurchasedAt:          it.PurchasedAt,
		ListingPriceCents:    it.ListingPriceCents,
		Length:               it.Length,
		Width:                it.Width,
		Height:               it.Height,
		MeasurementUnit:      string(it.MeasurementUnit),
		ExtraMeasurements:    it.ExtraMeasurements,
		WeightLbs:            it.WeightLbs,
		WeightOz:             it.WeightOz,
		Notes:                it.Notes,
		WhatnotNumber:        it.WhatnotNumber,
		SoldPriceCents:       it.SoldPriceCents,
		SoldAt:               it.SoldAt,
		CreatedAt:            it.CreatedAt,
		UpdatedAt:            it.UpdatedAt,
		SellingPlaces:        []string{},
		Labels:               []string{},
	}

	for _, sp := range it.Edges.SellingPlaces {
		out.SellingPlaces = append(out.SellingPlaces, sp.Name)
	}
	for _, l := range it.Edges.Labels {
		out.Labels = append(out.Labels, l.Name)
	}
	if it.Edges.SoldPlace != nil {
		name := it.Edges.SoldPlace.Name
		out.SoldPlace = &name
	}

	count, err := db.GetDBFromContext(ctx).ItemImage.Query().
		Where(itemimage.HasItemWith(item.ID(it.ID)), itemimage.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting item images: %w", err)
	}
	out.ImageCount = count

	return out, nil
}

// itemLoaded fetches one item with every edge itemOutput needs.
func itemLoaded(ctx context.Context, db *db_platform.Client, id string) (*generated.Item, error) {
	return db.GetDBFromContext(ctx).Item.Query().
		Where(item.ID(id), item.DeletedAtIsNil()).
		WithSellingPlaces().
		WithLabels().
		WithSoldPlace().
		Only(ctx)
}

// parseMeasurementUnit converts a wire string to the generated enum.
func parseMeasurementUnit(s string) (item.MeasurementUnit, error) {
	switch s {
	case "inch":
		return item.MeasurementUnitInch, nil
	case "cm":
		return item.MeasurementUnitCm, nil
	}
	return "", fmt.Errorf("measurement_unit must be inch or cm")
}

// parseItemStatus converts a wire string to the generated status enum.
func parseItemStatus(s string) (item.Status, error) {
	switch s {
	case "draft":
		return item.StatusDraft, nil
	case "listed":
		return item.StatusListed, nil
	case "sold":
		return item.StatusSold, nil
	case "archived":
		return item.StatusArchived, nil
	}
	return "", fmt.Errorf("invalid status %q", s)
}

// itemBody is the shared create/update payload. All optional fields are
// pointers so PATCH can distinguish "absent" from "cleared".
type itemBody struct {
	Name                 string     `json:"name" minLength:"1" maxLength:"300"`
	Description          *string    `json:"description,omitempty"`
	Category             *string    `json:"category,omitempty"`
	Condition            *string    `json:"condition,omitempty"`
	AcquisitionCostCents *int64     `json:"acquisition_cost_cents,omitempty" doc:"Purchase price in USD cents"`
	PurchasedAt          *time.Time `json:"purchased_at,omitempty"`
	ListingPriceCents    *int64     `json:"listing_price_cents,omitempty"`
	Length               *float64   `json:"length,omitempty"`
	Width                *float64   `json:"width,omitempty"`
	Height               *float64   `json:"height,omitempty"`
	MeasurementUnit      *string    `json:"measurement_unit,omitempty" enum:"inch,cm" doc:"Unit for length/width/height"`
	ExtraMeasurements    *string    `json:"extra_measurements,omitempty"`
	WeightLbs            *int       `json:"weight_lbs,omitempty"`
	WeightOz             *float64   `json:"weight_oz,omitempty"`
	Notes                *string    `json:"notes,omitempty"`
	WhatnotNumber        *string    `json:"whatnot_number,omitempty"`
	SellingPlaceIDs      []string   `json:"selling_place_ids,omitempty"`
	LabelIDs             []string   `json:"label_ids,omitempty"`
}

type getItemOutput struct {
	Body *ItemOutput `json:"body"`
}

type listItemsOutput struct {
	Body []*ItemOutput `json:"body"`
}

type itemIDInput struct {
	ID string `path:"id" doc:"Item ID"`
}

type listItemsInput struct {
	Query string `query:"query" maxLength:"200" doc:"Substring match against item name" required:"false"`
	// PlaceID filters to items listed on this selling place.
	PlaceID string `query:"place_id" required:"false"`
	// LabelID filters to items tagged with this label.
	LabelID string `query:"label_id" required:"false"`
	// WhatnotNumber filters to the item with this Whatnot listing ID.
	WhatnotNumber string `query:"whatnot_number" required:"false"`
	Status        string `query:"status" required:"false" enum:"draft,listed,sold,archived"`
}

type updateItemInput struct {
	ID   string `path:"id"`
	Body itemBody
}

type markSoldInput struct {
	ID   string `path:"id"`
	Body struct {
		SoldAt         time.Time `json:"sold_at" doc:"Date the item sold"`
		SoldPriceCents int64     `json:"sold_price_cents" doc:"Sale price in USD cents"`
		SoldPlaceID    string    `json:"sold_place_id" doc:"ID of the platform/place where it sold"`
	}
}

// registerItemCRUD registers the item create/list/get/update and mark-sold
// routes.
func registerItemCRUD(api huma.API, db *db_platform.Client) {
	client := db.GetDBFromContext(nil)

	huma.Register(api, huma.Operation{
		OperationID: "create-item",
		Method:      http.MethodPost,
		Path:        "/admin/items",
		Summary:     "Create an inventory item",
	}, func(ctx context.Context, in *struct{ Body itemBody }) (*getItemOutput, error) {
		ownerID, err := ensureOwner(ctx, client)
		if err != nil {
			return nil, err
		}

		create := client.Item.Create().
			SetName(in.Body.Name).
			SetOwnerID(ownerID).
			SetNillableDescription(in.Body.Description).
			SetNillableCategory(in.Body.Category).
			SetNillableCondition(in.Body.Condition).
			SetNillableAcquisitionCostCents(in.Body.AcquisitionCostCents).
			SetNillablePurchasedAt(in.Body.PurchasedAt).
			SetNillableListingPriceCents(in.Body.ListingPriceCents).
			SetNillableLength(in.Body.Length).
			SetNillableWidth(in.Body.Width).
			SetNillableHeight(in.Body.Height).
			SetNillableExtraMeasurements(in.Body.ExtraMeasurements).
			SetNillableWeightLbs(in.Body.WeightLbs).
			SetNillableWeightOz(in.Body.WeightOz).
			SetNillableNotes(in.Body.Notes).
			SetNillableWhatnotNumber(in.Body.WhatnotNumber)

		if in.Body.MeasurementUnit != nil {
			unit, err := parseMeasurementUnit(*in.Body.MeasurementUnit)
			if err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			create.SetMeasurementUnit(unit)
		}
		if err := validateItemEdges(ctx, client, in.Body.SellingPlaceIDs, in.Body.LabelIDs); err != nil {
			return nil, err
		}
		if len(in.Body.SellingPlaceIDs) > 0 {
			create.AddSellingPlaceIDs(in.Body.SellingPlaceIDs...)
		}
		if len(in.Body.LabelIDs) > 0 {
			create.AddLabelIDs(in.Body.LabelIDs...)
		}

		it, err := create.Save(ctx)
		if err != nil {
			if generated.IsConstraintError(err) {
				return nil, huma.Error409Conflict("an item with this Whatnot number already exists")
			}
			return nil, fmt.Errorf("creating item: %w", err)
		}

		loaded, err := itemLoaded(ctx, db, it.ID)
		if err != nil {
			return nil, err
		}
		o, err := itemOutput(ctx, db, loaded)
		if err != nil {
			return nil, err
		}
		return &getItemOutput{Body: o}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-items",
		Method:      http.MethodGet,
		Path:        "/admin/items",
		Summary:     "List inventory items with search filters",
	}, func(ctx context.Context, in *listItemsInput) (*listItemsOutput, error) {
		q := client.Item.Query().
			Where(item.DeletedAtIsNil()).
			WithSellingPlaces().
			WithLabels().
			WithSoldPlace().
			Order(generated.Desc(item.FieldCreatedAt))

		if in.Query != "" {
			q = q.Where(item.NameContainsFold(in.Query))
		}
		if in.PlaceID != "" {
			q = q.Where(item.HasSellingPlacesWith(sellingplace.IDEQ(in.PlaceID), sellingplace.DeletedAtIsNil()))
		}
		if in.LabelID != "" {
			q = q.Where(item.HasLabelsWith(label.IDEQ(in.LabelID), label.DeletedAtIsNil()))
		}
		if in.WhatnotNumber != "" {
			q = q.Where(item.WhatnotNumberEQ(in.WhatnotNumber))
		}
		if in.Status != "" {
			status, err := parseItemStatus(in.Status)
			if err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			q = q.Where(item.StatusEQ(status))
		}

		items, err := q.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing items: %w", err)
		}
		out := &listItemsOutput{Body: make([]*ItemOutput, 0, len(items))}
		for _, it := range items {
			o, err := itemOutput(ctx, db, it)
			if err != nil {
				return nil, err
			}
			out.Body = append(out.Body, o)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-item",
		Method:      http.MethodGet,
		Path:        "/admin/items/{id}",
		Summary:     "Get one inventory item",
	}, func(ctx context.Context, in *itemIDInput) (*getItemOutput, error) {
		loaded, err := itemLoaded(ctx, db, in.ID)
		if generated.IsNotFound(err) {
			return nil, huma.Error404NotFound("item not found")
		}
		if err != nil {
			return nil, fmt.Errorf("loading item: %w", err)
		}
		o, err := itemOutput(ctx, db, loaded)
		if err != nil {
			return nil, err
		}
		return &getItemOutput{Body: o}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-item",
		Method:      http.MethodPatch,
		Path:        "/admin/items/{id}",
		Summary:     "Update an inventory item",
	}, func(ctx context.Context, in *updateItemInput) (*getItemOutput, error) {
		update := client.Item.UpdateOneID(in.ID).
			SetName(in.Body.Name).
			SetNillableDescription(in.Body.Description).
			SetNillableCategory(in.Body.Category).
			SetNillableCondition(in.Body.Condition).
			SetNillableAcquisitionCostCents(in.Body.AcquisitionCostCents).
			SetNillablePurchasedAt(in.Body.PurchasedAt).
			SetNillableListingPriceCents(in.Body.ListingPriceCents).
			SetNillableLength(in.Body.Length).
			SetNillableWidth(in.Body.Width).
			SetNillableHeight(in.Body.Height).
			SetNillableExtraMeasurements(in.Body.ExtraMeasurements).
			SetNillableWeightLbs(in.Body.WeightLbs).
			SetNillableWeightOz(in.Body.WeightOz).
			SetNillableNotes(in.Body.Notes).
			SetNillableWhatnotNumber(in.Body.WhatnotNumber)

		if in.Body.MeasurementUnit != nil {
			unit, err := parseMeasurementUnit(*in.Body.MeasurementUnit)
			if err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			update.SetMeasurementUnit(unit)
		}

		// Full replacement of the M2M sets: the PWA always sends the
		// complete lists.
		if in.Body.SellingPlaceIDs != nil {
			if err := validateItemEdges(ctx, client, in.Body.SellingPlaceIDs, nil); err != nil {
				return nil, err
			}
			update.ClearSellingPlaces()
			update.AddSellingPlaceIDs(in.Body.SellingPlaceIDs...)
		}
		if in.Body.LabelIDs != nil {
			if err := validateItemEdges(ctx, client, nil, in.Body.LabelIDs); err != nil {
				return nil, err
			}
			update.ClearLabels()
			update.AddLabelIDs(in.Body.LabelIDs...)
		}

		it, err := update.Save(ctx)
		if err != nil {
			if generated.IsNotFound(err) {
				return nil, huma.Error404NotFound("item not found")
			}
			if generated.IsConstraintError(err) {
				return nil, huma.Error409Conflict("an item with this Whatnot number already exists")
			}
			return nil, fmt.Errorf("updating item: %w", err)
		}

		loaded, err := itemLoaded(ctx, db, it.ID)
		if err != nil {
			return nil, err
		}
		o, err := itemOutput(ctx, db, loaded)
		if err != nil {
			return nil, err
		}
		return &getItemOutput{Body: o}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "mark-item-sold",
		Method:      http.MethodPost,
		Path:        "/admin/items/{id}/sold",
		Summary:     "Mark an item as sold",
	}, func(ctx context.Context, in *markSoldInput) (*getItemOutput, error) {
		placeExists, err := client.SellingPlace.Query().
			Where(sellingplace.IDEQ(in.Body.SoldPlaceID), sellingplace.DeletedAtIsNil()).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("checking selling place: %w", err)
		}
		if !placeExists {
			return nil, huma.Error400BadRequest("unknown sold_place_id")
		}

		it, err := client.Item.UpdateOneID(in.ID).
			SetStatus(item.StatusSold).
			SetSoldAt(in.Body.SoldAt).
			SetSoldPriceCents(in.Body.SoldPriceCents).
			SetSoldPlaceID(in.Body.SoldPlaceID).
			Save(ctx)
		if err != nil {
			if generated.IsNotFound(err) {
				return nil, huma.Error404NotFound("item not found")
			}
			return nil, fmt.Errorf("marking item sold: %w", err)
		}

		loaded, err := itemLoaded(ctx, db, it.ID)
		if err != nil {
			return nil, err
		}
		o, err := itemOutput(ctx, db, loaded)
		if err != nil {
			return nil, err
		}
		return &getItemOutput{Body: o}, nil
	})
}

// validateItemEdges checks the referenced IDs exist, returning a huma 400
// naming the offending list when they don't.
func validateItemEdges(ctx context.Context, client *generated.Client, placeIDs, labelIDs []string) error {
	if len(placeIDs) > 0 {
		n, err := client.SellingPlace.Query().
			Where(sellingplace.IDIn(placeIDs...), sellingplace.DeletedAtIsNil()).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("checking selling places: %w", err)
		}
		if n != len(placeIDs) {
			return huma.Error400BadRequest("unknown selling_place_ids")
		}
	}
	if len(labelIDs) > 0 {
		n, err := client.Label.Query().
			Where(label.IDIn(labelIDs...), label.DeletedAtIsNil()).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("checking labels: %w", err)
		}
		if n != len(labelIDs) {
			return huma.Error400BadRequest("unknown label_ids")
		}
	}
	return nil
}