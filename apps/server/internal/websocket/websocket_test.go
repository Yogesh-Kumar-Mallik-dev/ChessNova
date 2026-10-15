package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/game"
	"chess-platform/server/internal/user"

	gorillaWs "github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHubRoomCreationAndBroadcast(t *testing.T) {
	hub := NewHub(nil, nil)

	// 1. Create/Get Room
	room1 := hub.getOrCreateRoom("game-123")
	if room1 == nil {
		t.Fatal("expected valid room")
	}
	if room1.gameID != "game-123" {
		t.Errorf("expected gameID game-123, got %s", room1.gameID)
	}

	// 2. Reuse existing room
	room2 := hub.getOrCreateRoom("game-123")
	if room1 != room2 {
		t.Error("expected getOrCreateRoom to return identical instance")
	}

	// 3. Broadcast to empty room (should not panic or deadlock)
	hub.Broadcast("game-123", ServerMessage{
		Type:   "test_message",
		GameID: "game-123",
	})

	// 4. Broadcast to non-existent room (should be a clean no-op)
	hub.Broadcast("game-999", ServerMessage{
		Type: "noop",
	})
}

func TestWebSocketOriginValidation(t *testing.T) {
	cfg := HubConfig{
		AllowedOrigins: []string{
			"http://localhost:5173",
			"https://chessnova.com",
			"tauri://localhost",
			"http://tauri.localhost",
		},
		AllowMissingOrigin: true,
		AllowDevLocalhost:  true,
	}

	tests := []struct {
		name    string
		origin  string
		allowed bool
	}{
		// Trusted origins
		{"Allowed configured origin", "http://localhost:5173", true},
		{"Allowed production origin", "https://chessnova.com", true},
		{"Allowed desktop tauri scheme", "tauri://localhost", true},
		{"Allowed desktop http tauri", "http://tauri.localhost", true},
		{"Allowed dev localhost with port", "http://localhost:3000", true},
		{"Allowed dev loopback 127.0.0.1", "http://127.0.0.1:8080", true},

		// Hostile origins
		{"Hostile external origin", "https://evil-attacker.com", false},
		{"Hostile subdomain trick", "http://localhost.evil.com", false},
		{"Hostile suffix attack", "http://chessnova.com.attacker.com", false},
		{"Hostile http instead of https prod", "http://chessnova.com", false},

		// Malformed origins
		{"Malformed empty scheme", "://broken", false},
		{"Malformed javascript scheme", "javascript:alert(1)", false},
		{"Malformed random text", "not-a-valid-origin", false},
		{"Malformed empty host", "http://", false},

		// Missing origin
		{"Missing origin when allowed", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateOrigin(tc.origin, cfg)
			if got != tc.allowed {
				t.Errorf("ValidateOrigin(%q) = %v; want %v", tc.origin, got, tc.allowed)
			}
		})
	}

	// Test missing origin rejection when disabled
	strictCfg := cfg
	strictCfg.AllowMissingOrigin = false
	if ValidateOrigin("", strictCfg) {
		t.Error("expected missing origin to be rejected when AllowMissingOrigin is false")
	}

	// Test same-host matching (e.g. Origin: https://chess.yogeshmallik.in with Host: chess.yogeshmallik.in)
	if !ValidateOrigin("https://chess.yogeshmallik.in", strictCfg, "chess.yogeshmallik.in") {
		t.Error("expected same-host origin to be allowed")
	}
	if !ValidateOrigin("https://chess.yogeshmallik.in", strictCfg, "chess.yogeshmallik.in:443") {
		t.Error("expected same-host origin with port to be allowed")
	}
}

func TestWebSocketUpgradeOriginRejection(t *testing.T) {
	cfg := HubConfig{
		AllowedOrigins:     []string{"http://localhost:5173"},
		AllowMissingOrigin: false,
		AllowDevLocalhost:  false,
	}
	hub := NewHub(nil, nil, cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, "test-game")
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 1. Connection with untrusted origin should fail
	dialer := gorillaWs.Dialer{}
	header := http.Header{}
	header.Set("Origin", "https://evil.com")
	_, resp, err := dialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected error dialing with untrusted origin")
	}
	if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", resp.StatusCode)
	}

	// 2. Connection with trusted origin should succeed
	header.Set("Origin", "http://localhost:5173")
	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("expected dial to succeed with trusted origin: %v", err)
	}
	conn.Close()
}

func readNextMessage(t *testing.T, conn *gorillaWs.Conn) ServerMessage {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("failed to read expected websocket message: %v", err)
		}
		var msg ServerMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		// Skip initial connection/ping notices to get to actual game events
		if msg.Type == "player_connected" || msg.Type == "ping" {
			continue
		}
		return msg
	}
}

func TestMultiplayerWebSocketMoveAndDrawFlow(t *testing.T) {
	jwtSecret := "super-secure-jwt-secret-at-least-32-chars-long"
	authService := auth.NewService(nil, jwtSecret, jwtSecret)
	gameService := game.NewService(nil, nil)
	cfg := HubConfig{
		AllowedOrigins:     []string{"http://localhost:5173"},
		AllowMissingOrigin: true,
	}
	hub := NewHub(gameService, authService, cfg)

	uWhite := &user.User{ID: primitive.NewObjectID(), Username: "alice"}
	uBlack := &user.User{ID: primitive.NewObjectID(), Username: "bob"}

	whiteTokens, _ := authService.GenerateTokenPair(uWhite)
	blackTokens, _ := authService.GenerateTokenPair(uBlack)

	whitePlayer := game.PlayerInfo{UserID: uWhite.ID.Hex(), Username: uWhite.Username, Rating: 1500}
	blackPlayer := game.PlayerInfo{UserID: uBlack.ID.Hex(), Username: uBlack.Username, Rating: 1500}
	tc := game.TimeControl{InitialSeconds: 300, Increment: 0}

	ag, err := gameService.CreateGame(whitePlayer, blackPlayer, tc)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ag.ID)
	}))
	defer server.Close()

	baseURL := "ws" + strings.TrimPrefix(server.URL, "http")

	dialer := gorillaWs.Dialer{}
	header := http.Header{"Origin": []string{"http://localhost:5173"}}

	// 1. Connect White Player
	whiteConn, _, err := dialer.Dial(baseURL+"?token="+whiteTokens.AccessToken, header)
	if err != nil {
		t.Fatalf("white player failed to connect: %v", err)
	}
	defer whiteConn.Close()

	// 2. Connect Black Player
	blackConn, _, err := dialer.Dial(baseURL+"?token="+blackTokens.AccessToken, header)
	if err != nil {
		t.Fatalf("black player failed to connect: %v", err)
	}
	defer blackConn.Close()

	// Drain initial game_started messages
	_ = readNextMessage(t, whiteConn)
	_ = readNextMessage(t, blackConn)

	// 3. Test Move Translation: White plays 1. e4
	err = whiteConn.WriteJSON(ClientMessage{
		Type: "move",
		From: "e2",
		To:   "e4",
	})
	if err != nil {
		t.Fatalf("white failed to send move: %v", err)
	}

	// Black client must receive move with SAN "e4"
	moveMsgForBlack := readNextMessage(t, blackConn)
	if moveMsgForBlack.Type != "move" || moveMsgForBlack.SAN != "e4" {
		t.Fatalf("expected black to receive move e4, got type=%s, san=%s", moveMsgForBlack.Type, moveMsgForBlack.SAN)
	}
	// White client also receives broadcast confirmation of move e4
	moveMsgForWhiteEcho := readNextMessage(t, whiteConn)
	if moveMsgForWhiteEcho.Type != "move" || moveMsgForWhiteEcho.SAN != "e4" {
		t.Fatalf("expected white to receive echoed move e4, got type=%s, san=%s", moveMsgForWhiteEcho.Type, moveMsgForWhiteEcho.SAN)
	}

	// 4. Test Move Translation: Black plays 1... e5
	err = blackConn.WriteJSON(ClientMessage{
		Type: "move",
		From: "e7",
		To:   "e5",
	})
	if err != nil {
		t.Fatalf("black failed to send move: %v", err)
	}

	// White client must receive move with SAN "e5"
	moveMsgForWhite := readNextMessage(t, whiteConn)
	if moveMsgForWhite.Type != "move" || moveMsgForWhite.SAN != "e5" {
		t.Fatalf("expected white to receive move e5, got type=%s, san=%s", moveMsgForWhite.Type, moveMsgForWhite.SAN)
	}
	// Black client also receives echoed move e5
	moveMsgForBlackEcho := readNextMessage(t, blackConn)
	if moveMsgForBlackEcho.Type != "move" || moveMsgForBlackEcho.SAN != "e5" {
		t.Fatalf("expected black to receive echoed move e5, got type=%s, san=%s", moveMsgForBlackEcho.Type, moveMsgForBlackEcho.SAN)
	}

	// 5. Test Draw Negotiation: White offers draw
	err = whiteConn.WriteJSON(ClientMessage{
		Type: "draw_offer",
	})
	if err != nil {
		t.Fatalf("white failed to offer draw: %v", err)
	}

	drawOfferMsg := readNextMessage(t, blackConn)
	if drawOfferMsg.Type != "draw_offer" {
		t.Fatalf("expected black to receive draw_offer, got type=%s", drawOfferMsg.Type)
	}
	drawOfferEcho := readNextMessage(t, whiteConn)
	if drawOfferEcho.Type != "draw_offer" {
		t.Fatalf("expected white to receive echoed draw_offer, got type=%s", drawOfferEcho.Type)
	}

	// 6. Black accepts draw
	err = blackConn.WriteJSON(ClientMessage{
		Type: "draw_accept",
	})
	if err != nil {
		t.Fatalf("black failed to accept draw: %v", err)
	}

	// Both receive game_finished or draw_accepted
	drawFinForWhite := readNextMessage(t, whiteConn)
	if drawFinForWhite.Type != "game_finished" && drawFinForWhite.Type != "draw_accepted" {
		t.Fatalf("expected white to receive game_finished/draw_accepted, got type=%s", drawFinForWhite.Type)
	}

	drawFinForBlack := readNextMessage(t, blackConn)
	if drawFinForBlack.Type != "game_finished" && drawFinForBlack.Type != "draw_accepted" {
		t.Fatalf("expected black to receive game_finished/draw_accepted, got type=%s", drawFinForBlack.Type)
	}
}

func TestMultiplayerResignationFlow(t *testing.T) {
	jwtSecret := "super-secure-jwt-secret-at-least-32-chars-long"
	authService := auth.NewService(nil, jwtSecret, jwtSecret)
	gameService := game.NewService(nil, nil)
	cfg := HubConfig{
		AllowedOrigins:     []string{"http://localhost:5173"},
		AllowMissingOrigin: true,
	}
	hub := NewHub(gameService, authService, cfg)

	uWhite := &user.User{ID: primitive.NewObjectID(), Username: "alice"}
	uBlack := &user.User{ID: primitive.NewObjectID(), Username: "bob"}

	whiteTokens, _ := authService.GenerateTokenPair(uWhite)
	blackTokens, _ := authService.GenerateTokenPair(uBlack)

	whitePlayer := game.PlayerInfo{UserID: uWhite.ID.Hex(), Username: uWhite.Username, Rating: 1500}
	blackPlayer := game.PlayerInfo{UserID: uBlack.ID.Hex(), Username: uBlack.Username, Rating: 1500}
	tc := game.TimeControl{InitialSeconds: 300, Increment: 0}

	ag, err := gameService.CreateGame(whitePlayer, blackPlayer, tc)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ag.ID)
	}))
	defer server.Close()

	baseURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := gorillaWs.Dialer{}
	header := http.Header{"Origin": []string{"http://localhost:5173"}}

	whiteConn, _, _ := dialer.Dial(baseURL+"?token="+whiteTokens.AccessToken, header)
	defer whiteConn.Close()

	blackConn, _, _ := dialer.Dial(baseURL+"?token="+blackTokens.AccessToken, header)
	defer blackConn.Close()

	_ = readNextMessage(t, whiteConn)
	_ = readNextMessage(t, blackConn)

	// White resigns
	err = whiteConn.WriteJSON(ClientMessage{
		Type: "resign",
	})
	if err != nil {
		t.Fatalf("white failed to send resign: %v", err)
	}

	// Black receives game_finished and resignation events, both indicating winner=black
	msg1 := readNextMessage(t, blackConn)
	msg2 := readNextMessage(t, blackConn)

	gotGameFinished := (msg1.Type == "game_finished" || msg2.Type == "game_finished")
	gotResignation := (msg1.Type == "resignation" || msg2.Type == "resignation")

	if !gotGameFinished || !gotResignation {
		t.Fatalf("expected black to receive game_finished and resignation, got %s and %s", msg1.Type, msg2.Type)
	}
	if msg1.Winner != "black" && msg2.Winner != "black" {
		t.Fatalf("expected winner to be black, got msg1.winner=%s msg2.winner=%s", msg1.Winner, msg2.Winner)
	}
}

func TestMultiplayerContinuousRealTimeMoveTranslation(t *testing.T) {
	jwtSecret := "super-secure-jwt-secret-at-least-32-chars-long"
	authService := auth.NewService(nil, jwtSecret, jwtSecret)
	gameService := game.NewService(nil, nil)
	cfg := HubConfig{
		AllowedOrigins:     []string{"http://localhost:5173"},
		AllowMissingOrigin: true,
	}
	hub := NewHub(gameService, authService, cfg)

	uWhite := &user.User{ID: primitive.NewObjectID(), Username: "alice"}
	uBlack := &user.User{ID: primitive.NewObjectID(), Username: "bob"}

	whiteTokens, _ := authService.GenerateTokenPair(uWhite)
	blackTokens, _ := authService.GenerateTokenPair(uBlack)

	whitePlayer := game.PlayerInfo{UserID: uWhite.ID.Hex(), Username: uWhite.Username, Rating: 1500}
	blackPlayer := game.PlayerInfo{UserID: uBlack.ID.Hex(), Username: uBlack.Username, Rating: 1500}
	tc := game.TimeControl{InitialSeconds: 300, Increment: 0}

	ag, err := gameService.CreateGame(whitePlayer, blackPlayer, tc)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r, ag.ID)
	}))
	defer server.Close()

	baseURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := gorillaWs.Dialer{}
	header := http.Header{"Origin": []string{"http://localhost:5173"}}

	whiteConn, _, err := dialer.Dial(baseURL+"?token="+whiteTokens.AccessToken, header)
	if err != nil {
		t.Fatalf("white player connection failed: %v", err)
	}
	defer whiteConn.Close()

	blackConn, _, err := dialer.Dial(baseURL+"?token="+blackTokens.AccessToken, header)
	if err != nil {
		t.Fatalf("black player connection failed: %v", err)
	}
	defer blackConn.Close()

	// Drain game_started messages
	_ = readNextMessage(t, whiteConn)
	_ = readNextMessage(t, blackConn)

	// Opening moves sequence: 1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5
	moves := []struct {
		player string
		from   string
		to     string
		san    string
		turn   string
	}{
		{"white", "e2", "e4", "e4", "black"},
		{"black", "e7", "e5", "e5", "white"},
		{"white", "g1", "f3", "Nf3", "black"},
		{"black", "b8", "c6", "Nc6", "white"},
		{"white", "f1", "c4", "Bc4", "black"},
		{"black", "f8", "c5", "Bc5", "white"},
	}

	for i, m := range moves {
		var activeConn, opponentConn *gorillaWs.Conn
		if m.player == "white" {
			activeConn = whiteConn
			opponentConn = blackConn
		} else {
			activeConn = blackConn
			opponentConn = whiteConn
		}

		err = activeConn.WriteJSON(ClientMessage{
			Type: "move",
			From: m.from,
			To:   m.to,
		})
		if err != nil {
			t.Fatalf("step %d (%s %s-%s): write move failed: %v", i+1, m.player, m.from, m.to, err)
		}

		oppMsg := readNextMessage(t, opponentConn)
		if oppMsg.Type != "move" || oppMsg.SAN != m.san || oppMsg.Turn != m.turn {
			t.Fatalf("step %d (%s): opponent expected move SAN=%s turn=%s, got type=%s san=%s turn=%s",
				i+1, m.player, m.san, m.turn, oppMsg.Type, oppMsg.SAN, oppMsg.Turn)
		}

		echoMsg := readNextMessage(t, activeConn)
		if echoMsg.Type != "move" || echoMsg.SAN != m.san || echoMsg.Turn != m.turn {
			t.Fatalf("step %d (%s): echo expected move SAN=%s turn=%s, got type=%s san=%s turn=%s",
				i+1, m.player, m.san, m.turn, echoMsg.Type, echoMsg.SAN, echoMsg.Turn)
		}
	}

	// Test out-of-turn rejection: Black tries to move when it's White's turn
	err = blackConn.WriteJSON(ClientMessage{
		Type: "move",
		From: "d7",
		To:   "d6",
	})
	if err != nil {
		t.Fatalf("failed to send out-of-turn move: %v", err)
	}

	errMsg := readNextMessage(t, blackConn)
	if errMsg.Type != "error" {
		t.Fatalf("expected error message on out-of-turn move, got type=%s", errMsg.Type)
	}
}

func TestMultiplayerMoveTranslationWithRedisCluster(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not available; skipping multi-instance cluster test")
	}

	jwtSecret := "super-secure-jwt-secret-at-least-32-chars-long"
	authService := auth.NewService(nil, jwtSecret, jwtSecret)

	gameService1 := game.NewService(nil, nil)
	gameService1.SetRedis(rdb)

	gameService2 := game.NewService(nil, nil)
	gameService2.SetRedis(rdb)

	cfg := HubConfig{
		AllowedOrigins:     []string{"http://localhost:5173"},
		AllowMissingOrigin: true,
	}

	// Instance 1
	hub1 := NewHub(gameService1, authService, cfg)
	hub1.SetRedis(rdb, "instance-1")

	// Instance 2
	hub2 := NewHub(gameService2, authService, cfg)
	hub2.SetRedis(rdb, "instance-2")

	// Allow Redis subscription goroutines to settle
	time.Sleep(100 * time.Millisecond)

	uWhite := &user.User{ID: primitive.NewObjectID(), Username: "alice"}
	uBlack := &user.User{ID: primitive.NewObjectID(), Username: "bob"}

	whiteTokens, _ := authService.GenerateTokenPair(uWhite)
	blackTokens, _ := authService.GenerateTokenPair(uBlack)

	whitePlayer := game.PlayerInfo{UserID: uWhite.ID.Hex(), Username: uWhite.Username, Rating: 1500}
	blackPlayer := game.PlayerInfo{UserID: uBlack.ID.Hex(), Username: uBlack.Username, Rating: 1500}
	tc := game.TimeControl{InitialSeconds: 300, Increment: 0}

	ag, err := gameService1.CreateGame(whitePlayer, blackPlayer, tc)
	if err != nil {
		t.Fatalf("failed to create game on instance 1: %v", err)
	}

	// Server 1 hosts Hub 1 (Instance 1)
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub1.HandleWebSocket(w, r, ag.ID)
	}))
	defer server1.Close()

	// Server 2 hosts Hub 2 (Instance 2)
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub2.HandleWebSocket(w, r, ag.ID)
	}))
	defer server2.Close()

	dialer := gorillaWs.Dialer{}
	header := http.Header{"Origin": []string{"http://localhost:5173"}}

	// White connects to Instance 1
	base1 := "ws" + strings.TrimPrefix(server1.URL, "http")
	whiteConn, _, err := dialer.Dial(base1+"?token="+whiteTokens.AccessToken, header)
	if err != nil {
		t.Fatalf("white failed to connect to server 1: %v", err)
	}
	defer whiteConn.Close()

	// Black connects to Instance 2
	base2 := "ws" + strings.TrimPrefix(server2.URL, "http")
	blackConn, _, err := dialer.Dial(base2+"?token="+blackTokens.AccessToken, header)
	if err != nil {
		t.Fatalf("black failed to connect to server 2: %v", err)
	}
	defer blackConn.Close()

	_ = readNextMessage(t, whiteConn)
	_ = readNextMessage(t, blackConn)

	// White plays 1. d4 on Instance 1
	err = whiteConn.WriteJSON(ClientMessage{
		Type: "move",
		From: "d2",
		To:   "d4",
	})
	if err != nil {
		t.Fatalf("white failed to send move: %v", err)
	}

	// Black on Instance 2 must receive 1. d4 via Redis Pub/Sub broadcast!
	msgForBlack := readNextMessage(t, blackConn)
	if msgForBlack.Type != "move" || msgForBlack.SAN != "d4" {
		t.Fatalf("cross-instance move translation failed: expected SAN d4, got type=%s san=%s", msgForBlack.Type, msgForBlack.SAN)
	}

	// White receives local echo on Instance 1
	msgForWhiteEcho := readNextMessage(t, whiteConn)
	if msgForWhiteEcho.Type != "move" || msgForWhiteEcho.SAN != "d4" {
		t.Fatalf("white echo failed: expected SAN d4, got type=%s san=%s", msgForWhiteEcho.Type, msgForWhiteEcho.SAN)
	}

	// Black plays 1... d5 on Instance 2
	err = blackConn.WriteJSON(ClientMessage{
		Type: "move",
		From: "d7",
		To:   "d5",
	})
	if err != nil {
		t.Fatalf("black failed to send move: %v", err)
	}

	// White on Instance 1 must receive 1... d5 via Redis Pub/Sub broadcast!
	msgForWhite := readNextMessage(t, whiteConn)
	if msgForWhite.Type != "move" || msgForWhite.SAN != "d5" {
		t.Fatalf("cross-instance move translation failed: expected SAN d5, got type=%s san=%s", msgForWhite.Type, msgForWhite.SAN)
	}
}

