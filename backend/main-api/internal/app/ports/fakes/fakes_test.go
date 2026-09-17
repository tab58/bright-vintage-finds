package fakes

import (
	"context"
	"errors"
	"strings"
	"testing"

	"main-api/internal/app/domain"
)

// The fakes stand in for the database in every service test, so the rules they
// reproduce are worth checking on their own: a fake that quietly disagrees
// with the real adapter turns a green service test into a false negative.

func TestItemRepoWhatnotNumberIsUnique(t *testing.T) {
	ctx := context.Background()
	repo := NewItemRepo()

	if _, err := repo.Create(ctx, domain.NewItem{Name: "a", WhatnotNumber: ptr("WN-1")}); err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err := repo.Create(ctx, domain.NewItem{Name: "b", WhatnotNumber: ptr("WN-1")})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second create err = %v, want conflict", err)
	}
	if err.Error() != "an item with this Whatnot number already exists" {
		t.Errorf("message = %q", err.Error())
	}

	// A number is only taken while some row holds it; empty stays free.
	if _, err := repo.Create(ctx, domain.NewItem{Name: "c"}); err != nil {
		t.Fatalf("create without a number: %v", err)
	}
	if _, err := repo.Create(ctx, domain.NewItem{Name: "d"}); err != nil {
		t.Fatalf("second create without a number: %v", err)
	}
}

func TestItemRepoListIsNewestFirstAndFiltered(t *testing.T) {
	ctx := context.Background()
	places, labels := NewSellingPlaceRepo(), NewLabelRepo()
	repo := NewItemRepo()
	repo.Places, repo.Labels = places, labels

	glass := labels.Add("Glassware")
	shop := places.Add("Local store")

	older, _ := repo.Create(ctx, domain.NewItem{Name: "Old vase", LabelIDs: []string{glass.ID}})
	newer, _ := repo.Create(ctx, domain.NewItem{Name: "New lamp", SellingPlaceIDs: []string{shop.ID}})

	all, err := repo.List(ctx, domain.ItemFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 || all[0].ID != newer.ID {
		t.Fatalf("List order = %v, want newest first", ids(all))
	}

	byName, _ := repo.List(ctx, domain.ItemFilter{Query: "VASE"})
	if len(byName) != 1 || byName[0].ID != older.ID {
		t.Errorf("name filter is case-insensitive substring, got %v", ids(byName))
	}

	byLabel, _ := repo.List(ctx, domain.ItemFilter{LabelID: glass.ID})
	if len(byLabel) != 1 || byLabel[0].ID != older.ID {
		t.Errorf("label filter = %v", ids(byLabel))
	}
	if got := byLabel[0].LabelNames(); len(got) != 1 || got[0] != "Glassware" {
		t.Errorf("edges should hydrate to names, got %v", got)
	}

	byPlace, _ := repo.List(ctx, domain.ItemFilter{PlaceID: shop.ID})
	if len(byPlace) != 1 || byPlace[0].ID != newer.ID {
		t.Errorf("place filter = %v", ids(byPlace))
	}

	draft := domain.StatusDraft
	byStatus, _ := repo.List(ctx, domain.ItemFilter{Status: &draft})
	if len(byStatus) != 2 {
		t.Errorf("both items are drafts, got %d", len(byStatus))
	}
}

func TestItemRepoSoftDeleteHidesFromReads(t *testing.T) {
	ctx := context.Background()
	repo := NewItemRepo()
	it, _ := repo.Create(ctx, domain.NewItem{Name: "gone"})

	if err := repo.SoftDelete(ctx, it.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	if _, err := repo.Get(ctx, it.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get after delete = %v, want not found", err)
	}
	if got, _ := repo.List(ctx, domain.ItemFilter{}); len(got) != 0 {
		t.Errorf("List after delete = %v, want empty", ids(got))
	}
	if _, err := repo.LiveStatusState(ctx, it.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("LiveStatusState after delete = %v, want not found", err)
	}
	// The update path looks a row up by id alone, so a soft-deleted item is
	// still visible there. The asymmetry is deliberate: it mirrors the query
	// the handler issues today.
	if _, err := repo.StatusState(ctx, it.ID); err != nil {
		t.Errorf("StatusState after delete = %v, want the row", err)
	}
}

func TestItemRepoUpdateAppliesTheTriState(t *testing.T) {
	ctx := context.Background()
	repo := NewItemRepo()
	it, _ := repo.Create(ctx, domain.NewItem{
		Name:          "lamp",
		Notes:         ptr("chipped"),
		WhatnotNumber: ptr("WN-9"),
		Category:      ptr("Lighting"),
	})

	// Leave: nothing named, nothing changes.
	got, err := repo.Update(ctx, it.ID, domain.ItemPatch{Name: "lamp"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Notes == nil || *got.Notes != "chipped" {
		t.Errorf("notes = %v, want unchanged", got.Notes)
	}

	// Set, then clear.
	got, _ = repo.Update(ctx, it.ID, domain.ItemPatch{
		Name:  "lamp",
		Notes: domain.Clearable(ptr("repaired")),
	})
	if got.Notes == nil || *got.Notes != "repaired" {
		t.Errorf("notes = %v, want repaired", got.Notes)
	}

	got, _ = repo.Update(ctx, it.ID, domain.ItemPatch{
		Name:          "lamp",
		Notes:         domain.Clearable(ptr("")),
		WhatnotNumber: domain.Clearable(ptr("")),
	})
	if got.Notes != nil || got.WhatnotNumber != nil {
		t.Errorf("clear left notes=%v whatnot=%v, want nil", got.Notes, got.WhatnotNumber)
	}
	if got.Category == nil || *got.Category != "Lighting" {
		t.Errorf("unmentioned category = %v, want untouched", got.Category)
	}

	if repo.LastPatch.Name != "lamp" {
		t.Errorf("LastPatch should record what the caller asked for")
	}
}

func TestItemRepoUpdateReplacesEdgeSetsWholesale(t *testing.T) {
	ctx := context.Background()
	labels := NewLabelRepo()
	repo := NewItemRepo()
	repo.Labels = labels
	a, b := labels.Add("A"), labels.Add("B")

	it, _ := repo.Create(ctx, domain.NewItem{Name: "x", LabelIDs: []string{a.ID}})

	// A nil slice leaves the set alone.
	got, _ := repo.Update(ctx, it.ID, domain.ItemPatch{Name: "x"})
	if len(got.Labels) != 1 {
		t.Errorf("nil label list should leave the set alone, got %v", got.LabelNames())
	}

	got, _ = repo.Update(ctx, it.ID, domain.ItemPatch{Name: "x", LabelIDs: []string{b.ID}})
	if len(got.Labels) != 1 || got.Labels[0].ID != b.ID {
		t.Errorf("labels = %v, want only B", got.LabelNames())
	}

	// An empty, non-nil slice clears the set.
	got, _ = repo.Update(ctx, it.ID, domain.ItemPatch{Name: "x", LabelIDs: []string{}})
	if len(got.Labels) != 0 {
		t.Errorf("labels = %v, want none", got.LabelNames())
	}
}

func TestItemRepoCatalogPagingWalksTheCursor(t *testing.T) {
	ctx := context.Background()
	labels := NewLabelRepo()
	repo := NewItemRepo()
	repo.Labels = labels
	glass := labels.Add("Glassware")

	listed := domain.StatusListed
	for _, name := range []string{"one", "two", "three", "four"} {
		if _, err := repo.Create(ctx, domain.NewItem{
			Name: name, Status: &listed, Category: ptr("Decor"), LabelIDs: []string{glass.ID},
		}); err != nil {
			t.Fatalf("seeding %s: %v", name, err)
		}
	}
	// A draft must never reach the catalog.
	if _, err := repo.Create(ctx, domain.NewItem{Name: "hidden"}); err != nil {
		t.Fatalf("seeding draft: %v", err)
	}

	first, err := repo.ListListed(ctx, domain.CatalogQuery{Limit: 2})
	if err != nil {
		t.Fatalf("ListListed: %v", err)
	}
	if len(first) != 2 || first[0].Name != "four" || first[1].Name != "three" {
		t.Fatalf("first page = %v, want four, three", names(first))
	}

	second, _ := repo.ListListed(ctx, domain.CatalogQuery{Limit: 2, Cursor: first[1].ID})
	if len(second) != 2 || second[0].Name != "two" || second[1].Name != "one" {
		t.Fatalf("second page = %v, want two, one", names(second))
	}

	last, _ := repo.ListListed(ctx, domain.CatalogQuery{Limit: 2, Cursor: second[1].ID})
	if len(last) != 0 {
		t.Errorf("past the end = %v, want empty", names(last))
	}

	byLabel, _ := repo.ListListed(ctx, domain.CatalogQuery{Limit: 10, Label: "Glassware"})
	if len(byLabel) != 4 {
		t.Errorf("label filter = %d items, want 4", len(byLabel))
	}
	byCategory, _ := repo.ListListed(ctx, domain.CatalogQuery{Limit: 10, Category: "Nope"})
	if len(byCategory) != 0 {
		t.Errorf("unknown category = %d items, want 0", len(byCategory))
	}
}

func TestItemRepoListedFacetsCoverListedItemsOnly(t *testing.T) {
	ctx := context.Background()
	labels := NewLabelRepo()
	repo := NewItemRepo()
	repo.Labels = labels
	shown, hidden := labels.Add("Shown"), labels.Add("Hidden")

	listed := domain.StatusListed
	repo.Create(ctx, domain.NewItem{Name: "a", Status: &listed, Category: ptr("Zebra"), LabelIDs: []string{shown.ID}})
	repo.Create(ctx, domain.NewItem{Name: "b", Status: &listed, Category: ptr("Apple")})
	repo.Create(ctx, domain.NewItem{Name: "c", Category: ptr("Draftonly"), LabelIDs: []string{hidden.ID}})

	categories, labelNames, err := repo.ListedFacets(ctx)
	if err != nil {
		t.Fatalf("ListedFacets: %v", err)
	}
	if len(categories) != 2 || categories[0] != "Apple" || categories[1] != "Zebra" {
		t.Errorf("categories = %v, want Apple, Zebra sorted", categories)
	}
	if len(labelNames) != 1 || labelNames[0] != "Shown" {
		t.Errorf("labels = %v, want only Shown", labelNames)
	}
}

func TestItemImageRepoOrdersByDisplayOrder(t *testing.T) {
	ctx := context.Background()
	repo := NewItemImageRepo()
	repo.Add(domain.ItemImage{ItemID: "i1", UploadKey: "b", DisplayOrder: 1})
	repo.Add(domain.ItemImage{ItemID: "i1", UploadKey: "a", DisplayOrder: 0})
	repo.Add(domain.ItemImage{ItemID: "i2", UploadKey: "c", DisplayOrder: 0})

	got, err := repo.ListForItem(ctx, "i1")
	if err != nil {
		t.Fatalf("ListForItem: %v", err)
	}
	if len(got) != 2 || got[0].UploadKey != "a" {
		t.Errorf("photos = %v, want display order", keys(got))
	}

	batch, _ := repo.ListForItems(ctx, []string{"i1", "i2", "i3"})
	if len(batch["i1"]) != 2 || len(batch["i2"]) != 1 {
		t.Errorf("batch = %v", batch)
	}
	if _, ok := batch["i3"]; ok {
		t.Error("an item with no photos should be absent from the batch, not empty")
	}

	if n, _ := repo.CountForItem(ctx, "i1"); n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}

func TestItemImageRepoDeletedRowsStayVisibleToTheDeletePath(t *testing.T) {
	ctx := context.Background()
	repo := NewItemImageRepo()
	gone := repo.Add(domain.ItemImage{ItemID: "i1", UploadKey: "old", DisplayOrder: 0})
	repo.Add(domain.ItemImage{ItemID: "i1", UploadKey: "new", DisplayOrder: 1})
	repo.SoftDelete(gone.ID)

	live, _ := repo.ListForItem(ctx, "i1")
	if len(live) != 1 {
		t.Errorf("live photos = %v, want one", keys(live))
	}
	// Deleting an item must clear objects it once had, so this path sees them.
	all, _ := repo.ListForItemIncludingDeleted(ctx, "i1")
	if len(all) != 2 {
		t.Errorf("all photos = %v, want two", keys(all))
	}

	if err := repo.HardDeleteForItem(ctx, "i1"); err != nil {
		t.Fatalf("HardDeleteForItem: %v", err)
	}
	if len(repo.Remaining()) != 0 {
		t.Errorf("rows remain after hard delete: %v", keys(repo.Remaining()))
	}
}

func TestSellingPlaceEnsureBuiltinsIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := NewSellingPlaceRepo()

	if err := repo.EnsureBuiltins(ctx, domain.BuiltinSellingPlaces); err != nil {
		t.Fatalf("EnsureBuiltins: %v", err)
	}
	first, _ := repo.List(ctx)
	if len(first) != len(domain.BuiltinSellingPlaces) {
		t.Fatalf("seeded %d places, want %d", len(first), len(domain.BuiltinSellingPlaces))
	}

	if err := repo.EnsureBuiltins(ctx, domain.BuiltinSellingPlaces); err != nil {
		t.Fatalf("second EnsureBuiltins: %v", err)
	}
	second, _ := repo.List(ctx)
	if len(second) != len(first) {
		t.Errorf("re-seeding duplicated places: %d then %d", len(first), len(second))
	}

	// A place the owner deleted stays deleted.
	if err := repo.SoftDelete(ctx, second[0].ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if err := repo.EnsureBuiltins(ctx, domain.BuiltinSellingPlaces); err != nil {
		t.Fatalf("third EnsureBuiltins: %v", err)
	}
	third, _ := repo.List(ctx)
	if len(third) != len(second)-1 {
		t.Errorf("seeding resurrected a deleted place: %d, want %d", len(third), len(second)-1)
	}
}

func TestFakeErrorInjection(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("boom")

	items := NewItemRepo()
	items.Err = boom
	if _, err := items.Get(ctx, "x"); !errors.Is(err, boom) {
		t.Errorf("item repo err = %v, want boom", err)
	}

	store := NewImageStore("bucket")
	store.PresignErr = boom
	if _, err := store.ViewURL(ctx, "bucket", "key", domain.PresignTTL); !errors.Is(err, boom) {
		t.Errorf("store err = %v, want boom", err)
	}
	// Uploads still work when only presigning is broken.
	if err := store.Upload(ctx, "bucket", "key", strings.NewReader("bytes")); err != nil {
		t.Fatalf("Upload = %v, want nil", err)
	}
	if got := string(store.Objects["bucket/key"]); got != "bytes" {
		t.Errorf("stored %q, want the whole body", got)
	}
}

func ptr[T any](v T) *T { return &v }

func ids(items []domain.Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

func names(items []domain.Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func keys(images []domain.ItemImage) []string {
	out := make([]string, 0, len(images))
	for _, img := range images {
		out = append(out, img.UploadKey)
	}
	return out
}
