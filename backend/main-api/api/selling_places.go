package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	db_platform "main-api/db"
	"main-api/db/generated"
	"main-api/db/generated/sellingplace"

	"github.com/danielgtaylor/huma/v2"
)

// SellingPlaceOutput is a selling place as returned by the API.
type SellingPlaceOutput struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	IsBuiltin bool       `json:"is_builtin"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func sellingPlaceOutput(sp *generated.SellingPlace) *SellingPlaceOutput {
	return &SellingPlaceOutput{
		ID:        sp.ID,
		Name:      sp.Name,
		IsBuiltin: sp.IsBuiltin,
		CreatedAt: sp.CreatedAt,
		DeletedAt: sp.DeletedAt,
	}
}

// getSellingPlaceOutput wraps a single place: without the Body field huma
// answers 204 and puts the fields in headers.
type getSellingPlaceOutput struct {
	Body *SellingPlaceOutput `json:"body"`
}

type listSellingPlacesOutput struct {
	Body []*SellingPlaceOutput `json:"body"`
}

type createSellingPlaceInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"100" doc:"Unique name of the place, e.g. a local store name"`
	}
}

type sellingPlaceIDInput struct {
	ID string `path:"id" doc:"Selling place ID"`
}

// registerSellingPlaces registers the /admin/selling-places CRUD routes.
func registerSellingPlaces(api huma.API, db *db_platform.Client) {
	client := db.GetDBFromContext(nil)

	huma.Register(api, huma.Operation{
		OperationID: "list-selling-places",
		Method:      http.MethodGet,
		Path:        "/admin/selling-places",
		Summary:     "List selling places",
	}, func(ctx context.Context, _ *struct{}) (*listSellingPlacesOutput, error) {
		places, err := client.SellingPlace.Query().
			Where(sellingplace.DeletedAtIsNil()).
			Order(generated.Asc(sellingplace.FieldName)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing selling places: %w", err)
		}
		out := &listSellingPlacesOutput{Body: make([]*SellingPlaceOutput, 0, len(places))}
		for _, sp := range places {
			out.Body = append(out.Body, sellingPlaceOutput(sp))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-selling-place",
		Method:      http.MethodPost,
		Path:        "/admin/selling-places",
		Summary:     "Add a custom selling place",
	}, func(ctx context.Context, in *createSellingPlaceInput) (*getSellingPlaceOutput, error) {
		sp, err := client.SellingPlace.Create().
			SetName(in.Body.Name).
			Save(ctx)
		if err != nil {
			if generated.IsConstraintError(err) {
				return nil, huma.Error409Conflict(fmt.Sprintf("selling place %q already exists", in.Body.Name))
			}
			return nil, fmt.Errorf("creating selling place: %w", err)
		}
		return &getSellingPlaceOutput{Body: sellingPlaceOutput(sp)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-selling-place",
		Method:      http.MethodDelete,
		Path:        "/admin/selling-places/{id}",
		Summary:     "Soft-delete a selling place",
	}, func(ctx context.Context, in *sellingPlaceIDInput) (*struct{}, error) {
		deletedAt := time.Now()
		err := client.SellingPlace.UpdateOneID(in.ID).
			SetDeletedAt(deletedAt).
			Exec(ctx)
		switch {
		case generated.IsNotFound(err):
			return nil, huma.Error404NotFound("selling place not found")
		case err != nil:
			return nil, fmt.Errorf("deleting selling place: %w", err)
		}
		return nil, nil
	})
}

var errSellingPlaceNotFound = errors.New("selling place not found")