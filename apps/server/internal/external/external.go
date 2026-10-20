package external

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

var validExternalUsername = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]{1,50}$`)

func validateExternalUsername(username string) error {
	trimmed := strings.TrimSpace(username)
	if !validExternalUsername.MatchString(trimmed) {
		return errors.New("invalid external username format")
	}
	return nil
}

func isPrivateOrLoopbackIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

type ExternalGame struct {
	Platform    string    `json:"platform"` // "chesscom" | "lichess"
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	White       string    `json:"white"`
	WhiteRating int       `json:"whiteRating"`
	Black       string    `json:"black"`
	BlackRating int       `json:"blackRating"`
	UserRating  int       `json:"userRating,omitempty"`
	Result      string    `json:"result"`
	TimeControl string    `json:"timeControl"`
	PlayedAt    time.Time `json:"playedAt"`
	PGN         string    `json:"pgn"`
}

type PlatformRatings struct {
	Bullet    int `json:"bullet,omitempty"`
	Blitz     int `json:"blitz,omitempty"`
	Rapid     int `json:"rapid,omitempty"`
	Classical int `json:"classical,omitempty"`
	Puzzle    int `json:"puzzle,omitempty"`
}

type PlatformComparison struct {
	Username    string           `json:"username"`
	ChessPasant PlatformRatings  `json:"chesspasant"`
	ChessNova   PlatformRatings  `json:"chessnova,omitempty"`
	ChessCom    *PlatformRatings `json:"chesscom,omitempty"`
	Lichess     *PlatformRatings `json:"lichess,omitempty"`
	ChessComUser string          `json:"chessComUser,omitempty"`
	LichessUser string           `json:"lichessUser,omitempty"`
}

type Service struct {
	client *http.Client
}

func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("stopped after 5 redirects")
				}
				host := req.URL.Hostname()
				if ips, err := net.LookupIP(host); err == nil {
					for _, ip := range ips {
						if isPrivateOrLoopbackIP(ip) {
							return errors.New("redirect to private or loopback address blocked")
						}
					}
				}
				return nil
			},
		},
	}
}

// FetchChessComStats fetches public rating stats from Chess.com
func (s *Service) FetchChessComStats(ctx context.Context, username string) (*PlatformRatings, error) {
	if username == "" {
		return nil, nil
	}
	if err := validateExternalUsername(username); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.chess.com/pub/player/%s/stats", strings.ToLower(username))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ChessPasant-App/1.0 (contact: support@chesspasant.com)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("chess.com user '%s' not found", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chess.com api error: %d", resp.StatusCode)
	}

	var data struct {
		ChessBullet struct {
			Last struct {
				Rating int `json:"rating"`
			} `json:"last"`
		} `json:"chess_bullet"`
		ChessBlitz struct {
			Last struct {
				Rating int `json:"rating"`
			} `json:"last"`
		} `json:"chess_blitz"`
		ChessRapid struct {
			Last struct {
				Rating int `json:"rating"`
			} `json:"last"`
		} `json:"chess_rapid"`
		ChessDaily struct {
			Last struct {
				Rating int `json:"rating"`
			} `json:"last"`
		} `json:"chess_daily"`
		Tactics struct {
			Highest struct {
				Rating int `json:"rating"`
			} `json:"highest"`
			Lowest struct {
				Rating int `json:"rating"`
			} `json:"lowest"`
		} `json:"tactics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	ratings := &PlatformRatings{
		Bullet:    data.ChessBullet.Last.Rating,
		Blitz:     data.ChessBlitz.Last.Rating,
		Rapid:     data.ChessRapid.Last.Rating,
		Classical: data.ChessDaily.Last.Rating,
		Puzzle:    data.Tactics.Highest.Rating,
	}

	return ratings, nil
}

// FetchLichessStats fetches public rating stats from Lichess
func (s *Service) FetchLichessStats(ctx context.Context, username string) (*PlatformRatings, error) {
	if username == "" {
		return nil, nil
	}
	if err := validateExternalUsername(username); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://lichess.org/api/user/%s", strings.ToLower(username))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ChessPasant-App/1.0 (contact: support@chesspasant.com)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("lichess user '%s' not found", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lichess api error: %d", resp.StatusCode)
	}

	var data struct {
		Perfs struct {
			Bullet struct {
				Rating int `json:"rating"`
			} `json:"bullet"`
			Blitz struct {
				Rating int `json:"rating"`
			} `json:"blitz"`
			Rapid struct {
				Rating int `json:"rating"`
			} `json:"rapid"`
			Classical struct {
				Rating int `json:"rating"`
			} `json:"classical"`
			Puzzle struct {
				Rating int `json:"rating"`
			} `json:"puzzle"`
		} `json:"perfs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	ratings := &PlatformRatings{
		Bullet:    data.Perfs.Bullet.Rating,
		Blitz:     data.Perfs.Blitz.Rating,
		Rapid:     data.Perfs.Rapid.Rating,
		Classical: data.Perfs.Classical.Rating,
		Puzzle:    data.Perfs.Puzzle.Rating,
	}

	return ratings, nil
}

// FetchLichessRecentGames fetches recent games from Lichess in PGN format
func (s *Service) FetchLichessRecentGames(ctx context.Context, username string, limit int) ([]ExternalGame, error) {
	if username == "" {
		return nil, nil
	}
	if err := validateExternalUsername(username); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}

	url := fmt.Sprintf("https://lichess.org/api/games/user/%s?max=%d&pgnInJson=true&clocks=false&evals=false&opening=true", strings.ToLower(username), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("User-Agent", "ChessPasant-App/1.0 (contact: support@chesspasant.com)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lichess games api returned status %d", resp.StatusCode)
	}

	var games []ExternalGame
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw struct {
			ID        string `json:"id"`
			Speed     string `json:"speed"`
			Status    string `json:"status"`
			CreatedAt int64  `json:"createdAt"`
			Winner    string `json:"winner"`
			Players   struct {
				White struct {
					User struct {
						Name string `json:"name"`
					} `json:"user"`
					Rating int `json:"rating"`
				} `json:"white"`
				Black struct {
					User struct {
						Name string `json:"name"`
					} `json:"user"`
					Rating int `json:"rating"`
				} `json:"black"`
			} `json:"players"`
			PGN string `json:"pgn"`
		}

		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		whiteName := raw.Players.White.User.Name
		if whiteName == "" {
			whiteName = "Anonymous"
		}
		blackName := raw.Players.Black.User.Name
		if blackName == "" {
			blackName = "Anonymous"
		}

		resultStr := "1/2-1/2"
		if raw.Winner == "white" {
			resultStr = "1-0"
		} else if raw.Winner == "black" {
			resultStr = "0-1"
		}

		playedAt := time.UnixMilli(raw.CreatedAt)
		if raw.CreatedAt == 0 {
			playedAt = time.Now().UTC()
		}

		userRating := 0
		if strings.EqualFold(whiteName, username) {
			userRating = raw.Players.White.Rating
		} else if strings.EqualFold(blackName, username) {
			userRating = raw.Players.Black.Rating
		}

		games = append(games, ExternalGame{
			Platform:    "lichess",
			ID:          raw.ID,
			URL:         fmt.Sprintf("https://lichess.org/%s", raw.ID),
			White:       whiteName,
			WhiteRating: raw.Players.White.Rating,
			Black:       blackName,
			BlackRating: raw.Players.Black.Rating,
			UserRating:  userRating,
			Result:      resultStr,
			TimeControl: raw.Speed,
			PlayedAt:    playedAt,
			PGN:         raw.PGN,
		})
	}

	return games, nil
}

// FetchChessComRecentGames fetches recent games from Chess.com archives across multiple months
func (s *Service) FetchChessComRecentGames(ctx context.Context, username string, limit int) ([]ExternalGame, error) {
	if username == "" {
		return nil, nil
	}
	if err := validateExternalUsername(username); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}

	// 1. Query the player's full monthly archives list
	archivesURL := fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", strings.ToLower(username))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archivesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ChessPasant-App/1.0 (contact: support@chesspasant.com)")

	resp, err := s.client.Do(req)
	var archiveURLs []string
	if err == nil && resp.StatusCode == http.StatusOK {
		var archData struct {
			Archives []string `json:"archives"`
		}
		if json.NewDecoder(resp.Body).Decode(&archData) == nil {
			archiveURLs = archData.Archives
		}
		resp.Body.Close()
	} else if resp != nil {
		resp.Body.Close()
	}

	// Fallback if archives endpoint returned empty or failed
	if len(archiveURLs) == 0 {
		now := time.Now().UTC()
		archiveURLs = []string{
			fmt.Sprintf("https://api.chess.com/pub/player/%s/games/%d/%02d", strings.ToLower(username), now.Year(), int(now.Month())),
		}
		prev := now.AddDate(0, -1, 0)
		archiveURLs = append([]string{
			fmt.Sprintf("https://api.chess.com/pub/player/%s/games/%d/%02d", strings.ToLower(username), prev.Year(), int(prev.Month())),
		}, archiveURLs...)
	}

	var allGames []ExternalGame

	// Iterate backwards from newest archive to older until limit is satisfied
	for i := len(archiveURLs) - 1; i >= 0 && len(allGames) < limit; i-- {
		u := archiveURLs[i]
		mReq, mErr := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if mErr != nil {
			continue
		}
		mReq.Header.Set("User-Agent", "ChessPasant-App/1.0 (contact: support@chesspasant.com)")

		mResp, mErr := s.client.Do(mReq)
		if mErr != nil || mResp.StatusCode != http.StatusOK {
			if mResp != nil {
				mResp.Body.Close()
			}
			continue
		}

		var data struct {
			Games []struct {
				URL         string `json:"url"`
				PGN         string `json:"pgn"`
				TimeControl string `json:"time_control"`
				EndTime     int64  `json:"end_time"`
				White       struct {
					Username string `json:"username"`
					Rating   int    `json:"rating"`
					Result   string `json:"result"`
				} `json:"white"`
				Black struct {
					Username string `json:"username"`
					Rating   int    `json:"rating"`
					Result   string `json:"result"`
				} `json:"black"`
			} `json:"games"`
		}

		bodyBytes, bErr := io.ReadAll(mResp.Body)
		mResp.Body.Close()
		if bErr != nil || json.Unmarshal(bodyBytes, &data) != nil {
			continue
		}

		for _, g := range data.Games {
			resultStr := "*"
			if g.White.Result == "win" {
				resultStr = "1-0"
			} else if g.Black.Result == "win" {
				resultStr = "0-1"
			} else {
				resultStr = "1/2-1/2"
			}

			parts := strings.Split(g.URL, "/")
			gameID := parts[len(parts)-1]

			userRating := 0
			if strings.EqualFold(g.White.Username, username) {
				userRating = g.White.Rating
			} else if strings.EqualFold(g.Black.Username, username) {
				userRating = g.Black.Rating
			}

			allGames = append(allGames, ExternalGame{
				Platform:    "chesscom",
				ID:          gameID,
				URL:         g.URL,
				White:       g.White.Username,
				WhiteRating: g.White.Rating,
				Black:       g.Black.Username,
				BlackRating: g.Black.Rating,
				UserRating:  userRating,
				Result:      resultStr,
				TimeControl: g.TimeControl,
				PlayedAt:    time.Unix(g.EndTime, 0),
				PGN:         g.PGN,
			})
		}
	}

	// Sort newest first
	sort.Slice(allGames, func(i, j int) bool {
		return allGames[i].PlayedAt.After(allGames[j].PlayedAt)
	})

	if len(allGames) > limit {
		allGames = allGames[:limit]
	}

	return allGames, nil
}
