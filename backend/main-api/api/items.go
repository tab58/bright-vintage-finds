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
	// IDs alongside the names: the PWA edits these lists as checkboxes.
	SellingPlaceIDs []string `json:"selling_place_ids"`
	LabelIDs        []string `json:"label_ids"`
	ImageCount      int      `json:"image_count"`
	// Presigned URL of the first image, so lists can show a thumbnail without
	// a request per item. Absent when the item has no images or storage is off.
	CoverImageURL *string `json:"cover_image_url,omitempty"`

	// Set while the item is listed; cleared when it goes back to draft.
	ListedAt *time.Time `json:"listed_at,omitempty"`
	// First time the item was ever listed; never cleared.
	FirstListedAt *time.Time `json:"first_listed_at,omitempty"`

	SoldPriceCents *int64     `json:"sold_price_cents,omitempty"`
	SoldAt         *time.Time `json:"sold_at,omitempty"`
	SoldPlace      *string    `json:"sold_place,omitempty"`
	SoldPlaceID    *string    `json:"sold_place_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// itemOutput maps a loaded item (with edges eager-loaded by itemLoaded) to
// its API representation.
func itemOutput(ctx context.Context, deps *AppDeps, it *generated.Item) (*ItemOutput, error) {
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
		ListedAt:             it.ListedAt,
		FirstListedAt:        it.FirstListedAt,
		CreatedAt:            it.CreatedAt,
		UpdatedAt:            it.UpdatedAt,
		SellingPlaces:        []string{},
		Labels:               []string{},
		SellingPlaceIDs:      []string{},
		LabelIDs:             []string{},
	}

	// Rows listed before first_listed_at existed only have listed_at to go on.
	if out.FirstListedAt == nil {
		out.FirstListedAt = it.ListedAt
	}

	for _, sp := range it.Edges.SellingPlaces {
		out.SellingPlaces = append(out.SellingPlaces, sp.Name)
		out.SellingPlaceIDs = append(out.SellingPlaceIDs, sp.ID)
	}
	for _, l := range it.Edges.Labels {
		out.Labels = append(out.Labels, l.Name)
		out.LabelIDs = append(out.LabelIDs, l.ID)
	}
	if it.Edges.SoldPlace != nil {
		name := it.Edges.SoldPlace.Name
		id := it.Edges.SoldPlace.ID
		out.SoldPlace = &name
		out.SoldPlaceID = &id
	}

	images, err := deps.DB.GetDBFromContext(ctx).ItemImage.Query().
		Where(itemimage.HasItemWith(item.ID(it.ID)), itemimage.DeletedAtIsNil()).
		Order(generated.Asc(itemimage.FieldDisplayOrder)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading item images: %w", err)
	}
	out.ImageCount = len(images)
	if len(images) > 0 && deps.Store != nil {
		cover, err := imageOutputWithURL(ctx, deps, images[0])
		if err != nil {
			return nil, err
		}
		out.CoverImageURL = &cover.URL
	}

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

	// Status moves the item through draft -> listed -> sold, with archived as
	// a side exit. The PWA enforces which moves it offers; the API only
	// validates the value.
	Status *string `json:"status,omitempty" enum:"draft,listed,sold,archived" doc:"New status for the item"`

	// Sale details, for correcting a sale after the fact. Marking an item sold
	// in the first place goes through POST /admin/items/{id}/sold.
	SoldPriceCents *int64     `json:"sold_price_cents,omitempty" doc:"Sale price in USD cents"`
	SoldAt         *time.Time `json:"sold_at,omitempty"`
	SoldPlaceID    *string    `json:"sold_place_id,omitempty" doc:"ID of the place where it sold"`
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

// isEmpty reports whether the client sent the field as an empty string, which
// the PWA uses to mean "clear it".
func isEmpty(s *string) bool { return s != nil && *s == "" }

// omitEmpty drops an empty string so it is never stored as "".
func omitEmpty(s *string) *string {
	if isEmpty(s) {
		return nil
	}
	return s
}

// applyItemClears clears the fields the client sent as an empty string.
// whatnot_number is unique when present, so a cleared value must become NULL
// rather than "", otherwise two cleared items collide.
func applyItemClears(update *generated.ItemUpdateOne, body itemBody) *generated.ItemUpdateOne {
	if isEmpty(body.Notes) {
		update.ClearNotes()
	}
	if isEmpty(body.WhatnotNumber) {
		update.ClearWhatnotNumber()
	}
	return update
}

// registerItemCRUD registers the item create/list/get/update and mark-sold
// routes.
func registerItemCRUD(api huma.API, deps *AppDeps) {
	db := deps.DB
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
			SetNillableNotes(omitEmpty(in.Body.Notes)).
			SetNillableWhatnotNumber(omitEmpty(in.Body.WhatnotNumber))

		if in.Body.Status != nil {
			status, err := parseItemStatus(*in.Body.Status)
			if err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			if status == item.StatusSold && in.Body.SoldPlaceID == nil {
				return nil, huma.Error400BadRequest("a sold item needs sold_place_id")
			}
			create.SetStatus(status)
			if status == item.StatusListed {
				now := time.Now().UTC()
				create.SetListedAt(now).SetFirstListedAt(now)
			}
		}

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
		o, err := itemOutput(ctx, deps, loaded)
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
			o, err := itemOutput(ctx, deps, it)
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
		o, err := itemOutput(ctx, deps, loaded)
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
			SetNillableNotes(omitEmpty(in.Body.Notes)).
			SetNillableWhatnotNumber(omitEmpty(in.Body.WhatnotNumber))

		applyItemClears(update, in.Body)

		update.SetNillableSoldPriceCents(in.Body.SoldPriceCents).
			SetNillableSoldAt(in.Body.SoldAt)
		if in.Body.SoldPlaceID != nil {
			placeExists, err := client.SellingPlace.Query().
				Where(sellingplace.IDEQ(*in.Body.SoldPlaceID), sellingplace.DeletedAtIsNil()).
				Exist(ctx)
			if err != nil {
				return nil, fmt.Errorf("checking selling place: %w", err)
			}
			if !placeExists {
				return nil, huma.Error400BadRequest("unknown sold_place_id")
			}
			update.SetSoldPlaceID(*in.Body.SoldPlaceID)
		}

		if in.Body.Status != nil {
			status, err := parseItemStatus(*in.Body.Status)
			if err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			current, err := client.Item.Query().
				Where(item.IDEQ(in.ID)).
				Select(item.FieldStatus, item.FieldSoldPlaceID, item.FieldFirstListedAt).
				Only(ctx)
			if generated.IsNotFound(err) {
				return nil, huma.Error404NotFound("item not found")
			}
			if err != nil {
				return nil, fmt.Errorf("loading item status: %w", err)
			}
			// A sold item always names where it sold: sales insight is built on
			// that edge. The place comes from this payload or is already set.
			if status == item.StatusSold && in.Body.SoldPlaceID == nil && current.SoldPlaceID == nil {
				return nil, huma.Error400BadRequest("a sold item needs sold_place_id")
			}
			update.SetStatus(status)
			// listed_at times the current listing: stamped on the way in,
			// cleared on the way back to draft, untouched when the status is
			// re-sent unchanged.
			switch {
			case status == item.StatusListed && current.Status != item.StatusListed:
				now := time.Now().UTC()
				update.SetListedAt(now)
				// first_listed_at is written once and then left alone.
				if current.FirstListedAt == nil {
					update.SetFirstListedAt(now)
				}
			case status == item.StatusDraft:
				update.ClearListedAt()
			}
		}

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
		o, err := itemOutput(ctx, deps, loaded)
		if err != nil {
			return nil, err
		}
		return &getItemOutput{Body: o}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-item",
		Method:      http.MethodDelete,
		Path:        "/admin/items/{id}",
		Summary:     "Delete a draft item that was never listed",
	}, func(ctx context.Context, in *itemIDInput) (*struct{}, error) {
		it, err := client.Item.Query().
			Where(item.IDEQ(in.ID), item.DeletedAtIsNil()).
			Select(item.FieldStatus, item.FieldFirstListedAt).
			Only(ctx)
		if generated.IsNotFound(err) {
			return nil, huma.Error404NotFound("item not found")
		}
		if err != nil {
			return nil, fmt.Errorf("loading item: %w", err)
		}
		// Deletable: a draft that never went out, or anything already archived
		// (archive is the deliberate step before disposal). A live listing must
		// be unlisted or archived first, and a sale is never deleted.
		neverListed := it.Status == item.StatusDraft && it.FirstListedAt == nil
		switch {
		case it.Status == item.StatusArchived || neverListed:
			// allowed
		case it.Status == item.StatusSold:
			return nil, huma.Error409Conflict("sold items are a record of the sale and cannot be deleted")
		default:
			return nil, huma.Error409Conflict("this item has been listed; archive it before deleting")
		}

		// The pictures go too: stored objects first, then their rows. Objects
		// are removed before the rows so nothing can be orphaned in the bucket
		// with no record of what it was.
		images, err := client.ItemImage.Query().
			Where(itemimage.HasItemWith(item.ID(in.ID))).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading item images: %w", err)
		}
		for _, img := range images {
			if deps.Store == nil {
				break
			}
			if err := deps.Store.DeleteFile(ctx, img.UploadBucket, img.UploadKey); err != nil {
				return nil, fmt.Errorf("deleting image object %s: %w", img.UploadKey, err)
			}
		}
		if _, err := client.ItemImage.Delete().
			Where(itemimage.HasItemWith(item.ID(in.ID))).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("deleting item image rows: %w", err)
		}

		if err := client.Item.UpdateOneID(in.ID).SetDeletedAt(time.Now().UTC()).Exec(ctx); err != nil {
			return nil, fmt.Errorf("deleting item: %w", err)
		}
		return nil, nil
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
		o, err := itemOutput(ctx, deps, loaded)
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