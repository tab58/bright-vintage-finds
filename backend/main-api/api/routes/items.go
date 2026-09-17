package routes

import (
	"context"
	"net/http"
	"time"

	"main-api/internal/app"
	"main-api/internal/app/domain"

	"github.com/danielgtaylor/huma/v2"
)

// ItemOutput is an inventory item as returned by the admin API.
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

// itemOutput maps an item and its photo facts to the API representation.
func itemOutput(v app.ItemView) *ItemOutput {
	it := v.Item
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
		SellingPlaces:        it.SellingPlaceNames(),
		Labels:               it.LabelNames(),
		SellingPlaceIDs:      it.SellingPlaceIDs(),
		LabelIDs:             it.LabelIDs(),
		ImageCount:           v.ImageCount,
		CoverImageURL:        v.CoverImageURL,
		ListedAt:             it.ListedAt,
		FirstListedAt:        it.EffectiveFirstListedAt(),
		SoldPriceCents:       it.SoldPriceCents,
		SoldAt:               it.SoldAt,
		CreatedAt:            it.CreatedAt,
		UpdatedAt:            it.UpdatedAt,
	}
	if it.SoldPlace != nil {
		name, id := it.SoldPlace.Name, it.SoldPlace.ID
		out.SoldPlace = &name
		out.SoldPlaceID = &id
	}
	return out
}

func itemOutputs(views []app.ItemView) []*ItemOutput {
	out := make([]*ItemOutput, 0, len(views))
	for _, v := range views {
		out = append(out, itemOutput(v))
	}
	return out
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

// itemInput hands the payload to the application unchanged; parsing the
// status and unit strings is the application's job, because the messages it
// rejects them with are part of the API contract.
func itemInput(b itemBody) app.ItemInput {
	return app.ItemInput{
		Name:                 b.Name,
		Description:          b.Description,
		Category:             b.Category,
		Condition:            b.Condition,
		AcquisitionCostCents: b.AcquisitionCostCents,
		PurchasedAt:          b.PurchasedAt,
		ListingPriceCents:    b.ListingPriceCents,
		Length:               b.Length,
		Width:                b.Width,
		Height:               b.Height,
		MeasurementUnit:      b.MeasurementUnit,
		ExtraMeasurements:    b.ExtraMeasurements,
		WeightLbs:            b.WeightLbs,
		WeightOz:             b.WeightOz,
		Notes:                b.Notes,
		WhatnotNumber:        b.WhatnotNumber,
		SellingPlaceIDs:      b.SellingPlaceIDs,
		LabelIDs:             b.LabelIDs,
		Status:               b.Status,
		SoldPriceCents:       b.SoldPriceCents,
		SoldAt:               b.SoldAt,
		SoldPlaceID:          b.SoldPlaceID,
	}
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

// RegisterItemCRUD registers the item create/list/get/update, delete and
// mark-sold routes.
func RegisterItemCRUD(api huma.API, a *app.Application) {
	huma.Register(api, huma.Operation{
		OperationID: "create-item",
		Method:      http.MethodPost,
		Path:        "/admin/items",
		Summary:     "Create an inventory item",
	}, func(ctx context.Context, in *struct{ Body itemBody }) (*getItemOutput, error) {
		v, err := a.CreateItem(ctx, itemInput(in.Body))
		if err != nil {
			return nil, toHTTP(err)
		}
		return &getItemOutput{Body: itemOutput(v)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-items",
		Method:      http.MethodGet,
		Path:        "/admin/items",
		Summary:     "List inventory items with search filters",
	}, func(ctx context.Context, in *listItemsInput) (*listItemsOutput, error) {
		views, err := a.ListItems(ctx, app.ItemFilterInput{
			Query:         in.Query,
			PlaceID:       in.PlaceID,
			LabelID:       in.LabelID,
			WhatnotNumber: in.WhatnotNumber,
			Status:        in.Status,
		})
		if err != nil {
			return nil, toHTTP(err)
		}
		return &listItemsOutput{Body: itemOutputs(views)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-item",
		Method:      http.MethodGet,
		Path:        "/admin/items/{id}",
		Summary:     "Get one inventory item",
	}, func(ctx context.Context, in *itemIDInput) (*getItemOutput, error) {
		v, err := a.GetItem(ctx, in.ID)
		if err != nil {
			return nil, toHTTP(err)
		}
		return &getItemOutput{Body: itemOutput(v)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-item",
		Method:      http.MethodPatch,
		Path:        "/admin/items/{id}",
		Summary:     "Update an inventory item",
	}, func(ctx context.Context, in *updateItemInput) (*getItemOutput, error) {
		v, err := a.UpdateItem(ctx, in.ID, itemInput(in.Body))
		if err != nil {
			return nil, toHTTP(err)
		}
		return &getItemOutput{Body: itemOutput(v)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-item",
		Method:      http.MethodDelete,
		Path:        "/admin/items/{id}",
		Summary:     "Delete a draft item that was never listed",
	}, func(ctx context.Context, in *itemIDInput) (*struct{}, error) {
		if err := a.DeleteItem(ctx, in.ID); err != nil {
			return nil, toHTTP(err)
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "mark-item-sold",
		Method:      http.MethodPost,
		Path:        "/admin/items/{id}/sold",
		Summary:     "Mark an item as sold",
	}, func(ctx context.Context, in *markSoldInput) (*getItemOutput, error) {
		v, err := a.MarkItemSold(ctx, in.ID, domain.Sale{
			SoldAt:         in.Body.SoldAt,
			SoldPriceCents: in.Body.SoldPriceCents,
			SoldPlaceID:    in.Body.SoldPlaceID,
		})
		if err != nil {
			return nil, toHTTP(err)
		}
		return &getItemOutput{Body: itemOutput(v)}, nil
	})
}
