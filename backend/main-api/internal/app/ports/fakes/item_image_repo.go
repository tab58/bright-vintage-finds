package fakes

import (
	"context"
	"sort"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.ItemImageRepository = (*FakeItemImageRepo)(nil)

// FakeItemImageRepo is an in-memory ports.ItemImageRepository.
type FakeItemImageRepo struct {
	Err error

	// ListForItemCalls and ListForItemsCalls count the two read shapes, so a
	// test can show the catalog batches its cover lookups instead of issuing
	// one query per item.
	ListForItemCalls  int
	ListForItemsCalls int

	ids     seq
	images  []domain.ItemImage
	deleted map[string]bool
}

// NewItemImageRepo returns an empty photo repository.
func NewItemImageRepo() *FakeItemImageRepo {
	return &FakeItemImageRepo{ids: seq{prefix: "img"}, deleted: map[string]bool{}}
}

// Add seeds a photo, assigning an id when it has none. The display order is
// the caller's to set: it decides which photo is the cover.
func (r *FakeItemImageRepo) Add(img domain.ItemImage) domain.ItemImage {
	if img.ID == "" {
		img.ID = r.ids.next()
	}
	r.images = append(r.images, img)
	return img
}

// SoftDelete marks a photo deleted, for seeding the state a deleted item
// leaves behind.
func (r *FakeItemImageRepo) SoftDelete(id string) { r.deleted[id] = true }

// Remaining returns every photo row still stored, so a test can show the
// deletion path cleared them.
func (r *FakeItemImageRepo) Remaining() []domain.ItemImage { return r.images }

func (r *FakeItemImageRepo) ListForItem(_ context.Context, itemID string) ([]domain.ItemImage, error) {
	r.ListForItemCalls++
	if r.Err != nil {
		return nil, r.Err
	}
	return r.forItem(itemID, false), nil
}

func (r *FakeItemImageRepo) ListForItems(_ context.Context, itemIDs []string) (map[string][]domain.ItemImage, error) {
	r.ListForItemsCalls++
	if r.Err != nil {
		return nil, r.Err
	}
	out := make(map[string][]domain.ItemImage, len(itemIDs))
	for _, id := range itemIDs {
		if imgs := r.forItem(id, false); len(imgs) > 0 {
			out[id] = imgs
		}
	}
	return out, nil
}

func (r *FakeItemImageRepo) ListForItemIncludingDeleted(_ context.Context, itemID string) ([]domain.ItemImage, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.forItem(itemID, true), nil
}

func (r *FakeItemImageRepo) CountForItem(_ context.Context, itemID string) (int, error) {
	if r.Err != nil {
		return 0, r.Err
	}
	return len(r.forItem(itemID, false)), nil
}

func (r *FakeItemImageRepo) Create(_ context.Context, in domain.NewItemImage) (domain.ItemImage, error) {
	if r.Err != nil {
		return domain.ItemImage{}, r.Err
	}
	contentType := in.ContentType
	img := domain.ItemImage{
		ID:           r.ids.next(),
		ItemID:       in.ItemID,
		UploadBucket: in.UploadBucket,
		UploadKey:    in.UploadKey,
		Filename:     in.Filename,
		ContentType:  &contentType,
		DisplayOrder: in.DisplayOrder,
	}
	r.images = append(r.images, img)
	return img, nil
}

func (r *FakeItemImageRepo) HardDeleteForItem(_ context.Context, itemID string) error {
	if r.Err != nil {
		return r.Err
	}
	kept := make([]domain.ItemImage, 0, len(r.images))
	for _, img := range r.images {
		if img.ItemID != itemID {
			kept = append(kept, img)
		}
	}
	r.images = kept
	return nil
}

// forItem returns an item's photos in display order.
func (r *FakeItemImageRepo) forItem(itemID string, includeDeleted bool) []domain.ItemImage {
	out := []domain.ItemImage{}
	for _, img := range r.images {
		if img.ItemID != itemID {
			continue
		}
		if !includeDeleted && r.deleted[img.ID] {
			continue
		}
		out = append(out, img)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayOrder < out[j].DisplayOrder })
	return out
}
