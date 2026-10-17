package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/game"
	"chess-platform/server/internal/observability"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// HubConfig configures origin policy and websocket behavior
type HubConfig struct {
	AllowedOrigins     []string
	AllowMissingOrigin bool
	AllowDevLocalhost  bool
}

// DefaultHubConfig returns secure default origins including Tauri and local dev ports
func DefaultHubConfig(customOrigins ...string) HubConfig {
	origins := []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://localhost:4173",
		"tauri://localhost",
		"http://tauri.localhost",
		"https://tauri.localhost",
	}
	for _, o := range customOrigins {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return HubConfig{
		AllowedOrigins:     origins,
		AllowMissingOrigin: true, // Allow non-browser desktop / test clients without Origin header
		AllowDevLocalhost:  true,
	}
}

// ValidateOrigin performs strict origin verification against trusted origins
func ValidateOrigin(origin string, cfg HubConfig, reqHost ...string) bool {
	if origin == "" {
		return cfg.AllowMissingOrigin
	}

	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" {
		return false
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" && scheme != "tauri" {
		return false
	}

	// For custom URI scheme (e.g. tauri://localhost)
	if scheme == "tauri" {
		full := "tauri://" + strings.ToLower(u.Host)
		for _, allowed := range cfg.AllowedOrigins {
			if strings.EqualFold(strings.TrimRight(allowed, "/"), full) {
				return true
			}
		}
		return false
	}

	if u.Host == "" {
		return false
	}

	originHost := strings.ToLower(u.Hostname())
	normOrigin := scheme + "://" + strings.ToLower(u.Host)

	// If the request was made to the same host (e.g. r.Host), it is same-origin and trusted
	if len(reqHost) > 0 && reqHost[0] != "" {
		hostOnly := strings.ToLower(strings.Split(reqHost[0], ":")[0])
		if originHost == hostOnly {
			return true
		}
	}

	// Check explicit allowlist
	for _, allowed := range cfg.AllowedOrigins {
		allowedTrimmed := strings.TrimRight(strings.ToLower(strings.TrimSpace(allowed)), "/")
		if allowedTrimmed == "*" || normOrigin == allowedTrimmed {
			return true
		}
		// Match by URL scheme and wildcard subdomain (e.g. https://*.yogeshmallik.in)
		if strings.HasPrefix(allowedTrimmed, "http://") || strings.HasPrefix(allowedTrimmed, "https://") {
			if au, err := url.Parse(allowedTrimmed); err == nil {
				if au.Scheme == scheme {
					if strings.EqualFold(au.Hostname(), originHost) {
						return true
					}
					if strings.HasPrefix(au.Hostname(), "*.") {
						suffix := au.Hostname()[1:]
						if strings.HasSuffix(originHost, suffix) {
							return true
						}
					}
				}
			}
		} else {
			if strings.EqualFold(allowedTrimmed, originHost) {
				return true
			}
			if strings.HasPrefix(allowedTrimmed, "*.") && strings.HasSuffix(originHost, allowedTrimmed[1:]) {
				return true
			}
		}
	}

	// Check dev localhost
	if cfg.AllowDevLocalhost {
		if originHost == "localhost" || originHost == "127.0.0.1" || originHost == "::1" {
			return true
		}
	}

	return false
}

type ClientMessage struct {
	Type      string `json:"type"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Promotion string `json:"promotion,omitempty"`
}

type WebMove struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
	SAN       string `json:"san,omitempty"`
}

type ServerMessage struct {
	Type        string            `json:"type"`
	GameID      string            `json:"gameId,omitempty"`
	Move        *WebMove          `json:"move,omitempty"`
	SAN         string            `json:"san,omitempty"`
	FEN         string            `json:"fen,omitempty"`
	Turn        string            `json:"turn,omitempty"`
	WhiteTime   int64             `json:"whiteTime,omitempty"`
	BlackTime   int64             `json:"blackTime,omitempty"`
	Status      game.Status       `json:"status,omitempty"`
	Outcome     string            `json:"outcome,omitempty"`
	Result      string            `json:"result,omitempty"`
	Winner      string            `json:"winner,omitempty"`
	IsCheck     bool              `json:"isCheck,omitempty"`
	OfferedBy   string            `json:"offeredBy,omitempty"`
	UserID      string            `json:"userId,omitempty"`
	Username    string            `json:"username,omitempty"`
	Error       string            `json:"error,omitempty"`
	White       *game.PlayerInfo  `json:"white,omitempty"`
	Black       *game.PlayerInfo  `json:"black,omitempty"`
	TimeControl *game.TimeControl `json:"timeControl,omitempty"`
	WhiteDiff   *int              `json:"whiteDiff,omitempty"`
	BlackDiff   *int              `json:"blackDiff,omitempty"`
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	gameID   string
	userID   string
	username string
	isPlayer bool
}

type GameRoom struct {
	gameID           string
	clients          map[*Client]bool
	disconnectTimers map[string]*time.Timer
	mu               sync.RWMutex
}

type Hub struct {
	rooms       map[string]*GameRoom
	mu          sync.RWMutex
	gameService *game.Service
	authService *auth.Service
	config      HubConfig
	upgrader    websocket.Upgrader
	rdb         *redis.Client
	instanceID  string
}

func (h *Hub) SetRedis(rdb *redis.Client, instanceID string) {
	h.rdb = rdb
	h.instanceID = instanceID
	if rdb != nil {
		go h.subscribeRedis()
	}
}

func NewHub(gameService *game.Service, authService *auth.Service, cfgs ...HubConfig) *Hub {
	var cfg HubConfig
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	} else {
		cfg = DefaultHubConfig()
	}

	h := &Hub{
		rooms:       make(map[string]*GameRoom),
		gameService: gameService,
		authService: authService,
		config:      cfg,
	}

	h.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return ValidateOrigin(r.Header.Get("Origin"), h.config, r.Host)
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	if gameService != nil {
		gameService.SetOnGameFinish(func(doc *game.GameDocument) {
			winner := ""
			if doc.Result == "1-0" {
				winner = "white"
			} else if doc.Result == "0-1" {
				winner = "black"
			}
			wDiff := doc.WhiteRatingDiff
			bDiff := doc.BlackRatingDiff
			h.Broadcast(doc.ID, ServerMessage{
				Type:      "game_finished",
				GameID:    doc.ID,
				Status:    doc.Status,
				Outcome:   doc.Outcome,
				Result:    doc.Result,
				Winner:    winner,
				FEN:       doc.FinalFEN,
				WhiteDiff: &wDiff,
				BlackDiff: &bDiff,
			})
		})
	}

	return h
}

func (h *Hub) getOrCreateRoom(gameID string) *GameRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[gameID]
	if !ok {
		room = &GameRoom{
			gameID:           gameID,
			clients:          make(map[*Client]bool),
			disconnectTimers: make(map[string]*time.Timer),
		}
		h.rooms[gameID] = room
	}
	return room
}

type RedisBroadcastEnvelope struct {
	GameID         string        `json:"gameId"`
	Msg            ServerMessage `json:"msg"`
	SenderInstance string        `json:"senderInstance"`
}

func (h *Hub) broadcastLocal(gameID string, msg ServerMessage) {
	h.mu.RLock()
	room, ok := h.rooms[gameID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	var dropped []*Client
	room.mu.RLock()
	for c := range room.clients {
		select {
		case c.send <- bytes:
		default:
			dropped = append(dropped, c)
		}
	}
	room.mu.RUnlock()

	if len(dropped) > 0 {
		room.mu.Lock()
		for _, c := range dropped {
			if _, exists := room.clients[c]; exists {
				close(c.send)
				delete(room.clients, c)
			}
		}
		room.mu.Unlock()
	}
}

func (h *Hub) Broadcast(gameID string, msg ServerMessage) {
	h.broadcastLocal(gameID, msg)

	if h.rdb != nil {
		env := RedisBroadcastEnvelope{
			GameID:         gameID,
			Msg:            msg,
			SenderInstance: h.instanceID,
		}
		if data, err := json.Marshal(env); err == nil {
			_ = h.rdb.Publish(context.Background(), "chess:broadcast", data).Err()
		}
	}
}

func (h *Hub) subscribeRedis() {
	if h.rdb == nil {
		return
	}
	ctx := context.Background()
	pubsub := h.rdb.Subscribe(ctx, "chess:broadcast")
	if _, err := pubsub.Receive(ctx); err != nil {
		return
	}
	ch := pubsub.Channel()
	for msg := range ch {
		var env RedisBroadcastEnvelope
		if err := json.Unmarshal([]byte(msg.Payload), &env); err == nil {
			if env.SenderInstance != h.instanceID {
				if h.gameService != nil {
					switch env.Msg.Type {
					case "move":
						from := ""
						to := ""
						if env.Msg.Move != nil {
							from = env.Msg.Move.From
							to = env.Msg.Move.To
						}
						h.gameService.ApplyRemoteMove(env.GameID, from, to, env.Msg.SAN, env.Msg.FEN, env.Msg.Turn, env.Msg.WhiteTime, env.Msg.BlackTime, env.Msg.Status)
					case "resignation", "game_finished", "draw_accepted":
						h.gameService.ApplyRemoteFinish(env.GameID, env.Msg.Status)
					}
				}
				h.broadcastLocal(env.GameID, env.Msg)
			}
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, gameID string) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// upgrader writes error status on check origin failure
		return
	}

	var userID string
	var username string

	// Extract token from query param or Authorization header
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		authH := r.Header.Get("Authorization")
		if strings.HasPrefix(authH, "Bearer ") {
			tokenStr = strings.TrimPrefix(authH, "Bearer ")
		}
	}

	if tokenStr != "" && h.authService != nil {
		claims, err := h.authService.ValidateAccessToken(tokenStr)
		if err == nil {
			userID = claims.UserID
			username = claims.Username
		}
	}

	room := h.getOrCreateRoom(gameID)

	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		gameID:   gameID,
		userID:   userID,
		username: username,
	}

	// Check if active game exists to send initial state
	if h.gameService != nil {
		ag, err := h.gameService.GetActiveGame(gameID)
		if err == nil {
			// A player must be an authenticated user whose ID matches either seat
			client.isPlayer = (userID != "" && (strings.EqualFold(ag.White.UserID, userID) || strings.EqualFold(ag.Black.UserID, userID)))
			wMs, bMs, _ := ag.Clock.CurrentTimes(ag.Status)
			initMsg := ServerMessage{
				Type:        "game_started",
				GameID:      ag.ID,
				FEN:         chess.ToFEN(ag.Engine.CurrentPosition),
				Turn:        ag.Engine.CurrentPosition.SideToMove.String(),
				WhiteTime:   wMs,
				BlackTime:   bMs,
				Status:      ag.Status,
				White:       &ag.White,
				Black:       &ag.Black,
				TimeControl: &ag.TimeControl,
				IsCheck:     chess.IsCheck(ag.Engine.CurrentPosition),
			}
			initBytes, _ := json.Marshal(initMsg)
			client.send <- initBytes
		}
	}

	room.mu.Lock()
	room.clients[client] = true
	// Cancel any active abandonment timer if this user reconnected
	if userID != "" {
		if t, ok := room.disconnectTimers[userID]; ok && t != nil {
			t.Stop()
			delete(room.disconnectTimers, userID)
		}
	}
	room.mu.Unlock()

	// Notify connection
	h.Broadcast(gameID, ServerMessage{
		Type:     "player_connected",
		GameID:   gameID,
		UserID:   userID,
		Username: username,
	})

	observability.DefaultMetrics.IncActiveWebSocket()
	go client.writePump()
	go client.readPump()
}

func (h *Hub) handlePlayerDisconnect(gameID, userID, username string) {
	if h.gameService == nil || userID == "" {
		return
	}

	h.mu.RLock()
	room, ok := h.rooms[gameID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	room.mu.Lock()
	delete(room.disconnectTimers, userID)
	for client := range room.clients {
		if strings.EqualFold(client.userID, userID) {
			room.mu.Unlock()
			return // User successfully reconnected before grace period expired
		}
	}
	room.mu.Unlock()

	ag, err := h.gameService.GetActiveGame(gameID)
	if err != nil || ag.Status != game.StatusPlaying {
		return
	}

	doc, err := h.gameService.Resign(context.Background(), gameID, userID)
	if err == nil {
		winner := "white"
		if doc.Result == "0-1" {
			winner = "black"
		}
		h.Broadcast(gameID, ServerMessage{
			Type:     "resignation",
			GameID:   gameID,
			Status:   doc.Status,
			Outcome:  "abandonment",
			Result:   doc.Result,
			Winner:   winner,
			UserID:   userID,
			Username: username,
		})
	}
}

func (c *Client) readPump() {
	defer func() {
		observability.DefaultMetrics.DecActiveWebSocket()
		room := c.hub.getOrCreateRoom(c.gameID)
		room.mu.Lock()
		delete(room.clients, c)

		// Check if the user has another active client connection in this room
		hasRemaining := false
		if c.userID != "" {
			for remaining := range room.clients {
				if strings.EqualFold(remaining.userID, c.userID) {
					hasRemaining = true
					break
				}
			}
		}

		if !hasRemaining && c.isPlayer && c.hub.gameService != nil && c.userID != "" {
			if prev, ok := room.disconnectTimers[c.userID]; ok && prev != nil {
				prev.Stop()
			}
			userID := c.userID
			gameID := c.gameID
			username := c.username
			room.disconnectTimers[c.userID] = time.AfterFunc(60*time.Second, func() {
				c.hub.handlePlayerDisconnect(gameID, userID, username)
			})
		}
		room.mu.Unlock()

		close(c.send)
		c.conn.Close()

		c.hub.Broadcast(c.gameID, ServerMessage{
			Type:     "player_disconnected",
			GameID:   c.gameID,
			UserID:   c.userID,
			Username: c.username,
		})
	}()

	c.conn.SetReadLimit(4096)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// Refresh read deadline on ANY received packet (including text pings and moves)
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var req ClientMessage
		if err := json.Unmarshal(message, &req); err != nil {
			c.sendError("invalid message format")
			continue
		}

		c.handleMessage(req)
	}
}

func (c *Client) handleMessage(msg ClientMessage) {
	ctx := context.Background()

	// Dynamically check player seat in active game in case seat was claimed after connection
	if !c.isPlayer && c.hub.gameService != nil && c.userID != "" {
		if ag, err := c.hub.gameService.GetActiveGame(c.gameID); err == nil {
			if strings.EqualFold(ag.White.UserID, c.userID) || strings.EqualFold(ag.Black.UserID, c.userID) {
				c.isPlayer = true
			}
		}
	}

	// Only players can perform moves, resign, or negotiate draws
	if msg.Type != "ping" && !c.isPlayer {
		c.sendError("only active players can submit moves or game actions")
		return
	}

	if c.hub.gameService == nil {
		c.sendError("game service unavailable")
		return
	}

	switch msg.Type {
	case "move":
		fromSq, err := chess.ParseSquare(msg.From)
		if err != nil {
			c.sendError("invalid from square: " + msg.From)
			return
		}
		toSq, err := chess.ParseSquare(msg.To)
		if err != nil {
			c.sendError("invalid to square: " + msg.To)
			return
		}

		var promo *chess.PieceType
		if msg.Promotion != "" {
			var pt chess.PieceType
			switch strings.ToLower(msg.Promotion) {
			case "q", "queen":
				pt = chess.Queen
			case "r", "rook":
				pt = chess.Rook
			case "b", "bishop":
				pt = chess.Bishop
			case "n", "knight":
				pt = chess.Knight
			}
			if pt != chess.NoPiece {
				promo = &pt
			}
		}

		chessMove := chess.NewMove(fromSq, toSq, promo)
		event, err := c.hub.gameService.MakeMove(ctx, c.gameID, c.userID, chessMove)
		if err != nil {
			c.sendError(err.Error())
			return
		}

		var promoStr string
		if event.Move.Promotion != nil {
			promoStr = event.Move.Promotion.String()
		}

		webMove := &WebMove{
			From:      event.Move.From.String(),
			To:        event.Move.To.String(),
			Promotion: promoStr,
			SAN:       event.SAN,
		}

		c.hub.Broadcast(c.gameID, ServerMessage{
			Type:      "move",
			GameID:    c.gameID,
			Move:      webMove,
			SAN:       event.SAN,
			FEN:       event.FEN,
			Turn:      event.Turn,
			WhiteTime: event.WhiteTimeMs,
			BlackTime: event.BlackTimeMs,
			Status:    event.Status,
			Outcome:   event.Outcome,
			Result:    event.Result,
			Winner:    event.Winner,
			IsCheck:   event.IsCheck,
		})

	case "resign":
		doc, err := c.hub.gameService.Resign(ctx, c.gameID, c.userID)
		if err != nil {
			c.sendError(err.Error())
			return
		}
		winner := "white"
		if doc.Result == "0-1" {
			winner = "black"
		}
		c.hub.Broadcast(c.gameID, ServerMessage{
			Type:    "resignation",
			GameID:  c.gameID,
			Status:  doc.Status,
			Outcome: doc.Outcome,
			Result:  doc.Result,
			Winner:  winner,
			UserID:  c.userID,
		})

	case "draw_offer":
		color, err := c.hub.gameService.OfferDraw(c.gameID, c.userID)
		if err != nil {
			c.sendError(err.Error())
			return
		}
		c.hub.Broadcast(c.gameID, ServerMessage{
			Type:      "draw_offer",
			GameID:    c.gameID,
			OfferedBy: color.String(),
			UserID:    c.userID,
		})

	case "draw_accept":
		doc, err := c.hub.gameService.AcceptDraw(ctx, c.gameID, c.userID)
		if err != nil {
			c.sendError(err.Error())
			return
		}
		c.hub.Broadcast(c.gameID, ServerMessage{
			Type:    "draw_accepted",
			GameID:  c.gameID,
			Status:  doc.Status,
			Outcome: doc.Outcome,
			Result:  doc.Result,
		})

	case "draw_decline":
		err := c.hub.gameService.DeclineDraw(c.gameID, c.userID)
		if err != nil {
			c.sendError(err.Error())
			return
		}
		c.hub.Broadcast(c.gameID, ServerMessage{
			Type:   "draw_declined",
			GameID: c.gameID,
			UserID: c.userID,
		})

	case "claim_draw":
		doc, err := c.hub.gameService.ClaimDraw(ctx, c.gameID, c.userID)
		if err != nil {
			c.sendError(err.Error())
			return
		}
		c.hub.Broadcast(c.gameID, ServerMessage{
			Type:    "draw_claimed",
			GameID:  c.gameID,
			Status:  doc.Status,
			Outcome: doc.Outcome,
			Result:  doc.Result,
		})

	case "ping":
		// Clock sync ping
		ag, err := c.hub.gameService.GetActiveGame(c.gameID)
		if err == nil {
			wMs, bMs, _ := ag.Clock.CurrentTimes(ag.Status)
			res, _ := json.Marshal(ServerMessage{
				Type:      "clock_update",
				GameID:    c.gameID,
				WhiteTime: wMs,
				BlackTime: bMs,
			})
			c.send <- res
		}
	}
}

func (c *Client) sendError(errStr string) {
	msg := ServerMessage{
		Type:  "error",
		Error: errStr,
	}
	bytes, _ := json.Marshal(msg)
	select {
	case c.send <- bytes:
	default:
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
