package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/ws-minoro/link-admin/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"uuid"
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

func (f *fakeUserGetter) GetUserByID(ctx context.Context, id uuid.UUID) (*repository.User, error) {
	for _, u := range f.usersByEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
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

type fakeRegistrar struct {
	registered []struct{ tenantName, email, passwordHash string }
	err        error
	nextTenant *repository.Tenant
	nextUser   *repository.User
}

func (f *fakeRegistrar) RegisterTenantOwner(ctx context.Context, tenantName, email, passwordHash string) (*repository.Tenant, *repository.User, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	f.registered = append(f.registered, struct{ tenantName, email, passwordHash string }{tenantName, email, passwordHash})

	tenant := f.nextTenant
	if tenant == nil {
		tenant = &repository.Tenant{ID: uuid.New(), Name: tenantName}
	}
	user := f.nextUser
	if user == nil {
		user = &repository.User{ID: uuid.New(), TenantID: tenant.ID, Email: email, PasswordHash: passwordHash, Role: "owner", IsActive: true}
	}
	return tenant, user, nil
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
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

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
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	_, _, err := svc.Login(context.Background(), "user@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownEmailFails(t *testing.T) {
	svc := NewAuthService(&fakeUserGetter{usersByEmail: map[string]*repository.User{}}, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

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
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

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
	issuer := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "issuer-secret-at-least-32-characters")
	verifier := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "different-secret-at-least-32-chars")

	access, _, err := issuer.Login(context.Background(), "user@example.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := verifier.ValidateToken(access); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for a token signed with a different secret, got %v", err)
	}
}

func TestValidateToken_RejectsGarbage(t *testing.T) {
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

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
	svc := NewAuthService(&fakeUserGetter{}, apiKeys, &fakeRegistrar{}, "test-secret-at-least-32-characters")

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
	svc := NewAuthService(&fakeUserGetter{}, apiKeys, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	if _, err := svc.ValidateAPIKey(context.Background(), rawKey); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for an inactive key, got %v", err)
	}
}

func TestValidateAPIKey_UnknownKeyRejected(t *testing.T) {
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{keysByHash: map[string]*repository.APIKey{}}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	if _, err := svc.ValidateAPIKey(context.Background(), "never-issued"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for an unknown key, got %v", err)
	}
}

func TestRegister_ValidInputCreatesAccountAndReturnsTokens(t *testing.T) {
	reg := &fakeRegistrar{}
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{}, reg, "test-secret-at-least-32-characters")

	access, refresh, err := svc.Register(context.Background(), "Acme Inc", "owner@acme.com", "supersecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}
	if len(reg.registered) != 1 {
		t.Fatalf("expected 1 registration call, got %d", len(reg.registered))
	}
	if reg.registered[0].tenantName != "Acme Inc" || reg.registered[0].email != "owner@acme.com" {
		t.Fatalf("unexpected registration call: %+v", reg.registered[0])
	}
	if reg.registered[0].passwordHash == "supersecret" {
		t.Fatal("the raw password must never reach the repository — only its hash")
	}
}

func TestRegister_IssuedTokenCarriesOwnerRole(t *testing.T) {
	reg := &fakeRegistrar{}
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{}, reg, "test-secret-at-least-32-characters")

	access, _, err := svc.Register(context.Background(), "Acme Inc", "owner@acme.com", "supersecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, err := svc.ValidateToken(access)
	if err != nil {
		t.Fatalf("unexpected error validating issued token: %v", err)
	}
	if claims.Role != "owner" {
		t.Fatalf("expected owner role, got %q", claims.Role)
	}
}

func TestRegister_MissingTenantNameRejected(t *testing.T) {
	reg := &fakeRegistrar{}
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{}, reg, "test-secret-at-least-32-characters")

	_, _, err := svc.Register(context.Background(), "  ", "owner@acme.com", "supersecret")
	if !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected ErrInvalidRegistration, got %v", err)
	}
	if len(reg.registered) != 0 {
		t.Fatal("repository must not be called when validation fails")
	}
}

func TestRegister_InvalidEmailRejected(t *testing.T) {
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	_, _, err := svc.Register(context.Background(), "Acme Inc", "not-an-email", "supersecret")
	if !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected ErrInvalidRegistration, got %v", err)
	}
}

func TestRegister_ShortPasswordRejected(t *testing.T) {
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	_, _, err := svc.Register(context.Background(), "Acme Inc", "owner@acme.com", "short")
	if !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected ErrInvalidRegistration, got %v", err)
	}
}

func TestRegister_DuplicateEmailPropagatesErrEmailTaken(t *testing.T) {
	reg := &fakeRegistrar{err: ErrEmailTaken}
	svc := NewAuthService(&fakeUserGetter{}, &fakeAPIKeyGetter{}, reg, "test-secret-at-least-32-characters")

	_, _, err := svc.Register(context.Background(), "Acme Inc", "owner@acme.com", "supersecret")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRefreshSession_ValidRefreshTokenIssuesNewWorkingPair(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{
		"user@example.com": {
			ID: userID, TenantID: tenantID, Email: "user@example.com",
			PasswordHash: hashPassword(t, "pw"), Role: "owner", IsActive: true,
		},
	}}
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	_, refresh, err := svc.Login(context.Background(), "user@example.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newAccess, newRefresh, err := svc.RefreshSession(context.Background(), refresh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Fatal("expected non-empty rotated tokens")
	}

	claims, err := svc.ValidateAccessToken(newAccess)
	if err != nil {
		t.Fatalf("new access token should be valid: %v", err)
	}
	if claims.TenantID != tenantID.String() || claims.Role != "owner" {
		t.Fatalf("unexpected claims after refresh: %+v", claims)
	}
}

func TestRefreshSession_RejectsAnAccessTokenUsedAsRefresh(t *testing.T) {
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{
		"user@example.com": {
			ID: uuid.New(), TenantID: uuid.New(), Email: "user@example.com",
			PasswordHash: hashPassword(t, "pw"), IsActive: true,
		},
	}}
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	access, _, err := svc.Login(context.Background(), "user@example.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, _, err := svc.RefreshSession(context.Background(), access); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized when an access token is presented as a refresh token, got %v", err)
	}
}

func TestRefreshSession_UnknownUserRejected(t *testing.T) {
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{
		"user@example.com": {
			ID: uuid.New(), TenantID: uuid.New(), Email: "user@example.com",
			PasswordHash: hashPassword(t, "pw"), IsActive: true,
		},
	}}
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	_, refresh, err := svc.Login(context.Background(), "user@example.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The user was deactivated/removed after the refresh token was issued.
	users.usersByEmail = map[string]*repository.User{}

	if _, _, err := svc.RefreshSession(context.Background(), refresh); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for a user no longer found, got %v", err)
	}
}

func TestValidateAccessToken_RejectsARefreshToken(t *testing.T) {
	users := &fakeUserGetter{usersByEmail: map[string]*repository.User{
		"user@example.com": {
			ID: uuid.New(), TenantID: uuid.New(), Email: "user@example.com",
			PasswordHash: hashPassword(t, "pw"), IsActive: true,
		},
	}}
	svc := NewAuthService(users, &fakeAPIKeyGetter{}, &fakeRegistrar{}, "test-secret-at-least-32-characters")

	_, refresh, err := svc.Login(context.Background(), "user@example.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := svc.ValidateAccessToken(refresh); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("a refresh token must never authenticate an API call, got %v", err)
	}
}
