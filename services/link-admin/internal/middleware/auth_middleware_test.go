package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
	"github.com/ws-minoro/link-admin/internal/service"
)

type fakeAuthValidator struct {
	validToken  string
	claims      *service.Claims
	validAPIKey string
	apiKeyUser  *repository.User
}

func (f *fakeAuthValidator) ValidateAccessToken(tokenStr string) (*service.Claims, error) {
	if tokenStr == f.validToken && f.claims != nil {
		return f.claims, nil
	}
	return nil, service.ErrUnauthorized
}

func (f *fakeAuthValidator) ValidateAPIKey(ctx context.Context, keyStr string) (*repository.User, error) {
	if keyStr == f.validAPIKey && f.apiKeyUser != nil {
		return f.apiKeyUser, nil
	}
	return nil, errors.New("invalid api key")
}

// newTestApp wires the middleware into a minimal Fiber app whose downstream
// handler echoes back whatever Authenticate put in c.Locals(UserContextKey),
// so tests can assert the tenant-scoping context was actually populated —
// not just that the request was allowed through.
func newTestApp(authSvc AuthValidator) *fiber.App {
	mw := NewAuthMiddleware(authSvc)
	app := fiber.New()
	app.Use(mw.Authenticate)
	app.Get("/protected", func(c *fiber.Ctx) error {
		claims, ok := c.Locals(UserContextKey).(*service.Claims)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "no claims in context"})
		}
		return c.JSON(fiber.Map{"tenant_id": claims.TenantID, "role": claims.Role})
	})
	return app
}

func TestAuthenticate_BearerToken_SetsClaimsInContext(t *testing.T) {
	tenantID := uuid.New().String()
	authSvc := &fakeAuthValidator{
		validToken: "good-token",
		claims:     &service.Claims{TenantID: tenantID, Role: "owner"},
	}
	app := newTestApp(authSvc)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthenticate_InvalidBearerToken_Rejected(t *testing.T) {
	authSvc := &fakeAuthValidator{validToken: "good-token"}
	app := newTestApp(authSvc)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// Regression test: the ApiKey branch used to validate the key but never
// call c.Locals(UserContextKey, ...), so every tenant-scoped handler behind
// a valid API key still got 401 from its own context lookup. This confirms
// the fix — a valid API key must leave usable claims in the request context.
func TestAuthenticate_ApiKey_PopulatesTenantContext(t *testing.T) {
	tenantID := uuid.New()
	authSvc := &fakeAuthValidator{
		validAPIKey: "good-api-key",
		apiKeyUser:  &repository.User{TenantID: tenantID, Role: "admin"},
	}
	app := newTestApp(authSvc)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "ApiKey good-api-key")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 (claims populated), got %d", resp.StatusCode)
	}
}

func TestAuthenticate_InvalidApiKey_Rejected(t *testing.T) {
	authSvc := &fakeAuthValidator{validAPIKey: "good-api-key"}
	app := newTestApp(authSvc)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "ApiKey wrong-key")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthenticate_CookieFallback_SetsClaimsInContext(t *testing.T) {
	tenantID := uuid.New().String()
	authSvc := &fakeAuthValidator{
		validToken: "cookie-token",
		claims:     &service.Claims{TenantID: tenantID, Role: "viewer"},
	}
	app := newTestApp(authSvc)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthenticate_NoCredentials_Rejected(t *testing.T) {
	app := newTestApp(&fakeAuthValidator{})

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
