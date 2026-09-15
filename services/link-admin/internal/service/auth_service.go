package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUnauthorized = errors.New("unauthorized")
var ErrEmailTaken = repository.ErrEmailTaken
var ErrInvalidRegistration = errors.New("invalid registration")

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour

	minPasswordLen = 8

	// TokenTypeAccess and TokenTypeRefresh keep the two token kinds from
	// being interchangeable: without this, a refresh token — meant only to
	// be exchanged for a new access token — could also authenticate any
	// API call directly, turning a 15-minute blast radius into 7 days.
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Claims struct {
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// UserGetter and APIKeyGetter are the subsets of their repositories
// AuthService depends on. Defined here so tests can inject fakes instead of
// requiring a live Postgres connection.
type UserGetter interface {
	GetUserByEmail(ctx context.Context, email string) (*repository.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*repository.User, error)
}

type APIKeyGetter interface {
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*repository.APIKey, error)
}

// Registrar is the subset of repository.RegistrationRepository AuthService
// depends on. Defined here so tests can inject a fake instead of requiring
// a live Postgres connection.
type Registrar interface {
	RegisterTenantOwner(ctx context.Context, tenantName, email, passwordHash string) (*repository.Tenant, *repository.User, error)
}

type AuthService struct {
	users         UserGetter
	apiKeys       APIKeyGetter
	registrations Registrar
	jwtSecret     string
}

func NewAuthService(users UserGetter, apiKeys APIKeyGetter, registrations Registrar, jwtSecret string) *AuthService {
	return &AuthService{users: users, apiKeys: apiKeys, registrations: registrations, jwtSecret: jwtSecret}
}

// Register creates a new tenant with its first user (owner) and returns a
// session for it, the same shape as Login — self-serve signup logs the
// caller straight in rather than requiring a separate login step.
func (s *AuthService) Register(ctx context.Context, tenantName, email, password string) (accessToken, refreshToken string, err error) {
	tenantName = strings.TrimSpace(tenantName)
	email = strings.TrimSpace(email)

	if tenantName == "" || email == "" || !strings.Contains(email, "@") || len(password) < minPasswordLen {
		return "", "", ErrInvalidRegistration
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	_, user, err := s.registrations.RegisterTenantOwner(ctx, tenantName, email, string(hash))
	if err != nil {
		return "", "", err
	}

	return s.issueTokenPair(user)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error) {
	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	return s.issueTokenPair(user)
}

// RefreshSession exchanges a valid, unexpired refresh token for a new
// access/refresh pair. The user is re-fetched from the repository (rather
// than trusting the claims alone) so an account deactivated after the
// refresh token was issued can't silently keep renewing its session.
func (s *AuthService) RefreshSession(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, err error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil || claims.TokenType != TokenTypeRefresh {
		return "", "", ErrUnauthorized
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return "", "", ErrUnauthorized
	}

	return s.issueTokenPair(user)
}

func (s *AuthService) issueTokenPair(user *repository.User) (accessToken, refreshToken string, err error) {
	accessToken, err = s.generateToken(user, AccessTokenTTL, TokenTypeAccess)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = s.generateToken(user, RefreshTokenTTL, TokenTypeRefresh)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// ValidateAccessToken validates a token and rejects it unless it was issued
// as an access token — a refresh token must never authenticate an API call
// directly, only be exchanged via RefreshSession.
func (s *AuthService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, ErrUnauthorized
	}
	return claims, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnauthorized
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrUnauthorized
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrUnauthorized
	}
	return claims, nil
}

func (s *AuthService) ValidateAPIKey(ctx context.Context, keyStr string) (*repository.User, error) {
	hash := sha256.Sum256([]byte(keyStr))
	keyHash := hex.EncodeToString(hash[:])

	apiKey, err := s.apiKeys.GetAPIKeyByHash(ctx, keyHash)
	if err != nil || !apiKey.IsActive {
		return nil, ErrUnauthorized
	}

	return &repository.User{
		TenantID: apiKey.TenantID,
		Role:     "admin",
	}, nil
}

func (s *AuthService) generateToken(user *repository.User, duration time.Duration, tokenType string) (string, error) {
	claims := &Claims{
		UserID:    user.ID.String(),
		TenantID:  user.TenantID.String(),
		Role:      user.Role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
