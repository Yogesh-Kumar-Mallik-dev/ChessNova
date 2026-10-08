package game

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"chess-platform/server/internal/chess"
	"chess-platform/server/internal/rating"

	"github.com/google/uuid"
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
	gameRepo    Repository
	ratingRepo  rating.Repository
	activeGames map[string]*ActiveGame
	mu          sync.RWMutex
	onGameFinish func(doc *GameDocument)
}

func NewService(gameRepo Repository, ratingRepo rating.Repository) *Service {
	s := &Service{
		gameRepo:    gameRepo,
		ratingRepo:  ratingRepo,
		activeGames: make(map[string]*ActiveGame),
	}
	go s.startTimeoutWatcher()
	return s
}

func (s *Service) SetOnGameFinish(cb func(doc *GameDocument)) {
	s.onGameFinish = cb
}

func (s *Service) CreateGame(white, black PlayerInfo, tc TimeControl) (*ActiveGame, error) {
	id := uuid.New().String()
	ag := NewActiveGame(id, white, black, tc)

	s.mu.Lock()
	s.activeGames[id] = ag
	s.mu.Unlock()

	return ag, nil
}

func (s *Service) GetActiveGame(id string) (*ActiveGame, error) {
	s.mu.RLock()
	ag, ok := s.activeGames[id]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrGameNotFound
	}
	return ag, nil
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

	if err := ag.Engine.Resign(color); err != nil {
		return nil, err
	}

	ag.Status = StatusFinished
	doc := s.finalizeGame(ctx, ag)
	return doc, nil
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
		wRating, _ := s.ratingRepo.GetRating(ctx, whiteOID, category)
		bRating, _ := s.ratingRepo.GetRating(ctx, blackOID, category)

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

	_ = s.gameRepo.Save(ctx, doc)

	if s.onGameFinish != nil {
		s.onGameFinish(doc)
	}

	return doc
}

func (s *Service) startTimeoutWatcher() {
	ticker := time.NewTicker(500 * time.Millisecond)
	for range ticker.C {
		s.checkAllTimeouts()
	}
}

func (s *Service) checkAllTimeouts() {
	s.mu.RLock()
	ids := make([]string, 0, len(s.activeGames))
	for id := range s.activeGames {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, id := range ids {
		ag, err := s.GetActiveGame(id)
		if err != nil {
			continue
		}

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
