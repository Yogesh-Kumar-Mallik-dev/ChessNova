package game

import (
	"sync"
	"time"

	"chess-platform/server/internal/chess"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Status string

const (
	StatusWaiting  Status = "waiting"
	StatusPlaying  Status = "playing"
	StatusFinished Status = "finished"
	StatusAborted  Status = "aborted"
)

type TimeControl struct {
	InitialSeconds int `bson:"initial" json:"initial"`
	Increment      int `bson:"increment" json:"increment"`
}

type PlayerInfo struct {
	UserID   string `bson:"userId" json:"userId"`
	Username string `bson:"username" json:"username"`
	Rating   int    `bson:"rating" json:"rating"`
}

type ClockState struct {
	WhiteTimeMs int64       `bson:"whiteTimeMs" json:"whiteTimeMs"`
	BlackTimeMs int64       `bson:"blackTimeMs" json:"blackTimeMs"`
	LastMoveAt  time.Time   `bson:"lastMoveAt" json:"lastMoveAt"`
	ActiveColor chess.Color `bson:"activeColor" json:"activeColor"`
}

func NewClock(tc TimeControl) ClockState {
	totalMs := int64(tc.InitialSeconds) * 1000
	return ClockState{
		WhiteTimeMs: totalMs,
		BlackTimeMs: totalMs,
		LastMoveAt:  time.Now().UTC(),
		ActiveColor: chess.White,
	}
}

// CurrentTimes calculates real-time remaining clock times without mutating state
func (c *ClockState) CurrentTimes(status Status) (whiteMs, blackMs int64, timedOut *chess.Color) {
	whiteMs = c.WhiteTimeMs
	blackMs = c.BlackTimeMs

	if status != StatusPlaying {
		return whiteMs, blackMs, nil
	}

	elapsed := time.Since(c.LastMoveAt).Milliseconds()
	if c.ActiveColor == chess.White {
		whiteMs -= elapsed
		if whiteMs <= 0 {
			whiteMs = 0
			col := chess.White
			return whiteMs, blackMs, &col
		}
	} else {
		blackMs -= elapsed
		if blackMs <= 0 {
			blackMs = 0
			col := chess.Black
			return whiteMs, blackMs, &col
		}
	}

	return whiteMs, blackMs, nil
}

// UpdateAfterMove calculates elapsed time, subtracts from active side, adds increment, switches active color
func (c *ClockState) UpdateAfterMove(incrementSeconds int) (timedOut bool) {
	now := time.Now().UTC()
	elapsed := now.Sub(c.LastMoveAt).Milliseconds()
	c.LastMoveAt = now

	incMs := int64(incrementSeconds) * 1000

	if c.ActiveColor == chess.White {
		c.WhiteTimeMs -= elapsed
		if c.WhiteTimeMs <= 0 {
			c.WhiteTimeMs = 0
			return true
		}
		c.WhiteTimeMs += incMs
		c.ActiveColor = chess.Black
	} else {
		c.BlackTimeMs -= elapsed
		if c.BlackTimeMs <= 0 {
			c.BlackTimeMs = 0
			return true
		}
		c.BlackTimeMs += incMs
		c.ActiveColor = chess.White
	}

	return false
}

type PersistedMove struct {
	Ply       int    `bson:"ply" json:"ply"`
	From      string `bson:"from" json:"from"`
	To        string `bson:"to" json:"to"`
	SAN       string `bson:"san" json:"san"`
	FEN       string `bson:"fen" json:"fen"`
	TimeSpent int64  `bson:"timeSpent" json:"timeSpent"`
}

type GameDocument struct {
	ID          string             `bson:"_id" json:"id"`
	White       PlayerInfo         `bson:"white" json:"white"`
	Black       PlayerInfo         `bson:"black" json:"black"`
	Variant     string             `bson:"variant" json:"variant"`
	TimeControl TimeControl        `bson:"timeControl" json:"timeControl"`
	Status      Status             `bson:"status" json:"status"`
	Result      string             `bson:"result" json:"result"`
	Outcome     string             `bson:"outcome" json:"outcome"`
	InitialFEN  string             `bson:"initialFen" json:"initialFen"`
	FinalFEN    string             `bson:"finalFen" json:"finalFen"`
	Moves       []PersistedMove    `bson:"moves" json:"moves"`
	PGN         string             `bson:"pgn" json:"pgn"`
	WhiteRatingDiff int            `bson:"whiteRatingDiff" json:"whiteRatingDiff"`
	BlackRatingDiff int            `bson:"blackRatingDiff" json:"blackRatingDiff"`
	StartedAt   time.Time          `bson:"startedAt" json:"startedAt"`
	FinishedAt  *time.Time         `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
}

// ActiveGame holds in-memory / live execution state
type ActiveGame struct {
	mu          sync.Mutex
	ID          string          `json:"id"`
	White       PlayerInfo      `json:"white"`
	Black       PlayerInfo      `json:"black"`
	Engine      *chess.Game     `json:"engine"`
	Clock       ClockState      `json:"clock"`
	TimeControl TimeControl     `json:"timeControl"`
	Status      Status          `json:"status"`
	StartedAt   time.Time       `json:"startedAt"`
	FinishedAt  *time.Time      `json:"finishedAt,omitempty"`
	Moves       []PersistedMove `json:"moves"`
}

func NewActiveGame(id string, white, black PlayerInfo, tc TimeControl) *ActiveGame {
	return &ActiveGame{
		ID:          id,
		White:       white,
		Black:       black,
		Engine:      chess.NewGame(),
		Clock:       NewClock(tc),
		TimeControl: tc,
		Status:      StatusPlaying,
		StartedAt:   time.Now().UTC(),
		Moves:       make([]PersistedMove, 0),
	}
}

func (ag *ActiveGame) Lock()   { ag.mu.Lock() }
func (ag *ActiveGame) Unlock() { ag.mu.Unlock() }

func ParsePlayerObjectID(s string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(s)
}
