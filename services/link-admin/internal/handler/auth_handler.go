package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ws-minoro/link-admin/internal/service"
)

const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
	// authStateCookie is a non-sensitive, JS-readable marker (no token
	// material) the frontend uses to know a session is active, since the
	// real tokens live in httpOnly cookies it cannot read.
	authStateCookie = "auth_state"
)

type AuthHandler struct {
	authSvc      *service.AuthService
	cookieSecure bool
}

func NewAuthHandler(authSvc *service.AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, cookieSecure: cookieSecure}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req struct {
		TenantName string `json:"tenant_name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	accessToken, refreshToken, err := h.authSvc.Register(c.Context(), req.TenantName, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRegistration):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tenant name, a valid email and a password with at least 8 characters are required"})
		case errors.Is(err, service.ErrEmailTaken):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already registered"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to register"})
		}
	}

	h.setSessionCookies(c, accessToken, refreshToken)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "ok"})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	accessToken, refreshToken, err := h.authSvc.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	h.setSessionCookies(c, accessToken, refreshToken)
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	refreshToken := c.Cookies(refreshTokenCookie)
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing refresh token"})
	}

	if _, err := h.authSvc.ValidateToken(refreshToken); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid refresh token"})
	}

	return c.JSON(fiber.Map{"message": "use login endpoint to get new tokens"})
}

// Logout clears every session cookie. Since access_token and refresh_token
// are httpOnly, only the server can clear them (the browser JS cannot).
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	h.clearSessionCookies(c)
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h *AuthHandler) setSessionCookies(c *fiber.Ctx, accessToken, refreshToken string) {
	c.Cookie(&fiber.Cookie{
		Name:     accessTokenCookie,
		Value:    accessToken,
		Path:     "/",
		MaxAge:   int(service.AccessTokenTTL.Seconds()),
		HTTPOnly: true,
		Secure:   h.cookieSecure,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
	c.Cookie(&fiber.Cookie{
		Name:     refreshTokenCookie,
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(service.RefreshTokenTTL.Seconds()),
		HTTPOnly: true,
		Secure:   h.cookieSecure,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
	c.Cookie(&fiber.Cookie{
		Name:     authStateCookie,
		Value:    "1",
		Path:     "/",
		MaxAge:   int(service.RefreshTokenTTL.Seconds()),
		HTTPOnly: false,
		Secure:   h.cookieSecure,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func (h *AuthHandler) clearSessionCookies(c *fiber.Ctx) {
	expired := time.Now().Add(-time.Hour)
	for _, name := range []string{accessTokenCookie, refreshTokenCookie, authStateCookie} {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  expired,
			HTTPOnly: name != authStateCookie,
			Secure:   h.cookieSecure,
			SameSite: fiber.CookieSameSiteLaxMode,
		})
	}
}
