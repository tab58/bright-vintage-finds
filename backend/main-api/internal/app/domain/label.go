package domain

import "time"

// Label is an item-type tag, e.g. Glassware.
type Label struct {
	ID        string
	Name      string
	CreatedAt time.Time
	DeletedAt *time.Time
}
