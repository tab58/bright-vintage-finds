package ent

import (
	"context"

	"main-api/db/generated"
	"main-api/db/generated/user"
	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.OwnerRepository = (*OwnerRepo)(nil)

// OwnerRepo resolves the single owner the inventory belongs to.
type OwnerRepo struct {
	client *generated.Client
}

// NewOwnerRepo returns an owner repository over the given Ent client.
func NewOwnerRepo(client *generated.Client) *OwnerRepo { return &OwnerRepo{client: client} }

// EnsureBuiltinOwner returns the owner's user id, creating the well-known row
// on first use. The row is created once and never deleted, so it looks the
// row up first and only creates on a miss.
func (r *OwnerRepo) EnsureBuiltinOwner(ctx context.Context) (string, error) {
	found, err := r.client.User.Query().
		Where(user.IdpID(domain.OwnerIDPID)).
		First(ctx)
	if err == nil {
		return found.ID, nil
	}
	if !generated.IsNotFound(err) {
		return "", domain.Internal("resolving owner user", err)
	}

	created, err := r.client.User.Create().
		SetIdpID(domain.OwnerIDPID).
		SetEmail(domain.OwnerEmail).
		SetFullName(domain.OwnerName).
		SetAccountStatus(user.AccountStatus(domain.OwnerStatus)).
		OnConflictColumns(user.FieldIdpID).
		UpdateEmail().
		ID(ctx)
	if err != nil {
		return "", domain.Internal("creating owner user", err)
	}
	return created, nil
}
