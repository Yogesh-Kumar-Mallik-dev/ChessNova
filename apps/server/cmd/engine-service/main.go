package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/review"
)

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	port := getEnv("PORT", "9090")
	secret := getEnv("ENGINE_SECRET", "chessnova-engine-secret-2026")

	eng := engine.NewStockfishEngine()
	defer eng.Close()

	revSrv := review.NewReviewService(eng)

	mux := http.NewServeMux()

	// Health check (public)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "chessnova-engine",
			"engine":  "Stockfish 19",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Auth middleware
	authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Engine-Secret")
			if token == "" {
				authHdr := r.Header.Get("Authorization")
				if strings.HasPrefix(authHdr, "Bearer ") {
					token = strings.TrimPrefix(authHdr, "Bearer ")
				}
			}
			if secret != "" && token != secret {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
			next(w, r)
		}
	}

	// POST /evaluate
	mux.HandleFunc("POST /evaluate", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FEN   string `json:"fen"`
			Depth int    `json:"depth"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Depth <= 0 || req.Depth > 16 {
			req.Depth = 10
		}
		eval, err := eng.EvaluateFen(r.Context(), req.FEN, req.Depth)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(eval)
	}))

	// POST /bot-move
	mux.HandleFunc("POST /bot-move", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FEN   string `json:"fen"`
			Level int    `json:"level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Level <= 0 {
			req.Level = 2
		}
		move, err := eng.GenerateBotMove(r.Context(), req.FEN, req.Level)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"move": move})
	}))

	// POST /review
	mux.HandleFunc("POST /review", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PGN   string             `json:"pgn"`
			Moves []review.InputMove `json:"moves"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var res *review.ReviewResult
		var err error
		if strings.TrimSpace(req.PGN) != "" {
			res, err = revSrv.AnalyzePGN(r.Context(), req.PGN)
		} else if len(req.Moves) > 0 {
			res, err = revSrv.AnalyzeMoves(r.Context(), req.Moves)
		} else {
			http.Error(w, "empty input", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 90 * time.Second,
	}

	go func() {
		log.Printf("ChessNova Remote Engine Service listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
