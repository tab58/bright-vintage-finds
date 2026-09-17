package app

import (
	"context"

	"main-api/internal/app/domain"
)

// The public catalog is a read-only projection of the same items the admin
// routes manage. What keeps private data out is the response mapping at the
// HTTP edge, which has its own output type.

// Catalog returns one page of the items currently for sale.
func (a *Application) Catalog(ctx context.Context, q domain.CatalogQuery) (CatalogPage, error) {
	limit := domain.ClampPageSize(q.Limit)
	// One row beyond the page tells us another page exists, without a count.
	q.Limit = limit + 1

	items, err := a.items.ListListed(ctx, q)
	if err != nil {
		return CatalogPage{}, err
	}

	page := CatalogPage{}
	if len(items) > limit {
		items = items[:limit]
		page.NextCursor = items[limit-1].ID
	}

	covers, counts, err := a.covers(ctx, items)
	if err != nil {
		return CatalogPage{}, err
	}

	page.Items = make([]ItemView, 0, len(items))
	for _, item := range items {
		v := ItemView{Item: item, ImageCount: counts[item.ID]}
		if url, ok := covers[item.ID]; ok {
			v.CoverImageURL = &url
		}
		page.Items = append(page.Items, v)
	}
	return page, nil
}

// CatalogFacets returns the categories and labels present among the items
// currently for sale, so the shop can offer filters it cannot derive from a
// single page.
func (a *Application) CatalogFacets(ctx context.Context) (categories, labels []string, err error) {
	return a.items.ListedFacets(ctx)
}

// CatalogImages returns the photos of an item that is for sale. Photos of an
// unlisted item stay private, so an item that is not listed reads as missing.
func (a *Application) CatalogImages(ctx context.Context, itemID string) ([]ImageView, error) {
	listed, err := a.items.ListedExists(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if !listed {
		return nil, domain.NotFound("item not found")
	}
	if a.store == nil {
		return []ImageView{}, nil
	}

	images, err := a.images.ListForItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	return a.imageViews(ctx, images)
}

// covers loads the cover URL and photo count for a whole page in one query,
// rather than one query per item as the admin list does. Without storage
// there are no URLs and no counts, which is what the catalog has always
// reported.
func (a *Application) covers(ctx context.Context, items []domain.Item) (map[string]string, map[string]int, error) {
	covers := make(map[string]string, len(items))
	counts := make(map[string]int, len(items))
	if len(items) == 0 || a.store == nil {
		return covers, counts, nil
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	byItem, err := a.images.ListForItems(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	for _, id := range ids {
		images := byItem[id]
		counts[id] = len(images)
		cover, ok := domain.Cover(images)
		if !ok {
			continue
		}
		url, err := a.store.ViewURL(ctx, cover.UploadBucket, cover.UploadKey, domain.PresignTTL)
		if err != nil {
			return nil, nil, err
		}
		covers[id] = url
	}
	return covers, counts, nil
}
