package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"chess-platform/server/internal/email"
	"chess-platform/server/internal/user"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserAlreadyExists  = errors.New("username or email already in use")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInsecureSecret     = errors.New("jwt secret must be non-empty and at least 16 characters")
	ErrOTPInvalid         = errors.New("invalid verification code")
	ErrOTPExpired         = errors.New("verification code expired or not found")
	ErrTooManyAttempts    = errors.New("too many incorrect attempts, please request a new code")
	ErrResendTooSoon      = errors.New("please wait at least 30 seconds before requesting another code")
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

type PendingRegistration struct {
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"passwordHash"`
	OTP          string    `json:"otp"`
	Attempts     int       `json:"attempts"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type Service struct {
	userRepo      user.Repository
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	redisClient   *redis.Client
	mailer        email.Mailer
	mu            sync.RWMutex
	pendingOTPs   map[string]*PendingRegistration
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
		pendingOTPs:   make(map[string]*PendingRegistration),
	}
}

func (s *Service) SetRedis(rdb *redis.Client) {
	s.redisClient = rdb
}

func (s *Service) SetMailer(m email.Mailer) {
	s.mailer = m
}

func generateSecureOTP() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return fmt.Sprintf("%06d", (time.Now().UnixNano()%900000)+100000)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
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

const (
	otpTTL        = 10 * time.Minute
	otpRedisKeyPfx = "auth:otp:reg:"
)

func (s *Service) savePending(ctx context.Context, p *PendingRegistration) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(p.Email))
	if s.redisClient != nil {
		data, err := json.Marshal(p)
		if err == nil {
			if err := s.redisClient.Set(ctx, otpRedisKeyPfx+normalizedEmail, data, otpTTL).Err(); err == nil {
				return nil
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingOTPs[normalizedEmail] = p
	return nil
}

func (s *Service) getPending(ctx context.Context, emailStr string) (*PendingRegistration, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(emailStr))
	if s.redisClient != nil {
		data, err := s.redisClient.Get(ctx, otpRedisKeyPfx+normalizedEmail).Bytes()
		if err == nil && len(data) > 0 {
			var p PendingRegistration
			if err := json.Unmarshal(data, &p); err == nil {
				return &p, nil
			}
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pendingOTPs[normalizedEmail]
	if !ok || time.Now().After(p.ExpiresAt) {
		return nil, ErrOTPExpired
	}
	return p, nil
}

func (s *Service) deletePending(ctx context.Context, emailStr string) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(emailStr))
	if s.redisClient != nil {
		_ = s.redisClient.Del(ctx, otpRedisKeyPfx+normalizedEmail).Err()
	}

	s.mu.Lock()
	delete(s.pendingOTPs, normalizedEmail)
	s.mu.Unlock()
}

func (s *Service) InitiateRegistration(ctx context.Context, username, emailStr, password string) (string, error) {
	username = strings.TrimSpace(username)
	emailStr = strings.TrimSpace(strings.ToLower(emailStr))

	if username == "" || emailStr == "" || password == "" {
		return "", errors.New("username, email, and password are required")
	}

	if existing, err := s.userRepo.FindByUsername(ctx, username); err == nil && existing != nil {
		return "", ErrUserAlreadyExists
	} else if err != nil && !isNotFoundError(err) {
		return "", err
	}

	if existing, err := s.userRepo.FindByEmail(ctx, emailStr); err == nil && existing != nil {
		return "", ErrUserAlreadyExists
	} else if err != nil && !isNotFoundError(err) {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	otp := generateSecureOTP()
	now := time.Now().UTC()

	pending := &PendingRegistration{
		Username:     username,
		Email:        emailStr,
		PasswordHash: string(hash),
		OTP:          otp,
		Attempts:     0,
		CreatedAt:    now,
		ExpiresAt:    now.Add(otpTTL),
	}

	if err := s.savePending(ctx, pending); err != nil {
		return "", fmt.Errorf("failed to save registration: %w", err)
	}

	devOtp := ""
	if s.mailer != nil && s.mailer.IsConfigured() {
		if err := s.mailer.SendVerificationOTP(ctx, emailStr, username, otp); err != nil {
			log.Printf("[AUTH] Warning: mailer failed to send to %s: %v", emailStr, err)
			devOtp = otp
		}
	} else {
		log.Printf("[AUTH OTP] Verification code for %s (%s): %s", username, emailStr, otp)
		devOtp = otp
	}

	return devOtp, nil
}

func (s *Service) VerifyRegistrationOTP(ctx context.Context, emailStr, otp string) (*user.User, *TokenPair, error) {
	emailStr = strings.TrimSpace(strings.ToLower(emailStr))
	otp = strings.TrimSpace(otp)

	pending, err := s.getPending(ctx, emailStr)
	if err != nil {
		return nil, nil, ErrOTPExpired
	}

	if time.Now().After(pending.ExpiresAt) {
		s.deletePending(ctx, emailStr)
		return nil, nil, ErrOTPExpired
	}

	pending.Attempts++
	if pending.Attempts > 5 {
		s.deletePending(ctx, emailStr)
		return nil, nil, ErrTooManyAttempts
	}

	if pending.OTP != otp {
		_ = s.savePending(ctx, pending)
		return nil, nil, ErrOTPInvalid
	}

	u := &user.User{
		Username:      pending.Username,
		Email:         pending.Email,
		PasswordHash:  pending.PasswordHash,
		Avatar:        "",
		EmailVerified: true,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			s.deletePending(ctx, emailStr)
			return nil, nil, ErrUserAlreadyExists
		}
		return nil, nil, err
	}

	s.deletePending(ctx, emailStr)

	tokens, err := s.GenerateTokenPair(u)
	if err != nil {
		return nil, nil, err
	}

	return u, tokens, nil
}

func (s *Service) ResendRegistrationOTP(ctx context.Context, emailStr string) (string, error) {
	emailStr = strings.TrimSpace(strings.ToLower(emailStr))

	pending, err := s.getPending(ctx, emailStr)
	if err != nil {
		return "", ErrOTPExpired
	}

	if time.Since(pending.CreatedAt) < 30*time.Second {
		return "", ErrResendTooSoon
	}

	newOTP := generateSecureOTP()
	now := time.Now().UTC()
	pending.OTP = newOTP
	pending.Attempts = 0
	pending.CreatedAt = now
	pending.ExpiresAt = now.Add(otpTTL)

	if err := s.savePending(ctx, pending); err != nil {
		return "", err
	}

	devOtp := ""
	if s.mailer != nil && s.mailer.IsConfigured() {
		if err := s.mailer.SendVerificationOTP(ctx, emailStr, pending.Username, newOTP); err != nil {
			log.Printf("[AUTH] Warning: mailer failed to send to %s: %v", emailStr, err)
			devOtp = newOTP
		}
	} else {
		log.Printf("[AUTH OTP RESEND] Verification code for %s (%s): %s", pending.Username, emailStr, newOTP)
		devOtp = newOTP
	}

	return devOtp, nil
}
