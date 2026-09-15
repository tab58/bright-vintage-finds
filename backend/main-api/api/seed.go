package api

import (
	"context"
	"fmt"

	db_platform "main-api/db"
	"main-api/db/generated/sellingplace"
)

// BuiltinSellingPlaces are the platforms the intake page offers out of the
// box. Seeded on boot; the owner can add custom places via the API.
var BuiltinSellingPlaces = []string{
	"Whatnot",
	"Local store",
	"Facebook",
	"Mercari",
	"eBay",
	"Poshmark",
}

// SeedBuiltinSellingPlaces inserts the builtin places idempotently, keyed on
// the unique name. Rows the owner deleted (soft) are not resurrected — the
// upsert only fills missing names.
func SeedBuiltinSellingPlaces(db *db_platform.Client) error {
	ctx := context.Background()
	client := db.GetDBFromContext(ctx)
	for _, name := range BuiltinSellingPlaces {
		err := client.SellingPlace.Create().
			SetName(name).
			SetIsBuiltin(true).
			OnConflictColumns(sellingplace.FieldName).
			UpdateIsBuiltin().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("seeding selling place %q: %w", name, err)
		}
	}
	return nil
}