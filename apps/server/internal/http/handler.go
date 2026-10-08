package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/game"
	"chess-platform/server/internal/matchmaking"
	"chess-platform/server/internal/puzzle"
	"chess-platform/server/internal/rating"
	"chess-platform/server/internal/review"
	"chess-platform/server/internal/user"
	"chess-platform/server/internal/websocket"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Router struct {
	mux           *http.ServeMux
	authService   *auth.Service
	userService   user.Repository
	gameService   *game.Service
	gameRepo      game.Repository
	ratingRepo    rating.Repository
	mmService     *matchmaking.Service
	puzzleRepo    puzzle.Repository
	engineService *engine.StockfishEngine
	reviewService *review.ReviewService
	wsHub         *websocket.Hub
	corsOrigin    string
}

func NewRouter(
	authService *auth.Service,
	userService user.Repository,
	gameService *game.Service,
	gameRepo game.Repository,
	ratingRepo rating.Repository,
	mmService *matchmaking.Service,
	puzzleRepo puzzle.Repository,
	engineService *engine.StockfishEngine,
	reviewService *review.ReviewService,
	wsHub *websocket.Hub,
	corsOrigin string,
) *Router {
	r := &Router{
		mux:           http.NewServeMux(),
		authService:   authService,
		userService:   userService,
		gameService:   gameService,
		gameRepo:      gameRepo,
		ratingRepo:    ratingRepo,
		mmService:     mmService,
		puzzleRepo:    puzzleRepo,
		engineService: engineService,
		reviewService: reviewService,
		wsHub:         wsHub,
		corsOrigin:    corsOrigin,
	}
	r.routes()
	return r
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Global CORS Handler
	w.Header().Set("Access-Control-Allow-Origin", rt.corsOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	rt.mux.ServeHTTP(w, r)
}

func (rt *Router) routes() {
	// Health
	rt.mux.HandleFunc("GET /health", rt.handleHealth)

	// Auth
	rt.mux.HandleFunc("POST /api/v1/auth/register", rt.handleRegister)
	rt.mux.HandleFunc("POST /api/v1/auth/login", rt.handleLogin)
	rt.mux.HandleFunc("POST /api/v1/auth/refresh", rt.handleRefresh)
	rt.mux.HandleFunc("POST /api/v1/auth/logout", rt.handleLogout)

	// Users
	rt.mux.HandleFunc("GET /api/v1/users/me", rt.withAuth(rt.handleGetMe))
	rt.mux.HandleFunc("GET /api/v1/users/{username}", rt.handleGetUser)

	// Matchmaking
	rt.mux.HandleFunc("POST /api/v1/matchmaking/join", rt.withAuth(rt.handleMatchmakingJoin))
	rt.mux.HandleFunc("POST /api/v1/matchmaking/leave", rt.withAuth(rt.handleMatchmakingLeave))
	rt.mux.HandleFunc("GET /api/v1/matchmaking/status", rt.withAuth(rt.handleMatchmakingStatus))

	// Games
	rt.mux.HandleFunc("GET /api/v1/games", rt.handleListGames)
	rt.mux.HandleFunc("POST /api/v1/games", rt.withAuth(rt.handleCreateGame))
	rt.mux.HandleFunc("GET /api/v1/games/{id}", rt.handleGetGame)
	rt.mux.HandleFunc("GET /api/v1/games/{id}/pgn", rt.handleGetGamePGN)
	rt.mux.HandleFunc("POST /api/v1/games/{id}/moves", rt.withAuth(rt.handleMakeMove))
	rt.mux.HandleFunc("POST /api/v1/games/{id}/join", rt.withAuth(rt.handleJoinGame))
	rt.mux.HandleFunc("POST /api/v1/games/{id}/resign", rt.withAuth(rt.handleResign))

	// Authoritative Chess Engine & Legal Move Rules
	rt.mux.HandleFunc("POST /api/v1/chess/legal-moves", rt.handleLegalMoves)
	rt.mux.HandleFunc("POST /api/v1/chess/move", rt.handleExecuteMove)

	// Authoritative Stockfish Analysis & Game Review
	rt.mux.HandleFunc("POST /api/v1/analysis/evaluate", rt.handleEvaluatePosition)
	rt.mux.HandleFunc("POST /api/v1/analysis/review", rt.handleReviewGame)
	rt.mux.HandleFunc("POST /api/v1/games/{id}/review", rt.handleReviewGameByID)

	// Leaderboards
	rt.mux.HandleFunc("GET /api/v1/leaderboard", rt.handleLeaderboard)

	// Puzzles
	rt.mux.HandleFunc("GET /api/v1/puzzles/random", rt.handleGetRandomPuzzle)
	rt.mux.HandleFunc("GET /api/v1/puzzles/{id}", rt.handleGetPuzzleByID)
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
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// Auth Handlers
type registerReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (rt *Router) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if len(req.Username) < 3 || len(req.Password) < 6 || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid username (min 3), email, or password (min 6)")
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

	// Broadcast move via WebSocket Hub
	rt.wsHub.Broadcast(id, websocket.ServerMessage{
		Type:      "move",
		GameID:    id,
		Move:      &res.Move,
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
	p, err := rt.puzzleRepo.GetRandomPuzzle(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "no puzzle found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (rt *Router) handleGetPuzzleByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	oid, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid puzzle ID")
		return
	}
	p, err := rt.puzzleRepo.GetByID(r.Context(), oid)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "puzzle not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type solvePuzzleReq struct {
	Success bool `json:"success"`
}

func (rt *Router) handleSolvePuzzle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	_, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid puzzle ID")
		return
	}

	var req solvePuzzleReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := rt.authService.ValidateAccessToken(tokenStr)
		if err == nil && claims != nil {
			userOID, _ := primitive.ObjectIDFromHex(claims.UserID)
			uRat, _ := rt.ratingRepo.GetRating(r.Context(), userOID, rating.Puzzles)
			if uRat == nil {
				uRat = &rating.UserRating{
					UserID:   userOID,
					Username: claims.Username,
					Category: rating.Puzzles,
					Rating:   rating.DefaultRating,
				}
			}

			oldRating := uRat.Rating
			diff := 10
			if req.Success {
				uRat.Rating += 12
				uRat.Wins++
				diff = 12
			} else {
				uRat.Rating -= 10
				if uRat.Rating < rating.MinRating {
					uRat.Rating = rating.MinRating
				}
				uRat.Losses++
				diff = uRat.Rating - oldRating
			}
			uRat.Games++
			uRat.UpdatedAt = time.Now().UTC()
			_ = rt.ratingRepo.UpsertRating(r.Context(), uRat)

			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success":   req.Success,
				"newRating": uRat.Rating,
				"diff":      diff,
				"games":     uRat.Games,
			})
			return
		}
	}

	// Guest fallback
	diff := 10
	guestRating := rating.DefaultRating
	if req.Success {
		diff = 12
		guestRating += diff
	} else {
		diff = -10
		guestRating += diff
		if guestRating < rating.MinRating {
			guestRating = rating.MinRating
			diff = guestRating - rating.DefaultRating
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   req.Success,
		"newRating": guestRating,
		"diff":      diff,
		"games":     1,
	})
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
