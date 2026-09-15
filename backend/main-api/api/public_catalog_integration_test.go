//go:build integration

// Integration tests for the unauthenticated /public catalog routes. They run
// against the local Docker Postgres (task up), like the admin API tests.
package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	db_platform "main-api/db"
	"main-api/db/generated"
	"main-api/db/generated/item"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
)

// publicAPI wires the real public catalog routes onto huma's test adapter.
func publicAPI(t *testing.T, client *db_platform.Client, store bool) humatest.TestAPI {
	t.Helper()
	deps := &AppDeps{DB: client}
	if store {
		deps.Store = stubStore{url: "https://storage.example"}
		deps.S3UploadBucket = "bucket"
	}
	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	registerPublicCatalog(api, deps)
	return api
}

// publicPage is the decoded /public/items response.
type publicPage struct {
	Items      []*PublicItemOutput `json:"items"`
	NextCursor string              `json:"next_cursor"`
}

func getPublicPage(t *testing.T, api humatest.TestAPI, path string) publicPage {
	t.Helper()
	resp := api.Get(path)
	require.Equal(t, 200, resp.Code, path)
	var page publicPage
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &page))
	return page
}

// newItem creates an item row directly, bypassing the admin handlers so a
// sold/archived fixture needs no sale ceremony.
func newItem(t *testing.T, gen *generated.Client, ownerID, name string, status item.Status) *generated.Item {
	t.Helper()
	ctx := context.Background()
	it, err := gen.Item.Create().
		SetName(name).
		SetOwnerID(ownerID).
		SetStatus(status).
		Save(ctx)
	require.NoError(t, err)
	return it
}

// Only listed, non-deleted items are public, and the public shape carries no
// private business data.
func TestPublicItemsOnlyShowsListed(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)
	owner, err := ensureOwner(ctx, gen)
	require.NoError(t, err)

	newItem(t, gen, owner, "draft lamp", item.StatusDraft)
	newItem(t, gen, owner, "sold vase", item.StatusSold)
	newItem(t, gen, owner, "archived chair", item.StatusArchived)
	gone := newItem(t, gen, owner, "deleted mirror", item.StatusListed)
	_, err = gen.Item.UpdateOneID(gone.ID).SetDeletedAt(time.Now()).Save(ctx)
	require.NoError(t, err)

	live, err := gen.Item.UpdateOneID(newItem(t, gen, owner, "brass teapot", item.StatusListed).ID).
		SetListingPriceCents(4500).
		SetAcquisitionCostCents(1200).
		SetNotes("haggled at the flea market").
		SetWhatnotNumber("WN-9").
		Save(ctx)
	require.NoError(t, err)

	api := publicAPI(t, client, true)
	page := getPublicPage(t, api, "/public/items")
	require.Len(t, page.Items, 1, "only the listed, undeleted item is public")
	require.Equal(t, live.ID, page.Items[0].ID)
	require.Equal(t, int64(4500), *page.Items[0].ListingPriceCents)
	require.Empty(t, page.NextCursor, "one page, no cursor")
	require.Nil(t, page.Items[0].CoverImageURL, "no photos yet")
	require.Equal(t, 0, page.Items[0].ImageCount)

	// The private columns must not reach the wire under any key.
	var raw struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(api.Get("/public/items").Body.Bytes(), &raw))
	for _, private := range []string{
		"acquisition_cost_cents", "purchased_at", "notes", "whatnot_number",
		"selling_places", "selling_place_ids", "sold_price_cents", "sold_at",
		"sold_place", "status",
	} {
		require.NotContains(t, raw.Items[0], private, "private field leaked")
	}
}

// The catalog can grow, so the grid pages by cursor: every item shows up once.
func TestPublicItemsPagination(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)
	owner, err := ensureOwner(ctx, gen)
	require.NoError(t, err)

	const total = 5
	for i := range total {
		newItem(t, gen, owner, string(rune('a'+i))+" piece", item.StatusListed)
	}

	api := publicAPI(t, client, false)
	seen := map[string]bool{}
	path := "/public/items?limit=2"
	pages := 0
	for {
		page := getPublicPage(t, api, path)
		pages++
		require.LessOrEqual(t, len(page.Items), 2)
		for _, it := range page.Items {
			require.False(t, seen[it.ID], "item %s repeated across pages", it.ID)
			seen[it.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		require.Less(t, pages, 10, "cursor is not advancing")
		path = "/public/items?limit=2&cursor=" + page.NextCursor
	}
	require.Len(t, seen, total, "pagination lost or duplicated items")
	require.Equal(t, 3, pages, "5 items at 2 per page")
}

// query / category / label narrow the result set server-side, because the
// client only ever holds one page.
func TestPublicItemsFilters(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)
	owner, err := ensureOwner(ctx, gen)
	require.NoError(t, err)

	pyrex, err := gen.Label.Create().SetName("pyrex").Save(ctx)
	require.NoError(t, err)

	bowl := newItem(t, gen, owner, "pyrex mixing bowl", item.StatusListed)
	_, err = gen.Item.UpdateOneID(bowl.ID).SetCategory("kitchen").AddLabelIDs(pyrex.ID).Save(ctx)
	require.NoError(t, err)

	lamp := newItem(t, gen, owner, "brass floor lamp", item.StatusListed)
	_, err = gen.Item.UpdateOneID(lamp.ID).SetCategory("lighting").Save(ctx)
	require.NoError(t, err)

	api := publicAPI(t, client, false)

	byQuery := getPublicPage(t, api, "/public/items?query=BRASS")
	require.Len(t, byQuery.Items, 1)
	require.Equal(t, lamp.ID, byQuery.Items[0].ID)

	byCategory := getPublicPage(t, api, "/public/items?category=kitchen")
	require.Len(t, byCategory.Items, 1)
	require.Equal(t, bowl.ID, byCategory.Items[0].ID)

	byLabel := getPublicPage(t, api, "/public/items?label=pyrex")
	require.Len(t, byLabel.Items, 1)
	require.Equal(t, bowl.ID, byLabel.Items[0].ID)
	require.Equal(t, []string{"pyrex"}, byLabel.Items[0].Labels)
}

// The lightbox reads photos by item id; unlisted photos must stay private.
func TestPublicItemImagesGatedByStatus(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)
	owner, err := ensureOwner(ctx, gen)
	require.NoError(t, err)

	listed := newItem(t, gen, owner, "oak side table", item.StatusListed)
	draft := newItem(t, gen, owner, "unfinished intake", item.StatusDraft)
	for _, seed := range []struct {
		itemID, key string
		order       int
	}{
		{listed.ID, "items/second.jpg", 1},
		{listed.ID, "items/cover.jpg", 0},
		{draft.ID, "items/secret.jpg", 0},
	} {
		_, err := gen.ItemImage.Create().
			SetItemID(seed.itemID).SetUploadBucket("bucket").
			SetUploadKey(seed.key).SetDisplayOrder(seed.order).Save(ctx)
		require.NoError(t, err)
	}

	api := publicAPI(t, client, true)

	resp := api.Get("/public/items/" + listed.ID + "/images")
	require.Equal(t, 200, resp.Code)
	var imgs []*PublicImageOutput
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &imgs))
	require.Len(t, imgs, 2)
	require.Equal(t, "https://storage.example/items/cover.jpg", imgs[0].URL, "ordered by display_order")

	// The storage key and bucket are internal.
	var rawImgs []map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &rawImgs))
	require.NotContains(t, rawImgs[0], "upload_key")
	require.NotContains(t, rawImgs[0], "upload_bucket")

	require.Equal(t, 404, api.Get("/public/items/"+draft.ID+"/images").Code, "draft photos are private")
	require.Equal(t, 404, api.Get("/public/items/nosuchid/images").Code)

	// The grid's cover URL comes from the same photos.
	page := getPublicPage(t, api, "/public/items")
	require.Len(t, page.Items, 1)
	require.Equal(t, 2, page.Items[0].ImageCount)
	require.Equal(t, "https://storage.example/items/cover.jpg", *page.Items[0].CoverImageURL)
}

// The filter lists advertise only what a visitor can actually find.
func TestPublicFiltersFromListedItemsOnly(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)
	owner, err := ensureOwner(ctx, gen)
	require.NoError(t, err)

	public, err := gen.Label.Create().SetName("mid-century").Save(ctx)
	require.NoError(t, err)
	hidden, err := gen.Label.Create().SetName("needs-repair").Save(ctx)
	require.NoError(t, err)

	shown := newItem(t, gen, owner, "walnut credenza", item.StatusListed)
	_, err = gen.Item.UpdateOneID(shown.ID).SetCategory("furniture").AddLabelIDs(public.ID).Save(ctx)
	require.NoError(t, err)

	draft := newItem(t, gen, owner, "cracked ashtray", item.StatusDraft)
	_, err = gen.Item.UpdateOneID(draft.ID).SetCategory("smoking").AddLabelIDs(hidden.ID).Save(ctx)
	require.NoError(t, err)

	resp := publicAPI(t, client, false).Get("/public/filters")
	require.Equal(t, 200, resp.Code)
	var filters struct {
		Categories []string `json:"categories"`
		Labels     []string `json:"labels"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &filters))
	require.Equal(t, []string{"furniture"}, filters.Categories)
	require.Equal(t, []string{"mid-century"}, filters.Labels)
}
