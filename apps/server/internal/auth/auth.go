package auth

import (
	"context"
	"errors"
	"time"

	"chess-platform/server/internal/user"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserAlreadyExists  = errors.New("username or email already in use")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type Claims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
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
	return &Service{
		userRepo:      userRepo,
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}
}

func (s *Service) Register(ctx context.Context, username, email, password string) (*user.User, *TokenPair, error) {
	if existing, _ := s.userRepo.FindByUsername(ctx, username); existing != nil {
		return nil, nil, ErrUserAlreadyExists
	}
	if existing, _ := s.userRepo.FindByEmail(ctx, email); existing != nil {
		return nil, nil, ErrUserAlreadyExists
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
		return nil, nil, err
	}

	tokens, err := s.GenerateTokenPair(u)
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*user.User, *TokenPair, error) {
	u, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
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
		return s.refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
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
		return s.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *Service) GenerateTokenPair(u *user.User) (*TokenPair, error) {
	now := time.Now().UTC()
	accessClaims := Claims{
		UserID:   u.ID.Hex(),
		Username: u.Username,
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
		UserID:   u.ID.Hex(),
		Username: u.Username,
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
