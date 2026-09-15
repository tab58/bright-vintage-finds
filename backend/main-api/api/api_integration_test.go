//go:build integration

// Integration tests for the inventory admin API. They run against the local
// Docker Postgres (task up) — the same stack the service runs against — and
// skip cleanly when the database is unreachable.
package api

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	db_platform "main-api/db"
	"main-api/db/generated/item"
	"main-api/db/generated/label"
	"main-api/db/generated/sellingplace"

	"github.com/stretchr/testify/require"
)

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