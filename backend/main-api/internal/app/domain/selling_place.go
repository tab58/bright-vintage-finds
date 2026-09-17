package domain

import "time"

// SellingPlace is a platform or venue an item is offered on or sold through.
type SellingPlace struct {
	ID        string
	Name      string
	IsBuiltin bool
	CreatedAt time.Time
	DeletedAt *time.Time
}

// BuiltinSellingPlaces are the platforms the intake page offers out of the
// box. Seeded on boot; the owner can add custom places through the API.
var BuiltinSellingPlaces = []string{
	"Whatnot",
	"Local store",
	"Facebook",
	"Mercari",
	"eBay",
	"Poshmark",
}
