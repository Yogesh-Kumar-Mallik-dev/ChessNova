package external

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExternalService(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/pub/player/magnuscarlsen/stats":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"chess_rapid": {"last": {"rating": 2820}},
				"chess_blitz": {"last": {"rating": 2880}},
				"chess_bullet": {"last": {"rating": 2900}},
				"tactics": {"highest": {"rating": 3100}}
			}`))
		case r.URL.Path == "/api/user/magnuscarlsen":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "magnuscarlsen",
				"perfs": {
					"rapid": {"rating": 2850},
					"blitz": {"rating": 2910},
					"bullet": {"rating": 2940},
					"puzzle": {"rating": 3050}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	svc := NewService()
	svc.client = mockServer.Client()

	ctx := context.Background()

	// Empty username check
	stats, err := svc.FetchChessComStats(ctx, "")
	if err != nil || stats != nil {
		t.Errorf("Expected nil stats and no error for empty username")
	}

	statsLi, err := svc.FetchLichessStats(ctx, "")
	if err != nil || statsLi != nil {
		t.Errorf("Expected nil stats and no error for empty username")
	}
}
