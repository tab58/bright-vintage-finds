// Package ent implements the repository ports on top of the Ent client. It is
// the only package that knows both the generated row types and the domain
// types, and the only one that turns database failures into domain errors.
package ent

import (
	"main-api/db/generated"
	"main-api/internal/app/domain"
)

// toItem maps a row to its domain form. Edges are copied when they were
// eager-loaded; a query that did not load them yields an item with empty
// edge slices rather than a partially-true one, so every read path that
// feeds an output mapping must load them.
func toItem(row *generated.Item) domain.Item {
	it := domain.Item{
		ID:                   row.ID,
		Name:                 row.Name,
		Description:          row.Description,
		Category:             row.Category,
		Condition:            row.Condition,
		Status:               domain.Status(row.Status),
		AcquisitionCostCents: row.AcquisitionCostCents,
		PurchasedAt:          row.PurchasedAt,
		ListingPriceCents:    row.ListingPriceCents,
		Length:               row.Length,
		Width:                row.Width,
		Height:               row.Height,
		MeasurementUnit:      domain.MeasurementUnit(row.MeasurementUnit),
		ExtraMeasurements:    row.ExtraMeasurements,
		WeightLbs:            row.WeightLbs,
		WeightOz:             row.WeightOz,
		Notes:                row.Notes,
		WhatnotNumber:        row.WhatnotNumber,
		ListedAt:             row.ListedAt,
		FirstListedAt:        row.FirstListedAt,
		SoldPriceCents:       row.SoldPriceCents,
		SoldAt:               row.SoldAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
		SellingPlaces:        make([]domain.SellingPlace, 0, len(row.Edges.SellingPlaces)),
		Labels:               make([]domain.Label, 0, len(row.Edges.Labels)),
	}
	for _, sp := range row.Edges.SellingPlaces {
		it.SellingPlaces = append(it.SellingPlaces, toSellingPlace(sp))
	}
	for _, l := range row.Edges.Labels {
		it.Labels = append(it.Labels, toLabel(l))
	}
	if row.Edges.SoldPlace != nil {
		place := toSellingPlace(row.Edges.SoldPlace)
		it.SoldPlace = &place
	}
	return it
}

func toItems(rows []*generated.Item) []domain.Item {
	items := make([]domain.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, toItem(row))
	}
	return items
}

func toSellingPlace(row *generated.SellingPlace) domain.SellingPlace {
	return domain.SellingPlace{
		ID:        row.ID,
		Name:      row.Name,
		IsBuiltin: row.IsBuiltin,
		CreatedAt: row.CreatedAt,
		DeletedAt: row.DeletedAt,
	}
}

func toLabel(row *generated.Label) domain.Label {
	return domain.Label{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		DeletedAt: row.DeletedAt,
	}
}

func toItemImage(row *generated.ItemImage) domain.ItemImage {
	img := domain.ItemImage{
		ID:           row.ID,
		UploadBucket: row.UploadBucket,
		UploadKey:    row.UploadKey,
		Filename:     row.Filename,
		ContentType:  row.ContentType,
		DisplayOrder: row.DisplayOrder,
	}
	if row.Edges.Item != nil {
		img.ItemID = row.Edges.Item.ID
	}
	return img
}

// wrap turns a database failure into a domain error. A missing row and a
// violated constraint carry the message the caller sees; anything else is
// internal, and its message stays in the logs.
func wrap(op string, err error, notFound, conflict string) error {
	switch {
	case err == nil:
		return nil
	case generated.IsNotFound(err) && notFound != "":
		return domain.NotFound(notFound)
	case generated.IsConstraintError(err) && conflict != "":
		return domain.Conflict(conflict)
	default:
		return domain.Internal(op, err)
	}
}
