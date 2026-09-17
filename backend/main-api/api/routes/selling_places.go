package routes

import (
	"context"
	"errors"
	"net/http"
	"time"

	"main-api/internal/app"
	"main-api/internal/app/domain"

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

func sellingPlaceOutput(sp domain.SellingPlace) *SellingPlaceOutput {
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

// RegisterSellingPlaces registers the /admin/selling-places CRUD routes.
// Places have no rules beyond storage, so the handlers talk to the repository
// port directly.
func RegisterSellingPlaces(api huma.API, a *app.Application) {
	huma.Register(api, huma.Operation{
		OperationID: "list-selling-places",
		Method:      http.MethodGet,
		Path:        "/admin/selling-places",
		Summary:     "List selling places",
	}, func(ctx context.Context, _ *struct{}) (*listSellingPlacesOutput, error) {
		places, err := a.ListSellingPlaces(ctx)
		if err != nil {
			return nil, toHTTP(err)
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
		sp, err := a.CreateSellingPlace(ctx, in.Body.Name)
		if err != nil {
			return nil, toHTTP(err)
		}
		return &getSellingPlaceOutput{Body: sellingPlaceOutput(sp)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-selling-place",
		Method:      http.MethodDelete,
		Path:        "/admin/selling-places/{id}",
		Summary:     "Soft-delete a selling place",
	}, func(ctx context.Context, in *sellingPlaceIDInput) (*struct{}, error) {
		if err := a.DeleteSellingPlace(ctx, in.ID); err != nil {
			return nil, toHTTP(err)
		}
		return nil, nil
	})
}

var errSellingPlaceNotFound = errors.New("selling place not found")
