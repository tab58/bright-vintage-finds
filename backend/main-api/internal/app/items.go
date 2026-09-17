package app

import (
	"context"

	"main-api/internal/app/domain"
)

// CreateItem stores a new item owned by the inventory's single owner.
func (a *Application) CreateItem(ctx context.Context, in ItemInput) (ItemView, error) {
	ownerID, err := a.owner.EnsureBuiltinOwner(ctx)
	if err != nil {
		return ItemView{}, err
	}
	newItem := buildNewItem(ownerID, in)

	if in.Status != nil {
		status, stamp, err := a.plannedCreateStatus(*in.Status, in.SoldPlaceID != nil)
		if err != nil {
			return ItemView{}, err
		}
		newItem.Status = &status
		newItem.ListedAt = stamp.ListedAt
		newItem.FirstListedAt = stamp.FirstListedAt
	}

	unit, err := parsedUnit(in.MeasurementUnit)
	if err != nil {
		return ItemView{}, err
	}
	newItem.MeasurementUnit = unit

	if err := a.checkEdges(ctx, in.SellingPlaceIDs, in.LabelIDs); err != nil {
		return ItemView{}, err
	}
	newItem.SellingPlaceIDs = in.SellingPlaceIDs
	newItem.LabelIDs = in.LabelIDs

	item, err := a.items.Create(ctx, newItem)
	if err != nil {
		return ItemView{}, err
	}
	return a.view(ctx, item)
}

// GetItem returns one live item.
func (a *Application) GetItem(ctx context.Context, id string) (ItemView, error) {
	item, err := a.items.Get(ctx, id)
	if err != nil {
		return ItemView{}, err
	}
	return a.view(ctx, item)
}

// ListItems returns live items newest first, narrowed by the given filters.
func (a *Application) ListItems(ctx context.Context, in ItemFilterInput) ([]ItemView, error) {
	filter := domain.ItemFilter{
		Query:         in.Query,
		PlaceID:       in.PlaceID,
		LabelID:       in.LabelID,
		WhatnotNumber: in.WhatnotNumber,
	}
	if in.Status != "" {
		status, err := domain.ParseStatus(in.Status)
		if err != nil {
			return nil, err
		}
		filter.Status = &status
	}

	items, err := a.items.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return a.views(ctx, items)
}

// UpdateItem applies a partial change to an item.
func (a *Application) UpdateItem(ctx context.Context, id string, in ItemInput) (ItemView, error) {
	patch := buildItemPatch(in)

	if in.SoldPlaceID != nil {
		if err := a.checkPlace(ctx, *in.SoldPlaceID); err != nil {
			return ItemView{}, err
		}
		patch.SoldPlaceID = in.SoldPlaceID
	}

	if in.Status != nil {
		status, stamp, err := a.plannedStatusChange(ctx, id, *in.Status, in.SoldPlaceID != nil)
		if err != nil {
			return ItemView{}, err
		}
		patch.Status = &status
		patch.ListedAt = stamp.ListedAt
		patch.FirstListedAt = stamp.FirstListedAt
		patch.ClearListedAt = stamp.ClearListedAt
	}

	unit, err := parsedUnit(in.MeasurementUnit)
	if err != nil {
		return ItemView{}, err
	}
	patch.MeasurementUnit = unit

	// A nil list leaves the set alone; a non-nil one replaces it wholesale.
	if err := a.checkPatchEdges(ctx, in); err != nil {
		return ItemView{}, err
	}
	patch.SellingPlaceIDs = in.SellingPlaceIDs
	patch.LabelIDs = in.LabelIDs

	item, err := a.items.Update(ctx, id, patch)
	if err != nil {
		return ItemView{}, err
	}
	return a.view(ctx, item)
}

// MarkItemSold records a sale against an item.
func (a *Application) MarkItemSold(ctx context.Context, id string, sale domain.Sale) (ItemView, error) {
	if err := a.checkPlace(ctx, sale.SoldPlaceID); err != nil {
		return ItemView{}, err
	}
	item, err := a.items.MarkSold(ctx, id, sale)
	if err != nil {
		return ItemView{}, err
	}
	return a.view(ctx, item)
}

// DeleteItem removes an item that the deletion rules allow, along with its
// photos. The stored objects go before their rows, so nothing is left in the
// bucket with no record of what it was.
func (a *Application) DeleteItem(ctx context.Context, id string) error {
	state, err := a.items.LiveStatusState(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.CanDelete(state.Status, state.FirstListedAt); err != nil {
		return err
	}
	if err := a.deleteStoredObjects(ctx, id); err != nil {
		return err
	}
	if err := a.images.HardDeleteForItem(ctx, id); err != nil {
		return err
	}
	return a.items.SoftDelete(ctx, id)
}

// buildNewItem copies the wire payload into the create shape. An empty string
// means "not given" on creation: it is dropped rather than stored, so a blank
// whatnot number never occupies the unique index.
func buildNewItem(ownerID string, in ItemInput) domain.NewItem {
	newItem := domain.NewItem{
		OwnerID:              ownerID,
		Name:                 in.Name,
		Description:          in.Description,
		Category:             in.Category,
		Condition:            in.Condition,
		AcquisitionCostCents: in.AcquisitionCostCents,
		PurchasedAt:          in.PurchasedAt,
		ListingPriceCents:    in.ListingPriceCents,
		Length:               in.Length,
		Width:                in.Width,
		Height:               in.Height,
		ExtraMeasurements:    in.ExtraMeasurements,
		WeightLbs:            in.WeightLbs,
		WeightOz:             in.WeightOz,
	}
	newItem.Notes, _ = domain.Clearable(in.Notes).Stored()
	newItem.WhatnotNumber, _ = domain.Clearable(in.WhatnotNumber).Stored()
	return newItem
}

// buildItemPatch copies the wire payload into the patch shape, where notes and
// whatnot number carry the absent/clear/set tri-state.
func buildItemPatch(in ItemInput) domain.ItemPatch {
	return domain.ItemPatch{
		Name:                 in.Name,
		Description:          in.Description,
		Category:             in.Category,
		Condition:            in.Condition,
		AcquisitionCostCents: in.AcquisitionCostCents,
		PurchasedAt:          in.PurchasedAt,
		ListingPriceCents:    in.ListingPriceCents,
		Length:               in.Length,
		Width:                in.Width,
		Height:               in.Height,
		ExtraMeasurements:    in.ExtraMeasurements,
		WeightLbs:            in.WeightLbs,
		WeightOz:             in.WeightOz,
		Notes:                domain.Clearable(in.Notes),
		WhatnotNumber:        domain.Clearable(in.WhatnotNumber),
		SoldPriceCents:       in.SoldPriceCents,
		SoldAt:               in.SoldAt,
	}
}

// plannedCreateStatus parses the status an item is created in and the listing
// timestamps it implies.
func (a *Application) plannedCreateStatus(status string, hasSoldPlace bool) (domain.Status, domain.StatusStamp, error) {
	parsed, err := domain.ParseStatus(status)
	if err != nil {
		return "", domain.StatusStamp{}, err
	}
	stamp, err := domain.PlanCreateStatus(parsed, hasSoldPlace, a.now())
	if err != nil {
		return "", domain.StatusStamp{}, err
	}
	return parsed, stamp, nil
}

// plannedStatusChange parses the status an item moves to and the listing
// timestamps the move implies, given what is stored today.
func (a *Application) plannedStatusChange(ctx context.Context, id, status string, patchSetsSoldPlace bool) (domain.Status, domain.StatusStamp, error) {
	parsed, err := domain.ParseStatus(status)
	if err != nil {
		return "", domain.StatusStamp{}, err
	}
	current, err := a.items.StatusState(ctx, id)
	if err != nil {
		return "", domain.StatusStamp{}, err
	}
	stamp, err := domain.PlanStatusChange(parsed, current, patchSetsSoldPlace, a.now())
	if err != nil {
		return "", domain.StatusStamp{}, err
	}
	return parsed, stamp, nil
}

// parsedUnit converts the wire measurement unit, when the payload carried one.
func parsedUnit(unit *string) (*domain.MeasurementUnit, error) {
	if unit == nil {
		return nil, nil
	}
	parsed, err := domain.ParseMeasurementUnit(*unit)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// checkPatchEdges validates the edge lists a patch replaces, skipping the ones
// it leaves alone.
func (a *Application) checkPatchEdges(ctx context.Context, in ItemInput) error {
	if in.SellingPlaceIDs != nil {
		if err := a.checkEdges(ctx, in.SellingPlaceIDs, nil); err != nil {
			return err
		}
	}
	if in.LabelIDs != nil {
		if err := a.checkEdges(ctx, nil, in.LabelIDs); err != nil {
			return err
		}
	}
	return nil
}

// checkEdges rejects references to places or labels that do not exist,
// naming the offending list.
func (a *Application) checkEdges(ctx context.Context, placeIDs, labelIDs []string) error {
	if len(placeIDs) > 0 {
		ok, err := a.places.AllExist(ctx, placeIDs)
		if err != nil {
			return err
		}
		if !ok {
			return domain.Invalid("unknown selling_place_ids")
		}
	}
	if len(labelIDs) > 0 {
		ok, err := a.labels.AllExist(ctx, labelIDs)
		if err != nil {
			return err
		}
		if !ok {
			return domain.Invalid("unknown label_ids")
		}
	}
	return nil
}

// checkPlace rejects a sold place that does not exist.
func (a *Application) checkPlace(ctx context.Context, id string) error {
	ok, err := a.places.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return domain.Invalid("unknown sold_place_id")
	}
	return nil
}

// deleteStoredObjects removes the objects of every photo an item ever had,
// soft-deleted rows included, because deleting an item must clear the objects
// the owner had already removed too.
func (a *Application) deleteStoredObjects(ctx context.Context, itemID string) error {
	if a.store == nil {
		return nil
	}
	images, err := a.images.ListForItemIncludingDeleted(ctx, itemID)
	if err != nil {
		return err
	}
	for _, img := range images {
		if err := a.store.Delete(ctx, img.UploadBucket, img.UploadKey); err != nil {
			return err
		}
	}
	return nil
}

// view attaches the photo facts to an item: the count always, the cover URL
// only when there is storage to presign against.
func (a *Application) view(ctx context.Context, item domain.Item) (ItemView, error) {
	images, err := a.images.ListForItem(ctx, item.ID)
	if err != nil {
		return ItemView{}, err
	}

	v := ItemView{Item: item, ImageCount: len(images)}
	cover, ok := domain.Cover(images)
	if !ok || a.store == nil {
		return v, nil
	}
	url, err := a.store.ViewURL(ctx, cover.UploadBucket, cover.UploadKey, domain.PresignTTL)
	if err != nil {
		return ItemView{}, err
	}
	v.CoverImageURL = &url
	return v, nil
}

// views maps a list of items, one photo lookup per item. The admin list is
// small and owner-only; the public catalog, which is not, batches instead.
func (a *Application) views(ctx context.Context, items []domain.Item) ([]ItemView, error) {
	out := make([]ItemView, 0, len(items))
	for _, item := range items {
		v, err := a.view(ctx, item)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
