package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"chess-platform/server/internal/auth"
	"chess-platform/server/internal/email"
	"chess-platform/server/internal/engine"
	"chess-platform/server/internal/game"
	"chess-platform/server/internal/external"
	internalHttp "chess-platform/server/internal/http"
	"chess-platform/server/internal/matchmaking"
	"chess-platform/server/internal/observability"
	"chess-platform/server/internal/puzzle"
	"chess-platform/server/internal/rating"
	"chess-platform/server/internal/review"
	"chess-platform/server/internal/storage"
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
	instanceID := getEnv("INSTANCE_ID", "instance-"+serverPort)
	observability.DefaultLogger.SetInstanceID(instanceID)
	observability.DefaultMetrics.SetInstanceID(instanceID)

	corsOrigin := getEnv("CORS_ORIGIN", "http://localhost:5173")
	jwtAccessSecret := getEnv("JWT_ACCESS_SECRET", "super-secret-access-key-production-strength-12345")
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "super-secret-refresh-key-production-strength-67890")
	resendAPIKey := getEnv("RESEND_API_KEY", "")
	smtpHost := getEnv("SMTP_HOST", "")
	smtpPort := getEnv("SMTP_PORT", "587")
	smtpUser := getEnv("SMTP_USER", "")
	smtpPass := getEnv("SMTP_PASS", "")
	smtpFrom := getEnv("SMTP_FROM", "ChessNova <noreply@chess.yogeshmallik.in>")

	observability.DefaultLogger.Info("Initializing ChessNova instance", map[string]interface{}{
		"instanceId": instanceID,
		"port":       serverPort,
	})

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
	puzzleService := puzzle.NewService(puzzleRepo)

	// 4. Initialize Domain Services
	mailer := email.NewMailer(email.Config{
		ResendAPIKey: resendAPIKey,
		Host:         smtpHost,
		Port:         smtpPort,
		User:         smtpUser,
		Password:     smtpPass,
		From:         smtpFrom,
	})
	authService := auth.NewService(userRepo, jwtAccessSecret, jwtRefreshSecret)
	authService.SetRedis(rdb)
	authService.SetMailer(mailer)
	gameService := game.NewService(gameRepo, ratingRepo)
	gameService.SetRedis(rdb)
	mmService := matchmaking.NewService(rdb, gameService)
	allowedOriginsStr := getEnv("ALLOWED_ORIGINS", corsOrigin)
	var customOrigins []string
	for _, o := range strings.Split(allowedOriginsStr, ",") {
		if tr := strings.TrimSpace(o); tr != "" {
			customOrigins = append(customOrigins, tr)
		}
	}
	wsCfg := websocket.DefaultHubConfig(customOrigins...)
	wsHub := websocket.NewHub(gameService, authService, wsCfg)
	wsHub.SetRedis(rdb, instanceID)

	// Forward matchmaking events to websocket hub
	mmService.SetOnMatched(func(res matchmaking.MatchResult) {
		observability.DefaultLogger.Info("Match found", map[string]interface{}{
			"gameId": res.GameID,
			"white":  res.White.Username,
			"black":  res.Black.Username,
		})
	})

	// 5. Initialize Engine & Review Services
	engineService := engine.NewStockfishEngine()
	reviewService := review.NewReviewService(engineService)

	// Background Game Pre-Evaluation Worker
	// Automatically analyzes finished multiplayer matches so Game Review is instant (<1ms)
	preEvalQueue := make(chan string, 100)
	go func() {
		for gameID := range preEvalQueue {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			doc, err := gameRepo.FindByID(ctx, gameID)
			if err == nil && doc != nil && !doc.IsEvaluated && len(doc.Moves) > 0 {
				var res *review.ReviewResult
				var rErr error
				if strings.TrimSpace(doc.PGN) != "" {
					res, rErr = reviewService.AnalyzePGN(ctx, doc.PGN)
				} else {
					inputMoves := make([]review.InputMove, len(doc.Moves))
					for i, m := range doc.Moves {
						inputMoves[i] = review.InputMove{
							From: m.From,
							To:   m.To,
							SAN:  m.SAN,
							FEN:  m.FEN,
						}
					}
					res, rErr = reviewService.AnalyzeMoves(ctx, inputMoves)
				}
				if rErr == nil && res != nil {
					_ = gameRepo.SaveReview(ctx, gameID, res)
					observability.DefaultLogger.Info("Background game pre-evaluation completed", map[string]interface{}{
						"gameId":        gameID,
						"whiteAccuracy": res.WhiteAccuracy,
						"blackAccuracy": res.BlackAccuracy,
					})
				}
			}
			cancel()
		}
	}()

	// Register background pre-evaluation hook when any game finishes
	gameService.AddOnGameFinish(func(doc *game.GameDocument) {
		if doc != nil && len(doc.Moves) > 0 {
			select {
			case preEvalQueue <- doc.ID:
			default:
				// Queue full under extreme load, will be evaluated on-demand if requested
			}
		}
	})

	// 6. Initialize Storage & External Platform Services
	storageService := storage.NewStorageService(storage.Config{
		Endpoint:         getEnv("S3_ENDPOINT", ""),
		Bucket:           getEnv("S3_BUCKET", "chessnova-avatars"),
		AccessKey:        getEnv("S3_ACCESS_KEY", ""),
		SecretKey:        getEnv("S3_SECRET_KEY", ""),
		Region:           getEnv("S3_REGION", "us-east-1"),
		UseSSL:           getEnv("S3_USE_SSL", "true") == "true",
		PublicURL:        getEnv("S3_PUBLIC_URL", ""),
		LocalFallbackDir: getEnv("LOCAL_UPLOAD_DIR", "uploads/avatars"),
	})
	externalService := external.NewService()

	// 7. Initialize Router
	router := internalHttp.NewRouter(
		authService,
		userRepo,
		gameService,
		gameRepo,
		ratingRepo,
		mmService,
		puzzleService,
		engineService,
		reviewService,
		storageService,
		externalService,
		wsHub,
		corsOrigin,
	)
	adminSecret := getEnv("ADMIN_SECRET", jwtAccessSecret)
	router.SetAdminSecret(adminSecret)

	loggedHandler := observability.HTTPLoggingMiddleware(observability.DefaultLogger, observability.DefaultMetrics, router)

	server := &http.Server{
		Addr:         ":" + serverPort,
		Handler:      loggedHandler,
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

	gameService.Close()

	log.Println("Server gracefully stopped.")
}
