package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"chess-platform/server/internal/user"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var errNotFound = errors.New("user not found")

type mockUserRepo struct {
	users map[string]*user.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*user.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, u *user.User) error {
	u.ID = primitive.NewObjectID()
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()
	m.users[u.Username] = u
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*user.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errNotFound
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, errNotFound
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errNotFound
}

func (m *mockUserRepo) EnsureIndexes(ctx context.Context) error {
	return nil
}

func TestAuthRegisterAndLogin(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewService(repo, "test-access-secret-12345", "test-refresh-secret-67890")

	ctx := context.Background()

	// 1. Register user
	u, tokens, err := svc.Register(ctx, "magnus", "magnus@example.com", "Password123!")
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if u.Username != "magnus" {
		t.Errorf("expected username magnus, got %s", u.Username)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Error("expected valid tokens")
	}

	// 2. Duplicate register should fail
	_, _, err = svc.Register(ctx, "magnus", "other@example.com", "Password123!")
	if err != ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}

	// 3. Login with correct password
	loginUser, loginTokens, err := svc.Login(ctx, "magnus", "Password123!")
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}
	if loginUser.ID != u.ID {
		t.Errorf("expected user ID %v, got %v", u.ID, loginUser.ID)
	}
	if loginTokens.AccessToken == "" {
		t.Error("expected valid login access token")
	}

	// 4. Login with invalid password
	_, _, err = svc.Login(ctx, "magnus", "wrongPassword")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	// 5. Validate Access Token
	claims, err := svc.ValidateAccessToken(loginTokens.AccessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}
	if claims.Username != "magnus" {
		t.Errorf("expected claims username magnus, got %s", claims.Username)
	}

	// 6. Refresh Token
	newTokens, err := svc.Refresh(ctx, loginTokens.RefreshToken)
	if err != nil {
		t.Fatalf("failed to refresh tokens: %v", err)
	}
	if newTokens.AccessToken == "" {
		t.Error("expected new access token on refresh")
	}

	// 7. Validate invalid token
	_, err = svc.ValidateAccessToken("invalid.token.string")
	if err == nil {
		t.Error("expected error validating malformed token")
	}
}
