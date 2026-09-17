package ent

import (
	"context"
	"time"

	"main-api/db/generated"
	"main-api/db/generated/label"
	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.LabelRepository = (*LabelRepo)(nil)

// LabelRepo stores item-type labels.
type LabelRepo struct {
	client *generated.Client
}

// NewLabelRepo returns a label repository over the given Ent client.
func NewLabelRepo(client *generated.Client) *LabelRepo { return &LabelRepo{client: client} }

func (r *LabelRepo) List(ctx context.Context) ([]domain.Label, error) {
	rows, err := r.client.Label.Query().
		Where(label.DeletedAtIsNil()).
		Order(generated.Asc(label.FieldName)).
		All(ctx)
	if err != nil {
		return nil, domain.Internal("listing labels", err)
	}
	labels := make([]domain.Label, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, toLabel(row))
	}
	return labels, nil
}

func (r *LabelRepo) Create(ctx context.Context, name string) (domain.Label, error) {
	row, err := r.client.Label.Create().SetName(name).Save(ctx)
	if err != nil {
		return domain.Label{}, wrap("creating label", err, "", labelTaken(name))
	}
	return toLabel(row), nil
}

func (r *LabelRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.client.Label.UpdateOneID(id).SetDeletedAt(time.Now()).Exec(ctx)
	return wrap("deleting label", err, "label not found", "")
}

func (r *LabelRepo) AllExist(ctx context.Context, ids []string) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	n, err := r.client.Label.Query().
		Where(label.IDIn(ids...), label.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		return false, domain.Internal("checking labels", err)
	}
	return n == len(ids), nil
}

func labelTaken(name string) string { return "label " + quote(name) + " already exists" }
