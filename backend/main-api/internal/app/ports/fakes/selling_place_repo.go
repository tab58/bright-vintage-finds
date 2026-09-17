package fakes

import (
	"context"
	"sort"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.SellingPlaceRepository = (*FakeSellingPlaceRepo)(nil)

// FakeSellingPlaceRepo is an in-memory ports.SellingPlaceRepository.
type FakeSellingPlaceRepo struct {
	Err error

	ids    seq
	places []domain.SellingPlace
}

// NewSellingPlaceRepo returns an empty selling-place repository.
func NewSellingPlaceRepo() *FakeSellingPlaceRepo {
	return &FakeSellingPlaceRepo{ids: seq{prefix: "plc"}}
}

// Add seeds a place by name and returns it.
func (r *FakeSellingPlaceRepo) Add(name string) domain.SellingPlace {
	sp := domain.SellingPlace{ID: r.ids.next(), Name: name, CreatedAt: FixedTime}
	r.places = append(r.places, sp)
	return sp
}

// NameOf returns a place's name, or "" when the id is unknown.
func (r *FakeSellingPlaceRepo) NameOf(id string) string {
	for _, sp := range r.places {
		if sp.ID == id {
			return sp.Name
		}
	}
	return ""
}

func (r *FakeSellingPlaceRepo) List(_ context.Context) ([]domain.SellingPlace, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	out := []domain.SellingPlace{}
	for _, sp := range r.places {
		if sp.DeletedAt == nil {
			out = append(out, sp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (r *FakeSellingPlaceRepo) Create(_ context.Context, name string) (domain.SellingPlace, error) {
	if r.Err != nil {
		return domain.SellingPlace{}, r.Err
	}
	for _, sp := range r.places {
		if sp.Name == name {
			return domain.SellingPlace{}, domain.Conflictf("selling place %q already exists", name)
		}
	}
	return r.Add(name), nil
}

func (r *FakeSellingPlaceRepo) SoftDelete(_ context.Context, id string) error {
	if r.Err != nil {
		return r.Err
	}
	for i, sp := range r.places {
		if sp.ID == id {
			deletedAt := FixedTime
			sp.DeletedAt = &deletedAt
			r.places[i] = sp
			return nil
		}
	}
	return domain.NotFound("selling place not found")
}

func (r *FakeSellingPlaceRepo) Exists(_ context.Context, id string) (bool, error) {
	if r.Err != nil {
		return false, r.Err
	}
	return r.liveExists(id), nil
}

func (r *FakeSellingPlaceRepo) AllExist(_ context.Context, ids []string) (bool, error) {
	if r.Err != nil {
		return false, r.Err
	}
	for _, id := range ids {
		if !r.liveExists(id) {
			return false, nil
		}
	}
	return true, nil
}

func (r *FakeSellingPlaceRepo) EnsureBuiltins(_ context.Context, names []string) error {
	if r.Err != nil {
		return r.Err
	}
	for _, name := range names {
		if r.nameTaken(name) {
			// Already present, deleted or not: the upsert only fills gaps and
			// never resurrects what the owner removed.
			continue
		}
		sp := r.Add(name)
		r.markBuiltin(sp.ID)
	}
	return nil
}

func (r *FakeSellingPlaceRepo) markBuiltin(id string) {
	for i, sp := range r.places {
		if sp.ID == id {
			sp.IsBuiltin = true
			r.places[i] = sp
			return
		}
	}
}

func (r *FakeSellingPlaceRepo) nameTaken(name string) bool {
	for _, sp := range r.places {
		if sp.Name == name {
			return true
		}
	}
	return false
}

func (r *FakeSellingPlaceRepo) liveExists(id string) bool {
	for _, sp := range r.places {
		if sp.ID == id && sp.DeletedAt == nil {
			return true
		}
	}
	return false
}
