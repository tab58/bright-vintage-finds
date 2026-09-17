package domain

// The inventory is single-owner. The owner is a well-known user row created
// on first use; its identity is not derived from Cloudflare Access headers.
const (
	OwnerIDPID  = "builtin-owner"
	OwnerEmail  = "owner@local"
	OwnerName   = "Owner"
	OwnerStatus = "active"
)
