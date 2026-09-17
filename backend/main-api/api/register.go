// Package api is the service's HTTP driving adapter. It builds the server and
// hands the route registration to api/routes, which owns the wire format.
package api

import (
	"main-api/api/routes"
	"main-api/internal/app"

	server "github.com/tab58/huma-http-server"
	"github.com/tab58/huma-http-server/router"
)

// NewServer constructs the HTTP server and registers the platform /healthz
// route, which is always on the auth/logging skip list. When an application
// is given, the inventory admin routes and the unauthenticated /public
// catalog routes are registered too.
func NewServer[A router.AuthInfo](cfg server.ServerConfig, builder router.AuthInfoBuilder[A], a *app.Application, opts ...server.ServerConfigOption) *server.Server[A] {
	opts = append(opts, server.WithSkipPaths([]string{"/healthz"}))
	srv := server.New(cfg, builder, opts...)

	routes.RegisterHealthz(srv.API())

	if a != nil {
		routes.RegisterSellingPlaces(srv.API(), a)
		routes.RegisterLabels(srv.API(), a)
		routes.RegisterItemCRUD(srv.API(), a)
		routes.RegisterPublicCatalog(srv.API(), a)
		if a.HasImageStore() {
			routes.RegisterItemImageRoutes(srv.API(), a)
		}
	}

	return srv
}
