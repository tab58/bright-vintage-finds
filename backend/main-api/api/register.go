package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	server "github.com/tab58/huma-http-server"
	"github.com/tab58/huma-http-server/router"
)

// NewServer constructs the HTTP server and registers the platform /healthz
// route, which is always on the auth/logging skip list. When deps carries a
// database client, the inventory admin routes and the unauthenticated
// /public catalog routes are registered too.
func NewServer[A router.AuthInfo](cfg server.ServerConfig, builder router.AuthInfoBuilder[A], deps *AppDeps, opts ...server.ServerConfigOption) *server.Server[A] {
	opts = append(opts, server.WithSkipPaths([]string{"/healthz"}))
	srv := server.New(cfg, builder, opts...)

	registerHealthz(srv.API())

	if deps != nil && deps.DB != nil {
		registerSellingPlaces(srv.API(), deps.DB)
		registerLabels(srv.API(), deps.DB)
		registerItemCRUD(srv.API(), deps)
		registerPublicCatalog(srv.API(), deps)
		if deps.Store != nil {
			registerItemImageRoutes(srv.API(), deps)
		}
	}

	return srv
}

// registerHealthz registers the platform healthcheck. Apps must not
// register their own /healthz.
func registerHealthz(api huma.API) {
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
