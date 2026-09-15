//go:build integration

// Integration tests for the inventory admin API. They run against the local
// Docker Postgres (task up) — the same stack the service runs against — and
// skip cleanly when the database is unreachable.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	db_platform "main-api/db"
	"main-api/db/generated/item"
	"main-api/db/generated/itemimage"
	"main-api/db/generated/label"
	"main-api/db/generated/sellingplace"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
)

// testAPI wires the real item routes onto huma's test adapter, so status
// transitions are exercised through the handlers the PWA calls.
func testAPI(t *testing.T, client *db_platform.Client) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	registerItemCRUD(api, &AppDeps{DB: client})
	registerSellingPlaces(api, client)
	registerLabels(api, client)
	return api
}

// decodeItem reads an ItemOutput out of a recorded response body.
func decodeItem(t *testing.T, body []byte) ItemOutput {
	t.Helper()
	var out ItemOutput
	require.NoError(t, json.Unmarshal(body, &out))
	return out
}

const localDBURL = "postgres://postgres:postgres@localhost:5432/maindb?sslmode=disable"

func testDB(t *testing.T) *db_platform.Client {
	t.Helper()
	url := os.Getenv("MAIN_DB_URL")
	if url == "" {
		url = localDBURL
	}
	client, err := db_platform.NewClient(db_platform.ClientConfig{ConnectionString: url})
	require.NoError(t, err, "cannot open DB; is the local docker stack up? (task up)")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Raw().PingContext(ctx); err != nil {
		t.Skipf("database unreachable, skipping integration test: %v", err)
	}
	return client
}

// ptr is a small generic pointer helper for test literals.
func ptr[T any](v T) *T { return &v }

func ptrTime(t time.Time) *time.Time { return &t }

// truncate wipes the inventory tables between tests. Single-user dev DB.
func truncate(t *testing.T, client *db_platform.Client) {
	t.Helper()
	_, err := client.Raw().Exec(`TRUNCATE items, item_images, labels, item_labels, selling_places, item_selling_places, users CASCADE`)
	require.NoError(t, err)
}

func TestSeedBuiltinSellingPlacesIdempotent(t *testing.T) {
	client := testDB(t)
	truncate(t, client)

	require.NoError(t, SeedBuiltinSellingPlaces(client))
	require.NoError(t, SeedBuiltinSellingPlaces(client)) // second run: no duplicates

	places, err := client.GetDBFromContext(context.Background()).SellingPlace.Query().All(context.Background())
	require.NoError(t, err)
	require.Len(t, places, len(BuiltinSellingPlaces))
	for _, p := range places {
		require.True(t, p.IsBuiltin)
	}
}

func TestSellingPlaceCRUD(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	db := client

	created, err := db.GetDBFromContext(ctx).SellingPlace.Create().SetName("Rust & Gold Antiques").Save(ctx)
	require.NoError(t, err)

	// Duplicate name → constraint error.
	_, err = db.GetDBFromContext(ctx).SellingPlace.Create().SetName("Rust & Gold Antiques").Save(ctx)
	require.Error(t, err)

	// Soft delete hides it from the default filter (the mixin is field-only;
	// ent's DeleteOne would hard-delete, so set deleted_at explicitly).
	_, err = db.GetDBFromContext(ctx).SellingPlace.Update().
		Where(sellingplace.IDEQ(created.ID)).
		SetDeletedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	_, err = db.GetDBFromContext(ctx).SellingPlace.Query().
		Where(sellingplace.IDEQ(created.ID), sellingplace.DeletedAtIsNil()).
		First(ctx)
	require.Error(t, err, "soft-deleted place should not be listed")
}

func TestItemLifecycleWithEdgesAndSearch(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	db := client
	gen := db.GetDBFromContext(ctx)

	require.NoError(t, SeedBuiltinSellingPlaces(client))
	place, err := gen.SellingPlace.Query().Where(sellingplace.NameEQ("Whatnot")).First(ctx)
	require.NoError(t, err)

	lbl, err := gen.Label.Create().SetName("Glassware").Save(ctx)
	require.NoError(t, err)

	ownerID, err := ensureOwner(ctx, gen)
	require.NoError(t, err)

	// Create an item wired to the place + label.
	it, err := gen.Item.Create().
		SetName("Fenton cobalt vase").
		SetOwnerID(ownerID).
		SetNillableAcquisitionCostCents(ptr(int64(2500))).
		SetNillablePurchasedAt(ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))).
		SetLength(8.5).SetMeasurementUnit(item.MeasurementUnitInch).
		SetWeightLbs(2).SetWeightOz(6.0).
		SetNillableWhatnotNumber(ptr("WH-123")).
		AddSellingPlaceIDs(place.ID).
		AddLabelIDs(lbl.ID).
		Save(ctx)
	require.NoError(t, err)

	// Round-trip with eager edges.
	loaded, err := itemLoaded(ctx, db, it.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Edges.SellingPlaces, 1)
	require.Len(t, loaded.Edges.Labels, 1)

	// Search: by name substring.
	found, err := gen.Item.Query().Where(item.DeletedAtIsNil(), item.NameContainsFold("fenton")).All(ctx)
	require.NoError(t, err)
	require.Len(t, found, 1)

	// Search: by selling place.
	byPlace, err := gen.Item.Query().
		Where(item.HasSellingPlacesWith(sellingplace.IDEQ(place.ID))).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, byPlace, 1)

	// Search: by Whatnot number.
	byWhatnot, err := gen.Item.Query().Where(item.WhatnotNumberEQ("WH-123")).All(ctx)
	require.NoError(t, err)
	require.Len(t, byWhatnot, 1)

	// Search: by label.
	byLabel, err := gen.Item.Query().Where(item.HasLabelsWith(label.IDEQ(lbl.ID))).All(ctx)
	require.NoError(t, err)
	require.Len(t, byLabel, 1)

	// Mark sold.
	sold, err := gen.Item.UpdateOneID(it.ID).
		SetStatus(item.StatusSold).
		SetSoldAt(time.Now()).
		SetSoldPriceCents(8000).
		SetSoldPlaceID(place.ID).
		Save(ctx)
	require.NoError(t, err)
	require.Equal(t, item.StatusSold, sold.Status)
	require.NotNil(t, sold.SoldPriceCents)
	require.Equal(t, int64(8000), *sold.SoldPriceCents)
}

func TestWhatnotNumberUniqueWhenSet(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)
	ownerID, err := ensureOwner(ctx, gen)
	require.NoError(t, err)

	_, err = gen.Item.Create().SetName("A").SetOwnerID(ownerID).SetWhatnotNumber("WN-1").Save(ctx)
	require.NoError(t, err)

	// Same whatnot number → constraint error.
	_, err = gen.Item.Create().SetName("B").SetOwnerID(ownerID).SetWhatnotNumber("WN-1").Save(ctx)
	require.Error(t, err)

	// NULL whatnot numbers are allowed repeatedly.
	for i := 0; i < 3; i++ {
		_, err = gen.Item.Create().SetName(fmt.Sprintf("no-whatnot-%d", i)).SetOwnerID(ownerID).Save(ctx)
		require.NoError(t, err)
	}
}

func TestEnsureOwnerIdempotent(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)

	id1, err := ensureOwner(ctx, gen)
	require.NoError(t, err)
	id2, err := ensureOwner(ctx, gen)
	require.NoError(t, err)
	require.Equal(t, id1, id2)

	users, err := gen.User.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, users, 1, "exactly one owner row must exist")
}

// Empty strings from the PWA mean "clear this field": they must land as NULL,
// not "", because whatnot_number is unique when present.
func TestClearFieldsWithEmptyString(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)

	owner, err := ensureOwner(ctx, gen)
	require.NoError(t, err)

	made := make([]string, 0, 2)
	for i, wn := range []string{"214", "215"} {
		it, err := gen.Item.Create().
			SetName(fmt.Sprintf("bowl %d", i)).
			SetOwnerID(owner).
			SetNotes("chip on rim").
			SetWhatnotNumber(wn).
			Save(ctx)
		require.NoError(t, err)
		made = append(made, it.ID)
	}

	for _, id := range made {
		body := itemBody{Name: "bowl", Notes: ptr(""), WhatnotNumber: ptr("")}
		updated, err := applyItemClears(gen.Item.UpdateOneID(id).
			SetName(body.Name).
			SetNillableNotes(omitEmpty(body.Notes)).
			SetNillableWhatnotNumber(omitEmpty(body.WhatnotNumber)), body).
			Save(ctx)
		require.NoError(t, err, "clearing both items must not collide on whatnot_number")
		require.Nil(t, updated.Notes)
		require.Nil(t, updated.WhatnotNumber)
	}

	cleared, err := gen.Item.Query().Where(item.WhatnotNumberIsNil()).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cleared)
}

// The PWA drives the flow with PATCH status; listed_at times the current
// listing. Transitions themselves are not guarded server-side by design.
func TestItemStatusFlow(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	api := testAPI(t, client)

	created := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "jadeite mug"}).Body.Bytes())
	require.Equal(t, "draft", created.Status)
	require.Nil(t, created.ListedAt)

	// draft -> listed stamps listed_at
	listed := decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
		"name": created.Name, "status": "listed",
	}).Body.Bytes())
	require.Equal(t, "listed", listed.Status)
	require.NotNil(t, listed.ListedAt)

	// re-sending the same status must not restart the listing clock
	again := decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
		"name": created.Name, "status": "listed",
	}).Body.Bytes())
	require.NotNil(t, again.ListedAt)
	require.Equal(t, listed.ListedAt.UnixNano(), again.ListedAt.UnixNano())

	// unlist -> back to draft, listed_at cleared
	unlisted := decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
		"name": created.Name, "status": "draft",
	}).Body.Bytes())
	require.Equal(t, "draft", unlisted.Status)
	require.Nil(t, unlisted.ListedAt)

	// archive from draft, then restore
	archived := decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
		"name": created.Name, "status": "archived",
	}).Body.Bytes())
	require.Equal(t, "archived", archived.Status)
	restored := decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
		"name": created.Name, "status": "draft",
	}).Body.Bytes())
	require.Equal(t, "draft", restored.Status)

	// archive from listed too
	api.Patch("/admin/items/"+created.ID, map[string]any{"name": created.Name, "status": "listed"})
	fromListed := decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
		"name": created.Name, "status": "archived",
	}).Body.Bytes())
	require.Equal(t, "archived", fromListed.Status)

	// the one server-side guard: the status value itself
	bad := api.Patch("/admin/items/"+created.ID, map[string]any{"name": created.Name, "status": "nonsense"})
	require.Equal(t, 422, bad.Code)
}

// A mis-keyed sale must be fixable without reopening the item.
func TestSaleCorrections(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	api := testAPI(t, client)
	require.NoError(t, SeedBuiltinSellingPlaces(client))

	var places []SellingPlaceOutput
	require.NoError(t, json.Unmarshal(api.Get("/admin/selling-places").Body.Bytes(), &places))
	require.NotEmpty(t, places)
	first, second := places[0], places[1]

	created := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "camp blanket"}).Body.Bytes())
	sold := decodeItem(t, api.Post("/admin/items/"+created.ID+"/sold", map[string]any{
		"sold_at":          time.Now().UTC().Format(time.RFC3339),
		"sold_price_cents": 4500,
		"sold_place_id":    first.ID,
	}).Body.Bytes())
	require.Equal(t, "sold", sold.Status)
	require.Equal(t, int64(4500), *sold.SoldPriceCents)

	fixed := decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
		"name":             created.Name,
		"sold_price_cents": 5200,
		"sold_place_id":    second.ID,
	}).Body.Bytes())
	require.Equal(t, int64(5200), *fixed.SoldPriceCents)
	require.Equal(t, second.ID, *fixed.SoldPlaceID)
	require.Equal(t, "sold", fixed.Status, "correcting a sale must not change the status")

	unknown := api.Patch("/admin/items/"+created.ID, map[string]any{
		"name": created.Name, "sold_place_id": "does-not-exist",
	})
	require.Equal(t, 400, unknown.Code)
}

// Huma answers 204 with header fields when a response struct has no Body, which
// silently gave the PWA an undefined place/label after every create.
func TestCreateReturnsJSONBody(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	api := testAPI(t, client)

	place := api.Post("/admin/selling-places", map[string]any{"name": "Pickup"})
	require.Equal(t, 200, place.Code)
	var createdPlace SellingPlaceOutput
	require.NoError(t, json.Unmarshal(place.Body.Bytes(), &createdPlace))
	require.NotEmpty(t, createdPlace.ID)
	require.Equal(t, "Pickup", createdPlace.Name)

	label := api.Post("/admin/labels", map[string]any{"name": "Glassware"})
	require.Equal(t, 200, label.Code)
	var createdLabel LabelOutput
	require.NoError(t, json.Unmarshal(label.Body.Bytes(), &createdLabel))
	require.NotEmpty(t, createdLabel.ID)
	require.Equal(t, "Glassware", createdLabel.Name)

	// The created place is usable straight away as a sale destination.
	item := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "hobnail vase"}).Body.Bytes())
	sold := decodeItem(t, api.Post("/admin/items/"+item.ID+"/sold", map[string]any{
		"sold_at":          time.Now().UTC().Format(time.RFC3339),
		"sold_price_cents": 2400,
		"sold_place_id":    createdPlace.ID,
	}).Body.Bytes())
	require.Equal(t, "sold", sold.Status)
	require.Equal(t, "Pickup", *sold.SoldPlace)
}

// Sold is the one status that carries data: an item cannot be sold nowhere,
// or the sales-by-platform numbers quietly lose rows.
func TestSoldRequiresPlace(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	api := testAPI(t, client)
	require.NoError(t, SeedBuiltinSellingPlaces(client))

	var places []SellingPlaceOutput
	require.NoError(t, json.Unmarshal(api.Get("/admin/selling-places").Body.Bytes(), &places))
	place := places[0]

	created := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "hobnail vase"}).Body.Bytes())

	// no place anywhere: refused
	bare := api.Patch("/admin/items/"+created.ID, map[string]any{"name": created.Name, "status": "sold"})
	require.Equal(t, 400, bare.Code)

	// place in the same payload: allowed
	sold := decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
		"name": created.Name, "status": "sold", "sold_place_id": place.ID,
	}).Body.Bytes())
	require.Equal(t, "sold", sold.Status)
	require.Equal(t, place.ID, *sold.SoldPlaceID)

	// place already on the row: re-sending sold is fine
	again := api.Patch("/admin/items/"+created.ID, map[string]any{"name": created.Name, "status": "sold"})
	require.Equal(t, 200, again.Code)

	// and it cannot be created sold without one either
	badCreate := api.Post("/admin/items", map[string]any{"name": "born sold", "status": "sold"})
	require.Equal(t, 400, badCreate.Code)
}

// The sold_place FK is RESTRICT: a place with sales against it cannot be
// hard-deleted, so past sales can never lose the platform they sold on.
func TestSoldPlaceCannotBeHardDeleted(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	api := testAPI(t, client)
	require.NoError(t, SeedBuiltinSellingPlaces(client))

	var places []SellingPlaceOutput
	require.NoError(t, json.Unmarshal(api.Get("/admin/selling-places").Body.Bytes(), &places))
	place := places[0]

	item := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "copper mould"}).Body.Bytes())
	sold := decodeItem(t, api.Post("/admin/items/"+item.ID+"/sold", map[string]any{
		"sold_at":          time.Now().UTC().Format(time.RFC3339),
		"sold_price_cents": 2400,
		"sold_place_id":    place.ID,
	}).Body.Bytes())
	require.Equal(t, place.ID, *sold.SoldPlaceID)

	_, err := client.Raw().Exec(`DELETE FROM selling_places WHERE id = $1`, place.ID)
	require.Error(t, err, "deleting a place with sales must be refused by the FK")

	// The API's soft delete still works and leaves the sale intact.
	require.Equal(t, 204, api.Delete("/admin/selling-places/"+place.ID).Code)
	after := decodeItem(t, api.Get("/admin/items/"+item.ID).Body.Bytes())
	require.Equal(t, place.ID, *after.SoldPlaceID)
}

// first_listed_at answers "how long from listing to sale", so it must survive
// an unlist/relist cycle that resets listed_at.
func TestFirstListedAtSurvivesRelist(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	api := testAPI(t, client)

	created := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "camp blanket"}).Body.Bytes())
	patch := func(status string) ItemOutput {
		return decodeItem(t, api.Patch("/admin/items/"+created.ID, map[string]any{
			"name": created.Name, "status": status,
		}).Body.Bytes())
	}

	listed := patch("listed")
	require.NotNil(t, listed.FirstListedAt)
	first := *listed.FirstListedAt

	unlisted := patch("draft")
	require.Nil(t, unlisted.ListedAt, "unlisting clears the current listing")
	require.NotNil(t, unlisted.FirstListedAt, "but not the first one")

	relisted := patch("listed")
	require.NotNil(t, relisted.ListedAt)
	require.Equal(t, first.UnixNano(), relisted.FirstListedAt.UnixNano(), "first listing is written once")
}

// stubStore is a minimal aws_s3.Client: only presigning matters here, and a
// local stub avoids pulling gomock into this module for one call.
type stubStore struct {
	url     string
	deleted *[]string
}

func (s stubStore) UploadFile(context.Context, string, string, io.Reader) error { return nil }
func (s stubStore) UploadFileWithMetadata(context.Context, string, string, io.Reader, map[string]string) error {
	return nil
}
func (s stubStore) DownloadFile(context.Context, string, string) (io.ReadCloser, error) {
	return nil, nil
}
func (s stubStore) DeleteFile(_ context.Context, _, key string) error {
	if s.deleted != nil {
		*s.deleted = append(*s.deleted, key)
	}
	return nil
}
func (s stubStore) FileExists(context.Context, string, string) (bool, error) { return true, nil }
func (s stubStore) Ping(context.Context, string) error                       { return nil }
func (s stubStore) PresignGetObject(_ context.Context, _, key string, _ time.Duration) (string, error) {
	return s.url + "/" + key, nil
}

// The inventory list shows a thumbnail per item, so item rows carry a presigned
// cover URL rather than the client fetching images row by row.
func TestListCarriesCoverImageURL(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)

	deps := &AppDeps{DB: client, Store: stubStore{url: "https://storage.example"}, S3UploadBucket: "bucket"}
	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	registerItemCRUD(api, deps)

	owner, err := ensureOwner(ctx, gen)
	require.NoError(t, err)
	it, err := gen.Item.Create().SetName("jadeite mug").SetOwnerID(owner).Save(ctx)
	require.NoError(t, err)

	// display_order decides the cover, not insertion order.
	_, err = gen.ItemImage.Create().
		SetItemID(it.ID).SetUploadBucket("bucket").SetUploadKey("items/second.jpg").SetDisplayOrder(1).Save(ctx)
	require.NoError(t, err)
	_, err = gen.ItemImage.Create().
		SetItemID(it.ID).SetUploadBucket("bucket").SetUploadKey("items/cover.jpg").SetDisplayOrder(0).Save(ctx)
	require.NoError(t, err)

	var listed []ItemOutput
	require.NoError(t, json.Unmarshal(api.Get("/admin/items").Body.Bytes(), &listed))
	require.Len(t, listed, 1)
	require.Equal(t, 2, listed[0].ImageCount)
	require.NotNil(t, listed[0].CoverImageURL)
	require.Equal(t, "https://storage.example/items/cover.jpg", *listed[0].CoverImageURL)
}

// Without object storage the list still works; it just has no thumbnails.
func TestListWithoutStorageHasNoCover(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	api := testAPI(t, client)

	created := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "no photos"}).Body.Bytes())
	require.Nil(t, created.CoverImageURL)
	require.Equal(t, 0, created.ImageCount)
}

// Drafts that never went out can be deleted outright; anything that reached
// listed is part of the record and must be archived instead.
func TestDeleteOnlyNeverListedDrafts(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	api := testAPI(t, client)

	draft := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "mis-typed item"}).Body.Bytes())
	require.Equal(t, 204, api.Delete("/admin/items/"+draft.ID).Code)

	var remaining []ItemOutput
	require.NoError(t, json.Unmarshal(api.Get("/admin/items").Body.Bytes(), &remaining))
	require.Empty(t, remaining, "a deleted draft leaves the list")
	require.Equal(t, 404, api.Get("/admin/items/"+draft.ID).Code)
	require.Equal(t, 404, api.Delete("/admin/items/"+draft.ID).Code, "deleting twice is not found")

	// listed, then unlisted: back in draft but no longer deletable
	listed := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "real stock"}).Body.Bytes())
	api.Patch("/admin/items/"+listed.ID, map[string]any{"name": listed.Name, "status": "listed"})
	unlisted := decodeItem(t, api.Patch("/admin/items/"+listed.ID, map[string]any{
		"name": listed.Name, "status": "draft",
	}).Body.Bytes())
	require.Equal(t, "draft", unlisted.Status)
	require.NotNil(t, unlisted.FirstListedAt)
	require.Equal(t, 409, api.Delete("/admin/items/"+listed.ID).Code)

	// ...but archiving it is the deliberate step that makes disposal allowed.
	api.Patch("/admin/items/"+listed.ID, map[string]any{"name": listed.Name, "status": "archived"})
	require.Equal(t, 204, api.Delete("/admin/items/"+listed.ID).Code)

	// A sale is never deleted, archived or not.
	require.NoError(t, SeedBuiltinSellingPlaces(client))
	var places []SellingPlaceOutput
	require.NoError(t, json.Unmarshal(api.Get("/admin/selling-places").Body.Bytes(), &places))
	sold := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "sold stock"}).Body.Bytes())
	api.Post("/admin/items/"+sold.ID+"/sold", map[string]any{
		"sold_at":          time.Now().UTC().Format(time.RFC3339),
		"sold_price_cents": 1000,
		"sold_place_id":    places[0].ID,
	})
	require.Equal(t, 409, api.Delete("/admin/items/"+sold.ID).Code)
}

// Deleting a draft takes its pictures with it: the stored objects are removed,
// not just the rows, so the bucket does not fill with orphans.
func TestDeleteDraftRemovesImages(t *testing.T) {
	client := testDB(t)
	truncate(t, client)
	ctx := context.Background()
	gen := client.GetDBFromContext(ctx)

	var deleted []string
	deps := &AppDeps{DB: client, Store: stubStore{url: "https://storage.example", deleted: &deleted}, S3UploadBucket: "bucket"}
	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	registerItemCRUD(api, deps)

	created := decodeItem(t, api.Post("/admin/items", map[string]any{"name": "junk intake"}).Body.Bytes())
	for _, key := range []string{"items/a.jpg", "items/b.jpg"} {
		_, err := gen.ItemImage.Create().
			SetItemID(created.ID).SetUploadBucket("bucket").SetUploadKey(key).SetDisplayOrder(0).Save(ctx)
		require.NoError(t, err)
	}

	require.Equal(t, 204, api.Delete("/admin/items/"+created.ID).Code)
	require.ElementsMatch(t, []string{"items/a.jpg", "items/b.jpg"}, deleted, "both objects removed from storage")

	rows, err := gen.ItemImage.Query().Where(itemimage.HasItemWith(item.ID(created.ID))).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rows, "image rows go with the objects")
}
