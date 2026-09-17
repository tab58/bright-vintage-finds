package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports/fakes"
)

func TestCreateAttributesTheItemToTheBuiltinOwner(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	if _, err := h.app.CreateItem(ctx, ItemInput{Name: "vase"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if h.owner.Calls != 1 {
		t.Errorf("owner resolved %d times, want 1", h.owner.Calls)
	}
	if h.items.LastOwnerID != h.owner.ID {
		t.Errorf("owner id = %q, want %q", h.items.LastOwnerID, h.owner.ID)
	}
}

func TestCreateDropsBlankNotesAndWhatnotNumber(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	// A blank whatnot number stored as "" would occupy the unique index and
	// collide with the next blank one, so creation drops it entirely.
	first, err := h.app.CreateItem(ctx, ItemInput{Name: "a", Notes: ptr(""), WhatnotNumber: ptr("")})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if first.Item.Notes != nil || first.Item.WhatnotNumber != nil {
		t.Errorf("blank fields stored: notes=%v whatnot=%v", first.Item.Notes, first.Item.WhatnotNumber)
	}

	if _, err := h.app.CreateItem(ctx, ItemInput{Name: "b", WhatnotNumber: ptr("")}); err != nil {
		t.Fatalf("second blank whatnot number should not collide: %v", err)
	}
}

func TestCreateStatusRules(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		in         ItemInput
		wantErr    string
		wantKind   domain.Kind
		wantListed bool
		wantStatus domain.Status
	}{
		{
			name:       "no status defaults to draft with no timestamps",
			in:         ItemInput{Name: "a"},
			wantStatus: domain.StatusDraft,
		},
		{
			name:       "created as listed is stamped as listed now",
			in:         ItemInput{Name: "a", Status: ptr("listed")},
			wantStatus: domain.StatusListed,
			wantListed: true,
		},
		{
			name:     "an unknown status is rejected with the wire message",
			in:       ItemInput{Name: "a", Status: ptr("pending")},
			wantErr:  `invalid status "pending"`,
			wantKind: domain.KindInvalid,
		},
		{
			name:     "created as sold needs a place",
			in:       ItemInput{Name: "a", Status: ptr("sold")},
			wantErr:  "a sold item needs sold_place_id",
			wantKind: domain.KindInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			got, err := h.app.CreateItem(ctx, tt.in)

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				if domain.KindOf(err) != tt.wantKind {
					t.Errorf("kind = %d, want %d", domain.KindOf(err), tt.wantKind)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if got.Item.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", got.Item.Status, tt.wantStatus)
			}
			if tt.wantListed {
				if got.Item.ListedAt == nil || !got.Item.ListedAt.Equal(fakes.FixedTime) {
					t.Errorf("listed_at = %v, want the clock's time", got.Item.ListedAt)
				}
				if got.Item.FirstListedAt == nil {
					t.Error("first_listed_at should be stamped on a first listing")
				}
			} else if got.Item.ListedAt != nil {
				t.Errorf("listed_at = %v, want none", got.Item.ListedAt)
			}
		})
	}
}

func TestCreateRejectsUnknownEdges(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	place := h.places.Add("Whatnot")

	_, err := h.app.CreateItem(ctx, ItemInput{Name: "a", SellingPlaceIDs: []string{place.ID, "nope"}})
	if err == nil || err.Error() != "unknown selling_place_ids" {
		t.Fatalf("err = %v, want unknown selling_place_ids", err)
	}

	_, err = h.app.CreateItem(ctx, ItemInput{Name: "a", LabelIDs: []string{"nope"}})
	if err == nil || err.Error() != "unknown label_ids" {
		t.Fatalf("err = %v, want unknown label_ids", err)
	}

	// Nothing was written on the way out.
	items, _ := h.app.ListItems(ctx, ItemFilterInput{})
	if len(items) != 0 {
		t.Errorf("rejected creates left %d items behind", len(items))
	}
}

func TestCreateRejectsAnUnparseableStatusBeforeCheckingEdges(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	// The handler validated in this order, and which error the client sees
	// when a payload is wrong in two ways is part of the contract.
	_, err := h.app.CreateItem(ctx, ItemInput{Name: "a", Status: ptr("bogus"), LabelIDs: []string{"nope"}})
	if err == nil || err.Error() != `invalid status "bogus"` {
		t.Fatalf("err = %v, want the status error first", err)
	}
}

func TestUpdateAppliesTheClearableTriState(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	created, _ := h.app.CreateItem(ctx, ItemInput{Name: "lamp", Notes: ptr("chipped"), WhatnotNumber: ptr("WN-1")})

	// Absent: left alone.
	got, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "lamp"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Item.Notes == nil || *got.Item.Notes != "chipped" {
		t.Errorf("notes = %v, want unchanged", got.Item.Notes)
	}
	if h.items.LastPatch.Notes.Action != domain.FieldLeave {
		t.Errorf("patch action = %d, want leave", h.items.LastPatch.Notes.Action)
	}

	// Empty string: cleared, not stored as "".
	got, err = h.app.UpdateItem(ctx, created.Item.ID, ItemInput{
		Name:          "lamp",
		Notes:         ptr(""),
		WhatnotNumber: ptr(""),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Item.Notes != nil || got.Item.WhatnotNumber != nil {
		t.Errorf("clear left notes=%v whatnot=%v", got.Item.Notes, got.Item.WhatnotNumber)
	}
	if h.items.LastPatch.WhatnotNumber.Action != domain.FieldClear {
		t.Errorf("patch action = %d, want clear", h.items.LastPatch.WhatnotNumber.Action)
	}
}

func TestUpdateStatusTimestamps(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		seedStatus   string
		next         string
		wantListedAt bool
		wantFirstAt  bool
	}{
		{name: "draft to listed stamps both", seedStatus: "", next: "listed", wantListedAt: true, wantFirstAt: true},
		{name: "listed to draft clears listed_at but keeps first", seedStatus: "listed", next: "draft", wantListedAt: false, wantFirstAt: true},
		{name: "listed to archived keeps both", seedStatus: "listed", next: "archived", wantListedAt: true, wantFirstAt: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			in := ItemInput{Name: "x"}
			if tt.seedStatus != "" {
				in.Status = ptr(tt.seedStatus)
			}
			created, err := h.app.CreateItem(ctx, in)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			got, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x", Status: ptr(tt.next)})
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if (got.Item.ListedAt != nil) != tt.wantListedAt {
				t.Errorf("listed_at set = %v, want %v", got.Item.ListedAt != nil, tt.wantListedAt)
			}
			if (got.Item.FirstListedAt != nil) != tt.wantFirstAt {
				t.Errorf("first_listed_at set = %v, want %v", got.Item.FirstListedAt != nil, tt.wantFirstAt)
			}
		})
	}
}

func TestUpdateFirstListedAtSurvivesRelisting(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	created, _ := h.app.CreateItem(ctx, ItemInput{Name: "x", Status: ptr("listed")})
	firstListed := *created.Item.FirstListedAt

	// Unlist, then list again: the original listing date is history and must
	// not move.
	if _, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x", Status: ptr("draft")}); err != nil {
		t.Fatalf("unlist: %v", err)
	}
	got, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x", Status: ptr("listed")})
	if err != nil {
		t.Fatalf("relist: %v", err)
	}
	if got.Item.FirstListedAt == nil || !got.Item.FirstListedAt.Equal(firstListed) {
		t.Errorf("first_listed_at = %v, want %v", got.Item.FirstListedAt, firstListed)
	}
	if h.items.LastPatch.FirstListedAt != nil {
		t.Error("relisting should not rewrite first_listed_at")
	}
}

func TestUpdateRejectsAnUnknownSoldPlace(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	created, _ := h.app.CreateItem(ctx, ItemInput{Name: "x"})

	_, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x", SoldPlaceID: ptr("nope")})
	if err == nil || err.Error() != "unknown sold_place_id" {
		t.Fatalf("err = %v, want unknown sold_place_id", err)
	}
}

func TestUpdateToSoldNeedsAPlaceFromEitherSide(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	place := h.places.Add("eBay")
	created, _ := h.app.CreateItem(ctx, ItemInput{Name: "x", Status: ptr("listed")})

	_, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x", Status: ptr("sold")})
	if err == nil || err.Error() != "a sold item needs sold_place_id" {
		t.Fatalf("err = %v, want the sold-place error", err)
	}

	// With the place in the same payload it goes through, and the item then
	// carries one for any later status edit.
	if _, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{
		Name: "x", Status: ptr("sold"), SoldPlaceID: ptr(place.ID),
	}); err != nil {
		t.Fatalf("sold with a place: %v", err)
	}
	if _, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x", Status: ptr("sold")}); err != nil {
		t.Errorf("re-sending sold with a stored place: %v", err)
	}
}

func TestUpdateReplacesEdgeSetsOnlyWhenSent(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	a, b := h.labels.Add("A"), h.labels.Add("B")
	created, _ := h.app.CreateItem(ctx, ItemInput{Name: "x", LabelIDs: []string{a.ID}})

	got, _ := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x"})
	if len(got.Item.Labels) != 1 {
		t.Errorf("omitting the list should leave it alone, got %v", got.Item.LabelNames())
	}

	got, _ = h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x", LabelIDs: []string{b.ID}})
	if len(got.Item.Labels) != 1 || got.Item.Labels[0].ID != b.ID {
		t.Errorf("labels = %v, want only B", got.Item.LabelNames())
	}

	_, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "x", LabelIDs: []string{"nope"}})
	if err == nil || err.Error() != "unknown label_ids" {
		t.Errorf("err = %v, want unknown label_ids", err)
	}
}

func TestMarkSoldRecordsTheSale(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	place := h.places.Add("Whatnot")
	created, _ := h.app.CreateItem(ctx, ItemInput{Name: "x", Status: ptr("listed")})

	soldAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	got, err := h.app.MarkItemSold(ctx, created.Item.ID, domain.Sale{
		SoldAt: soldAt, SoldPriceCents: 4200, SoldPlaceID: place.ID,
	})
	if err != nil {
		t.Fatalf("MarkSold: %v", err)
	}
	if got.Item.Status != domain.StatusSold {
		t.Errorf("status = %q, want sold", got.Item.Status)
	}
	if got.Item.SoldPlace == nil || got.Item.SoldPlace.Name != "Whatnot" {
		t.Errorf("sold place = %v, want Whatnot", got.Item.SoldPlace)
	}
	if got.Item.SoldPriceCents == nil || *got.Item.SoldPriceCents != 4200 {
		t.Errorf("sold price = %v", got.Item.SoldPriceCents)
	}

	_, err = h.app.MarkItemSold(ctx, created.Item.ID, domain.Sale{SoldPlaceID: "nope"})
	if err == nil || err.Error() != "unknown sold_place_id" {
		t.Errorf("err = %v, want unknown sold_place_id", err)
	}
}

func TestDeleteEnforcesTheRulesAndClearsPhotos(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	place := h.places.Add("eBay")

	draft, _ := h.app.CreateItem(ctx, ItemInput{Name: "draft"})
	h.addPhoto(draft.Item.ID, "items/d/1.jpg", 0)
	gone := h.addPhoto(draft.Item.ID, "items/d/0.jpg", 1)
	// A photo the owner already removed still has an object in the bucket.
	h.images.SoftDelete(gone.ID)

	if err := h.app.DeleteItem(ctx, draft.Item.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(h.store.Deleted) != 2 {
		t.Errorf("objects deleted = %v, want both the live and the removed photo", h.store.Deleted)
	}
	if len(h.images.Remaining()) != 0 {
		t.Error("photo rows should be gone")
	}
	if _, err := h.app.GetItem(ctx, draft.Item.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get after delete = %v, want not found", err)
	}

	listed, _ := h.app.CreateItem(ctx, ItemInput{Name: "listed", Status: ptr("listed")})
	err := h.app.DeleteItem(ctx, listed.Item.ID)
	if err == nil || err.Error() != "this item has been listed; archive it before deleting" {
		t.Errorf("deleting a listing = %v", err)
	}

	sold, _ := h.app.CreateItem(ctx, ItemInput{Name: "sold", Status: ptr("listed")})
	h.app.MarkItemSold(ctx, sold.Item.ID, domain.Sale{SoldPlaceID: place.ID})
	err = h.app.DeleteItem(ctx, sold.Item.ID)
	if err == nil || err.Error() != "sold items are a record of the sale and cannot be deleted" {
		t.Errorf("deleting a sale = %v", err)
	}

	if err := h.app.DeleteItem(ctx, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("deleting an unknown item = %v, want not found", err)
	}
}

func TestDeleteWithoutStorageStillRemovesTheRows(t *testing.T) {
	ctx := context.Background()
	h := newHarnessWithoutStorage(t)
	draft, _ := h.app.CreateItem(ctx, ItemInput{Name: "draft"})
	h.addPhoto(draft.Item.ID, "items/d/1.jpg", 0)

	if err := h.app.DeleteItem(ctx, draft.Item.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(h.images.Remaining()) != 0 {
		t.Error("photo rows should be gone even with no bucket to clear")
	}
}

func TestViewCarriesPhotoCountAndCover(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	created, _ := h.app.CreateItem(ctx, ItemInput{Name: "x"})
	h.addPhoto(created.Item.ID, "items/x/second.jpg", 1)
	h.addPhoto(created.Item.ID, "items/x/first.jpg", 0)

	got, err := h.app.GetItem(ctx, created.Item.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ImageCount != 2 {
		t.Errorf("image count = %d, want 2", got.ImageCount)
	}
	// The cover is the first photo in display order, not the first uploaded.
	if got.CoverImageURL == nil || *got.CoverImageURL != "https://storage.test/uploads/items/x/first.jpg" {
		t.Errorf("cover = %v", got.CoverImageURL)
	}
}

func TestViewWithoutStorageCountsPhotosButHasNoCover(t *testing.T) {
	ctx := context.Background()
	h := newHarnessWithoutStorage(t)
	created, _ := h.app.CreateItem(ctx, ItemInput{Name: "x"})
	h.addPhoto(created.Item.ID, "items/x/first.jpg", 0)

	got, err := h.app.GetItem(ctx, created.Item.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// The admin list counts photos from the database, which is there whether
	// or not the bucket is.
	if got.ImageCount != 1 {
		t.Errorf("image count = %d, want 1", got.ImageCount)
	}
	if got.CoverImageURL != nil {
		t.Errorf("cover = %v, want none without storage", *got.CoverImageURL)
	}
}

func TestListFiltersAndRejectsABadStatus(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	h.app.CreateItem(ctx, ItemInput{Name: "old vase"})
	h.app.CreateItem(ctx, ItemInput{Name: "new lamp", Status: ptr("listed")})

	all, err := h.app.ListItems(ctx, ItemFilterInput{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 || all[0].Item.Name != "new lamp" {
		t.Errorf("List = %v, want newest first", viewNames(all))
	}

	listed, _ := h.app.ListItems(ctx, ItemFilterInput{Status: "listed"})
	if len(listed) != 1 || listed[0].Item.Name != "new lamp" {
		t.Errorf("status filter = %v", viewNames(listed))
	}

	if _, err := h.app.ListItems(ctx, ItemFilterInput{Status: "bogus"}); err == nil ||
		err.Error() != `invalid status "bogus"` {
		t.Errorf("err = %v, want the status message", err)
	}
}

func TestRepositoryFailuresStayInternal(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	h.items.Err = errors.New("connection reset")

	_, err := h.app.ListItems(ctx, ItemFilterInput{})
	if !errors.Is(err, h.items.Err) {
		t.Fatalf("err = %v, want the cause", err)
	}
	// A bare infrastructure failure must not be reported as a client mistake.
	if domain.KindOf(err) != domain.KindInternal {
		t.Errorf("kind = %d, want internal", domain.KindOf(err))
	}
}

func viewNames(views []ItemView) []string {
	out := make([]string, 0, len(views))
	for _, v := range views {
		out = append(out, v.Item.Name)
	}
	return out
}

func TestMeasurementUnitIsParsedFromTheWire(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		unit    *string
		want    domain.MeasurementUnit
		wantErr string
	}{
		{name: "absent leaves the stored default", want: domain.UnitInch},
		{name: "inch is stored", unit: ptr("inch"), want: domain.UnitInch},
		{name: "cm is stored", unit: ptr("cm"), want: domain.UnitCm},
		{name: "an unknown unit is rejected", unit: ptr("furlong"), wantErr: "measurement_unit must be inch or cm"},
		{name: "an empty unit is rejected", unit: ptr(""), wantErr: "measurement_unit must be inch or cm"},
	}

	for _, tt := range tests {
		t.Run("create: "+tt.name, func(t *testing.T) {
			h := newHarness(t)

			v, err := h.app.CreateItem(ctx, ItemInput{Name: "vase", MeasurementUnit: tt.unit})
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Create error = %v, want %q", err, tt.wantErr)
				}
				if domain.KindOf(err) != domain.KindInvalid {
					t.Errorf("kind = %v, want invalid", domain.KindOf(err))
				}
				return
			}
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if v.Item.MeasurementUnit != tt.want {
				t.Errorf("unit = %q, want %q", v.Item.MeasurementUnit, tt.want)
			}
		})

		t.Run("update: "+tt.name, func(t *testing.T) {
			h := newHarness(t)

			created, err := h.app.CreateItem(ctx, ItemInput{Name: "vase", MeasurementUnit: ptr("cm")})
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			v, err := h.app.UpdateItem(ctx, created.Item.ID, ItemInput{Name: "vase", MeasurementUnit: tt.unit})
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("Update error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			// An absent unit leaves the stored one alone, which is cm here.
			want := tt.want
			if tt.unit == nil {
				want = domain.UnitCm
			}
			if v.Item.MeasurementUnit != want {
				t.Errorf("unit = %q, want %q", v.Item.MeasurementUnit, want)
			}
		})
	}
}
