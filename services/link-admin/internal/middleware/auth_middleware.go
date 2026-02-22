package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ws-minoro/link-admin/internal/service"
)

const UserContextKey = "user_claims"

type AuthMiddleware struct {
	authSvc *service.AuthService
}

func NewAuthMiddleware(authSvc *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authSvc: authSvc}
}

func (m *AuthMiddleware) Authenticate(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.authSvc.ValidateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		c.Locals(UserContextKey, claims)
		return c.Next()
	}

	if strings.HasPrefix(authHeader, "ApiKey ") {
		key := strings.TrimPrefix(authHeader, "ApiKey ")
		_, err := m.authSvc.ValidateAPIKey(c.Context(), key)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid api key"})
		}
		return c.Next()
	}

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization"})
}
