package domain

import "time"

// Status is where an item sits in the selling flow: draft -> listed -> sold,
// with archived as a side exit. Which moves are offered is the client's call;
// the domain only validates the value and the bookkeeping each move implies.
type Status string

const (
	StatusDraft    Status = "draft"
	StatusListed   Status = "listed"
	StatusSold     Status = "sold"
	StatusArchived Status = "archived"
)

// ParseStatus converts a wire string to a Status.
func ParseStatus(s string) (Status, error) {
	switch Status(s) {
	case StatusDraft, StatusListed, StatusSold, StatusArchived:
		return Status(s), nil
	}
	return "", Invalidf("invalid status %q", s)
}

// MeasurementUnit is the unit the length/width/height fields are in.
type MeasurementUnit string

const (
	UnitInch MeasurementUnit = "inch"
	UnitCm   MeasurementUnit = "cm"
)

// ParseMeasurementUnit converts a wire string to a MeasurementUnit.
func ParseMeasurementUnit(s string) (MeasurementUnit, error) {
	switch MeasurementUnit(s) {
	case UnitInch, UnitCm:
		return MeasurementUnit(s), nil
	}
	return "", Invalid("measurement_unit must be inch or cm")
}

// Item is one piece of inventory. Optional columns stay pointers so "not set"
// survives the round trip through the repository; a nil pointer means the
// column is NULL, never the zero value.
type Item struct {
	ID          string
	Name        string
	Description *string
	Category    *string
	Condition   *string
	Status      Status

	AcquisitionCostCents *int64
	PurchasedAt          *time.Time
	ListingPriceCents    *int64

	Length            *float64
	Width             *float64
	Height            *float64
	MeasurementUnit   MeasurementUnit
	ExtraMeasurements *string
	WeightLbs         *int
	WeightOz          *float64
	Notes             *string

	WhatnotNumber *string

	SellingPlaces []SellingPlace
	Labels        []Label
	SoldPlace     *SellingPlace

	// ListedAt times the current listing: stamped on the way in, cleared on
	// the way back to draft. FirstListedAt is written once and never cleared.
	ListedAt      *time.Time
	FirstListedAt *time.Time

	SoldPriceCents *int64
	SoldAt         *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// EffectiveFirstListedAt is FirstListedAt, falling back to ListedAt for rows
// written before the column existed: those only have the one timestamp to go
// on. Deletion rules read the raw field instead, so a legacy row is not
// treated as never-listed.
func (i Item) EffectiveFirstListedAt() *time.Time {
	if i.FirstListedAt != nil {
		return i.FirstListedAt
	}
	return i.ListedAt
}

// SellingPlaceIDs returns the IDs of the places this item is offered on.
func (i Item) SellingPlaceIDs() []string {
	ids := make([]string, 0, len(i.SellingPlaces))
	for _, sp := range i.SellingPlaces {
		ids = append(ids, sp.ID)
	}
	return ids
}

// SellingPlaceNames returns the names of the places this item is offered on.
func (i Item) SellingPlaceNames() []string {
	names := make([]string, 0, len(i.SellingPlaces))
	for _, sp := range i.SellingPlaces {
		names = append(names, sp.Name)
	}
	return names
}

// LabelIDs returns the IDs of the labels tagged on this item.
func (i Item) LabelIDs() []string {
	ids := make([]string, 0, len(i.Labels))
	for _, l := range i.Labels {
		ids = append(ids, l.ID)
	}
	return ids
}

// LabelNames returns the names of the labels tagged on this item.
func (i Item) LabelNames() []string {
	names := make([]string, 0, len(i.Labels))
	for _, l := range i.Labels {
		names = append(names, l.Name)
	}
	return names
}
