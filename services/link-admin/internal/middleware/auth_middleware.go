package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ws-minoro/link-admin/internal/repository"
	"github.com/ws-minoro/link-admin/internal/service"
)

const UserContextKey = "user_claims"

// AuthValidator is the subset of service.AuthService the middleware depends
// on. Defined here so tests can inject a fake instead of requiring a live
// Postgres connection and real JWT signing.
type AuthValidator interface {
	ValidateAccessToken(tokenStr string) (*service.Claims, error)
	ValidateAPIKey(ctx context.Context, keyStr string) (*repository.User, error)
}

type AuthMiddleware struct {
	authSvc AuthValidator
}

func NewAuthMiddleware(authSvc AuthValidator) *AuthMiddleware {
	return &AuthMiddleware{authSvc: authSvc}
}

func (m *AuthMiddleware) Authenticate(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if strings.HasPrefix(authHeader, "Bearer ") {
		return m.authenticateJWT(c, strings.TrimPrefix(authHeader, "Bearer "))
	}

	if strings.HasPrefix(authHeader, "ApiKey ") {
		key := strings.TrimPrefix(authHeader, "ApiKey ")
		user, err := m.authSvc.ValidateAPIKey(c.Context(), key)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid api key"})
		}
		c.Locals(UserContextKey, &service.Claims{
			UserID:   user.ID.String(),
			TenantID: user.TenantID.String(),
			Role:     user.Role,
		})
		return c.Next()
	}

	// No Authorization header — the frontend authenticates via an httpOnly
	// session cookie instead of a manually attached header.
	if cookieToken := c.Cookies("access_token"); cookieToken != "" {
		return m.authenticateJWT(c, cookieToken)
	}

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization"})
}

func (m *AuthMiddleware) authenticateJWT(c *fiber.Ctx, token string) error {
	claims, err := m.authSvc.ValidateAccessToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}
	c.Locals(UserContextKey, claims)
	return c.Next()
}
