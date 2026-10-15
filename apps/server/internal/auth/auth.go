package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"chess-platform/server/internal/user"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserAlreadyExists  = errors.New("username or email already in use")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInsecureSecret     = errors.New("jwt secret must be non-empty and at least 16 characters")
)

type Claims struct {
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	TokenType string `json:"tokenType"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type Service struct {
	userRepo      user.Repository
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewService(userRepo user.Repository, accessSecret, refreshSecret string) *Service {
	if len(accessSecret) < 16 || len(refreshSecret) < 16 {
		panic("JWT access and refresh secrets must each be at least 16 characters for security")
	}

	return &Service{
		userRepo:      userRepo,
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  7 * 24 * time.Hour,
		refreshExpiry: 30 * 24 * time.Hour,
	}
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return true
	}
	errLower := strings.ToLower(err.Error())
	return strings.Contains(errLower, "not found")
}

func (s *Service) Register(ctx context.Context, username, email, password string) (*user.User, *TokenPair, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(strings.ToLower(email))

	if username == "" || email == "" || password == "" {
		return nil, nil, errors.New("username, email, and password are required")
	}

	if existing, err := s.userRepo.FindByUsername(ctx, username); err == nil && existing != nil {
		return nil, nil, ErrUserAlreadyExists
	} else if err != nil && !isNotFoundError(err) {
		return nil, nil, err
	}

	if existing, err := s.userRepo.FindByEmail(ctx, email); err == nil && existing != nil {
		return nil, nil, ErrUserAlreadyExists
	} else if err != nil && !isNotFoundError(err) {
		return nil, nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	u := &user.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Avatar:       "",
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, nil, ErrUserAlreadyExists
		}
		return nil, nil, err
	}

	tokens, err := s.GenerateTokenPair(u)
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*user.User, *TokenPair, error) {
	username = strings.TrimSpace(username)
	u, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.GenerateTokenPair(u)
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	token, err := jwt.ParseWithClaims(refreshToken, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	objID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	u, err := s.userRepo.FindByID(ctx, objID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return s.GenerateTokenPair(u)
}

func (s *Service) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *Service) GenerateTokenPair(u *user.User) (*TokenPair, error) {
	now := time.Now().UTC()
	accessClaims := Claims{
		UserID:    u.ID.Hex(),
		Username:  u.Username,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   u.ID.Hex(),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.accessSecret)
	if err != nil {
		return nil, err
	}

	refreshClaims := Claims{
		UserID:    u.ID.Hex(),
		Username:  u.Username,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   u.ID.Hex(),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessExpiry.Seconds()),
	}, nil
}
