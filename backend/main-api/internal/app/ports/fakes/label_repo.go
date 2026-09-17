package fakes

import (
	"context"
	"sort"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.LabelRepository = (*FakeLabelRepo)(nil)

// FakeLabelRepo is an in-memory ports.LabelRepository.
type FakeLabelRepo struct {
	Err error

	ids    seq
	labels []domain.Label
}

// NewLabelRepo returns an empty label repository.
func NewLabelRepo() *FakeLabelRepo { return &FakeLabelRepo{ids: seq{prefix: "lbl"}} }

// Add seeds a label by name and returns it.
func (r *FakeLabelRepo) Add(name string) domain.Label {
	l := domain.Label{ID: r.ids.next(), Name: name, CreatedAt: FixedTime}
	r.labels = append(r.labels, l)
	return l
}

// NameOf returns a label's name, or "" when the id is unknown.
func (r *FakeLabelRepo) NameOf(id string) string {
	for _, l := range r.labels {
		if l.ID == id {
			return l.Name
		}
	}
	return ""
}

func (r *FakeLabelRepo) List(_ context.Context) ([]domain.Label, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	out := []domain.Label{}
	for _, l := range r.labels {
		if l.DeletedAt == nil {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (r *FakeLabelRepo) Create(_ context.Context, name string) (domain.Label, error) {
	if r.Err != nil {
		return domain.Label{}, r.Err
	}
	// The name is unique across live and deleted rows alike, as the index is.
	for _, l := range r.labels {
		if l.Name == name {
			return domain.Label{}, domain.Conflictf("label %q already exists", name)
		}
	}
	return r.Add(name), nil
}

func (r *FakeLabelRepo) SoftDelete(_ context.Context, id string) error {
	if r.Err != nil {
		return r.Err
	}
	for i, l := range r.labels {
		if l.ID == id {
			deletedAt := FixedTime
			l.DeletedAt = &deletedAt
			r.labels[i] = l
			return nil
		}
	}
	return domain.NotFound("label not found")
}

func (r *FakeLabelRepo) AllExist(_ context.Context, ids []string) (bool, error) {
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

func (r *FakeLabelRepo) liveExists(id string) bool {
	for _, l := range r.labels {
		if l.ID == id && l.DeletedAt == nil {
			return true
		}
	}
	return false
}
