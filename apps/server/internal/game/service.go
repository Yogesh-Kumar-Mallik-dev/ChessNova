package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/observability"
	"chess-platform/server/internal/rating"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrGameNotFound     = errors.New("game not found")
	ErrNotPlayerInGame  = errors.New("user is not a player in this game")
	ErrNotPlayerTurn    = errors.New("not your turn")
	ErrGameNotPlaying   = errors.New("game is not currently active")
	ErrPlayerTimedOut   = errors.New("player time expired")
)

type MoveEvent struct {
	GameID       string      `json:"gameId"`
	Move         chess.Move  `json:"move"`
	SAN          string      `json:"san"`
	FEN          string      `json:"fen"`
	Turn         string      `json:"turn"`
	WhiteTimeMs  int64       `json:"whiteTime"`
	BlackTimeMs  int64       `json:"blackTime"`
	IsCheck      bool        `json:"isCheck"`
	Status       Status      `json:"status"`
	Outcome      string      `json:"outcome,omitempty"`
	Result       string      `json:"result,omitempty"`
	Winner       string      `json:"winner,omitempty"`
	FinishedDoc  *GameDocument `json:"finishedDoc,omitempty"`
}

type Service struct {
	gameRepo     Repository
	ratingRepo   rating.Repository
	activeGames  map[string]*ActiveGame
	mu           sync.RWMutex
	onGameFinish func(doc *GameDocument)
	stopChan     chan struct{}
	stopOnce     sync.Once
	rdb          *redis.Client
}

func NewService(gameRepo Repository, ratingRepo rating.Repository) *Service {
	s := &Service{
		gameRepo:    gameRepo,
		ratingRepo:  ratingRepo,
		activeGames: make(map[string]*ActiveGame),
		stopChan:    make(chan struct{}),
	}
	go s.startTimeoutWatcher()
	return s
}

func (s *Service) SetRedis(rdb *redis.Client) {
	s.rdb = rdb
}

func (s *Service) syncToRedis(ag *ActiveGame) {
	if s.rdb == nil || ag == nil {
		return
	}
	ag.Lock()
	defer ag.Unlock()
	data, err := json.Marshal(ag)
	if err == nil {
		_ = s.rdb.Set(context.Background(), fmt.Sprintf("active_game:%s", ag.ID), data, 24*time.Hour).Err()
	}
}

func (s *Service) Close() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})
}

func (s *Service) SetOnGameFinish(cb func(doc *GameDocument)) {
	s.onGameFinish = cb
}

func (s *Service) CreateGame(white, black PlayerInfo, tc TimeControl) (*ActiveGame, error) {
	id := uuid.New().String()
	ag := NewActiveGame(id, white, black, tc)

	s.mu.Lock()
	s.activeGames[id] = ag
	count := int64(len(s.activeGames))
	s.mu.Unlock()

	observability.DefaultMetrics.SetActiveGames(count)
	s.syncToRedis(ag)

	return ag, nil
}

func (s *Service) GetActiveGame(id string) (*ActiveGame, error) {
	s.mu.RLock()
	ag, ok := s.activeGames[id]
	s.mu.RUnlock()
	if ok {
		return ag, nil
	}

	if s.rdb != nil {
		data, err := s.rdb.Get(context.Background(), fmt.Sprintf("active_game:%s", id)).Result()
		if err == nil {
			var loaded ActiveGame
			if err := json.Unmarshal([]byte(data), &loaded); err == nil {
				loaded.Engine = chess.NewGame()
				for _, m := range loaded.Moves {
					mv, err := chess.ParseUCIMove(m.From + m.To)
					if err == nil {
						_, _ = loaded.Engine.MakeMove(mv)
					}
				}
				s.mu.Lock()
				s.activeGames[loaded.ID] = &loaded
				s.mu.Unlock()
				return &loaded, nil
			}
		}
	}

	return nil, ErrGameNotFound
}

func (s *Service) GetActiveGameForUser(userID string) *ActiveGame {
	if userID == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ag := range s.activeGames {
		ag.Lock()
		st := ag.Status
		isPlayer := (ag.White.UserID == userID || ag.Black.UserID == userID)
		ag.Unlock()
		if st == StatusPlaying && isPlayer {
			return ag
		}
	}
	return nil
}

func (s *Service) JoinGame(gameID string, player PlayerInfo) (*ActiveGame, error) {
	ag, err := s.GetActiveGame(gameID)
	if err != nil {
		return nil, err
	}

	ag.Lock()
	defer ag.Unlock()

	// If player is already white or black, return active game
	if ag.White.UserID == player.UserID || ag.Black.UserID == player.UserID {
		return ag, nil
	}

	// Claim open/guest seat
	if ag.Black.UserID == "guest-opponent" || ag.Black.UserID == "" {
		ag.Black = player
		return ag, nil
	}
	if ag.White.UserID == "guest-opponent" || ag.White.UserID == "" {
		ag.White = player
		return ag, nil
	}

	return nil, errors.New("game room is already full")
}

func (s *Service) MakeMove(ctx context.Context, gameID, userID string, m chess.Move) (*MoveEvent, error) {
	ag, err := s.GetActiveGame(gameID)
	if err != nil {
		return nil, err
	}

	ag.Lock()
	defer ag.Unlock()

	if ag.Status != StatusPlaying {
		return nil, ErrGameNotPlaying
	}

	// 1. Verify player
	var playerColor chess.Color
	if ag.White.UserID == userID {
		playerColor = chess.White
	} else if ag.Black.UserID == userID {
		playerColor = chess.Black
	} else {
		return nil, ErrNotPlayerInGame
	}

	// 2. Verify turn
	if ag.Engine.CurrentPosition.SideToMove != playerColor {
		return nil, ErrNotPlayerTurn
	}

	// 3. Check clock timeout
	wMs, bMs, timedOutColor := ag.Clock.CurrentTimes(ag.Status)
	if timedOutColor != nil && *timedOutColor == playerColor {
		ag.Status = StatusFinished
		_ = ag.Engine.Timeout(playerColor)
		doc := s.finalizeGame(ctx, ag)
		return &MoveEvent{
			GameID:      ag.ID,
			WhiteTimeMs: 0,
			BlackTimeMs: bMs,
			Status:      StatusFinished,
			Outcome:     string(chess.OutcomeTimeout),
			Result:      ag.Engine.Result(),
			Winner:      playerColor.Other().String(),
			FinishedDoc: doc,
		}, ErrPlayerTimedOut
	}

	// 4. Validate & apply move in engine
	rec, err := ag.Engine.MakeMove(m)
	if err != nil {
		return nil, err
	}

	// 5. Update clock
	ag.Clock.UpdateAfterMove(ag.TimeControl.Increment)
	wMs, bMs, _ = ag.Clock.CurrentTimes(ag.Status)

	// 6. Record move
	timeSpent := int64(0)
	ag.Moves = append(ag.Moves, PersistedMove{
		Ply:       rec.Ply,
		From:      m.From.String(),
		To:        m.To.String(),
		SAN:       rec.SAN,
		FEN:       rec.FENAfter,
		TimeSpent: timeSpent,
	})

	// 7. Check if game finished
	isFinished := ag.Engine.IsOver()
	var doc *GameDocument
	if isFinished {
		ag.Status = StatusFinished
		doc = s.finalizeGame(ctx, ag)
	}

	winnerStr := ""
	if ag.Engine.Winner != nil {
		winnerStr = ag.Engine.Winner.String()
	}

	event := &MoveEvent{
		GameID:      ag.ID,
		Move:        m,
		SAN:         rec.SAN,
		FEN:         rec.FENAfter,
		Turn:        ag.Engine.CurrentPosition.SideToMove.String(),
		WhiteTimeMs: wMs,
		BlackTimeMs: bMs,
		IsCheck:     chess.IsCheck(ag.Engine.CurrentPosition),
		Status:      ag.Status,
		Outcome:     string(ag.Engine.Outcome),
		Result:      ag.Engine.Result(),
		Winner:      winnerStr,
		FinishedDoc: doc,
	}

	if !isFinished {
		s.syncToRedis(ag)
	}

	return event, nil
}

func (s *Service) Resign(ctx context.Context, gameID, userID string) (*GameDocument, error) {
	ag, err := s.GetActiveGame(gameID)
	if err != nil {
		return nil, err
	}

	ag.Lock()
	defer ag.Unlock()

	if ag.Status != StatusPlaying {
		// Game is already finished; return current document state idempotently
		if s.gameRepo != nil {
			doc, errRepo := s.gameRepo.FindByID(ctx, gameID)
			if errRepo == nil && doc != nil {
				return doc, nil
			}
		}
		return nil, ErrGameNotPlaying
	}

	var color chess.Color
	if strings.EqualFold(ag.White.UserID, userID) {
		color = chess.White
	} else if strings.EqualFold(ag.Black.UserID, userID) {
		color = chess.Black
	} else {
		return nil, ErrNotPlayerInGame
	}

	if err := ag.Engine.Resign(color); err != nil {
		// If engine was already over, still finalize status cleanly
		if ag.Engine.Outcome != chess.OutcomePlaying {
			ag.Status = StatusFinished
			doc := s.finalizeGame(ctx, ag)
			return doc, nil
		}
		return nil, err
	}

	ag.Status = StatusFinished
	doc := s.finalizeGame(ctx, ag)
	return doc, nil
}

// CloseAllActiveGames terminates and finalizes all active games across the service
func (s *Service) CloseAllActiveGames(ctx context.Context) int {
	s.mu.Lock()
	games := make([]*ActiveGame, 0, len(s.activeGames))
	for _, ag := range s.activeGames {
		games = append(games, ag)
	}
	s.mu.Unlock()

	closed := 0
	for _, ag := range games {
		ag.Lock()
		if ag.Status == StatusPlaying {
			ag.Status = StatusFinished
			if ag.Engine != nil && ag.Engine.Outcome == chess.OutcomePlaying {
				ag.Engine.Outcome = chess.OutcomeAborted
			}
			_ = s.finalizeGame(ctx, ag)
			closed++
		}
		ag.Unlock()
	}

	// Purge all active game keys in Redis
	if s.rdb != nil {
		keys, err := s.rdb.Keys(ctx, "active_game:*").Result()
		if err == nil && len(keys) > 0 {
			_ = s.rdb.Del(ctx, keys...).Err()
		}
	}

	s.mu.Lock()
	s.activeGames = make(map[string]*ActiveGame)
	s.mu.Unlock()
	observability.DefaultMetrics.SetActiveGames(0)

	return closed
}

func (s *Service) OfferDraw(gameID, userID string) (chess.Color, error) {
	ag, err := s.GetActiveGame(gameID)
	if err != nil {
		return chess.White, err
	}

	ag.Lock()
	defer ag.Unlock()

	if ag.Status != StatusPlaying {
		return chess.White, ErrGameNotPlaying
	}

	var color chess.Color
	if ag.White.UserID == userID {
		color = chess.White
	} else if ag.Black.UserID == userID {
		color = chess.Black
	} else {
		return chess.White, ErrNotPlayerInGame
	}

	return color, ag.Engine.OfferDraw(color)
}

func (s *Service) AcceptDraw(ctx context.Context, gameID, userID string) (*GameDocument, error) {
	ag, err := s.GetActiveGame(gameID)
	if err != nil {
		return nil, err
	}

	ag.Lock()
	defer ag.Unlock()

	if ag.Status != StatusPlaying {
		return nil, ErrGameNotPlaying
	}

	var color chess.Color
	if ag.White.UserID == userID {
		color = chess.White
	} else if ag.Black.UserID == userID {
		color = chess.Black
	} else {
		return nil, ErrNotPlayerInGame
	}

	if err := ag.Engine.AcceptDraw(color); err != nil {
		return nil, err
	}

	ag.Status = StatusFinished
	doc := s.finalizeGame(ctx, ag)
	return doc, nil
}

func (s *Service) DeclineDraw(gameID, userID string) error {
	ag, err := s.GetActiveGame(gameID)
	if err != nil {
		return err
	}

	ag.Lock()
	defer ag.Unlock()

	var color chess.Color
	if ag.White.UserID == userID {
		color = chess.White
	} else if ag.Black.UserID == userID {
		color = chess.Black
	} else {
		return ErrNotPlayerInGame
	}

	return ag.Engine.DeclineDraw(color)
}

func (s *Service) ClaimDraw(ctx context.Context, gameID, userID string) (*GameDocument, error) {
	ag, err := s.GetActiveGame(gameID)
	if err != nil {
		return nil, err
	}

	ag.Lock()
	defer ag.Unlock()

	if ag.Status != StatusPlaying {
		return nil, ErrGameNotPlaying
	}

	if ag.White.UserID != userID && ag.Black.UserID != userID {
		return nil, ErrNotPlayerInGame
	}

	if err := ag.Engine.ClaimDraw(); err != nil {
		return nil, err
	}

	ag.Status = StatusFinished
	doc := s.finalizeGame(ctx, ag)
	return doc, nil
}

func (s *Service) finalizeGame(ctx context.Context, ag *ActiveGame) *GameDocument {
	now := time.Now().UTC()
	ag.FinishedAt = &now

	pgnStr := ag.Engine.ToPGN(map[string]string{
		"Event":     "Online Game",
		"Site":      "Antigravity Chess",
		"Date":      ag.StartedAt.Format("2006.01.02"),
		"Round":     "1",
		"White":     ag.White.Username,
		"Black":     ag.Black.Username,
		"Result":    ag.Engine.Result(),
		"TimeControl": fmt.Sprintf("%d+%d", ag.TimeControl.InitialSeconds, ag.TimeControl.Increment),
	})

	// Elo Rating update
	category := rating.DetermineCategory(ag.TimeControl.InitialSeconds)
	var whiteDiff, blackDiff int

	whiteOID, wErr := primitive.ObjectIDFromHex(ag.White.UserID)
	blackOID, bErr := primitive.ObjectIDFromHex(ag.Black.UserID)

	if wErr == nil && bErr == nil && s.ratingRepo != nil {
		wRating, errW := s.ratingRepo.GetRating(ctx, whiteOID, category)
		if errW != nil || wRating == nil {
			wRating = &rating.UserRating{
				UserID:   whiteOID,
				Username: ag.White.Username,
				Category: category,
				Rating:   rating.DefaultRating,
			}
		}
		bRating, errB := s.ratingRepo.GetRating(ctx, blackOID, category)
		if errB != nil || bRating == nil {
			bRating = &rating.UserRating{
				UserID:   blackOID,
				Username: ag.Black.Username,
				Category: category,
				Rating:   rating.DefaultRating,
			}
		}

		var scoreA float64
		switch ag.Engine.Result() {
		case "1-0":
			scoreA = 1.0
			wRating.Wins++
			bRating.Losses++
		case "0-1":
			scoreA = 0.0
			wRating.Losses++
			bRating.Wins++
		case "1/2-1/2":
			scoreA = 0.5
			wRating.Draws++
			bRating.Draws++
		}

		if ag.Engine.Outcome != chess.OutcomeAborted {
			newW, newB, diffW, diffB := rating.CalculateElo(wRating.Rating, bRating.Rating, scoreA, rating.DefaultKFactor)
			wRating.Rating = newW
			bRating.Rating = newB
			wRating.Games++
			bRating.Games++
			whiteDiff = diffW
			blackDiff = diffB

			wRating.Username = ag.White.Username
			bRating.Username = ag.Black.Username

			_ = s.ratingRepo.UpsertRating(ctx, wRating)
			_ = s.ratingRepo.UpsertRating(ctx, bRating)
		}
	}

	doc := &GameDocument{
		ID:              ag.ID,
		White:           ag.White,
		Black:           ag.Black,
		Variant:         "standard",
		TimeControl:     ag.TimeControl,
		Status:          ag.Status,
		Result:          ag.Engine.Result(),
		Outcome:         string(ag.Engine.Outcome),
		InitialFEN:      chess.ToFEN(ag.Engine.InitialPosition),
		FinalFEN:        chess.ToFEN(ag.Engine.CurrentPosition),
		Moves:           ag.Moves,
		PGN:             pgnStr,
		WhiteRatingDiff: whiteDiff,
		BlackRatingDiff: blackDiff,
		StartedAt:       ag.StartedAt,
		FinishedAt:      ag.FinishedAt,
	}

	if s.gameRepo != nil {
		_ = s.gameRepo.Save(ctx, doc)
	}

	if s.rdb != nil {
		_ = s.rdb.Del(ctx, fmt.Sprintf("active_game:%s", ag.ID)).Err()
	}

	if s.onGameFinish != nil {
		s.onGameFinish(doc)
	}

	// Schedule memory eviction of finished game after a 5-minute grace period
	// so finished rooms have time to wrap up without unbounded memory growth
	go func(gameID string) {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		delete(s.activeGames, gameID)
		count := int64(len(s.activeGames))
		s.mu.Unlock()
		observability.DefaultMetrics.SetActiveGames(count)
	}(ag.ID)

	return doc
}

func (s *Service) startTimeoutWatcher() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAllTimeouts()
		}
	}
}

func (s *Service) checkAllTimeouts() {
	s.mu.RLock()
	games := make([]*ActiveGame, 0, len(s.activeGames))
	for _, ag := range s.activeGames {
		games = append(games, ag)
	}
	s.mu.RUnlock()

	if len(games) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, ag := range games {
		ag.Lock()
		if ag.Status == StatusPlaying {
			_, _, timedOut := ag.Clock.CurrentTimes(ag.Status)
			if timedOut != nil {
				ag.Status = StatusFinished
				_ = ag.Engine.Timeout(*timedOut)
				_ = s.finalizeGame(ctx, ag)
			}
		}
		ag.Unlock()
	}
}
