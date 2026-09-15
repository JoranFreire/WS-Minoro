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
)

type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// UserGetter and APIKeyGetter are the subsets of their repositories
// AuthService depends on. Defined here so tests can inject fakes instead of
// requiring a live Postgres connection.
type UserGetter interface {
	GetUserByEmail(ctx context.Context, email string) (*repository.User, error)
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

	accessToken, err = s.generateToken(user, AccessTokenTTL)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = s.generateToken(user, RefreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error) {
	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	accessToken, err = s.generateToken(user, AccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = s.generateToken(user, RefreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
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

func (s *AuthService) generateToken(user *repository.User, duration time.Duration) (string, error) {
	claims := &Claims{
		UserID:   user.ID.String(),
		TenantID: user.TenantID.String(),
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
