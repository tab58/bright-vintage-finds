package api

import (
	"context"
	"fmt"
	"net/http"

	"main-api/db/generated"
	"main-api/db/generated/item"
	"main-api/db/generated/itemimage"
	"main-api/db/generated/label"

	"github.com/danielgtaylor/huma/v2"
)

// PublicItemOutput is the public projection of a listed item: what an
// anonymous visitor to the shop front page may see. It deliberately omits
// acquisition cost, notes, Whatnot numbers, selling places and sold data.
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

const (
	publicPageDefault = 24
	publicPageMax     = 60
)

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

// registerPublicCatalog registers the unauthenticated read-only catalog
// routes. They are outside /admin, so the Cloudflare Access guard does not
// apply: everything they return must be safe for anyone to read.
func registerPublicCatalog(api huma.API, deps *AppDeps) {
	client := deps.DB.GetDBFromContext(nil)

	huma.Register(api, huma.Operation{
		OperationID: "list-public-items",
		Method:      http.MethodGet,
		Path:        "/public/items",
		Summary:     "List the items currently for sale",
	}, func(ctx context.Context, in *listPublicItemsInput) (*listPublicItemsOutput, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = publicPageDefault
		}
		if limit > publicPageMax {
			limit = publicPageMax
		}

		// ponytail: KSUIDs sort by creation, so id-desc is "newest first" and
		// doubles as the pagination cursor. Ordering by listing date instead
		// would need a composite cursor.
		q := client.Item.Query().
			Where(item.DeletedAtIsNil(), item.StatusEQ(item.StatusListed)).
			WithLabels().
			Order(generated.Desc(item.FieldID)).
			Limit(limit + 1)

		if in.Cursor != "" {
			q = q.Where(item.IDLT(in.Cursor))
		}
		if in.Query != "" {
			q = q.Where(item.NameContainsFold(in.Query))
		}
		if in.Category != "" {
			q = q.Where(item.CategoryEQ(in.Category))
		}
		if in.Label != "" {
			q = q.Where(item.HasLabelsWith(label.NameEQ(in.Label), label.DeletedAtIsNil()))
		}

		items, err := q.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing public items: %w", err)
		}

		out := &listPublicItemsOutput{}
		// The extra row asked for above only tells us another page exists.
		if len(items) > limit {
			items = items[:limit]
			out.Body.NextCursor = items[limit-1].ID
		}

		covers, counts, err := publicCovers(ctx, deps, items)
		if err != nil {
			return nil, err
		}

		out.Body.Items = make([]*PublicItemOutput, 0, len(items))
		for _, it := range items {
			o := publicItemOutput(it)
			o.ImageCount = counts[it.ID]
			if url, ok := covers[it.ID]; ok {
				o.CoverImageURL = &url
			}
			out.Body.Items = append(out.Body.Items, o)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-public-filters",
		Method:      http.MethodGet,
		Path:        "/public/filters",
		Summary:     "Categories and labels present among the items for sale",
	}, func(ctx context.Context, _ *struct{}) (*publicFiltersOutput, error) {
		categories, err := client.Item.Query().
			Where(item.DeletedAtIsNil(), item.StatusEQ(item.StatusListed), item.CategoryNotNil()).
			Order(generated.Asc(item.FieldCategory)).
			GroupBy(item.FieldCategory).
			Strings(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing public categories: %w", err)
		}

		labels, err := client.Label.Query().
			Where(label.DeletedAtIsNil(), label.HasItemsWith(
				item.DeletedAtIsNil(), item.StatusEQ(item.StatusListed))).
			Order(generated.Asc(label.FieldName)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing public labels: %w", err)
		}

		out := &publicFiltersOutput{}
		out.Body.Categories = categories
		out.Body.Labels = make([]string, 0, len(labels))
		for _, l := range labels {
			out.Body.Labels = append(out.Body.Labels, l.Name)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-public-item-images",
		Method:      http.MethodGet,
		Path:        "/public/items/{id}/images",
		Summary:     "Photos of an item that is for sale",
	}, func(ctx context.Context, in *itemIDInput) (*listPublicImagesOutput, error) {
		// Unlisted photos stay private, so the item's status gates the lookup.
		listed, err := client.Item.Query().
			Where(item.ID(in.ID), item.DeletedAtIsNil(), item.StatusEQ(item.StatusListed)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("checking public item %s: %w", in.ID, err)
		}
		if !listed {
			return nil, huma.Error404NotFound("item not found")
		}

		out := &listPublicImagesOutput{Body: []*PublicImageOutput{}}
		if deps.Store == nil {
			return out, nil
		}

		imgs, err := client.ItemImage.Query().
			Where(itemimage.HasItemWith(item.ID(in.ID)), itemimage.DeletedAtIsNil()).
			Order(generated.Asc(itemimage.FieldDisplayOrder)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing public item images: %w", err)
		}
		for _, ii := range imgs {
			full, err := imageOutputWithURL(ctx, deps, ii)
			if err != nil {
				return nil, err
			}
			out.Body = append(out.Body, &PublicImageOutput{
				ID:           full.ID,
				URL:          full.URL,
				DisplayOrder: full.DisplayOrder,
			})
		}
		return out, nil
	})
}

// publicItemOutput maps a loaded item (labels eager-loaded) to its public
// shape, leaving the image fields to the caller.
func publicItemOutput(it *generated.Item) *PublicItemOutput {
	out := &PublicItemOutput{
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
		Labels:            []string{},
	}
	for _, l := range it.Edges.Labels {
		out.Labels = append(out.Labels, l.Name)
	}
	return out
}

// publicCovers returns the presigned cover URL and photo count per item ID,
// loading every photo of the page in one query rather than one query per item.
func publicCovers(ctx context.Context, deps *AppDeps, items []*generated.Item) (map[string]string, map[string]int, error) {
	counts := make(map[string]int, len(items))
	covers := make(map[string]string, len(items))
	if len(items) == 0 || deps.Store == nil {
		return covers, counts, nil
	}

	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}

	imgs, err := deps.DB.GetDBFromContext(ctx).ItemImage.Query().
		Where(itemimage.HasItemWith(item.IDIn(ids...)), itemimage.DeletedAtIsNil()).
		Order(generated.Asc(itemimage.FieldDisplayOrder)).
		WithItem().
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("loading public cover images: %w", err)
	}

	for _, ii := range imgs {
		if ii.Edges.Item == nil {
			continue
		}
		itemID := ii.Edges.Item.ID
		counts[itemID]++
		if _, seen := covers[itemID]; seen {
			continue
		}
		full, err := imageOutputWithURL(ctx, deps, ii)
		if err != nil {
			return nil, nil, err
		}
		covers[itemID] = full.URL
	}
	return covers, counts, nil
}
