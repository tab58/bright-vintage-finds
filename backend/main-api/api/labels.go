package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	db_platform "main-api/db"
	"main-api/db/generated"
	"main-api/db/generated/label"

	"github.com/danielgtaylor/huma/v2"
)

// LabelOutput is an item-type label as returned by the API.
type LabelOutput struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func labelOutput(l *generated.Label) *LabelOutput {
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

// registerLabels registers the /admin/labels CRUD routes.
func registerLabels(api huma.API, db *db_platform.Client) {
	client := db.GetDBFromContext(nil)

	huma.Register(api, huma.Operation{
		OperationID: "list-labels",
		Method:      http.MethodGet,
		Path:        "/admin/labels",
		Summary:     "List item-type labels",
	}, func(ctx context.Context, _ *struct{}) (*listLabelsOutput, error) {
		labels, err := client.Label.Query().
			Where(label.DeletedAtIsNil()).
			Order(generated.Asc(label.FieldName)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing labels: %w", err)
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
		l, err := client.Label.Create().
			SetName(in.Body.Name).
			Save(ctx)
		if err != nil {
			if generated.IsConstraintError(err) {
				return nil, huma.Error409Conflict(fmt.Sprintf("label %q already exists", in.Body.Name))
			}
			return nil, fmt.Errorf("creating label: %w", err)
		}
		return &getLabelOutput{Body: labelOutput(l)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-label",
		Method:      http.MethodDelete,
		Path:        "/admin/labels/{id}",
		Summary:     "Soft-delete an item-type label",
	}, func(ctx context.Context, in *labelIDInput) (*struct{}, error) {
		deletedAt := time.Now()
		err := client.Label.UpdateOneID(in.ID).
			SetDeletedAt(deletedAt).
			Exec(ctx)
		switch {
		case generated.IsNotFound(err):
			return nil, huma.Error404NotFound("label not found")
		case err != nil:
			return nil, fmt.Errorf("deleting label: %w", err)
		}
		return nil, nil
	})
}