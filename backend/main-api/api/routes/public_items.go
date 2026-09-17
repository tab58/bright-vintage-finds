package routes

import (
	"context"
	"net/http"

	"main-api/internal/app"
	"main-api/internal/app/domain"

	"github.com/danielgtaylor/huma/v2"
)

// PublicItemOutput is an item as the shop front sees it. It deliberately
// carries no acquisition cost, notes, Whatnot number, selling places, sold
// data or storage keys: this response is readable by anyone.
type PublicItemOutput struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	Condition   *string `json:"condition,omitempty"`

	ListingPriceCents *int64 `json:"listing_price_cents,omitempty"`

	Length            *float64 `json:"length,omitempty"`
	Width             *float64 `json:"width,omitempty"`
	Height            *float64 `json:"height,omitempty"`
	MeasurementUnit   string   `json:"measurement_unit"`
	ExtraMeasurements *string  `json:"extra_measurements,omitempty"`

	Labels     []string `json:"labels"`
	ImageCount int      `json:"image_count"`
	// Presigned URL of the first photo; absent when the item has none or
	// storage is off.
	CoverImageURL *string `json:"cover_image_url,omitempty"`
}

// PublicImageOutput is a photo of a listed item: a view URL and its place in
// the sequence, without the storage key or bucket.
type PublicImageOutput struct {
	ID           string `json:"id"`
	URL          string `json:"url" doc:"Presigned GET URL, valid briefly"`
	DisplayOrder int    `json:"display_order"`
}

type listPublicItemsInput struct {
	Limit    int    `query:"limit" default:"24" minimum:"1" maximum:"60" doc:"Page size"`
	Cursor   string `query:"cursor" maxLength:"40" doc:"next_cursor from the previous page"`
	Query    string `query:"query" maxLength:"200" doc:"Substring match against item name"`
	Category string `query:"category" maxLength:"200"`
	Label    string `query:"label" maxLength:"200" doc:"Label name"`
}

type listPublicItemsOutput struct {
	Body struct {
		Items []*PublicItemOutput `json:"items"`
		// NextCursor is absent on the last page.
		NextCursor string `json:"next_cursor,omitempty"`
	}
}

type publicFiltersOutput struct {
	Body struct {
		Categories []string `json:"categories"`
		Labels     []string `json:"labels"`
	}
}

type listPublicImagesOutput struct {
	Body []*PublicImageOutput `json:"body"`
}

// RegisterPublicCatalog registers the unauthenticated read-only catalog
// routes. They are outside /admin, so the Cloudflare Access guard does not
// apply: everything they return must be safe for anyone to read.
func RegisterPublicCatalog(api huma.API, a *app.Application) {
	huma.Register(api, huma.Operation{
		OperationID: "list-public-items",
		Method:      http.MethodGet,
		Path:        "/public/items",
		Summary:     "List the items currently for sale",
	}, func(ctx context.Context, in *listPublicItemsInput) (*listPublicItemsOutput, error) {
		page, err := a.Catalog(ctx, domain.CatalogQuery{
			Limit:    in.Limit,
			Cursor:   in.Cursor,
			Query:    in.Query,
			Category: in.Category,
			Label:    in.Label,
		})
		if err != nil {
			return nil, toHTTP(err)
		}

		out := &listPublicItemsOutput{}
		out.Body.NextCursor = page.NextCursor
		out.Body.Items = make([]*PublicItemOutput, 0, len(page.Items))
		for _, v := range page.Items {
			out.Body.Items = append(out.Body.Items, publicItemOutput(v))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-public-filters",
		Method:      http.MethodGet,
		Path:        "/public/filters",
		Summary:     "Categories and labels present among the items for sale",
	}, func(ctx context.Context, _ *struct{}) (*publicFiltersOutput, error) {
		categories, labels, err := a.CatalogFacets(ctx)
		if err != nil {
			return nil, toHTTP(err)
		}
		out := &publicFiltersOutput{}
		out.Body.Categories = categories
		out.Body.Labels = labels
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-public-item-images",
		Method:      http.MethodGet,
		Path:        "/public/items/{id}/images",
		Summary:     "Photos of an item that is for sale",
	}, func(ctx context.Context, in *itemIDInput) (*listPublicImagesOutput, error) {
		views, err := a.CatalogImages(ctx, in.ID)
		if err != nil {
			return nil, toHTTP(err)
		}
		out := &listPublicImagesOutput{Body: make([]*PublicImageOutput, 0, len(views))}
		for _, v := range views {
			out.Body = append(out.Body, &PublicImageOutput{
				ID:           v.Image.ID,
				URL:          v.URL,
				DisplayOrder: v.Image.DisplayOrder,
			})
		}
		return out, nil
	})
}

// publicItemOutput maps an item to its public shape. Only the fields named
// here ever reach an unauthenticated caller.
func publicItemOutput(v app.ItemView) *PublicItemOutput {
	it := v.Item
	return &PublicItemOutput{
		ID:                it.ID,
		Name:              it.Name,
		Description:       it.Description,
		Category:          it.Category,
		Condition:         it.Condition,
		ListingPriceCents: it.ListingPriceCents,
		Length:            it.Length,
		Width:             it.Width,
		Height:            it.Height,
		MeasurementUnit:   string(it.MeasurementUnit),
		ExtraMeasurements: it.ExtraMeasurements,
		Labels:            it.LabelNames(),
		ImageCount:        v.ImageCount,
		CoverImageURL:     v.CoverImageURL,
	}
}
