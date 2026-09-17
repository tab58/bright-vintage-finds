package app

import (
	"context"
	"testing"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports/fakes"
)

// harness wires an Application over in-memory ports, so a test can drive the
// use cases without Postgres or S3 and still assert on what reached the
// repositories.
type harness struct {
	items  *fakes.FakeItemRepo
	images *fakes.FakeItemImageRepo
	places *fakes.FakeSellingPlaceRepo
	labels *fakes.FakeLabelRepo
	owner  *fakes.FakeOwnerRepo
	store  *fakes.FakeImageStore

	app *Application
}

// newHarness builds a harness with object storage configured.
func newHarness(t *testing.T) *harness {
	t.Helper()
	return build(fakes.NewImageStore("uploads"))
}

// newHarnessWithoutStorage builds a harness for a deployment that has no
// object storage, which is a supported configuration: the server still boots
// and the inventory still works, only without photos.
func newHarnessWithoutStorage(t *testing.T) *harness {
	t.Helper()
	return build(nil)
}

func build(store *fakes.FakeImageStore) *harness {
	h := &harness{
		items:  fakes.NewItemRepo(),
		images: fakes.NewItemImageRepo(),
		places: fakes.NewSellingPlaceRepo(),
		labels: fakes.NewLabelRepo(),
		owner:  fakes.NewOwnerRepo(),
		store:  store,
	}
	h.items.Places, h.items.Labels = h.places, h.labels

	cfg := Config{
		Items:  h.items,
		Images: h.images,
		Places: h.places,
		Labels: h.labels,
		Owner:  h.owner,
		Now:    fakes.Clock(),
		NewID:  fakes.IDs("obj"),
	}
	// A nil *FakeImageStore must not become a non-nil interface holding a nil
	// pointer: the use cases check the port for nil to decide whether storage
	// exists at all.
	if store != nil {
		cfg.Store = store
	}

	h.app = New(cfg)
	return h
}

// addPhoto records a photo row for an item, as an upload would.
func (h *harness) addPhoto(itemID, key string, order int) domain.ItemImage {
	return h.images.Add(domain.ItemImage{
		ItemID:       itemID,
		UploadBucket: "uploads",
		UploadKey:    key,
		DisplayOrder: order,
	})
}

func ptr[T any](v T) *T { return &v }

func TestNewFillsTheOptionalPorts(t *testing.T) {
	t.Run("a config without a clock, id generator or store gets the defaults", func(t *testing.T) {
		a := New(Config{Items: fakes.NewItemRepo(), Images: fakes.NewItemImageRepo()})

		if a.now == nil || a.now().IsZero() {
			t.Error("clock not defaulted to the system clock")
		}
		if a.newID == nil || a.newID() == "" {
			t.Error("id generator not defaulted")
		}
		if a.store != nil {
			t.Error("store set from a config that carried none")
		}
	})

	t.Run("a config that names them keeps them", func(t *testing.T) {
		a := New(Config{Store: fakes.NewImageStore("uploads"), Now: fakes.Clock(), NewID: fakes.IDs("obj")})

		if got := a.now(); !got.Equal(fakes.FixedTime) {
			t.Errorf("now = %v, want %v", got, fakes.FixedTime)
		}
		if got := a.newID(); got != "obj1" {
			t.Errorf("newID = %q, want %q", got, "obj1")
		}
		if !a.HasImageStore() {
			t.Error("HasImageStore = false with a store configured")
		}
	})
}

func TestHasImageStore(t *testing.T) {
	if !newHarness(t).app.HasImageStore() {
		t.Error("HasImageStore = false with storage configured")
	}
	if newHarnessWithoutStorage(t).app.HasImageStore() {
		t.Error("HasImageStore = true without storage")
	}

	// The HTTP adapter asks before it knows whether there is an application:
	// a deployment without a database has none at all.
	var missing *Application
	if missing.HasImageStore() {
		t.Error("HasImageStore = true on a nil application")
	}
}

func TestLabelCRUD(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	empty, err := h.app.ListLabels(ctx)
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("labels = %d, want 0", len(empty))
	}

	created, err := h.app.CreateLabel(ctx, "glassware")
	if err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}
	if created.Name != "glassware" || created.ID == "" {
		t.Fatalf("created = %+v, want a named label with an id", created)
	}

	listed, err := h.app.ListLabels(ctx)
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("labels = %d, want 1", len(listed))
	}

	if err := h.app.DeleteLabel(ctx, created.ID); err != nil {
		t.Fatalf("DeleteLabel: %v", err)
	}
	afterDelete, err := h.app.ListLabels(ctx)
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	if len(afterDelete) != 0 {
		t.Errorf("a soft-deleted label is still listed: %+v", afterDelete)
	}

	for _, id := range []string{"", "lbl-missing"} {
		if err := h.app.DeleteLabel(ctx, id); err == nil {
			t.Errorf("DeleteLabel(%q) = nil, want an error", id)
		}
	}
}

func TestSellingPlaceCRUD(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	empty, err := h.app.ListSellingPlaces(ctx)
	if err != nil {
		t.Fatalf("ListSellingPlaces: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("places = %d, want 0", len(empty))
	}

	created, err := h.app.CreateSellingPlace(ctx, "Whatnot")
	if err != nil {
		t.Fatalf("CreateSellingPlace: %v", err)
	}
	if created.Name != "Whatnot" || created.ID == "" {
		t.Fatalf("created = %+v, want a named place with an id", created)
	}

	listed, err := h.app.ListSellingPlaces(ctx)
	if err != nil {
		t.Fatalf("ListSellingPlaces: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("places = %d, want 1", len(listed))
	}

	if err := h.app.DeleteSellingPlace(ctx, created.ID); err != nil {
		t.Fatalf("DeleteSellingPlace: %v", err)
	}
	afterDelete, err := h.app.ListSellingPlaces(ctx)
	if err != nil {
		t.Fatalf("ListSellingPlaces: %v", err)
	}
	if len(afterDelete) != 0 {
		t.Errorf("a soft-deleted place is still listed: %+v", afterDelete)
	}

	for _, id := range []string{"", "place-missing"} {
		if err := h.app.DeleteSellingPlace(ctx, id); err == nil {
			t.Errorf("DeleteSellingPlace(%q) = nil, want an error", id)
		}
	}
}

func TestSeedBuiltinPlacesIsIdempotent(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	// Boot runs the seed every time, so the second pass must add nothing.
	for pass := 1; pass <= 2; pass++ {
		if err := h.app.SeedBuiltinPlaces(ctx); err != nil {
			t.Fatalf("SeedBuiltinPlaces pass %d: %v", pass, err)
		}
		places, err := h.app.ListSellingPlaces(ctx)
		if err != nil {
			t.Fatalf("ListSellingPlaces: %v", err)
		}
		if len(places) != len(domain.BuiltinSellingPlaces) {
			t.Fatalf("pass %d: places = %d, want %d", pass, len(places), len(domain.BuiltinSellingPlaces))
		}
	}
}
