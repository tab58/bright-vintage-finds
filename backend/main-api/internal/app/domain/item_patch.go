package domain

import "time"

// FieldAction is what a partial update does to one nullable column.
type FieldAction uint8

const (
	// FieldLeave means the client did not mention the field.
	FieldLeave FieldAction = iota
	// FieldSet means the client sent a value to store.
	FieldSet
	// FieldClear means the client sent an empty string, the PWA's way of
	// saying "clear it". The column becomes NULL rather than "" — whatnot_number
	// is unique when present, so two cleared items would otherwise collide.
	FieldClear
)

// ClearableString is a string column a partial update can leave alone, set,
// or clear. Only the columns the PWA lets the owner blank out use it.
type ClearableString struct {
	Action FieldAction
	Value  string
}

// Clearable reads the three-way intent out of an optional wire field.
func Clearable(p *string) ClearableString {
	switch {
	case p == nil:
		return ClearableString{Action: FieldLeave}
	case *p == "":
		return ClearableString{Action: FieldClear}
	default:
		return ClearableString{Action: FieldSet, Value: *p}
	}
}

// Stored returns the value to persist and whether to persist anything, for
// callers that treat "clear" and "leave" the same way — creation, where
// there is nothing to clear.
func (c ClearableString) Stored() (*string, bool) {
	if c.Action != FieldSet {
		return nil, false
	}
	v := c.Value
	return &v, true
}

// NewItem is everything needed to create an item. Nil fields are left unset.
type NewItem struct {
	Name                 string
	Description          *string
	Category             *string
	Condition            *string
	AcquisitionCostCents *int64
	PurchasedAt          *time.Time
	ListingPriceCents    *int64
	Length               *float64
	Width                *float64
	Height               *float64
	MeasurementUnit      *MeasurementUnit
	ExtraMeasurements    *string
	WeightLbs            *int
	WeightOz             *float64
	// Notes and WhatnotNumber are already normalized: an empty string means
	// "not given" on creation, and the caller drops it rather than storing ""
	// in a column that is unique when present.
	Notes           *string
	WhatnotNumber   *string
	SellingPlaceIDs []string
	LabelIDs        []string

	Status *Status
	// ListedAt and FirstListedAt are stamped by PlanCreateStatus, not by the
	// caller.
	ListedAt      *time.Time
	FirstListedAt *time.Time

	OwnerID string
}

// ItemPatch is a partial update. Name is always written, because the wire
// payload requires it. Every pointer field is "leave alone" when nil. The two
// ClearableString fields are the only columns a client can blank out, and a
// nil ID slice means "leave the set alone" while a non-nil one replaces it
// wholesale — the PWA always sends complete lists.
type ItemPatch struct {
	Name                 string
	Description          *string
	Category             *string
	Condition            *string
	AcquisitionCostCents *int64
	PurchasedAt          *time.Time
	ListingPriceCents    *int64
	Length               *float64
	Width                *float64
	Height               *float64
	MeasurementUnit      *MeasurementUnit
	ExtraMeasurements    *string
	WeightLbs            *int
	WeightOz             *float64
	Notes                ClearableString
	WhatnotNumber        ClearableString
	SellingPlaceIDs      []string
	LabelIDs             []string

	Status         *Status
	SoldPriceCents *int64
	SoldAt         *time.Time
	SoldPlaceID    *string

	// Timestamp bookkeeping decided by PlanStatusChange.
	ListedAt      *time.Time
	FirstListedAt *time.Time
	ClearListedAt bool
}

// ItemFilter narrows the admin item list. Zero-valued fields do not filter.
type ItemFilter struct {
	Query         string
	PlaceID       string
	LabelID       string
	WhatnotNumber string
	Status        *Status
}

// Page-size bounds for the public catalog.
const (
	CatalogPageDefault = 24
	CatalogPageMax     = 60
)

// CatalogQuery is one page of the public catalog. Cursor is the ID of the
// last item on the previous page; KSUIDs sort by creation, so id-desc is
// "newest first" and doubles as the cursor.
type CatalogQuery struct {
	Limit    int
	Cursor   string
	Query    string
	Category string
	Label    string
}

// ClampPageSize bounds a requested page size, filling in the default when the
// client asked for nothing.
func ClampPageSize(limit int) int {
	switch {
	case limit <= 0:
		return CatalogPageDefault
	case limit > CatalogPageMax:
		return CatalogPageMax
	default:
		return limit
	}
}

// Sale records an item changing hands. Marking an item sold goes through its
// own operation rather than a patch: it always writes the same four fields
// and never touches the rest of the item.
type Sale struct {
	SoldAt         time.Time
	SoldPriceCents int64
	SoldPlaceID    string
}
