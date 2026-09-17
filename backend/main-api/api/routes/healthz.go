package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type healthzOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

// RegisterHealthz registers the platform healthcheck. Apps must not register
// their own /healthz; NewServer keeps this path on the auth/logging skip list.
func RegisterHealthz(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "healthcheck",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Platform healthcheck endpoint",
	}, func(context.Context, *struct{}) (*healthzOutput, error) {
		out := &healthzOutput{}
		out.Body.Status = "ok"
		return out, nil
	})
}
