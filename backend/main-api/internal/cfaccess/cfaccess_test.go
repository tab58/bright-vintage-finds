package cfaccess

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/golang-jwt/jwt/v5"
)

const testAUD = "test-aud-tag"
const testKID = "test-key-1"

// newJWKSServer serves a JWKS document for the given RSA public key.
func newJWKSServer(t *testing.T, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()
	jwks := map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"kid": testKID,
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("failed to encode JWKS: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

// runMiddleware sends one request through mw and reports whether next was
// called plus the recorded response.
func runMiddleware(t *testing.T, mw Middleware, path, assertion string) (bool, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if assertion != "" {
		req.Header.Set(AssertionHeader, assertion)
	}
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{Method: http.MethodGet, Path: path}, req, rec)

	nextCalled := false
	mw(ctx, func(huma.Context) { nextCalled = true })
	return nextCalled, rec
}

func TestNewConfigModes(t *testing.T) {
	tests := []struct {
		name       string
		teamDomain string
		aud        string
		devBypass  bool
		wantErr    bool
	}{
		{"fully configured", "team.cloudflareaccess.com", testAUD, false, false},
		{"unconfigured dev", "", "", true, false},
		{"unconfigured production", "", "", false, false},
		{"only team domain", "team.cloudflareaccess.com", "", false, true},
		{"only aud", "", testAUD, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// use a local JWKS URL for the configured case so the
			// constructor does not reach out to a real team domain
			var mw Middleware
			var err error
			if tt.teamDomain != "" && tt.aud != "" {
				key, keyErr := rsa.GenerateKey(rand.Reader, 2048)
				if keyErr != nil {
					t.Fatalf("failed to generate key: %v", keyErr)
				}
				srv := newJWKSServer(t, &key.PublicKey)
				mw, err = newWithJWKSURL(srv.URL, tt.aud)
			} else {
				mw, err = New(tt.teamDomain, tt.aud, tt.devBypass)
			}

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected constructor error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mw == nil {
				t.Fatal("expected middleware, got nil")
			}
		})
	}
}

func TestUnconfiguredModes(t *testing.T) {
	tests := []struct {
		name      string
		devBypass bool
		path      string
		wantNext  bool
		wantCode  int
	}{
		{"dev bypass opens admin path", true, "/admin/items", true, 0},
		{"production fails closed on admin path", false, "/admin/items", false, http.StatusUnauthorized},
		{"production fails closed on bare admin path", false, "/admin", false, http.StatusUnauthorized},
		{"production leaves public path open", false, "/healthz", true, 0},
		{"production leaves admin-prefixed word open", false, "/administrata", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw, err := New("", "", tt.devBypass)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			nextCalled, rec := runMiddleware(t, mw, tt.path, "")
			if nextCalled != tt.wantNext {
				t.Fatalf("nextCalled = %v, want %v", nextCalled, tt.wantNext)
			}
			if !tt.wantNext && rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func TestConfiguredVerification(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	srv := newJWKSServer(t, &key.PublicKey)
	mw, err := newWithJWKSURL(srv.URL, testAUD)
	if err != nil {
		t.Fatalf("failed to build middleware: %v", err)
	}

	validClaims := func() jwt.MapClaims {
		return jwt.MapClaims{
			"aud":   testAUD,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
			"email": "owner@example.com",
		}
	}

	tests := []struct {
		name      string
		path      string
		assertion string
		wantNext  bool
	}{
		{"non-admin path passes without assertion", "/items", "", true},
		{"valid assertion on admin path", "/admin/items", signToken(t, key, validClaims()), true},
		{"missing assertion on admin path", "/admin/items", "", false},
		{"expired token", "/admin/items", signToken(t, key, jwt.MapClaims{
			"aud": testAUD,
			"exp": time.Now().Add(-time.Hour).Unix(),
		}), false},
		{"wrong audience", "/admin/items", signToken(t, key, jwt.MapClaims{
			"aud": "some-other-app",
			"exp": time.Now().Add(time.Hour).Unix(),
		}), false},
		{"missing expiry", "/admin/items", signToken(t, key, jwt.MapClaims{
			"aud": testAUD,
		}), false},
		{"signed by unknown key", "/admin/items", signToken(t, otherKey, validClaims()), false},
		{"garbage assertion", "/admin/items", "not-a-jwt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled, rec := runMiddleware(t, mw, tt.path, tt.assertion)
			if nextCalled != tt.wantNext {
				t.Fatalf("nextCalled = %v, want %v (status %d)", nextCalled, tt.wantNext, rec.Code)
			}
			if !tt.wantNext && rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}
