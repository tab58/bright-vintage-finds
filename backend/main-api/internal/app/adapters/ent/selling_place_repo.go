package ent

import (
	"context"
	"strconv"
	"time"

	"main-api/db/generated"
	"main-api/db/generated/sellingplace"
	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.SellingPlaceRepository = (*SellingPlaceRepo)(nil)

// SellingPlaceRepo stores the platforms and venues items are sold on.
type SellingPlaceRepo struct {
	client *generated.Client
}

// NewSellingPlaceRepo returns a selling-place repository over the given Ent
// client.
func NewSellingPlaceRepo(client *generated.Client) *SellingPlaceRepo {
	return &SellingPlaceRepo{client: client}
}

func (r *SellingPlaceRepo) List(ctx context.Context) ([]domain.SellingPlace, error) {
	rows, err := r.client.SellingPlace.Query().
		Where(sellingplace.DeletedAtIsNil()).
		Order(generated.Asc(sellingplace.FieldName)).
		All(ctx)
	if err != nil {
		return nil, domain.Internal("listing selling places", err)
	}
	places := make([]domain.SellingPlace, 0, len(rows))
	for _, row := range rows {
		places = append(places, toSellingPlace(row))
	}
	return places, nil
}

func (r *SellingPlaceRepo) Create(ctx context.Context, name string) (domain.SellingPlace, error) {
	row, err := r.client.SellingPlace.Create().SetName(name).Save(ctx)
	if err != nil {
		return domain.SellingPlace{}, wrap("creating selling place", err, "", placeTaken(name))
	}
	return toSellingPlace(row), nil
}

func (r *SellingPlaceRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.client.SellingPlace.UpdateOneID(id).SetDeletedAt(time.Now()).Exec(ctx)
	return wrap("deleting selling place", err, "selling place not found", "")
}

func (r *SellingPlaceRepo) Exists(ctx context.Context, id string) (bool, error) {
	found, err := r.client.SellingPlace.Query().
		Where(sellingplace.IDEQ(id), sellingplace.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return false, domain.Internal("checking selling place", err)
	}
	return found, nil
}

func (r *SellingPlaceRepo) AllExist(ctx context.Context, ids []string) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	n, err := r.client.SellingPlace.Query().
		Where(sellingplace.IDIn(ids...), sellingplace.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		return false, domain.Internal("checking selling places", err)
	}
	return n == len(ids), nil
}

func (r *SellingPlaceRepo) EnsureBuiltins(ctx context.Context, names []string) error {
	for _, name := range names {
		// Keyed on the unique name, so the upsert only fills missing rows.
		// Places the owner soft-deleted are not resurrected.
		err := r.client.SellingPlace.Create().
			SetName(name).
			SetIsBuiltin(true).
			OnConflictColumns(sellingplace.FieldName).
			UpdateIsBuiltin().
			Exec(ctx)
		if err != nil {
			return domain.Internal("seeding selling place "+quote(name), err)
		}
	}
	return nil
}

func placeTaken(name string) string { return "selling place " + quote(name) + " already exists" }

// quote renders a name the way %q does, which is how the conflict messages
// have always read.
func quote(s string) string { return strconv.Quote(s) }
