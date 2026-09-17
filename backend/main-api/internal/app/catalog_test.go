package app

import (
	"context"
	"errors"
	"testing"

	"main-api/internal/app/domain"
)

// seedCatalog lists n items named one..n, oldest first, and returns them.
func seedCatalog(t *testing.T, h *harness, names ...string) []ItemView {
	t.Helper()
	ctx := context.Background()
	out := make([]ItemView, 0, len(names))
	for _, name := range names {
		v, err := h.app.CreateItem(ctx, ItemInput{Name: name, Status: ptr("listed")})
		if err != nil {
			t.Fatalf("seeding %s: %v", name, err)
		}
		out = append(out, v)
	}
	return out
}

func TestCatalogShowsOnlyListedItems(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	place := h.places.Add("eBay")

	seedCatalog(t, h, "for sale")
	h.app.CreateItem(ctx, ItemInput{Name: "draft"})
	archived, _ := h.app.CreateItem(ctx, ItemInput{Name: "archived", Status: ptr("archived")})
	sold, _ := h.app.CreateItem(ctx, ItemInput{Name: "sold", Status: ptr("listed")})
	h.app.MarkItemSold(ctx, sold.Item.ID, domain.Sale{SoldPlaceID: place.ID})
	_ = archived

	page, err := h.app.Catalog(ctx, domain.CatalogQuery{})
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Item.Name != "for sale" {
		t.Errorf("catalog = %v, want only the listed item", viewNames(page.Items))
	}
}

func TestCatalogPagesWithACursor(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	seedCatalog(t, h, "one", "two", "three")

	first, err := h.app.Catalog(ctx, domain.CatalogQuery{Limit: 2})
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if got := viewNames(first.Items); len(got) != 2 || got[0] != "three" || got[1] != "two" {
		t.Fatalf("first page = %v, want three, two", got)
	}
	if first.NextCursor == "" {
		t.Fatal("a full page with more behind it must carry a cursor")
	}

	second, _ := h.app.Catalog(ctx, domain.CatalogQuery{Limit: 2, Cursor: first.NextCursor})
	if got := viewNames(second.Items); len(got) != 1 || got[0] != "one" {
		t.Fatalf("second page = %v, want one", got)
	}
	if second.NextCursor != "" {
		t.Errorf("last page carries a cursor %q, so the client would keep asking", second.NextCursor)
	}
}

func TestCatalogPageExactlyFullHasNoCursor(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	seedCatalog(t, h, "one", "two")

	// Two items and a page size of two: the extra row the service asks for
	// comes back empty, so there is nothing more to fetch.
	page, err := h.app.Catalog(ctx, domain.CatalogQuery{Limit: 2})
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor != "" {
		t.Errorf("page = %v cursor = %q, want both items and no cursor", viewNames(page.Items), page.NextCursor)
	}
}

func TestCatalogBatchesCoverLookups(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	items := seedCatalog(t, h, "one", "two", "three")
	for _, v := range items {
		h.addPhoto(v.Item.ID, "items/"+v.Item.ID+"/a.jpg", 0)
		h.addPhoto(v.Item.ID, "items/"+v.Item.ID+"/b.jpg", 1)
	}
	h.images.ListForItemCalls, h.images.ListForItemsCalls = 0, 0

	page, err := h.app.Catalog(ctx, domain.CatalogQuery{})
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("page = %v", viewNames(page.Items))
	}
	for _, v := range page.Items {
		if v.ImageCount != 2 {
			t.Errorf("%s image count = %d, want 2", v.Item.Name, v.ImageCount)
		}
		if v.CoverImageURL == nil {
			t.Errorf("%s has no cover", v.Item.Name)
		}
	}

	// A public page can be large, so its covers come from one query, not one
	// per item.
	if h.images.ListForItemsCalls != 1 || h.images.ListForItemCalls != 0 {
		t.Errorf("lookups: batched=%d per-item=%d, want 1 and 0",
			h.images.ListForItemsCalls, h.images.ListForItemCalls)
	}
}

func TestCatalogWithoutStorageReportsNoPhotos(t *testing.T) {
	ctx := context.Background()
	h := newHarnessWithoutStorage(t)
	items := seedCatalog(t, h, "one")
	h.addPhoto(items[0].Item.ID, "items/x/a.jpg", 0)

	page, err := h.app.Catalog(ctx, domain.CatalogQuery{})
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	// Unlike the admin list, the catalog reports no count either: without a
	// bucket it never loads the photo rows at all.
	if page.Items[0].ImageCount != 0 || page.Items[0].CoverImageURL != nil {
		t.Errorf("count = %d cover = %v, want neither", page.Items[0].ImageCount, page.Items[0].CoverImageURL)
	}
}

func TestCatalogFiltersByCategoryAndLabel(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	glass := h.labels.Add("Glassware")

	h.app.CreateItem(ctx, ItemInput{Name: "vase", Status: ptr("listed"), Category: ptr("Decor"), LabelIDs: []string{glass.ID}})
	h.app.CreateItem(ctx, ItemInput{Name: "chair", Status: ptr("listed"), Category: ptr("Furniture")})

	byCategory, _ := h.app.Catalog(ctx, domain.CatalogQuery{Category: "Decor"})
	if got := viewNames(byCategory.Items); len(got) != 1 || got[0] != "vase" {
		t.Errorf("category filter = %v", got)
	}

	byLabel, _ := h.app.Catalog(ctx, domain.CatalogQuery{Label: "Glassware"})
	if got := viewNames(byLabel.Items); len(got) != 1 || got[0] != "vase" {
		t.Errorf("label filter = %v", got)
	}

	byQuery, _ := h.app.Catalog(ctx, domain.CatalogQuery{Query: "CHAIR"})
	if got := viewNames(byQuery.Items); len(got) != 1 || got[0] != "chair" {
		t.Errorf("query filter = %v", got)
	}
}

func TestCatalogFacetsComeFromListedItemsOnly(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	shown, hidden := h.labels.Add("Shown"), h.labels.Add("Hidden")

	h.app.CreateItem(ctx, ItemInput{Name: "a", Status: ptr("listed"), Category: ptr("Decor"), LabelIDs: []string{shown.ID}})
	h.app.CreateItem(ctx, ItemInput{Name: "b", Category: ptr("Draftonly"), LabelIDs: []string{hidden.ID}})

	categories, labels, err := h.app.CatalogFacets(ctx)
	if err != nil {
		t.Fatalf("CatalogFacets: %v", err)
	}
	// A draft-only category would advertise a filter that returns nothing.
	if len(categories) != 1 || categories[0] != "Decor" {
		t.Errorf("categories = %v", categories)
	}
	if len(labels) != 1 || labels[0] != "Shown" {
		t.Errorf("labels = %v", labels)
	}
}

func TestCatalogImagesAreGatedByStatus(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	listed := seedCatalog(t, h, "for sale")[0]
	h.addPhoto(listed.Item.ID, "items/l/b.jpg", 1)
	h.addPhoto(listed.Item.ID, "items/l/a.jpg", 0)

	got, err := h.app.CatalogImages(ctx, listed.Item.ID)
	if err != nil {
		t.Fatalf("CatalogImages: %v", err)
	}
	if len(got) != 2 || got[0].Image.UploadKey != "items/l/a.jpg" {
		t.Errorf("photos = %v, want display order", got)
	}
	if got[0].URL != "https://storage.test/uploads/items/l/a.jpg" {
		t.Errorf("url = %q", got[0].URL)
	}

	// The lightbox must not be a back door to photos of an unlisted item.
	draft, _ := h.app.CreateItem(ctx, ItemInput{Name: "draft"})
	h.addPhoto(draft.Item.ID, "items/d/a.jpg", 0)
	if _, err := h.app.CatalogImages(ctx, draft.Item.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("draft photos = %v, want not found", err)
	}
	if _, err := h.app.CatalogImages(ctx, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown item = %v, want not found", err)
	}
}

func TestCatalogImagesWithoutStorageIsEmptyNotNil(t *testing.T) {
	ctx := context.Background()
	h := newHarnessWithoutStorage(t)
	listed := seedCatalog(t, h, "for sale")[0]
	h.addPhoto(listed.Item.ID, "items/l/a.jpg", 0)

	got, err := h.app.CatalogImages(ctx, listed.Item.ID)
	if err != nil {
		t.Fatalf("CatalogImages: %v", err)
	}
	// The wire contract is an empty array, never null.
	if got == nil || len(got) != 0 {
		t.Errorf("photos = %v, want an empty slice", got)
	}
}
