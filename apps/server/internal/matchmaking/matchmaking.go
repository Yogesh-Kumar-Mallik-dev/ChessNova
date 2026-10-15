package matchmaking

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"chess-platform/server/internal/game"
	"chess-platform/server/internal/observability"

	"github.com/redis/go-redis/v9"
)

type Ticket struct {
	ID             string           `json:"id"`
	UserID         string           `json:"userId"`
	Username       string           `json:"username"`
	Rating         int              `json:"rating"`
	TimeControl    game.TimeControl `json:"timeControl"`
	CreatedAt      time.Time        `json:"createdAt"`
	MatchedGameID  string           `json:"matchedGameId,omitempty"`
	AssignedColor  string           `json:"assignedColor,omitempty"`
}

type MatchResult struct {
	GameID        string           `json:"gameId"`
	White         game.PlayerInfo  `json:"white"`
	Black         game.PlayerInfo  `json:"black"`
	TimeControl   game.TimeControl `json:"timeControl"`
}

type Service struct {
	rdb         *redis.Client
	gameService *game.Service
	mu          sync.Mutex
	localQueue  []*Ticket
	onMatched   func(result MatchResult)
}

func NewService(rdb *redis.Client, gameService *game.Service) *Service {
	s := &Service{
		rdb:         rdb,
		gameService: gameService,
		localQueue:  make([]*Ticket, 0),
	}
	go s.matchmakingLoop()
	return s
}

func (s *Service) SetOnMatched(cb func(result MatchResult)) {
	s.onMatched = cb
}

func (s *Service) JoinQueue(ctx context.Context, t *Ticket) error {
	if s.gameService != nil {
		if ag := s.gameService.GetActiveGameForUser(t.UserID); ag != nil {
			// Auto-resign / forfeit any previous abandoned or stuck match
			// so the user is never blocked by "you already have an active game in progress"
			_, _ = s.gameService.Resign(ctx, ag.ID, t.UserID)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Avoid duplicates for same user
	for _, item := range s.localQueue {
		if item.UserID == t.UserID {
			return nil
		}
	}

	t.CreatedAt = time.Now().UTC()
	s.localQueue = append(s.localQueue, t)

	// Also store in Redis if redis client is available
	if s.rdb != nil {
		data, _ := json.Marshal(t)
		key := fmt.Sprintf("mm:ticket:%s", t.UserID)
		_ = s.rdb.Set(ctx, key, data, 5*time.Minute).Err()
		_ = s.rdb.SAdd(ctx, "mm:active_users", t.UserID).Err()
	}

	observability.DefaultMetrics.SetMatchmakingQueueSize(int64(len(s.localQueue)))

	return nil
}

func (s *Service) LeaveQueue(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]*Ticket, 0, len(s.localQueue))
	for _, t := range s.localQueue {
		if t.UserID != userID {
			filtered = append(filtered, t)
		}
	}
	s.localQueue = filtered

	if s.rdb != nil {
		_ = s.rdb.Del(ctx, fmt.Sprintf("mm:ticket:%s", userID)).Err()
		_ = s.rdb.SRem(ctx, "mm:active_users", userID).Err()
	}

	observability.DefaultMetrics.SetMatchmakingQueueSize(int64(len(s.localQueue)))

	return nil
}

func (s *Service) GetStatus(ctx context.Context, userID string) (*Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.localQueue {
		if t.UserID == userID {
			return t, nil
		}
	}

	if s.rdb != nil {
		data, err := s.rdb.Get(ctx, fmt.Sprintf("mm:ticket:%s", userID)).Result()
		if err == nil {
			var t Ticket
			if json.Unmarshal([]byte(data), &t) == nil {
				return &t, nil
			}
		}
	}

	return nil, nil
}

func (s *Service) matchmakingLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		s.tryMatch()
	}
}

func (s *Service) tryMatch() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var queue []*Ticket
	ctx := context.Background()

	if s.rdb != nil {
		userIDs, err := s.rdb.SMembers(ctx, "mm:active_users").Result()
		if err == nil && len(userIDs) > 0 {
			for _, uID := range userIDs {
				data, err := s.rdb.Get(ctx, fmt.Sprintf("mm:ticket:%s", uID)).Result()
				if err == nil {
					var tk Ticket
					if json.Unmarshal([]byte(data), &tk) == nil && tk.MatchedGameID == "" {
						queue = append(queue, &tk)
					}
				}
			}
		}
	}

	if len(queue) == 0 {
		queue = s.localQueue
	}

	observability.DefaultMetrics.SetMatchmakingQueueSize(int64(len(queue)))

	if len(queue) < 2 {
		return
	}

	now := time.Now().UTC()
	matchedIndices := make(map[int]bool)

	for i := 0; i < len(queue); i++ {
		if matchedIndices[i] {
			continue
		}
		ticketA := queue[i]
		waitA := now.Sub(ticketA.CreatedAt).Seconds()
		tolA := toleranceForWait(waitA)

		for j := i + 1; j < len(queue); j++ {
			if matchedIndices[j] {
				continue
			}
			ticketB := queue[j]

			// Must have same time control
			if ticketA.TimeControl.InitialSeconds != ticketB.TimeControl.InitialSeconds ||
				ticketA.TimeControl.Increment != ticketB.TimeControl.Increment {
				continue
			}

			// Same user cannot match with themselves
			if ticketA.UserID == ticketB.UserID {
				continue
			}

			waitB := now.Sub(ticketB.CreatedAt).Seconds()
			tolB := toleranceForWait(waitB)

			ratingDiff := int(math.Abs(float64(ticketA.Rating - ticketB.Rating)))
			allowedTolerance := int(math.Max(float64(tolA), float64(tolB)))

			if ratingDiff <= allowedTolerance {
				// We have a match!
				matchedIndices[i] = true
				matchedIndices[j] = true

				if s.rdb != nil {
					_ = s.rdb.SRem(ctx, "mm:active_users", ticketA.UserID, ticketB.UserID).Err()
				}
				s.createMatchedGame(ticketA, ticketB)
				break
			}
		}
	}

	// Remove matched tickets from queue
	if len(matchedIndices) > 0 {
		matchedUserIDs := make(map[string]bool)
		for idx := range matchedIndices {
			matchedUserIDs[queue[idx].UserID] = true
		}
		remaining := make([]*Ticket, 0, len(s.localQueue))
		for _, t := range s.localQueue {
			if !matchedUserIDs[t.UserID] {
				remaining = append(remaining, t)
			}
		}
		s.localQueue = remaining
	}
}

func toleranceForWait(seconds float64) int {
	switch {
	case seconds < 10:
		return 50
	case seconds < 20:
		return 100
	case seconds < 30:
		return 200
	default:
		return 300
	}
}

func (s *Service) createMatchedGame(a, b *Ticket) {
	// Randomly assign white and black
	var whitePlayer, blackPlayer game.PlayerInfo
	var colorA, colorB string

	if rand.Intn(2) == 0 {
		whitePlayer = game.PlayerInfo{UserID: a.UserID, Username: a.Username, Rating: a.Rating}
		blackPlayer = game.PlayerInfo{UserID: b.UserID, Username: b.Username, Rating: b.Rating}
		colorA = "white"
		colorB = "black"
	} else {
		whitePlayer = game.PlayerInfo{UserID: b.UserID, Username: b.Username, Rating: b.Rating}
		blackPlayer = game.PlayerInfo{UserID: a.UserID, Username: a.Username, Rating: a.Rating}
		colorA = "black"
		colorB = "white"
	}

	ag, err := s.gameService.CreateGame(whitePlayer, blackPlayer, a.TimeControl)
	if err != nil {
		return
	}

	a.MatchedGameID = ag.ID
	a.AssignedColor = colorA
	b.MatchedGameID = ag.ID
	b.AssignedColor = colorB

	if s.rdb != nil {
		ctx := context.Background()
		dataA, _ := json.Marshal(a)
		dataB, _ := json.Marshal(b)
		_ = s.rdb.Set(ctx, fmt.Sprintf("mm:ticket:%s", a.UserID), dataA, 2*time.Minute).Err()
		_ = s.rdb.Set(ctx, fmt.Sprintf("mm:ticket:%s", b.UserID), dataB, 2*time.Minute).Err()
	}

	res := MatchResult{
		GameID:      ag.ID,
		White:       whitePlayer,
		Black:       blackPlayer,
		TimeControl: a.TimeControl,
	}

	if s.onMatched != nil {
		s.onMatched(res)
	}
}
