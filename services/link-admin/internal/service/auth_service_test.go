package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// sha256Hex mirrors the hashing ValidateAPIKey performs internally, so tests
// can seed the fake repository with the hash it will actually look up.
func sha256Hex(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

type fakeUserGetter struct {
	usersByEmail map[string]*repository.User
}

func (f *fakeUserGetter) GetUserByEmail(ctx context.Context, email string) (*repository.User, error) {
	u, ok := f.usersByEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

type fakeAPIKeyGetter struct {
	keysByHash map[string]*repository.APIKey
}

func (f *fakeAPIKeyGetter) GetAPIKeyByHash(ctx context.Context, keyHash string) (*repository.APIKey, error) {
	k, ok := f.keysByHash[keyHash]
	if !ok {
		return nil, errors.New("not found")
	}
	return k, nil
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

func TestLogin_ValidCredentialsReturnsTokens(t *testing.T) {
	tenantID := uuid.New()
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{
		"user@example.com": {
			ID:           uuid.New(),
			TenantID:     tenantID,
			Email:        "user@example.com",
			PasswordHash: hashPassword(t, "correct-password"),
			Role:         "owner",
			IsActive:     true,
		},
	}}
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, "test-secret-at-least-32-characters")

	access, refresh, err := svc.Login(context.Background(), "user@example.com", "correct-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}
	if access == refresh {
		t.Fatal("access and refresh tokens must not be identical")
	}
}

func TestLogin_WrongPasswordFails(t *testing.T) {
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{
		"user@example.com": {
			Email:        "user@example.com",
			PasswordHash: hashPassword(t, "correct-password"),
			IsActive:     true,
		},
	}}
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, "test-secret-at-least-32-characters")

	_, _, err := svc.Login(context.Background(), "user@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownEmailFails(t *testing.T) {
	svc := NewAuthService(&fakeUserGetter{usersByEmail: map[string]*repository.User{}}, &fakeAPIKeyGetter{}, "test-secret-at-least-32-characters")

	_, _, err := svc.Login(context.Background(), "ghost@example.com", "whatever")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateToken_AcceptsTokenItIssued(t *testing.T) {
	tenantID := uuid.New()
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{
		"user@example.com": {
			ID:           uuid.New(),
			TenantID:     tenantID,
			Email:        "user@example.com",
			PasswordHash: hashPassword(t, "pw"),
			Role:         "admin",
			IsActive:     true,
		},
	}}
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, "test-secret-at-least-32-characters")

	access, _, err := svc.Login(context.Background(), "user@example.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, err := svc.ValidateToken(access)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.TenantID != tenantID.String() || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestValidateToken_RejectsTokenSignedWithDifferentSecret(t *testing.T) {
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{
		"user@example.com": {
			ID:           uuid.New(),
			TenantID:     uuid.New(),
			Email:        "user@example.com",
			PasswordHash: hashPassword(t, "pw"),
			IsActive:     true,
		},
	}}
	issuer := NewAuthService(users, &fakeAPIKeyGetter{}, "issuer-secret-at-least-32-characters")
	verifier := NewAuthService(users, &fakeAPIKeyGetter{}, "different-secret-at-least-32-chars")

	access, _, err := issuer.Login(context.Background(), "user@example.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := verifier.ValidateToken(access); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for a token signed with a different secret, got %v", err)
	}
}

func TestValidateToken_RejectsGarbage(t *testing.T) {
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{}, "test-secret-at-least-32-characters")

	if _, err := svc.ValidateToken("not-a-jwt"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestValidateAPIKey_ValidActiveKeyReturnsTenantScopedUser(t *testing.T) {
	tenantID := uuid.New()
	// Hash must match what ValidateAPIKey computes internally (sha256 hex of the raw key).
	rawKey := "test-api-key-12345"
	keyHash := sha256Hex(rawKey)
	apiKeys := &fakeAPIKeyGetter{keysByHash: map[string]*repository.APIKey{
		keyHash: {TenantID: tenantID, IsActive: true},
	}}
	svc := NewAuthService(&fakeUserGetter{}, apiKeys, "test-secret-at-least-32-characters")

	user, err := svc.ValidateAPIKey(context.Background(), rawKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.TenantID != tenantID {
		t.Fatalf("expected tenant %s, got %s", tenantID, user.TenantID)
	}
}

func TestValidateAPIKey_InactiveKeyRejected(t *testing.T) {
	rawKey := "test-api-key-12345"
	keyHash := sha256Hex(rawKey)
	apiKeys := &fakeAPIKeyGetter{keysByHash: map[string]*repository.APIKey{
		keyHash: {TenantID: uuid.New(), IsActive: false},
	}}
	svc := NewAuthService(&fakeUserGetter{}, apiKeys, "test-secret-at-least-32-characters")

	if _, err := svc.ValidateAPIKey(context.Background(), rawKey); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for an inactive key, got %v", err)
	}
}

func TestValidateAPIKey_UnknownKeyRejected(t *testing.T) {
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{keysByHash: map[string]*repository.APIKey{}}, "test-secret-at-least-32-characters")

	if _, err := svc.ValidateAPIKey(context.Background(), "never-issued"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for an unknown key, got %v", err)
	}
}
