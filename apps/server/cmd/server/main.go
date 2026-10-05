package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/game"
	internalHttp "chess-platform/server/internal/http"
	"chess-platform/server/internal/matchmaking"
	"chess-platform/server/internal/puzzle"
	"chess-platform/server/internal/rating"
	"chess-platform/server/internal/review"
	"chess-platform/server/internal/user"
	"chess-platform/server/internal/websocket"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	log.Println("Starting Antigravity Chess Platform Server...")

	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017")
	mongoDBName := getEnv("MONGODB_DATABASE", "chess")
	redisURL := getEnv("REDIS_URL", "localhost:6379")
	serverPort := getEnv("SERVER_PORT", "8080")
	corsOrigin := getEnv("CORS_ORIGIN", "http://localhost:5173")
	jwtAccessSecret := getEnv("JWT_ACCESS_SECRET", "super-secret-access-key-production-strength-12345")
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "super-secret-refresh-key-production-strength-67890")

	// 1. Connect MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		_ = mongoClient.Disconnect(context.Background())
	}()

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}
	log.Println("Connected to MongoDB successfully.")
	db := mongoClient.Database(mongoDBName)

	// 2. Connect Redis
	var rdb *redis.Client
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		// Try as plain host:port
		rdb = redis.NewClient(&redis.Options{
			Addr: redisURL,
		})
	} else {
		rdb = redis.NewClient(opt)
	}
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Redis ping failed (%v), continuing with local fallback...", err)
	} else {
		log.Println("Connected to Redis successfully.")
	}

	// 3. Initialize Repositories
	userRepo := user.NewMongoRepository(db)
	_ = userRepo.EnsureIndexes(context.Background())

	gameRepo := game.NewMongoRepository(db)
	_ = gameRepo.EnsureIndexes(context.Background())

	ratingRepo := rating.NewMongoRepository(db)
	_ = ratingRepo.EnsureIndexes(context.Background())

	puzzleRepo := puzzle.NewMongoRepository(db)
	_ = puzzleRepo.SeedInitialPuzzles(context.Background())

	// 4. Initialize Domain Services
	authService := auth.NewService(userRepo, jwtAccessSecret, jwtRefreshSecret)
	gameService := game.NewService(gameRepo, ratingRepo)
	mmService := matchmaking.NewService(rdb, gameService)
	wsHub := websocket.NewHub(gameService, authService)

	// Forward matchmaking events to websocket hub
	mmService.SetOnMatched(func(res matchmaking.MatchResult) {
		log.Printf("Match found: Game %s (%s vs %s)", res.GameID, res.White.Username, res.Black.Username)
	})

	// 5. Initialize Engine & Review Services
	engineService := engine.NewStockfishEngine()
	reviewService := review.NewReviewService(engineService)

	// 6. Initialize Router
	router := internalHttp.NewRouter(
		authService,
		userRepo,
		gameService,
		gameRepo,
		ratingRepo,
		mmService,
		puzzleRepo,
		engineService,
		reviewService,
		wsHub,
		corsOrigin,
	)

	server := &http.Server{
		Addr:         ":" + serverPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Chess Server listening on port %s (CORS Origin: %s)", serverPort, corsOrigin)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-shutdownChan
	log.Println("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server gracefully stopped.")
}
