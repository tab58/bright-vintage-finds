package app

import (
	"io"
	"time"

	"main-api/internal/app/domain"
)

// ItemView is an item together with the photo facts a response carries but
// the item row does not hold: how many photos it has and where to see the
// first one. Assembling it is a read-model concern, not a domain rule.
type ItemView struct {
	// Item is the stored item, with its selling places, labels and sold
	// place loaded.
	Item domain.Item
	// ImageCount is how many live photos the item has.
	ImageCount int
	// CoverImageURL is a short-lived URL for the first photo. It is absent
	// when the item has none, or when the deployment has no object storage.
	CoverImageURL *string
}

// ImageView is a photo with a URL the caller can fetch it from.
type ImageView struct {
	// Image is the stored photo row.
	Image domain.ItemImage
	// URL is a short-lived GET URL for the object, reachable by the client.
	URL string
}

// CatalogPage is one page of the public catalog.
type CatalogPage struct {
	// Items are the listed items on this page, newest first.
	Items []ItemView
	// NextCursor is the id to page from. It is empty on the last page.
	NextCursor string
}

// ItemInput is the create/update payload in the shape the wire uses: status
// and measurement unit arrive as strings and are parsed here, so the HTTP
// adapter stays a transport concern. A nil pointer means the payload did not
// carry the field; on update that leaves the stored value alone.
type ItemInput struct {
	// Name is the item's display name. It is the one required field.
	Name string
	// Description is the long-form copy shown with the listing.
	Description *string
	// Category is the free-text grouping the public catalog filters on.
	Category *string
	// Condition is the free-text condition note, e.g. "mint", "as-is".
	Condition *string
	// AcquisitionCostCents is what the item cost to buy, in cents.
	AcquisitionCostCents *int64
	// PurchasedAt is when the item was bought.
	PurchasedAt *time.Time
	// ListingPriceCents is the asking price, in cents.
	ListingPriceCents *int64
	// Length is the longest dimension, in MeasurementUnit.
	Length *float64
	// Width is the width, in MeasurementUnit.
	Width *float64
	// Height is the height, in MeasurementUnit.
	Height *float64
	// MeasurementUnit is the unit the dimensions are in: "inch" or "cm".
	// Anything else is rejected.
	MeasurementUnit *string
	// ExtraMeasurements is free text for dimensions the columns do not cover.
	ExtraMeasurements *string
	// WeightLbs is the whole-pound part of the shipping weight.
	WeightLbs *int
	// WeightOz is the ounces part of the shipping weight.
	WeightOz *float64
	// Notes is the owner's private note. It is clearable: an empty string
	// clears the column rather than storing "".
	Notes *string
	// WhatnotNumber is the item's number in a Whatnot show. It is clearable,
	// and unique when present, so a cleared one must become NULL.
	WhatnotNumber *string
	// SellingPlaceIDs are the places the item is offered on. On update a nil
	// slice leaves the set alone and a non-nil one replaces it wholesale.
	SellingPlaceIDs []string
	// LabelIDs are the item's labels, replaced on the same nil/non-nil rule
	// as SellingPlaceIDs.
	LabelIDs []string

	// Status is the wire status: "draft", "listed", "sold" or "archived".
	// Anything else is rejected, and the move decides the listing timestamps.
	Status *string

	// SoldPriceCents is what the item sold for, in cents, for correcting a
	// sale after the fact.
	SoldPriceCents *int64
	// SoldAt is when the item sold, for the same correction.
	SoldAt *time.Time
	// SoldPlaceID is where it sold. A sold item always names one, from this
	// payload or from what is already stored.
	SoldPlaceID *string
}

// ItemFilterInput narrows the admin item list, in the shape the query string
// delivers it: an empty field does not filter, and the status is parsed here.
type ItemFilterInput struct {
	// Query is a free-text search over the item's text columns.
	Query string
	// PlaceID keeps only items offered at that selling place.
	PlaceID string
	// LabelID keeps only items carrying that label.
	LabelID string
	// WhatnotNumber keeps only the item with that number.
	WhatnotNumber string
	// Status keeps only items in that status: "draft", "listed", "sold" or
	// "archived". Anything else is rejected.
	Status string
}

// UploadInput is a photo arriving from a client.
type UploadInput struct {
	// ItemID is the item the photo belongs to. It must be a live item.
	ItemID string
	// Filename is the client's name for the file, used for the object key's
	// extension and stored alongside the row.
	Filename string
	// ContentType is the upload's media type: image/jpeg, image/png or
	// image/webp. Anything else is rejected.
	ContentType string
	// Size is the upload's size in bytes, capped at 15 MB.
	Size int64
	// Body is the photo's bytes, streamed straight into storage.
	Body io.Reader
}
