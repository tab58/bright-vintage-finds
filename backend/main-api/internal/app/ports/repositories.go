// Package ports declares what the application core needs from the outside
// world. Every interface here is defined by its consumer — the services — and
// implemented by a driven adapter in internal/app/adapters. Nothing in this
// package knows about Ent, S3 or HTTP.
package ports

import (
	"context"

	"main-api/internal/app/domain"
)

// ItemRepository stores inventory items. Implementations return
// domain.NotFound for a missing or soft-deleted row and domain.Conflict for a
// unique-constraint collision; every other failure is wrapped as
// domain.Internal.
type ItemRepository interface {
	// Create stores a new item and returns it with its edges loaded.
	Create(ctx context.Context, in domain.NewItem) (domain.Item, error)

	// Get returns one live item with selling places, labels and sold place
	// loaded.
	Get(ctx context.Context, id string) (domain.Item, error)

	// List returns live items newest first, narrowed by f.
	List(ctx context.Context, f domain.ItemFilter) ([]domain.Item, error)

	// ListListed returns listed items for the public catalog, newest first by
	// KSUID id, starting after q.Cursor. It returns up to q.Limit rows and
	// does no paging arithmetic: the caller asks for one more row than the
	// page holds to learn whether another page exists.
	ListListed(ctx context.Context, q domain.CatalogQuery) ([]domain.Item, error)

	// ListedExists reports whether id names a live, listed item. The public
	// photo route gates on it: photos of an unlisted item stay private.
	ListedExists(ctx context.Context, id string) (bool, error)

	// ListedFacets returns the category values and label names present among
	// listed items, each sorted ascending.
	ListedFacets(ctx context.Context) (categories, labels []string, err error)

	// StatusState reads the fields a status change depends on. It matches the
	// update path, which looks the row up by id alone: a soft-deleted item is
	// still found here, and the write that follows fails on its own terms.
	StatusState(ctx context.Context, id string) (domain.ItemStatusState, error)

	// LiveStatusState reads the same fields for a row that must still be
	// live, as deletion requires.
	LiveStatusState(ctx context.Context, id string) (domain.ItemStatusState, error)

	// Update applies a partial change and returns the item with edges loaded.
	Update(ctx context.Context, id string, p domain.ItemPatch) (domain.Item, error)

	// MarkSold records a sale and returns the item with edges loaded.
	MarkSold(ctx context.Context, id string, sale domain.Sale) (domain.Item, error)

	// SoftDelete stamps deleted_at, hiding the item from every read path.
	SoftDelete(ctx context.Context, id string) error
}

// ItemImageRepository stores the rows describing uploaded photos. The objects
// themselves live behind ImageStore.
type ItemImageRepository interface {
	// ListForItem returns an item's live photos in display order.
	ListForItem(ctx context.Context, itemID string) ([]domain.ItemImage, error)

	// ListForItems returns live photos for several items in one query, keyed
	// by item id and in display order within each item. A catalog page reads
	// its covers this way rather than one query per item.
	ListForItems(ctx context.Context, itemIDs []string) (map[string][]domain.ItemImage, error)

	// ListForItemIncludingDeleted returns every photo row of an item, soft
	// deleted ones included, because deleting an item must clear the objects
	// it once had as well as the ones it still shows.
	ListForItemIncludingDeleted(ctx context.Context, itemID string) ([]domain.ItemImage, error)

	// CountForItem counts an item's live photos, which decides the next
	// display order.
	CountForItem(ctx context.Context, itemID string) (int, error)

	// Create records an uploaded photo.
	Create(ctx context.Context, in domain.NewItemImage) (domain.ItemImage, error)

	// HardDeleteForItem removes an item's photo rows outright. The rows go
	// after the objects, so nothing is left in the bucket with no record of
	// what it was.
	HardDeleteForItem(ctx context.Context, itemID string) error
}

// LabelRepository stores item-type labels.
type LabelRepository interface {
	List(ctx context.Context) ([]domain.Label, error)
	Create(ctx context.Context, name string) (domain.Label, error)
	SoftDelete(ctx context.Context, id string) error
	// AllExist reports whether every id names a live label.
	AllExist(ctx context.Context, ids []string) (bool, error)
}

// SellingPlaceRepository stores the platforms and venues items are sold on.
type SellingPlaceRepository interface {
	List(ctx context.Context) ([]domain.SellingPlace, error)
	Create(ctx context.Context, name string) (domain.SellingPlace, error)
	SoftDelete(ctx context.Context, id string) error
	// Exists reports whether id names a live selling place.
	Exists(ctx context.Context, id string) (bool, error)
	// AllExist reports whether every id names a live selling place.
	AllExist(ctx context.Context, ids []string) (bool, error)
	// EnsureBuiltins inserts the builtin places idempotently, keyed on the
	// unique name. Places the owner soft-deleted are not resurrected.
	EnsureBuiltins(ctx context.Context, names []string) error
}

// OwnerRepository resolves the single owner the inventory belongs to.
type OwnerRepository interface {
	// EnsureBuiltinOwner returns the owner's user id, creating the well-known
	// row on first use.
	EnsureBuiltinOwner(ctx context.Context) (string, error)
}
