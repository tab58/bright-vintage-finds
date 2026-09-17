package app

import (
	"context"

	"main-api/internal/app/domain"
)

// ListItemImages returns an item's photos with fresh view URLs.
func (a *Application) ListItemImages(ctx context.Context, itemID string) ([]ImageView, error) {
	if a.store == nil {
		return nil, domain.Unavailable("object storage is not configured")
	}
	images, err := a.images.ListForItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	return a.imageViews(ctx, images)
}

// UploadItemImage streams a photo into storage and records it against the
// item. The object is written before the row, so a failure leaves an
// unreferenced object rather than a row pointing at nothing.
func (a *Application) UploadItemImage(ctx context.Context, in UploadInput) (domain.ItemImage, error) {
	if a.store == nil {
		return domain.ItemImage{}, domain.Unavailable("object storage is not configured")
	}

	ext, err := domain.ValidateUpload(in.Size, in.ContentType)
	if err != nil {
		return domain.ItemImage{}, err
	}

	// The item must exist before we spend storage on it.
	if _, err := a.items.LiveStatusState(ctx, in.ItemID); err != nil {
		return domain.ItemImage{}, err
	}

	bucket := a.store.Bucket()
	key := domain.ObjectKey(in.ItemID, a.newID(), ext, in.Filename)
	if err := a.store.Upload(ctx, bucket, key, in.Body); err != nil {
		return domain.ItemImage{}, err
	}

	// Next display order: after the current last photo.
	count, err := a.images.CountForItem(ctx, in.ItemID)
	if err != nil {
		return domain.ItemImage{}, err
	}

	return a.images.Create(ctx, domain.NewItemImage{
		ItemID:       in.ItemID,
		UploadBucket: bucket,
		UploadKey:    key,
		Filename:     domain.FilenameOrNil(in.Filename),
		ContentType:  in.ContentType,
		DisplayOrder: count,
	})
}

// imageViews attaches a fresh URL to each photo.
func (a *Application) imageViews(ctx context.Context, images []domain.ItemImage) ([]ImageView, error) {
	out := make([]ImageView, 0, len(images))
	for _, img := range images {
		url, err := a.store.ViewURL(ctx, img.UploadBucket, img.UploadKey, domain.PresignTTL)
		if err != nil {
			return nil, err
		}
		out = append(out, ImageView{Image: img, URL: url})
	}
	return out, nil
}
