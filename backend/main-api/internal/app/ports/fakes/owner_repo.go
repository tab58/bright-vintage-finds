package fakes

import (
	"context"

	"main-api/internal/app/ports"
)

var _ ports.OwnerRepository = (*FakeOwnerRepo)(nil)

// FakeOwnerRepo is an in-memory ports.OwnerRepository. Calls counts how often
// the owner was resolved, so a test can show the lookup is not repeated per
// item.
type FakeOwnerRepo struct {
	Err   error
	ID    string
	Calls int
}

// NewOwnerRepo returns an owner repository resolving to a fixed id.
func NewOwnerRepo() *FakeOwnerRepo { return &FakeOwnerRepo{ID: "owner-1"} }

func (r *FakeOwnerRepo) EnsureBuiltinOwner(_ context.Context) (string, error) {
	r.Calls++
	if r.Err != nil {
		return "", r.Err
	}
	return r.ID, nil
}
