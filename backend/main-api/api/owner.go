package api

import (
	"context"
	"fmt"

	"main-api/db/generated"
	"main-api/db/generated/user"

	"github.com/danielgtaylor/huma/v2"
)

// The inventory is single-owner. The owner is a well-known User row seeded at
// boot; its identity is not derived from Cloudflare Access headers.
const (
	ownerIDPID  = "builtin-owner"
	ownerEmail  = "owner@local"
	ownerName   = "Owner"
	ownerStatus = "active"
)

// ensureOwner returns the single owner's user ID, creating the well-known
// owner row on first use (idempotent, keyed on idp_id).
func ensureOwner(ctx context.Context, client *generated.Client) (string, error) {
	// The owner row is created once and never deleted; lookup first, then
	// create on miss.
	found, err := client.User.Query().
		Where(user.IdpID(ownerIDPID)).
		First(ctx)
	if err == nil {
		return found.ID, nil
	}
	if !generated.IsNotFound(err) {
		return "", huma.Error500InternalServerError(fmt.Sprintf("resolving owner user: %v", err))
	}

	created, err := client.User.Create().
		SetIdpID(ownerIDPID).
		SetEmail(ownerEmail).
		SetFullName(ownerName).
		SetAccountStatus(user.AccountStatusActive).
		OnConflictColumns(user.FieldIdpID).
		UpdateEmail().
		ID(ctx)
	if err != nil {
		return "", huma.Error500InternalServerError(fmt.Sprintf("creating owner user: %v", err))
	}
	return created, nil
}