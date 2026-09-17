package fakes

import (
	"context"
	"sort"
	"strings"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.ItemRepository = (*FakeItemRepo)(nil)

// FakeItemRepo is an in-memory ports.ItemRepository. Items are held in
// insertion order, which stands in for KSUID order: the last one added is the
// newest.
type FakeItemRepo struct {
	Err error

	// Places and Labels, when set, resolve edge ids to full records so items
	// read back carry the names an output mapping needs.
	Places *FakeSellingPlaceRepo
	Labels *FakeLabelRepo

	// LastPatch is the most recent Update argument, so a test can assert what
	// the service asked for rather than only what came back.
	LastPatch domain.ItemPatch
	// LastOwnerID is the owner the most recent Create was attributed to.
	LastOwnerID string

	ids      seq
	items    []domain.Item
	placeIDs map[string][]string
	labelIDs map[string][]string
	deleted  map[string]bool
}

// NewItemRepo returns an empty item repository.
func NewItemRepo() *FakeItemRepo {
	return &FakeItemRepo{
		ids:      seq{prefix: "itm"},
		placeIDs: map[string][]string{},
		labelIDs: map[string][]string{},
		deleted:  map[string]bool{},
	}
}

// Add seeds an item directly, assigning an id when it has none, and returns
// the stored record.
func (r *FakeItemRepo) Add(it domain.Item) domain.Item {
	if it.ID == "" {
		it.ID = r.ids.next()
	}
	if it.Status == "" {
		it.Status = domain.StatusDraft
	}
	if it.MeasurementUnit == "" {
		it.MeasurementUnit = domain.UnitInch
	}
	if it.CreatedAt.IsZero() {
		it.CreatedAt = FixedTime
		it.UpdatedAt = FixedTime
	}
	r.placeIDs[it.ID] = idsOfPlaces(it.SellingPlaces)
	r.labelIDs[it.ID] = idsOfLabels(it.Labels)
	r.items = append(r.items, it)
	return it
}

// SoftDeletedIDs reports which items have been soft deleted, so a test can
// check the deletion happened without a read path that hides it.
func (r *FakeItemRepo) SoftDeletedIDs() []string {
	out := make([]string, 0, len(r.deleted))
	for id := range r.deleted {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (r *FakeItemRepo) Create(_ context.Context, in domain.NewItem) (domain.Item, error) {
	if r.Err != nil {
		return domain.Item{}, r.Err
	}
	if err := r.checkWhatnotFree(in.WhatnotNumber, ""); err != nil {
		return domain.Item{}, err
	}
	r.LastOwnerID = in.OwnerID

	it := domain.Item{
		ID:                   r.ids.next(),
		Name:                 in.Name,
		Description:          in.Description,
		Category:             in.Category,
		Condition:            in.Condition,
		Status:               domain.StatusDraft,
		AcquisitionCostCents: in.AcquisitionCostCents,
		PurchasedAt:          in.PurchasedAt,
		ListingPriceCents:    in.ListingPriceCents,
		Length:               in.Length,
		Width:                in.Width,
		Height:               in.Height,
		MeasurementUnit:      domain.UnitInch,
		ExtraMeasurements:    in.ExtraMeasurements,
		WeightLbs:            in.WeightLbs,
		WeightOz:             in.WeightOz,
		Notes:                in.Notes,
		WhatnotNumber:        in.WhatnotNumber,
		ListedAt:             in.ListedAt,
		FirstListedAt:        in.FirstListedAt,
		CreatedAt:            FixedTime,
		UpdatedAt:            FixedTime,
	}
	if in.Status != nil {
		it.Status = *in.Status
	}
	if in.MeasurementUnit != nil {
		it.MeasurementUnit = *in.MeasurementUnit
	}

	r.items = append(r.items, it)
	r.placeIDs[it.ID] = append([]string(nil), in.SellingPlaceIDs...)
	r.labelIDs[it.ID] = append([]string(nil), in.LabelIDs...)
	return r.hydrate(it), nil
}

func (r *FakeItemRepo) Get(_ context.Context, id string) (domain.Item, error) {
	if r.Err != nil {
		return domain.Item{}, r.Err
	}
	it, ok := r.live(id)
	if !ok {
		return domain.Item{}, domain.NotFound("item not found")
	}
	return r.hydrate(it), nil
}

func (r *FakeItemRepo) List(_ context.Context, f domain.ItemFilter) ([]domain.Item, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	out := []domain.Item{}
	// Newest first.
	for i := len(r.items) - 1; i >= 0; i-- {
		it := r.items[i]
		if r.deleted[it.ID] {
			continue
		}
		if f.Query != "" && !containsFold(it.Name, f.Query) {
			continue
		}
		if f.PlaceID != "" && !contains(r.placeIDs[it.ID], f.PlaceID) {
			continue
		}
		if f.LabelID != "" && !contains(r.labelIDs[it.ID], f.LabelID) {
			continue
		}
		if f.WhatnotNumber != "" && (it.WhatnotNumber == nil || *it.WhatnotNumber != f.WhatnotNumber) {
			continue
		}
		if f.Status != nil && it.Status != *f.Status {
			continue
		}
		out = append(out, r.hydrate(it))
	}
	return out, nil
}

func (r *FakeItemRepo) ListListed(_ context.Context, q domain.CatalogQuery) ([]domain.Item, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	out := []domain.Item{}
	for i := len(r.items) - 1; i >= 0; i-- {
		if len(out) == q.Limit {
			break
		}
		it := r.items[i]
		if r.deleted[it.ID] || it.Status != domain.StatusListed {
			continue
		}
		// The cursor is the previous page's last id; ids descend.
		if q.Cursor != "" && it.ID >= q.Cursor {
			continue
		}
		if q.Query != "" && !containsFold(it.Name, q.Query) {
			continue
		}
		if q.Category != "" && (it.Category == nil || *it.Category != q.Category) {
			continue
		}
		if q.Label != "" && !contains(r.labelNames(it.ID), q.Label) {
			continue
		}
		out = append(out, r.hydrate(it))
	}
	return out, nil
}

func (r *FakeItemRepo) ListedExists(_ context.Context, id string) (bool, error) {
	if r.Err != nil {
		return false, r.Err
	}
	it, ok := r.live(id)
	return ok && it.Status == domain.StatusListed, nil
}

func (r *FakeItemRepo) ListedFacets(_ context.Context) ([]string, []string, error) {
	if r.Err != nil {
		return nil, nil, r.Err
	}
	catSet, labelSet := map[string]bool{}, map[string]bool{}
	for _, it := range r.items {
		if r.deleted[it.ID] || it.Status != domain.StatusListed {
			continue
		}
		if it.Category != nil {
			catSet[*it.Category] = true
		}
		for _, name := range r.labelNames(it.ID) {
			labelSet[name] = true
		}
	}
	return sortedKeys(catSet), sortedKeys(labelSet), nil
}

func (r *FakeItemRepo) StatusState(_ context.Context, id string) (domain.ItemStatusState, error) {
	if r.Err != nil {
		return domain.ItemStatusState{}, r.Err
	}
	// Soft-deleted rows are still visible here, matching the update path.
	for _, it := range r.items {
		if it.ID == id {
			return statusStateOf(it), nil
		}
	}
	return domain.ItemStatusState{}, domain.NotFound("item not found")
}

func (r *FakeItemRepo) LiveStatusState(_ context.Context, id string) (domain.ItemStatusState, error) {
	if r.Err != nil {
		return domain.ItemStatusState{}, r.Err
	}
	it, ok := r.live(id)
	if !ok {
		return domain.ItemStatusState{}, domain.NotFound("item not found")
	}
	return statusStateOf(it), nil
}

func (r *FakeItemRepo) Update(_ context.Context, id string, p domain.ItemPatch) (domain.Item, error) {
	r.LastPatch = p
	if r.Err != nil {
		return domain.Item{}, r.Err
	}
	i, ok := r.indexOf(id)
	if !ok {
		return domain.Item{}, domain.NotFound("item not found")
	}
	if p.WhatnotNumber.Action == domain.FieldSet {
		if err := r.checkWhatnotFree(&p.WhatnotNumber.Value, id); err != nil {
			return domain.Item{}, err
		}
	}

	it := r.items[i]
	it.Name = p.Name
	setPtr(&it.Description, p.Description)
	setPtr(&it.Category, p.Category)
	setPtr(&it.Condition, p.Condition)
	setPtr(&it.AcquisitionCostCents, p.AcquisitionCostCents)
	setPtr(&it.PurchasedAt, p.PurchasedAt)
	setPtr(&it.ListingPriceCents, p.ListingPriceCents)
	setPtr(&it.Length, p.Length)
	setPtr(&it.Width, p.Width)
	setPtr(&it.Height, p.Height)
	setPtr(&it.ExtraMeasurements, p.ExtraMeasurements)
	setPtr(&it.WeightLbs, p.WeightLbs)
	setPtr(&it.WeightOz, p.WeightOz)
	setPtr(&it.SoldPriceCents, p.SoldPriceCents)
	setPtr(&it.SoldAt, p.SoldAt)
	applyClearable(&it.Notes, p.Notes)
	applyClearable(&it.WhatnotNumber, p.WhatnotNumber)

	if p.MeasurementUnit != nil {
		it.MeasurementUnit = *p.MeasurementUnit
	}
	if p.Status != nil {
		it.Status = *p.Status
	}
	if p.SoldPlaceID != nil {
		it.SoldPlace = &domain.SellingPlace{ID: *p.SoldPlaceID, Name: r.placeName(*p.SoldPlaceID)}
	}
	setPtr(&it.ListedAt, p.ListedAt)
	setPtr(&it.FirstListedAt, p.FirstListedAt)
	if p.ClearListedAt {
		it.ListedAt = nil
	}
	it.UpdatedAt = FixedTime

	if p.SellingPlaceIDs != nil {
		r.placeIDs[id] = append([]string(nil), p.SellingPlaceIDs...)
	}
	if p.LabelIDs != nil {
		r.labelIDs[id] = append([]string(nil), p.LabelIDs...)
	}

	r.items[i] = it
	return r.hydrate(it), nil
}

func (r *FakeItemRepo) MarkSold(_ context.Context, id string, sale domain.Sale) (domain.Item, error) {
	if r.Err != nil {
		return domain.Item{}, r.Err
	}
	i, ok := r.indexOf(id)
	if !ok {
		return domain.Item{}, domain.NotFound("item not found")
	}
	it := r.items[i]
	it.Status = domain.StatusSold
	it.SoldAt = &sale.SoldAt
	it.SoldPriceCents = &sale.SoldPriceCents
	it.SoldPlace = &domain.SellingPlace{ID: sale.SoldPlaceID, Name: r.placeName(sale.SoldPlaceID)}
	it.UpdatedAt = FixedTime
	r.items[i] = it
	return r.hydrate(it), nil
}

func (r *FakeItemRepo) SoftDelete(_ context.Context, id string) error {
	if r.Err != nil {
		return r.Err
	}
	if _, ok := r.indexOf(id); !ok {
		return domain.NotFound("item not found")
	}
	r.deleted[id] = true
	return nil
}

// checkWhatnotFree mirrors the unique index on whatnot_number: a value may
// repeat only across rows that have none.
func (r *FakeItemRepo) checkWhatnotFree(number *string, exceptID string) error {
	if number == nil || *number == "" {
		return nil
	}
	for _, it := range r.items {
		if it.ID == exceptID || it.WhatnotNumber == nil {
			continue
		}
		if *it.WhatnotNumber == *number {
			return domain.Conflict("an item with this Whatnot number already exists")
		}
	}
	return nil
}

func (r *FakeItemRepo) indexOf(id string) (int, bool) {
	for i, it := range r.items {
		if it.ID == id {
			return i, true
		}
	}
	return 0, false
}

func (r *FakeItemRepo) live(id string) (domain.Item, bool) {
	i, ok := r.indexOf(id)
	if !ok || r.deleted[id] {
		return domain.Item{}, false
	}
	return r.items[i], true
}

// hydrate fills the edge slices from the linked place and label repositories,
// falling back to id-only records when none are wired up.
func (r *FakeItemRepo) hydrate(it domain.Item) domain.Item {
	it.SellingPlaces = make([]domain.SellingPlace, 0, len(r.placeIDs[it.ID]))
	for _, id := range r.placeIDs[it.ID] {
		it.SellingPlaces = append(it.SellingPlaces, domain.SellingPlace{ID: id, Name: r.placeName(id)})
	}
	it.Labels = make([]domain.Label, 0, len(r.labelIDs[it.ID]))
	for _, id := range r.labelIDs[it.ID] {
		it.Labels = append(it.Labels, domain.Label{ID: id, Name: r.labelName(id)})
	}
	return it
}

func (r *FakeItemRepo) placeName(id string) string {
	if r.Places == nil {
		return ""
	}
	return r.Places.NameOf(id)
}

func (r *FakeItemRepo) labelName(id string) string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels.NameOf(id)
}

func (r *FakeItemRepo) labelNames(itemID string) []string {
	names := make([]string, 0, len(r.labelIDs[itemID]))
	for _, id := range r.labelIDs[itemID] {
		names = append(names, r.labelName(id))
	}
	return names
}

func statusStateOf(it domain.Item) domain.ItemStatusState {
	return domain.ItemStatusState{
		Status:        it.Status,
		HasSoldPlace:  it.SoldPlace != nil,
		FirstListedAt: it.FirstListedAt,
	}
}

func setPtr[T any](dst **T, src *T) {
	if src != nil {
		*dst = src
	}
}

func applyClearable(dst **string, c domain.ClearableString) {
	switch c.Action {
	case domain.FieldSet:
		v := c.Value
		*dst = &v
	case domain.FieldClear:
		*dst = nil
	}
}

func idsOfPlaces(places []domain.SellingPlace) []string {
	ids := make([]string, 0, len(places))
	for _, p := range places {
		ids = append(ids, p.ID)
	}
	return ids
}

func idsOfLabels(labels []domain.Label) []string {
	ids := make([]string, 0, len(labels))
	for _, l := range labels {
		ids = append(ids, l.ID)
	}
	return ids
}

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
