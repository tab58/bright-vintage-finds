package routes

import (
	"context"
	"net/http"
	"time"

	"main-api/internal/app"
	"main-api/internal/app/domain"

	"github.com/danielgtaylor/huma/v2"
)

// LabelOutput is an item-type label as returned by the API.
type LabelOutput struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func labelOutput(l domain.Label) *LabelOutput {
	return &LabelOutput{
		ID:        l.ID,
		Name:      l.Name,
		CreatedAt: l.CreatedAt,
		DeletedAt: l.DeletedAt,
	}
}

// getLabelOutput wraps a single label for the same reason as places: a
// response struct without a Body field becomes a 204 with header fields.
type getLabelOutput struct {
	Body *LabelOutput `json:"body"`
}

type listLabelsOutput struct {
	Body []*LabelOutput `json:"body"`
}

type createLabelInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"100" doc:"Unique label name, e.g. Glassware"`
	}
}

type labelIDInput struct {
	ID string `path:"id" doc:"Label ID"`
}

// RegisterLabels registers the /admin/labels CRUD routes. Labels have no
// rules beyond storage, so the handlers talk to the repository port directly.
func RegisterLabels(api huma.API, a *app.Application) {
	huma.Register(api, huma.Operation{
		OperationID: "list-labels",
		Method:      http.MethodGet,
		Path:        "/admin/labels",
		Summary:     "List item-type labels",
	}, func(ctx context.Context, _ *struct{}) (*listLabelsOutput, error) {
		labels, err := a.ListLabels(ctx)
		if err != nil {
			return nil, toHTTP(err)
		}
		out := &listLabelsOutput{Body: make([]*LabelOutput, 0, len(labels))}
		for _, l := range labels {
			out.Body = append(out.Body, labelOutput(l))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-label",
		Method:      http.MethodPost,
		Path:        "/admin/labels",
		Summary:     "Add an item-type label",
	}, func(ctx context.Context, in *createLabelInput) (*getLabelOutput, error) {
		l, err := a.CreateLabel(ctx, in.Body.Name)
		if err != nil {
			return nil, toHTTP(err)
		}
		return &getLabelOutput{Body: labelOutput(l)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-label",
		Method:      http.MethodDelete,
		Path:        "/admin/labels/{id}",
		Summary:     "Soft-delete an item-type label",
	}, func(ctx context.Context, in *labelIDInput) (*struct{}, error) {
		if err := a.DeleteLabel(ctx, in.ID); err != nil {
			return nil, toHTTP(err)
		}
		return nil, nil
	})
}
