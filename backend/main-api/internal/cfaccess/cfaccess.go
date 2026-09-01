// Package cfaccess guards /admin routes behind Cloudflare Access.
//
// Cloudflare Access authenticates admin requests at the edge and injects a
// Cf-Access-Jwt-Assertion header (RS256 JWT). This middleware independently
// verifies that assertion so the backend never trusts the edge blindly:
// signature against the team's JWKS, audience against the Access app's AUD
// tag. Unconfigured servers fail closed on admin paths, except in
// development where there is no Cloudflare edge in front of the server.
package cfaccess

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AssertionHeader is the header Cloudflare Access sets on requests that
// passed an Access policy.
const AssertionHeader = "Cf-Access-Jwt-Assertion"

const adminPathPrefix = "/admin"

// Middleware is the huma middleware shape the server accepts
// (server.WithMiddleware).
type Middleware = func(ctx huma.Context, next func(huma.Context))

// New builds the admin-route guard.
//
// teamDomain is the Zero Trust team domain (e.g. "team.cloudflareaccess.com")
// and aud the Access application's AUD tag; set both or neither. When both
// are set, admin requests must carry a valid assertion. When neither is set:
// devBypass true leaves admin paths open (local development), devBypass false
// denies them outright (fail closed).
func New(teamDomain, aud string, devBypass bool) (Middleware, error) {
	configured := teamDomain != "" && aud != ""
	if !configured {
		if teamDomain != "" || aud != "" {
			return nil, fmt.Errorf("cfaccess: CF_ACCESS_TEAM_DOMAIN and CF_ACCESS_AUD must be set together")
		}
		if devBypass {
			return func(c huma.Context, next func(huma.Context)) { next(c) }, nil
		}
		return func(c huma.Context, next func(huma.Context)) {
			if isAdminPath(c.URL().Path) {
				deny(c)
				return
			}
			next(c)
		}, nil
	}

	jwksURL := fmt.Sprintf("https://%s/cdn-cgi/access/certs", teamDomain)
	return newWithJWKSURL(jwksURL, aud)
}

// newWithJWKSURL is the configured-mode constructor, split out so tests can
// point it at a local JWKS server.
func newWithJWKSURL(jwksURL, aud string) (Middleware, error) {
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("cfaccess: failed to load JWKS from %s: %w", jwksURL, err)
	}

	return func(c huma.Context, next func(huma.Context)) {
		if !isAdminPath(c.URL().Path) {
			next(c)
			return
		}
		raw := c.Header(AssertionHeader)
		if raw == "" {
			deny(c)
			return
		}
		if _, err := jwt.Parse(raw, jwks.Keyfunc,
			jwt.WithAudience(aud),
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithExpirationRequired(),
		); err != nil {
			deny(c)
			return
		}
		next(c)
	}, nil
}

func isAdminPath(path string) bool {
	return path == adminPathPrefix || strings.HasPrefix(path, adminPathPrefix+"/")
}

func deny(c huma.Context) {
	c.SetHeader("Content-Type", "application/problem+json")
	c.SetStatus(http.StatusUnauthorized)
	// ponytail: static body; switch to huma.WriteErr if richer errors needed
	fmt.Fprint(c.BodyWriter(), `{"title":"Unauthorized","status":401,"detail":"missing or invalid Cloudflare Access assertion"}`)
}
