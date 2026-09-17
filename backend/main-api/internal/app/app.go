// Package app is the inventory application: the use cases the HTTP adapter
// calls. Items, their photos, the public catalog's read-only projection of
// them, and the labels and selling places they hang off. It works entirely
// over the ports in internal/app/ports, so it knows nothing about Ent, S3 or
// huma — cmd/app builds the driven adapters and hands them in.
package app

import (
	"context"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports"

	"github.com/segmentio/ksuid"
)

// Config is the set of ports an Application runs over. Every field is an
// interface, so a test wires the in-memory fakes and a deployment wires the
// Ent and S3 adapters through the same constructor.
type Config struct {
	// Items stores inventory items. Required.
	Items ports.ItemRepository
	// Images stores the rows describing uploaded photos. Required.
	Images ports.ItemImageRepository
	// Labels stores item-type labels. Required.
	Labels ports.LabelRepository
	// Places stores the platforms and venues items sell on. Required.
	Places ports.SellingPlaceRepository
	// Owner resolves the single owner the inventory belongs to. Required.
	Owner ports.OwnerRepository
	// Store is object storage for photos. Optional: nil means the deployment
	// has none, and the photo use cases report that rather than failing on a
	// missing bucket.
	Store ports.ImageStore
	// Now reads the current time. Optional: defaults to ports.SystemClock.
	Now ports.Clock
	// NewID mints the opaque id in an object key. Optional: defaults to KSUID.
	NewID ports.IDGenerator
}

// Application is the inventory use cases. Build one with New; its ports are
// unexported, so every read and write goes through the methods below.
type Application struct {
	items  ports.ItemRepository
	images ports.ItemImageRepository
	labels ports.LabelRepository
	places ports.SellingPlaceRepository
	owner  ports.OwnerRepository
	// store is nil when the deployment has no object storage; every use of it
	// is guarded, and a nil store means photos have no URLs.
	store ports.ImageStore
	now   ports.Clock
	newID ports.IDGenerator
}

// New wires an Application over the ports in cfg, filling in the optional
// ones that were left nil.
func New(cfg Config) *Application {
	if cfg.Now == nil {
		cfg.Now = ports.SystemClock
	}
	if cfg.NewID == nil {
		cfg.NewID = func() string { return ksuid.New().String() }
	}
	return &Application{
		items:  cfg.Items,
		images: cfg.Images,
		labels: cfg.Labels,
		places: cfg.Places,
		owner:  cfg.Owner,
		store:  cfg.Store,
		now:    cfg.Now,
		newID:  cfg.NewID,
	}
}

// HasImageStore reports whether photos can be stored and served. The HTTP
// adapter registers the photo routes only when they can be served, and calls
// this on a nil Application when the deployment has no database either.
func (a *Application) HasImageStore() bool { return a != nil && a.store != nil }

// ListLabels returns the live labels.
func (a *Application) ListLabels(ctx context.Context) ([]domain.Label, error) {
	return a.labels.List(ctx)
}

// CreateLabel adds a label.
func (a *Application) CreateLabel(ctx context.Context, name string) (domain.Label, error) {
	return a.labels.Create(ctx, name)
}

// DeleteLabel soft deletes a label, leaving the items that carry it alone.
func (a *Application) DeleteLabel(ctx context.Context, id string) error {
	return a.labels.SoftDelete(ctx, id)
}

// ListSellingPlaces returns the live selling places.
func (a *Application) ListSellingPlaces(ctx context.Context) ([]domain.SellingPlace, error) {
	return a.places.List(ctx)
}

// CreateSellingPlace adds a selling place.
func (a *Application) CreateSellingPlace(ctx context.Context, name string) (domain.SellingPlace, error) {
	return a.places.Create(ctx, name)
}

// DeleteSellingPlace soft deletes a selling place, leaving the items sold
// there alone.
func (a *Application) DeleteSellingPlace(ctx context.Context, id string) error {
	return a.places.SoftDelete(ctx, id)
}

// SeedBuiltinPlaces inserts the builtin selling places idempotently. It runs
// at boot so a broken seed surfaces at deploy time, not when the intake page
// first loads.
func (a *Application) SeedBuiltinPlaces(ctx context.Context) error {
	return a.places.EnsureBuiltins(ctx, domain.BuiltinSellingPlaces)
}
