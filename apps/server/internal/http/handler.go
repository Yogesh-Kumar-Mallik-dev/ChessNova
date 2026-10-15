package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/observability"
	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/external"
	"chess-platform/server/internal/game"
	"chess-platform/server/internal/matchmaking"
	"chess-platform/server/internal/puzzle"
	"chess-platform/server/internal/rating"
	"chess-platform/server/internal/review"
	"chess-platform/server/internal/storage"
	"chess-platform/server/internal/user"
	"chess-platform/server/internal/websocket"
	"io"
	"regexp"
	"sort"
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

type Router struct {
	mux             *http.ServeMux
	authService     *auth.Service
	userService     user.Repository
	gameService     *game.Service
	gameRepo        game.Repository
	ratingRepo      rating.Repository
	mmService       *matchmaking.Service
	puzzleService   *puzzle.Service
	engineService   *engine.StockfishEngine
	reviewService   *review.ReviewService
	storageService  storage.StorageService
	externalService *external.Service
	wsHub           *websocket.Hub
	corsOrigin      string
}

func NewRouter(
	authService *auth.Service,
	userService user.Repository,
	gameService *game.Service,
	gameRepo game.Repository,
	ratingRepo rating.Repository,
	mmService *matchmaking.Service,
	puzzleService *puzzle.Service,
	engineService *engine.StockfishEngine,
	reviewService *review.ReviewService,
	storageService storage.StorageService,
	externalService *external.Service,
	wsHub *websocket.Hub,
	corsOrigin string,
) *Router {
	if storageService == nil {
		storageService = storage.NewStorageService(storage.Config{})
	}
	if externalService == nil {
		externalService = external.NewService()
	}

	r := &Router{
		mux:             http.NewServeMux(),
		authService:     authService,
		userService:     userService,
		gameService:     gameService,
		gameRepo:        gameRepo,
		ratingRepo:      ratingRepo,
		mmService:       mmService,
		puzzleService:   puzzleService,
		engineService:   engineService,
		reviewService:   reviewService,
		storageService:  storageService,
		externalService: externalService,
		wsHub:           wsHub,
		corsOrigin:      corsOrigin,
	}
	r.routes()
	return r
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Security Headers Best Practices
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Global CORS Handler
	w.Header().Set("Access-Control-Allow-Origin", rt.corsOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	rt.mux.ServeHTTP(w, r)
}

func (rt *Router) routes() {
	// Health & Metrics
	rt.mux.HandleFunc("GET /health", rt.handleHealth)
	rt.mux.HandleFunc("GET /metrics", observability.DefaultMetrics.PrometheusHandler())
	rt.mux.HandleFunc("GET /api/v1/metrics", observability.DefaultMetrics.JSONHandler())

	// Auth
	rt.mux.HandleFunc("POST /api/v1/auth/register", rt.handleRegister)
	rt.mux.HandleFunc("POST /api/v1/auth/register/send-otp", rt.handleRegisterSendOTP)
	rt.mux.HandleFunc("POST /api/v1/auth/register/verify-otp", rt.handleRegisterVerifyOTP)
	rt.mux.HandleFunc("POST /api/v1/auth/register/resend-otp", rt.handleRegisterResendOTP)
	rt.mux.HandleFunc("POST /api/v1/auth/login", rt.handleLogin)
	rt.mux.HandleFunc("POST /api/v1/auth/refresh", rt.handleRefresh)
	rt.mux.HandleFunc("POST /api/v1/auth/logout", rt.handleLogout)

	// Users & Profiles
	rt.mux.HandleFunc("GET /api/v1/users/me", rt.withAuth(rt.handleGetMe))
	rt.mux.HandleFunc("GET /api/v1/users/{username}", rt.handleGetUser)
	rt.mux.HandleFunc("PUT /api/v1/users/profile", rt.withAuth(rt.handleUpdateProfile))
	rt.mux.HandleFunc("POST /api/v1/users/avatar", rt.withAuth(rt.handleUploadAvatar))
	rt.mux.HandleFunc("POST /api/v1/users/link-accounts", rt.withAuth(rt.handleLinkAccounts))
	rt.mux.HandleFunc("GET /api/v1/users/{username}/external-stats", rt.handleGetExternalStats)
	rt.mux.HandleFunc("GET /api/v1/users/{username}/external-games", rt.handleGetExternalGames)

	// Static local avatars fallback
	rt.mux.Handle("GET /uploads/avatars/", http.StripPrefix("/uploads/avatars/", http.FileServer(http.Dir("uploads/avatars"))))

	// Matchmaking
	rt.mux.HandleFunc("POST /api/v1/matchmaking/join", rt.withAuth(rt.handleMatchmakingJoin))
	rt.mux.HandleFunc("POST /api/v1/matchmaking/leave", rt.withAuth(rt.handleMatchmakingLeave))
	rt.mux.HandleFunc("GET /api/v1/matchmaking/status", rt.withAuth(rt.handleMatchmakingStatus))

	// Games
	rt.mux.HandleFunc("GET /api/v1/games", rt.handleListGames)
	rt.mux.HandleFunc("GET /api/v1/games/active", rt.withAuth(rt.handleGetActiveGameForUser))
	rt.mux.HandleFunc("POST /api/v1/games", rt.withAuth(rt.handleCreateGame))
	rt.mux.HandleFunc("GET /api/v1/games/{id}", rt.handleGetGame)
	rt.mux.HandleFunc("GET /api/v1/games/{id}/pgn", rt.handleGetGamePGN)
	rt.mux.HandleFunc("POST /api/v1/games/{id}/moves", rt.withAuth(rt.handleMakeMove))
	rt.mux.HandleFunc("POST /api/v1/games/{id}/join", rt.withAuth(rt.handleJoinGame))
	rt.mux.HandleFunc("POST /api/v1/games/{id}/resign", rt.withAuth(rt.handleResign))
	rt.mux.HandleFunc("POST /api/v1/games/{id}/claim-draw", rt.withAuth(rt.handleClaimDraw))

	// Authoritative Chess Engine & Legal Move Rules
	rt.mux.HandleFunc("POST /api/v1/chess/legal-moves", rt.handleLegalMoves)
	rt.mux.HandleFunc("POST /api/v1/chess/move", rt.handleExecuteMove)
	rt.mux.HandleFunc("POST /api/v1/chess/bot-move", rt.handleBotMove)

	// Authoritative Stockfish Analysis & Game Review
	rt.mux.HandleFunc("POST /api/v1/analysis/evaluate", rt.handleEvaluatePosition)
	rt.mux.HandleFunc("POST /api/v1/analysis/review", rt.handleReviewGame)
	rt.mux.HandleFunc("POST /api/v1/games/{id}/review", rt.handleReviewGameByID)

	// Leaderboards
	rt.mux.HandleFunc("GET /api/v1/leaderboard", rt.handleLeaderboard)

	// Puzzles
	rt.mux.HandleFunc("GET /api/v1/puzzles/random", rt.handleGetRandomPuzzle)
	rt.mux.HandleFunc("GET /api/v1/puzzles/daily", rt.handleGetDailyPuzzle)
	rt.mux.HandleFunc("GET /api/v1/puzzles/weekly", rt.handleGetWeeklyPuzzles)
	rt.mux.HandleFunc("GET /api/v1/puzzles/{id}", rt.handleGetPuzzleByID)
	rt.mux.HandleFunc("GET /api/v1/puzzles/{id}/hint", rt.handleGetPuzzleHint)
	rt.mux.HandleFunc("GET /api/v1/puzzles/{id}/solution", rt.handleGetPuzzleSolution)
	rt.mux.HandleFunc("POST /api/v1/puzzles/{id}/solve", rt.handleSolvePuzzle)

	// WebSockets
	rt.mux.HandleFunc("/ws/game/{id}", rt.handleWebSocket)
}

func (rt *Router) withAuth(next func(w http.ResponseWriter, r *http.Request, claims *auth.Claims)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "authorization token required")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := rt.authService.ValidateAccessToken(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "AUTH_INVALID", "invalid or expired access token")
			return
		}

		next(w, r, claims)
	}
}

func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	instance := os.Getenv("INSTANCE_ID")
	if instance == "" {
		instance = "default"
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"instance": instance,
		"time":     time.Now().UTC().Format(time.RFC3339),
	})
}

// Auth Handlers
type registerReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (rt *Router) handleRegister(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB payload limit
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if !usernameRegex.MatchString(req.Username) {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid username: 3-30 characters (letters, numbers, underscores, hyphens)")
		return
	}

	if !emailRegex.MatchString(req.Email) {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid email address format")
		return
	}

	if len(req.Password) < 6 || len(req.Password) > 128 {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "password must be between 6 and 128 characters")
		return
	}

	u, tokens, err := rt.authService.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusConflict, "USER_EXISTS", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user":   u,
		"tokens": tokens,
	})
}

func (rt *Router) handleRegisterSendOTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB payload limit
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if !usernameRegex.MatchString(req.Username) {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid username: 3-30 characters (letters, numbers, underscores, hyphens)")
		return
	}
	if !emailRegex.MatchString(req.Email) {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid email address")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "password must be at least 6 characters")
		return
	}

	err := rt.authService.InitiateRegistration(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			writeError(w, http.StatusConflict, "USER_EXISTS", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "OTP_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Verification code sent to email",
		"email":     req.Email,
		"expiresIn": 600,
	})
}

type verifyOTPReq struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func (rt *Router) handleRegisterVerifyOTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req verifyOTPReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.OTP = strings.TrimSpace(req.OTP)

	u, tokens, err := rt.authService.VerifyRegistrationOTP(r.Context(), req.Email, req.OTP)
	if err != nil {
		if errors.Is(err, auth.ErrOTPExpired) {
			writeError(w, http.StatusBadRequest, "OTP_EXPIRED", "Verification code expired or not found. Please request a new code.")
			return
		}
		if errors.Is(err, auth.ErrTooManyAttempts) {
			writeError(w, http.StatusTooManyRequests, "OTP_TOO_MANY_ATTEMPTS", "Too many incorrect attempts. Please request a new code.")
			return
		}
		if errors.Is(err, auth.ErrOTPInvalid) {
			writeError(w, http.StatusBadRequest, "OTP_INVALID", "Invalid verification code.")
			return
		}
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			writeError(w, http.StatusConflict, "USER_EXISTS", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "VERIFY_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user":   u,
		"tokens": tokens,
	})
}

type resendOTPReq struct {
	Email string `json:"email"`
}

func (rt *Router) handleRegisterResendOTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req resendOTPReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	err := rt.authService.ResendRegistrationOTP(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, auth.ErrResendTooSoon) {
			writeError(w, http.StatusTooManyRequests, "RESEND_TOO_SOON", err.Error())
			return
		}
		if errors.Is(err, auth.ErrOTPExpired) {
			writeError(w, http.StatusBadRequest, "NO_PENDING_REGISTRATION", "No pending registration found for this email. Please register again.")
			return
		}
		writeError(w, http.StatusInternalServerError, "RESEND_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "New verification code sent",
		"email":     req.Email,
		"expiresIn": 600,
	})
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (rt *Router) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	u, tokens, err := rt.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "LOGIN_FAILED", "invalid credentials")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user":   u,
		"tokens": tokens,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

func (rt *Router) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	tokens, err := rt.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "REFRESH_FAILED", "invalid refresh token")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

func (rt *Router) handleLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// User Handlers
func (rt *Router) handleGetMe(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	objID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid user ID")
		return
	}

	u, err := rt.userService.FindByID(r.Context(), objID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	ratingsList := make(map[string]int)
	for _, cat := range []rating.Category{rating.Bullet, rating.Blitz, rating.Rapid, rating.Classical} {
		ur, _ := rt.ratingRepo.GetRating(r.Context(), u.ID, cat)
		if ur != nil {
			ratingsList[string(cat)] = ur.Rating
		} else {
			ratingsList[string(cat)] = rating.DefaultRating
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user":    u,
		"ratings": ratingsList,
	})
}

func (rt *Router) handleGetUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	u, err := rt.userService.FindByUsername(r.Context(), username)
	if err != nil {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		return
	}

	ratingsList := make(map[string]interface{})
	for _, cat := range []rating.Category{rating.Bullet, rating.Blitz, rating.Rapid, rating.Classical} {
		ur, _ := rt.ratingRepo.GetRating(r.Context(), u.ID, cat)
		if ur != nil {
			ratingsList[string(cat)] = ur
		}
	}

	// Fetch recent games
	recentGames, _ := rt.gameRepo.FindUserGames(r.Context(), u.ID.Hex(), 10, 0)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user":        u,
		"ratings":     ratingsList,
		"recentGames": recentGames,
	})
}

type updateProfileReq struct {
	Bio     string `json:"bio"`
	Country string `json:"country"`
}

func (rt *Router) handleUpdateProfile(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req updateProfileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if len(req.Bio) > 500 {
		writeError(w, http.StatusBadRequest, "BIO_TOO_LONG", "bio cannot exceed 500 characters")
		return
	}
	if len(req.Country) > 50 {
		writeError(w, http.StatusBadRequest, "COUNTRY_TOO_LONG", "country cannot exceed 50 characters")
		return
	}

	userOID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "invalid user id")
		return
	}

	if err := rt.userService.UpdateProfile(r.Context(), userOID, strings.TrimSpace(req.Bio), strings.TrimSpace(req.Country)); err != nil {
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "profile updated successfully"})
}

func (rt *Router) handleUploadAvatar(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	// 6MB maximum request size (5MB file + multipart boundary headers)
	r.Body = http.MaxBytesReader(w, r.Body, 6*1024*1024)
	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
		writeError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "avatar file exceeds maximum limit of 5MB")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "MISSING_FILE", "avatar image file is required in 'avatar' field")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "READ_ERROR", "failed to read uploaded file")
		return
	}

	userOID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "invalid user id")
		return
	}

	avatarURL, err := rt.storageService.UploadAvatar(r.Context(), claims.UserID, fileBytes, header.Header.Get("Content-Type"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "UPLOAD_FAILED", err.Error())
		return
	}

	if err := rt.userService.UpdateAvatar(r.Context(), userOID, avatarURL); err != nil {
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "failed to save avatar to user profile")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"avatar": avatarURL,
	})
}

type linkAccountsReq struct {
	ChessComUsername string `json:"chessComUsername"`
	LichessUsername  string `json:"lichessUsername"`
}

func (rt *Router) handleLinkAccounts(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req linkAccountsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	chessComUser := strings.TrimSpace(req.ChessComUsername)
	lichessUser := strings.TrimSpace(req.LichessUsername)

	// Verify accounts if provided
	if chessComUser != "" {
		if _, err := rt.externalService.FetchChessComStats(r.Context(), chessComUser); err != nil {
			writeError(w, http.StatusBadRequest, "CHESSCOM_VERIFICATION_FAILED", fmt.Sprintf("Chess.com account '%s' could not be verified", chessComUser))
			return
		}
	}

	if lichessUser != "" {
		if _, err := rt.externalService.FetchLichessStats(r.Context(), lichessUser); err != nil {
			writeError(w, http.StatusBadRequest, "LICHESS_VERIFICATION_FAILED", fmt.Sprintf("Lichess account '%s' could not be verified", lichessUser))
			return
		}
	}

	userOID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "invalid user id")
		return
	}

	if err := rt.userService.UpdateLinkedAccounts(r.Context(), userOID, chessComUser, lichessUser); err != nil {
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", "failed to link external accounts")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":          "accounts linked successfully",
		"chessComUsername": chessComUser,
		"lichessUsername":  lichessUser,
	})
}

func (rt *Router) handleGetExternalStats(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	u, err := rt.userService.FindByUsername(r.Context(), username)
	if err != nil {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		return
	}

	// Fetch ChessNova ratings
	var novaRatings external.PlatformRatings
	for _, cat := range []rating.Category{rating.Bullet, rating.Blitz, rating.Rapid, rating.Classical} {
		ur, _ := rt.ratingRepo.GetRating(r.Context(), u.ID, cat)
		val := rating.DefaultRating
		if ur != nil {
			val = ur.Rating
		}
		switch cat {
		case rating.Bullet:
			novaRatings.Bullet = val
		case rating.Blitz:
			novaRatings.Blitz = val
		case rating.Rapid:
			novaRatings.Rapid = val
		case rating.Classical:
			novaRatings.Classical = val
		}
	}

	comparison := external.PlatformComparison{
		Username:     u.Username,
		ChessNova:    novaRatings,
		ChessComUser: u.ChessComUsername,
		LichessUser:  u.LichessUsername,
	}

	var wg sync.WaitGroup
	if u.ChessComUsername != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats, _ := rt.externalService.FetchChessComStats(r.Context(), u.ChessComUsername)
			if stats != nil {
				comparison.ChessCom = stats
			}
		}()
	}

	if u.LichessUsername != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats, _ := rt.externalService.FetchLichessStats(r.Context(), u.LichessUsername)
			if stats != nil {
				comparison.Lichess = stats
			}
		}()
	}

	wg.Wait()
	writeJSON(w, http.StatusOK, comparison)
}

func (rt *Router) handleGetExternalGames(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	u, err := rt.userService.FindByUsername(r.Context(), username)
	if err != nil {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	platform := strings.ToLower(r.URL.Query().Get("platform"))
	var allGames []external.ExternalGame
	var mu sync.Mutex
	var wg sync.WaitGroup

	if (platform == "" || platform == "all" || platform == "lichess") && u.LichessUsername != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			games, _ := rt.externalService.FetchLichessRecentGames(r.Context(), u.LichessUsername, limit)
			if len(games) > 0 {
				mu.Lock()
				allGames = append(allGames, games...)
				mu.Unlock()
			}
		}()
	}

	if (platform == "" || platform == "all" || platform == "chesscom") && u.ChessComUsername != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			games, _ := rt.externalService.FetchChessComRecentGames(r.Context(), u.ChessComUsername, limit)
			if len(games) > 0 {
				mu.Lock()
				allGames = append(allGames, games...)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Sort newest first
	sort.Slice(allGames, func(i, j int) bool {
		return allGames[i].PlayedAt.After(allGames[j].PlayedAt)
	})

	if len(allGames) > limit {
		allGames = allGames[:limit]
	}

	writeJSON(w, http.StatusOK, allGames)
}

// Matchmaking Handlers
type joinQueueReq struct {
	InitialSeconds int `json:"initial"`
	Increment      int `json:"increment"`
}

func (rt *Router) handleMatchmakingJoin(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	var req joinQueueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.InitialSeconds = 180
		req.Increment = 0
	}
	if req.InitialSeconds <= 0 {
		req.InitialSeconds = 180
	}

	userOID, _ := primitive.ObjectIDFromHex(claims.UserID)
	cat := rating.DetermineCategory(req.InitialSeconds)
	userRating, _ := rt.ratingRepo.GetRating(r.Context(), userOID, cat)
	ratVal := rating.DefaultRating
	if userRating != nil {
		ratVal = userRating.Rating
	}

	tc := game.TimeControl{InitialSeconds: req.InitialSeconds, Increment: req.Increment}
	ticket := &matchmaking.Ticket{
		ID:          randomHex(8),
		UserID:      claims.UserID,
		Username:    claims.Username,
		Rating:      ratVal,
		TimeControl: tc,
	}

	if err := rt.mmService.JoinQueue(r.Context(), ticket); err != nil {
		writeError(w, http.StatusInternalServerError, "QUEUE_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}

func (rt *Router) handleMatchmakingLeave(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	_ = rt.mmService.LeaveQueue(r.Context(), claims.UserID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "left matchmaking queue"})
}

func (rt *Router) handleMatchmakingStatus(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	ticket, err := rt.mmService.GetStatus(r.Context(), claims.UserID)
	if err != nil || ticket == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "idle"})
		return
	}

	if ticket.MatchedGameID != "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "matched",
			"gameId": ticket.MatchedGameID,
			"color":  ticket.AssignedColor,
		})
		// Auto leave queue
		_ = rt.mmService.LeaveQueue(r.Context(), claims.UserID)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "waiting",
		"ticket": ticket,
	})
}

// Game Handlers
func (rt *Router) handleListGames(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	var list []game.GameDocument
	var err error
	if userID != "" {
		list, err = rt.gameRepo.FindUserGames(r.Context(), userID, limit, 0)
	} else {
		list, err = rt.gameRepo.FindRecentGames(r.Context(), limit)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if list == nil {
		list = []game.GameDocument{}
	}
	writeJSON(w, http.StatusOK, list)
}

type createGameReq struct {
	InitialSeconds int    `json:"initial"`
	Increment      int    `json:"increment"`
	OpponentID     string `json:"opponentId,omitempty"`
}

func (rt *Router) handleCreateGame(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	var req createGameReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.InitialSeconds <= 0 {
		req.InitialSeconds = 300
	}

	tc := game.TimeControl{InitialSeconds: req.InitialSeconds, Increment: req.Increment}
	cat := rating.DetermineCategory(req.InitialSeconds)
	userOID, _ := primitive.ObjectIDFromHex(claims.UserID)
	uRat, _ := rt.ratingRepo.GetRating(r.Context(), userOID, cat)
	ratVal := rating.DefaultRating
	if uRat != nil {
		ratVal = uRat.Rating
	}

	whitePlayer := game.PlayerInfo{UserID: claims.UserID, Username: claims.Username, Rating: ratVal}
	blackPlayer := game.PlayerInfo{UserID: "guest-opponent", Username: "Guest", Rating: rating.DefaultRating}

	if req.OpponentID != "" {
		oppOID, err := primitive.ObjectIDFromHex(req.OpponentID)
		if err == nil {
			oppUser, err := rt.userService.FindByID(r.Context(), oppOID)
			if err == nil {
				oppRat, _ := rt.ratingRepo.GetRating(r.Context(), oppOID, cat)
				oRating := rating.DefaultRating
				if oppRat != nil {
					oRating = oppRat.Rating
				}
				blackPlayer = game.PlayerInfo{UserID: oppUser.ID.Hex(), Username: oppUser.Username, Rating: oRating}
			}
		}
	}

	ag, err := rt.gameService.CreateGame(whitePlayer, blackPlayer, tc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GAME_CREATE_FAIL", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ag)
}

func (rt *Router) handleGetActiveGameForUser(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	if rt.gameService == nil || claims == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ag := rt.gameService.GetActiveGameForUser(claims.UserID)
	if ag == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	wMs, bMs, _ := ag.Clock.CurrentTimes(ag.Status)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":          ag.ID,
		"white":       ag.White,
		"black":       ag.Black,
		"status":      ag.Status,
		"timeControl": ag.TimeControl,
		"fen":         chess.ToFEN(ag.Engine.CurrentPosition),
		"turn":        ag.Engine.CurrentPosition.SideToMove.String(),
		"whiteTime":   wMs,
		"blackTime":   bMs,
		"moves":       ag.Moves,
		"result":      ag.Engine.Result(),
		"isCheck":     chess.IsCheck(ag.Engine.CurrentPosition),
	})
}

func (rt *Router) handleGetGame(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Check active first
	ag, err := rt.gameService.GetActiveGame(id)
	if err == nil {
		wMs, bMs, _ := ag.Clock.CurrentTimes(ag.Status)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":          ag.ID,
			"white":       ag.White,
			"black":       ag.Black,
			"status":      ag.Status,
			"timeControl": ag.TimeControl,
			"fen":         chess.ToFEN(ag.Engine.CurrentPosition),
			"turn":        ag.Engine.CurrentPosition.SideToMove.String(),
			"whiteTime":   wMs,
			"blackTime":   bMs,
			"moves":       ag.Moves,
			"result":      ag.Engine.Result(),
			"isCheck":     chess.IsCheck(ag.Engine.CurrentPosition),
		})
		return
	}

	doc, err := rt.gameRepo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "game not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (rt *Router) handleGetGamePGN(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	doc, err := rt.gameRepo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "game not found")
		return
	}

	w.Header().Set("Content-Type", "application/x-chess-pgn")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"game-%s.pgn\"", id))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(doc.PGN))
}

type moveReq struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
}

func (rt *Router) handleMakeMove(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	id := r.PathValue("id")
	var req moveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	fromSq, err := chess.ParseSquare(req.From)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SQUARE", "invalid from square")
		return
	}
	toSq, err := chess.ParseSquare(req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SQUARE", "invalid to square")
		return
	}

	var promo *chess.PieceType
	if req.Promotion != "" {
		var pt chess.PieceType
		switch req.Promotion {
		case "q", "queen", "Q":
			pt = chess.Queen
		case "r", "rook", "R":
			pt = chess.Rook
		case "b", "bishop", "B":
			pt = chess.Bishop
		case "n", "knight", "N":
			pt = chess.Knight
		}
		if pt != chess.NoPiece {
			promo = &pt
		}
	}

	chessMove := chess.NewMove(fromSq, toSq, promo)
	res, err := rt.gameService.MakeMove(r.Context(), id, claims.UserID, chessMove)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ILLEGAL_MOVE", err.Error())
		return
	}

	var promoStr string
	if res.Move.Promotion != nil {
		promoStr = res.Move.Promotion.String()
	}
	webMove := &websocket.WebMove{
		From:      res.Move.From.String(),
		To:        res.Move.To.String(),
		Promotion: promoStr,
		SAN:       res.SAN,
	}

	// Broadcast move via WebSocket Hub
	rt.wsHub.Broadcast(id, websocket.ServerMessage{
		Type:      "move",
		GameID:    id,
		Move:      webMove,
		SAN:       res.SAN,
		FEN:       res.FEN,
		Turn:      res.Turn,
		WhiteTime: res.WhiteTimeMs,
		BlackTime: res.BlackTimeMs,
		Status:    res.Status,
		Outcome:   res.Outcome,
		Result:    res.Result,
		Winner:    res.Winner,
		IsCheck:   res.IsCheck,
	})

	writeJSON(w, http.StatusOK, res)
}

func (rt *Router) handleResign(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	id := r.PathValue("id")
	doc, err := rt.gameService.Resign(r.Context(), id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "RESIGN_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (rt *Router) handleClaimDraw(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	id := r.PathValue("id")
	doc, err := rt.gameService.ClaimDraw(r.Context(), id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CLAIM_DRAW_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (rt *Router) handleJoinGame(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	id := r.PathValue("id")
	userOID, _ := primitive.ObjectIDFromHex(claims.UserID)
	uRat, _ := rt.ratingRepo.GetRating(r.Context(), userOID, rating.Blitz)
	ratVal := rating.DefaultRating
	if uRat != nil {
		ratVal = uRat.Rating
	}

	player := game.PlayerInfo{
		UserID:   claims.UserID,
		Username: claims.Username,
		Rating:   ratVal,
	}

	ag, err := rt.gameService.JoinGame(id, player)
	if err != nil {
		writeError(w, http.StatusBadRequest, "JOIN_ERROR", err.Error())
		return
	}

	// Broadcast update to websocket
	rt.wsHub.Broadcast(id, websocket.ServerMessage{
		Type:        "game_started",
		GameID:      ag.ID,
		FEN:         chess.ToFEN(ag.Engine.CurrentPosition),
		Turn:        ag.Engine.CurrentPosition.SideToMove.String(),
		WhiteTime:   ag.Clock.WhiteTimeMs,
		BlackTime:   ag.Clock.BlackTimeMs,
		Status:      ag.Status,
		White:       &ag.White,
		Black:       &ag.Black,
		TimeControl: &ag.TimeControl,
	})

	writeJSON(w, http.StatusOK, ag)
}


// Leaderboard Handler
func (rt *Router) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	catStr := r.URL.Query().Get("category")
	cat := rating.Category(catStr)
	if cat != rating.Bullet && cat != rating.Blitz && cat != rating.Rapid && cat != rating.Classical {
		cat = rating.Blitz
	}

	list, err := rt.ratingRepo.GetLeaderboard(r.Context(), cat, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if list == nil {
		list = []rating.UserRating{}
	}
	writeJSON(w, http.StatusOK, list)
}

// Puzzle Handlers
func (rt *Router) handleGetRandomPuzzle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := puzzle.QueryFilter{
		Category: strings.ToLower(strings.TrimSpace(q.Get("category"))),
	}
	if tierStr := q.Get("tier"); tierStr != "" {
		if val, err := strconv.Atoi(tierStr); err == nil && val >= 1 && val <= 5 {
			filter.Tier = val
		}
	}
	if minStr := q.Get("minRating"); minStr != "" {
		if val, err := strconv.Atoi(minStr); err == nil && val > 0 {
			filter.MinRating = val
		}
	}
	if maxStr := q.Get("maxRating"); maxStr != "" {
		if val, err := strconv.Atoi(maxStr); err == nil && val > 0 {
			filter.MaxRating = val
		}
	}

	pub, err := rt.puzzleService.GetRandom(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "no puzzle found matching criteria")
		return
	}
	writeJSON(w, http.StatusOK, pub)
}

func (rt *Router) handleGetDailyPuzzle(w http.ResponseWriter, r *http.Request) {
	pub, err := rt.puzzleService.GetDaily(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "no daily puzzle found")
		return
	}
	writeJSON(w, http.StatusOK, pub)
}

func (rt *Router) handleGetWeeklyPuzzles(w http.ResponseWriter, r *http.Request) {
	list, err := rt.puzzleService.GetWeekly(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (rt *Router) handleGetPuzzleByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	oid, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid puzzle ID")
		return
	}
	pub, err := rt.puzzleService.GetByID(r.Context(), oid)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "puzzle not found")
		return
	}
	writeJSON(w, http.StatusOK, pub)
}

func (rt *Router) handleGetPuzzleHint(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	oid, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid puzzle ID")
		return
	}

	ply := 0
	if plyStr := r.URL.Query().Get("plyIndex"); plyStr != "" {
		if val, err := strconv.Atoi(plyStr); err == nil {
			ply = val
		}
	}

	from, to, hint, err := rt.puzzleService.GetHint(r.Context(), oid, ply)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "puzzle not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plyIndex": ply,
		"from":     from,
		"to":       to,
		"hint":     hint,
	})
}

func (rt *Router) handleGetPuzzleSolution(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	oid, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid puzzle ID")
		return
	}
	p, err := rt.puzzleService.GetSolution(r.Context(), oid)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "puzzle not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       p.ID,
		"solution": p.Solution,
		"rating":   p.Rating,
		"themes":   p.Themes,
	})
}

type solvePuzzleReq struct {
	Move          string `json:"move,omitempty"`          // e.g. "e1e8" or "f2g3" (UCI format)
	PlyIndex      int    `json:"plyIndex"`               // 0, 2, 4... player's move turn index
	Success       *bool  `json:"success,omitempty"`       // backward-compatibility fallback
	IsRetry       bool   `json:"isRetry,omitempty"`       // if player is retrying an already failed puzzle
	HasAssistance bool   `json:"hasAssistance,omitempty"` // if player viewed hint or full solution
}

func (rt *Router) handleSolvePuzzle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	oid, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid puzzle ID")
		return
	}

	p, err := rt.puzzleService.GetSolution(r.Context(), oid)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "puzzle not found")
		return
	}

	var req solvePuzzleReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Fetch current user rating for puzzle elo calculation
	var claims *auth.Claims
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, _ = rt.authService.ValidateAccessToken(tokenStr)
	}

	playerRating := rating.DefaultRating
	var uRat *rating.UserRating

	if claims != nil {
		userOID, _ := primitive.ObjectIDFromHex(claims.UserID)
		uRat, _ = rt.ratingRepo.GetRating(r.Context(), userOID, rating.Puzzles)
		if uRat == nil {
			uRat = &rating.UserRating{
				UserID:   userOID,
				Username: claims.Username,
				Category: rating.Puzzles,
				Rating:   rating.DefaultRating,
			}
		}
		playerRating = uRat.Rating
	}

	// Verify move using domain service
	verResult := rt.puzzleService.VerifyMove(p, puzzle.SolveAttempt{
		Move:          req.Move,
		PlyIndex:      req.PlyIndex,
		Success:       req.Success,
		IsRetry:       req.IsRetry,
		HasAssistance: req.HasAssistance,
		PlayerRating:  playerRating,
	})

	newRating := playerRating
	diff := 0

	// Rating adjustments rules:
	// 1. If player had assistance (viewed hint/solution), no rating gains are awarded.
	// 2. If player is retrying a previously failed attempt, do not deduct rating multiple times.
	shouldApplyRating := false
	var score float64

	if verResult.Correct && verResult.Finished {
		if !req.HasAssistance && !req.IsRetry {
			score = 1.0
			shouldApplyRating = true
		}
	} else if !verResult.Correct {
		if !req.IsRetry && !req.HasAssistance {
			score = 0.0
			shouldApplyRating = true
		}
	}

	if shouldApplyRating {
		newRating, _, diff, _ = rating.CalculateElo(playerRating, p.Rating, score, rating.DefaultKFactor)
		if uRat != nil {
			uRat.Rating = newRating
			if score == 1.0 {
				uRat.Wins++
			} else {
				uRat.Losses++
			}
			uRat.Games++
			uRat.UpdatedAt = time.Now().UTC()
			_ = rt.ratingRepo.UpsertRating(r.Context(), uRat)
		}
	}

	verResult.NewRating = newRating
	verResult.Diff = diff

	writeJSON(w, http.StatusOK, verResult)
}

// WebSocket Handler
func (rt *Router) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")
	rt.wsHub.HandleWebSocket(w, r, gameID)
}

// Helpers
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	errID := randomHex(4)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"id":      fmt.Sprintf("ERR_%s", strings.ToUpper(errID)),
			"code":    code,
			"message": message,
		},
	})
}

func randomHex(bytesLen int) string {
	b := make([]byte, bytesLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// -----------------------------------------------------------------------------
// Authoritative Chess Engine & Legal Move Rules
// -----------------------------------------------------------------------------

type legalMovesReq struct {
	FEN    string `json:"fen"`
	Square string `json:"square,omitempty"`
}

type legalTargetRes struct {
	To        string `json:"to"`
	SAN       string `json:"san"`
	IsCapture bool   `json:"isCapture"`
	Category  string `json:"category"`
}

func (rt *Router) handleLegalMoves(w http.ResponseWriter, r *http.Request) {
	var req legalMovesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.FEN == "" {
		req.FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	}

	pos, err := chess.FromFEN(req.FEN)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FEN", err.Error())
		return
	}

	legalMoves := chess.LegalMoves(pos)
	targets := make([]legalTargetRes, 0)

	for _, m := range legalMoves {
		if req.Square != "" && m.From.String() != req.Square {
			continue
		}
		san, _ := chess.MoveToSAN(pos, m)
		fromPiece := pos.Board.Get(m.From)
		isCastle := fromPiece.Type == chess.King && (m.From.File()-m.To.File() == 2 || m.To.File()-m.From.File() == 2)
		isCap := !pos.Board.IsEmpty(m.To) || (fromPiece.Type == chess.Pawn && pos.EnPassant != nil && m.To == *pos.EnPassant)
		cat := "normal"
		if isCap {
			cat = "capture"
		} else if isCastle {
			cat = "castle"
		}

		targets = append(targets, legalTargetRes{
			To:        m.To.String(),
			SAN:       san,
			IsCapture: isCap,
			Category:  cat,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"fen":     req.FEN,
		"targets": targets,
	})
}

type executeMoveReq struct {
	FEN       string `json:"fen"`
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
}

func (rt *Router) handleExecuteMove(w http.ResponseWriter, r *http.Request) {
	var req executeMoveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	pos, err := chess.FromFEN(req.FEN)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FEN", err.Error())
		return
	}

	fromSq, err := chess.ParseSquare(req.From)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SQUARE", "invalid from square")
		return
	}
	toSq, err := chess.ParseSquare(req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SQUARE", "invalid to square")
		return
	}

	var promo *chess.PieceType
	if req.Promotion != "" {
		switch strings.ToLower(req.Promotion) {
		case "q", "queen":
			pt := chess.Queen
			promo = &pt
		case "r", "rook":
			pt := chess.Rook
			promo = &pt
		case "b", "bishop":
			pt := chess.Bishop
			promo = &pt
		case "n", "knight":
			pt := chess.Knight
			promo = &pt
		}
	}

	m := chess.Move{From: fromSq, To: toSq, Promotion: promo}
	if err := chess.ValidateMove(pos, m); err != nil {
		writeError(w, http.StatusBadRequest, "ILLEGAL_MOVE", "move is illegal in position: "+err.Error())
		return
	}

	san, err := chess.MoveToSAN(pos, m)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "NOTATION_ERROR", err.Error())
		return
	}

	nextPos, err := chess.ApplyMove(pos, m)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "MOVE_ERROR", err.Error())
		return
	}

	newFEN := chess.ToFEN(nextPos)
	inCheck := chess.IsCheck(nextPos)
	isCheckmate := chess.IsCheckmate(nextPos)
	isStalemate := chess.IsStalemate(nextPos)
	isDraw := chess.IsDraw(nextPos)

	turnStr := "white"
	if nextPos.SideToMove == chess.Black {
		turnStr = "black"
	}

	fromPiece := pos.Board.Get(m.From)
	isCapture := !pos.Board.IsEmpty(m.To) || (fromPiece.Type == chess.Pawn && pos.EnPassant != nil && m.To == *pos.EnPassant)
	isPromotion := promo != nil

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":       true,
		"newFen":      newFEN,
		"san":         san,
		"isCapture":   isCapture,
		"isPromotion": isPromotion,
		"isCheck":     inCheck,
		"isCheckmate": isCheckmate,
		"isStalemate": isStalemate,
		"isDraw":      isDraw,
		"turn":        turnStr,
	})
}

type botMoveReq struct {
	FEN   string `json:"fen"`
	Level int    `json:"level"`
}

func (rt *Router) handleBotMove(w http.ResponseWriter, r *http.Request) {
	var req botMoveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.FEN == "" {
		writeError(w, http.StatusBadRequest, "INVALID_FEN", "fen is required")
		return
	}
	if req.Level < 1 || req.Level > 5 {
		req.Level = 2
	}

	pos, err := chess.FromFEN(req.FEN)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FEN", err.Error())
		return
	}

	botUCI, err := rt.engineService.GenerateBotMove(r.Context(), req.FEN, req.Level)
	if err != nil || botUCI == "" {
		writeError(w, http.StatusInternalServerError, "ENGINE_ERROR", "failed to generate bot move")
		return
	}

	parsedM, err := chess.ParseUCIMove(botUCI)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ENGINE_ERROR", "invalid move format from engine: "+err.Error())
		return
	}

	san, err := chess.MoveToSAN(pos, parsedM)
	if err != nil {
		san = botUCI
	}

	nextPos, err := chess.ApplyMove(pos, parsedM)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "MOVE_ERROR", err.Error())
		return
	}

	fromPiece := pos.Board.Get(parsedM.From)
	isCapture := !pos.Board.IsEmpty(parsedM.To) || (fromPiece.Type == chess.Pawn && pos.EnPassant != nil && parsedM.To == *pos.EnPassant)
	isPromotion := parsedM.Promotion != nil

	turnStr := "white"
	if nextPos.SideToMove == chess.Black {
		turnStr = "black"
	}

	promoStr := ""
	if parsedM.Promotion != nil {
		switch *parsedM.Promotion {
		case chess.Queen:
			promoStr = "q"
		case chess.Rook:
			promoStr = "r"
		case chess.Bishop:
			promoStr = "b"
		case chess.Knight:
			promoStr = "n"
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":       true,
		"uci":         botUCI,
		"from":        parsedM.From.String(),
		"to":          parsedM.To.String(),
		"promotion":   promoStr,
		"san":         san,
		"newFen":      chess.ToFEN(nextPos),
		"isCapture":   isCapture,
		"isPromotion": isPromotion,
		"isCheck":     chess.IsCheck(nextPos),
		"isCheckmate": chess.IsCheckmate(nextPos),
		"isStalemate": chess.IsStalemate(nextPos),
		"isDraw":      chess.IsDraw(nextPos),
		"turn":        turnStr,
	})
}


// -----------------------------------------------------------------------------
// Authoritative Stockfish Position Evaluation & Game Review
// -----------------------------------------------------------------------------

type evaluateReq struct {
	FEN   string `json:"fen"`
	Depth int    `json:"depth"`
}

func (rt *Router) handleEvaluatePosition(w http.ResponseWriter, r *http.Request) {
	var req evaluateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}
	if req.FEN == "" {
		req.FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	}
	if req.Depth <= 0 || req.Depth > 20 {
		req.Depth = 12
	}

	eval, err := rt.engineService.EvaluateFen(r.Context(), req.FEN, req.Depth)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ENGINE_ERROR", err.Error())
		return
	}

	winChance := review.CentipawnsToWinChance(eval.ScoreCp)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"fen":       req.FEN,
		"scoreCp":   eval.ScoreCp,
		"isMate":    eval.IsMate,
		"mateIn":    eval.MateIn,
		"bestMove":  eval.BestMove,
		"pv":        eval.PV,
		"depth":     eval.Depth,
		"winChance": winChance,
	})
}

type reviewReq struct {
	PGN   string             `json:"pgn,omitempty"`
	Moves []review.InputMove `json:"moves,omitempty"`
}

func (rt *Router) handleReviewGame(w http.ResponseWriter, r *http.Request) {
	var req reviewReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	var result *review.ReviewResult
	var err error

	if strings.TrimSpace(req.PGN) != "" {
		result, err = rt.reviewService.AnalyzePGN(r.Context(), req.PGN)
	} else if len(req.Moves) > 0 {
		result, err = rt.reviewService.AnalyzeMoves(r.Context(), req.Moves)
	} else {
		writeError(w, http.StatusBadRequest, "EMPTY_INPUT", "either pgn or moves list must be provided")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "REVIEW_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (rt *Router) handleReviewGameByID(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("id")
	gameDoc, err := rt.gameRepo.FindByID(r.Context(), gameID)
	if err != nil {
		writeError(w, http.StatusNotFound, "GAME_NOT_FOUND", "game not found in database")
		return
	}

	var result *review.ReviewResult
	if strings.TrimSpace(gameDoc.PGN) != "" {
		result, err = rt.reviewService.AnalyzePGN(r.Context(), gameDoc.PGN)
	} else if len(gameDoc.Moves) > 0 {
		inputMoves := make([]review.InputMove, len(gameDoc.Moves))
		for i, m := range gameDoc.Moves {
			inputMoves[i] = review.InputMove{
				From: m.From,
				To:   m.To,
				SAN:  m.SAN,
				FEN:  m.FEN,
			}
		}
		result, err = rt.reviewService.AnalyzeMoves(r.Context(), inputMoves)
	} else {
		writeError(w, http.StatusBadRequest, "NO_MOVES", "game has no moves to review")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "REVIEW_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
