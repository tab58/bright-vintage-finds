package ent

import (
	"context"

	"main-api/db/generated"
	"main-api/db/generated/item"
	"main-api/db/generated/itemimage"
	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.ItemImageRepository = (*ItemImageRepo)(nil)

// ItemImageRepo stores the rows describing uploaded photos.
type ItemImageRepo struct {
	client *generated.Client
}

// NewItemImageRepo returns a photo repository over the given Ent client.
func NewItemImageRepo(client *generated.Client) *ItemImageRepo {
	return &ItemImageRepo{client: client}
}

func (r *ItemImageRepo) ListForItem(ctx context.Context, itemID string) ([]domain.ItemImage, error) {
	rows, err := r.client.ItemImage.Query().
		Where(itemimage.HasItemWith(item.ID(itemID)), itemimage.DeletedAtIsNil()).
		Order(generated.Asc(itemimage.FieldDisplayOrder)).
		All(ctx)
	if err != nil {
		return nil, domain.Internal("listing item images", err)
	}
	return withItemID(rows, itemID), nil
}

func (r *ItemImageRepo) ListForItems(ctx context.Context, itemIDs []string) (map[string][]domain.ItemImage, error) {
	out := map[string][]domain.ItemImage{}
	if len(itemIDs) == 0 {
		return out, nil
	}

	// One query for the whole page rather than one per item.
	rows, err := r.client.ItemImage.Query().
		Where(itemimage.HasItemWith(item.IDIn(itemIDs...)), itemimage.DeletedAtIsNil()).
		Order(generated.Asc(itemimage.FieldDisplayOrder)).
		WithItem().
		All(ctx)
	if err != nil {
		return nil, domain.Internal("loading cover images", err)
	}

	for _, row := range rows {
		if row.Edges.Item == nil {
			continue
		}
		img := toItemImage(row)
		out[img.ItemID] = append(out[img.ItemID], img)
	}
	return out, nil
}

func (r *ItemImageRepo) ListForItemIncludingDeleted(ctx context.Context, itemID string) ([]domain.ItemImage, error) {
	rows, err := r.client.ItemImage.Query().
		Where(itemimage.HasItemWith(item.ID(itemID))).
		Order(generated.Asc(itemimage.FieldDisplayOrder)).
		All(ctx)
	if err != nil {
		return nil, domain.Internal("loading item images", err)
	}
	return withItemID(rows, itemID), nil
}

func (r *ItemImageRepo) CountForItem(ctx context.Context, itemID string) (int, error) {
	n, err := r.client.ItemImage.Query().
		Where(itemimage.HasItemWith(item.ID(itemID)), itemimage.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		return 0, domain.Internal("counting images", err)
	}
	return n, nil
}

func (r *ItemImageRepo) Create(ctx context.Context, in domain.NewItemImage) (domain.ItemImage, error) {
	row, err := r.client.ItemImage.Create().
		SetItemID(in.ItemID).
		SetUploadBucket(in.UploadBucket).
		SetUploadKey(in.UploadKey).
		SetNillableFilename(in.Filename).
		SetContentType(in.ContentType).
		SetDisplayOrder(in.DisplayOrder).
		Save(ctx)
	if err != nil {
		return domain.ItemImage{}, domain.Internal("recording image", err)
	}
	img := toItemImage(row)
	img.ItemID = in.ItemID
	return img, nil
}

func (r *ItemImageRepo) HardDeleteForItem(ctx context.Context, itemID string) error {
	if _, err := r.client.ItemImage.Delete().
		Where(itemimage.HasItemWith(item.ID(itemID))).
		Exec(ctx); err != nil {
		return domain.Internal("deleting item image rows", err)
	}
	return nil
}

// withItemID maps rows that were fetched for one known item, where the item
// edge is not loaded but the owner is known from the query.
func withItemID(rows []*generated.ItemImage, itemID string) []domain.ItemImage {
	images := make([]domain.ItemImage, 0, len(rows))
	for _, row := range rows {
		img := toItemImage(row)
		img.ItemID = itemID
		images = append(images, img)
	}
	return images
}
