package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/game"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev/CORS
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ClientMessage struct {
	Type      string `json:"type"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Promotion string `json:"promotion,omitempty"`
}

type ServerMessage struct {
	Type        string            `json:"type"`
	GameID      string            `json:"gameId,omitempty"`
	Move        *chess.Move       `json:"move,omitempty"`
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
	gameID  string
	clients map[*Client]bool
	mu      sync.RWMutex
}

type Hub struct {
	rooms       map[string]*GameRoom
	mu          sync.RWMutex
	gameService *game.Service
	authService *auth.Service
}

func NewHub(gameService *game.Service, authService *auth.Service) *Hub {
	h := &Hub{
		rooms:       make(map[string]*GameRoom),
		gameService: gameService,
		authService: authService,
	}

	gameService.SetOnGameFinish(func(doc *game.GameDocument) {
		winner := ""
		if doc.Result == "1-0" {
			winner = "white"
		} else if doc.Result == "0-1" {
			winner = "black"
		}
		h.Broadcast(doc.ID, ServerMessage{
			Type:      "game_finished",
			GameID:    doc.ID,
			Status:    doc.Status,
			Outcome:   doc.Outcome,
			Result:    doc.Result,
			Winner:    winner,
			FEN:       doc.FinalFEN,
		})
	})

	return h
}

func (h *Hub) getOrCreateRoom(gameID string) *GameRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[gameID]
	if !ok {
		room = &GameRoom{
			gameID:  gameID,
			clients: make(map[*Client]bool),
		}
		h.rooms[gameID] = room
	}
	return room
}

func (h *Hub) Broadcast(gameID string, msg ServerMessage) {
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

	room.mu.RLock()
	defer room.mu.RUnlock()
	for c := range room.clients {
		select {
		case c.send <- bytes:
		default:
			close(c.send)
			delete(room.clients, c)
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, gameID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	var userID string
	var username string

	// Extract token from query param or header
	tokenStr := r.URL.Query().Get("token")
	if tokenStr != "" {
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
	ag, err := h.gameService.GetActiveGame(gameID)
	if err == nil {
		client.isPlayer = (ag.White.UserID == userID || ag.Black.UserID == userID)
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

	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	// Notify connection
	h.Broadcast(gameID, ServerMessage{
		Type:     "player_connected",
		GameID:   gameID,
		UserID:   userID,
		Username: username,
	})

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		room := c.hub.getOrCreateRoom(c.gameID)
		room.mu.Lock()
		delete(room.clients, c)
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

	switch msg.Type {
	case "move":
		fromSq, err := chess.ParseSquare(msg.From)
		if err != nil {
			c.sendError("invalid from square")
			return
		}
		toSq, err := chess.ParseSquare(msg.To)
		if err != nil {
			c.sendError("invalid to square")
			return
		}

		var promo *chess.PieceType
		if msg.Promotion != "" {
			var pt chess.PieceType
			switch msg.Promotion {
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
		event, err := c.hub.gameService.MakeMove(ctx, c.gameID, c.userID, chessMove)
		if err != nil {
			c.sendError(err.Error())
			return
		}

		c.hub.Broadcast(c.gameID, ServerMessage{
			Type:      "move",
			GameID:    c.gameID,
			Move:      &event.Move,
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
