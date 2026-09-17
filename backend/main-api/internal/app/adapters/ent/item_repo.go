package ent

import (
	"context"
	"time"

	"main-api/db/generated"
	"main-api/db/generated/item"
	"main-api/db/generated/label"
	"main-api/db/generated/sellingplace"
	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.ItemRepository = (*ItemRepo)(nil)

// whatnotTaken is the collision the unique index on whatnot_number produces.
const whatnotTaken = "an item with this Whatnot number already exists"

// itemMissing is what every item read reports for a row that is absent or
// soft deleted.
const itemMissing = "item not found"

// ItemRepo stores items in Postgres through Ent.
type ItemRepo struct {
	client *generated.Client
}

// NewItemRepo returns an item repository over the given Ent client.
func NewItemRepo(client *generated.Client) *ItemRepo { return &ItemRepo{client: client} }

func (r *ItemRepo) Create(ctx context.Context, in domain.NewItem) (domain.Item, error) {
	create := r.client.Item.Create().
		SetName(in.Name).
		SetOwnerID(in.OwnerID).
		SetNillableDescription(in.Description).
		SetNillableCategory(in.Category).
		SetNillableCondition(in.Condition).
		SetNillableAcquisitionCostCents(in.AcquisitionCostCents).
		SetNillablePurchasedAt(in.PurchasedAt).
		SetNillableListingPriceCents(in.ListingPriceCents).
		SetNillableLength(in.Length).
		SetNillableWidth(in.Width).
		SetNillableHeight(in.Height).
		SetNillableExtraMeasurements(in.ExtraMeasurements).
		SetNillableWeightLbs(in.WeightLbs).
		SetNillableWeightOz(in.WeightOz).
		SetNillableNotes(in.Notes).
		SetNillableWhatnotNumber(in.WhatnotNumber)

	if in.Status != nil {
		create.SetStatus(item.Status(*in.Status))
	}
	if in.MeasurementUnit != nil {
		create.SetMeasurementUnit(item.MeasurementUnit(*in.MeasurementUnit))
	}
	if in.ListedAt != nil {
		create.SetListedAt(*in.ListedAt)
	}
	if in.FirstListedAt != nil {
		create.SetFirstListedAt(*in.FirstListedAt)
	}
	if len(in.SellingPlaceIDs) > 0 {
		create.AddSellingPlaceIDs(in.SellingPlaceIDs...)
	}
	if len(in.LabelIDs) > 0 {
		create.AddLabelIDs(in.LabelIDs...)
	}

	row, err := create.Save(ctx)
	if err != nil {
		return domain.Item{}, wrap("creating item", err, "", whatnotTaken)
	}
	return r.Get(ctx, row.ID)
}

func (r *ItemRepo) Get(ctx context.Context, id string) (domain.Item, error) {
	row, err := r.loaded(ctx, id)
	if err != nil {
		return domain.Item{}, wrap("loading item", err, itemMissing, "")
	}
	return toItem(row), nil
}

func (r *ItemRepo) List(ctx context.Context, f domain.ItemFilter) ([]domain.Item, error) {
	q := r.client.Item.Query().
		Where(item.DeletedAtIsNil()).
		WithSellingPlaces().
		WithLabels().
		WithSoldPlace().
		Order(generated.Desc(item.FieldCreatedAt))

	if f.Query != "" {
		q = q.Where(item.NameContainsFold(f.Query))
	}
	if f.PlaceID != "" {
		q = q.Where(item.HasSellingPlacesWith(sellingplace.IDEQ(f.PlaceID), sellingplace.DeletedAtIsNil()))
	}
	if f.LabelID != "" {
		q = q.Where(item.HasLabelsWith(label.IDEQ(f.LabelID), label.DeletedAtIsNil()))
	}
	if f.WhatnotNumber != "" {
		q = q.Where(item.WhatnotNumberEQ(f.WhatnotNumber))
	}
	if f.Status != nil {
		q = q.Where(item.StatusEQ(item.Status(*f.Status)))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, domain.Internal("listing items", err)
	}
	return toItems(rows), nil
}

func (r *ItemRepo) ListListed(ctx context.Context, cq domain.CatalogQuery) ([]domain.Item, error) {
	// KSUIDs sort by creation, so id-desc is "newest first" and doubles as the
	// pagination cursor. Ordering by listing date instead would need a
	// composite cursor.
	q := r.client.Item.Query().
		Where(item.DeletedAtIsNil(), item.StatusEQ(item.StatusListed)).
		WithLabels().
		Order(generated.Desc(item.FieldID)).
		Limit(cq.Limit)

	if cq.Cursor != "" {
		q = q.Where(item.IDLT(cq.Cursor))
	}
	if cq.Query != "" {
		q = q.Where(item.NameContainsFold(cq.Query))
	}
	if cq.Category != "" {
		q = q.Where(item.CategoryEQ(cq.Category))
	}
	if cq.Label != "" {
		q = q.Where(item.HasLabelsWith(label.NameEQ(cq.Label), label.DeletedAtIsNil()))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, domain.Internal("listing public items", err)
	}
	return toItems(rows), nil
}

func (r *ItemRepo) ListedExists(ctx context.Context, id string) (bool, error) {
	// Unlisted photos stay private, so the item's status gates the lookup.
	found, err := r.client.Item.Query().
		Where(item.ID(id), item.DeletedAtIsNil(), item.StatusEQ(item.StatusListed)).
		Exist(ctx)
	if err != nil {
		return false, domain.Internal("checking public item "+id, err)
	}
	return found, nil
}

func (r *ItemRepo) ListedFacets(ctx context.Context) ([]string, []string, error) {
	categories, err := r.client.Item.Query().
		Where(item.DeletedAtIsNil(), item.StatusEQ(item.StatusListed), item.CategoryNotNil()).
		Order(generated.Asc(item.FieldCategory)).
		GroupBy(item.FieldCategory).
		Strings(ctx)
	if err != nil {
		return nil, nil, domain.Internal("listing public categories", err)
	}

	rows, err := r.client.Label.Query().
		Where(label.DeletedAtIsNil(), label.HasItemsWith(
			item.DeletedAtIsNil(), item.StatusEQ(item.StatusListed))).
		Order(generated.Asc(label.FieldName)).
		All(ctx)
	if err != nil {
		return nil, nil, domain.Internal("listing public labels", err)
	}

	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return categories, names, nil
}

func (r *ItemRepo) StatusState(ctx context.Context, id string) (domain.ItemStatusState, error) {
	// The update path looks the row up by id alone; a soft-deleted item is
	// still found here and the write that follows fails on its own terms.
	row, err := r.client.Item.Query().
		Where(item.IDEQ(id)).
		Select(item.FieldStatus, item.FieldSoldPlaceID, item.FieldFirstListedAt).
		Only(ctx)
	if err != nil {
		return domain.ItemStatusState{}, wrap("loading item status", err, itemMissing, "")
	}
	return statusState(row), nil
}

func (r *ItemRepo) LiveStatusState(ctx context.Context, id string) (domain.ItemStatusState, error) {
	row, err := r.client.Item.Query().
		Where(item.IDEQ(id), item.DeletedAtIsNil()).
		Select(item.FieldStatus, item.FieldSoldPlaceID, item.FieldFirstListedAt).
		Only(ctx)
	if err != nil {
		return domain.ItemStatusState{}, wrap("loading item", err, itemMissing, "")
	}
	return statusState(row), nil
}

func (r *ItemRepo) Update(ctx context.Context, id string, p domain.ItemPatch) (domain.Item, error) {
	update := r.client.Item.UpdateOneID(id).
		SetName(p.Name).
		SetNillableDescription(p.Description).
		SetNillableCategory(p.Category).
		SetNillableCondition(p.Condition).
		SetNillableAcquisitionCostCents(p.AcquisitionCostCents).
		SetNillablePurchasedAt(p.PurchasedAt).
		SetNillableListingPriceCents(p.ListingPriceCents).
		SetNillableLength(p.Length).
		SetNillableWidth(p.Width).
		SetNillableHeight(p.Height).
		SetNillableExtraMeasurements(p.ExtraMeasurements).
		SetNillableWeightLbs(p.WeightLbs).
		SetNillableWeightOz(p.WeightOz).
		SetNillableSoldPriceCents(p.SoldPriceCents).
		SetNillableSoldAt(p.SoldAt)

	// A cleared column becomes NULL rather than "": whatnot_number is unique
	// when present, so two cleared items would otherwise collide.
	switch p.Notes.Action {
	case domain.FieldSet:
		update.SetNotes(p.Notes.Value)
	case domain.FieldClear:
		update.ClearNotes()
	}
	switch p.WhatnotNumber.Action {
	case domain.FieldSet:
		update.SetWhatnotNumber(p.WhatnotNumber.Value)
	case domain.FieldClear:
		update.ClearWhatnotNumber()
	}

	if p.SoldPlaceID != nil {
		update.SetSoldPlaceID(*p.SoldPlaceID)
	}
	if p.Status != nil {
		update.SetStatus(item.Status(*p.Status))
	}
	if p.MeasurementUnit != nil {
		update.SetMeasurementUnit(item.MeasurementUnit(*p.MeasurementUnit))
	}
	if p.ListedAt != nil {
		update.SetListedAt(*p.ListedAt)
	}
	if p.FirstListedAt != nil {
		update.SetFirstListedAt(*p.FirstListedAt)
	}
	if p.ClearListedAt {
		update.ClearListedAt()
	}

	// Full replacement of the M2M sets: the client always sends complete lists.
	if p.SellingPlaceIDs != nil {
		update.ClearSellingPlaces().AddSellingPlaceIDs(p.SellingPlaceIDs...)
	}
	if p.LabelIDs != nil {
		update.ClearLabels().AddLabelIDs(p.LabelIDs...)
	}

	row, err := update.Save(ctx)
	if err != nil {
		return domain.Item{}, wrap("updating item", err, itemMissing, whatnotTaken)
	}
	return r.Get(ctx, row.ID)
}

func (r *ItemRepo) MarkSold(ctx context.Context, id string, sale domain.Sale) (domain.Item, error) {
	row, err := r.client.Item.UpdateOneID(id).
		SetStatus(item.StatusSold).
		SetSoldAt(sale.SoldAt).
		SetSoldPriceCents(sale.SoldPriceCents).
		SetSoldPlaceID(sale.SoldPlaceID).
		Save(ctx)
	if err != nil {
		return domain.Item{}, wrap("marking item sold", err, itemMissing, "")
	}
	return r.Get(ctx, row.ID)
}

func (r *ItemRepo) SoftDelete(ctx context.Context, id string) error {
	// deleted_at is persistence bookkeeping, not a business decision, so the
	// adapter stamps it rather than taking it from the caller's clock.
	err := r.client.Item.UpdateOneID(id).SetDeletedAt(time.Now().UTC()).Exec(ctx)
	return wrap("deleting item", err, itemMissing, "")
}

// loaded fetches one live item with every edge an output mapping needs.
func (r *ItemRepo) loaded(ctx context.Context, id string) (*generated.Item, error) {
	return r.client.Item.Query().
		Where(item.ID(id), item.DeletedAtIsNil()).
		WithSellingPlaces().
		WithLabels().
		WithSoldPlace().
		Only(ctx)
}

func statusState(row *generated.Item) domain.ItemStatusState {
	return domain.ItemStatusState{
		Status:        domain.Status(row.Status),
		HasSoldPlace:  row.SoldPlaceID != nil,
		FirstListedAt: row.FirstListedAt,
	}
}
