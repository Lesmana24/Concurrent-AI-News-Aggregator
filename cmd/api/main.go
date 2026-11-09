package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "news-aggregator/docs"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/cache"
	"news-aggregator/internal/clustering"
	"news-aggregator/internal/crawler"
	deliveryHttp "news-aggregator/internal/delivery/http"
	"news-aggregator/internal/middleware"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"news-aggregator/internal/scheduler"
	"news-aggregator/internal/service"
	"news-aggregator/internal/telegram"
	"news-aggregator/web"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title           Concurrent AI News Aggregator API
// @version         1.0
// @description     High-performance production API in Go that concurrently summarizes news articles with real-time web scraping, multi-provider AI resilience (Groq/NVIDIA/Gemini), Server-Sent Events (SSE) streaming Q&A, pgvector semantic search, and media bias comparative analysis.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-KEY
func main() {
	// Initialize Structured Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load Environment Variables
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, reading from system environment variables")
	}

	// Database Connection
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Configure Database Connection Pool
	poolCfg := repository.DBPoolConfigFromEnv()
	if err := repository.ConfigureDBPool(db, poolCfg); err != nil {
		slog.Error("Database connection pool configuration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Auto-Migrate Core Schemas
	if err := db.AutoMigrate(
		&models.User{},
		&models.Article{},
		&models.TelegramSubscription{},
		&models.TopicCluster{},
	); err != nil {
		slog.Error("Database schema migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize Postgres Repository
	repo := repository.NewPostgresRepository(db)

	// Initialize L1 Redis Distributed Cache with Graceful Fallback
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPass := os.Getenv("REDIS_PASSWORD")
	redisCache := cache.NewRedisCache(redisHost, redisPort, redisPass)
	defer redisCache.Close()

	// Initialize AI Service (Groq -> NVIDIA NIM -> Google AI Studio -> Local-Fallback)
	aiService := ai.NewDefaultAIService()
	embeddingSvc := ai.NewDefaultEmbeddingService()

	// Initialize Service with L1 Cache and Embedding
	svc := service.NewNewsService(repo, aiService).WithCache(redisCache).WithEmbedding(embeddingSvc)

	// Root context with graceful termination signal listener (SIGINT / SIGTERM)
	appCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	// Initialize Autonomous News Harvester Engine
	crawlerEnabled := strings.EqualFold(os.Getenv("CRAWLER_ENABLED"), "true")
	crawlerInterval, _ := strconv.Atoi(os.Getenv("CRAWLER_INTERVAL_MINUTES"))
	if crawlerInterval <= 0 {
		crawlerInterval = 15
	}
	crawlerMax, _ := strconv.Atoi(os.Getenv("CRAWLER_MAX_PER_FEED"))
	if crawlerMax <= 0 {
		crawlerMax = 3
	}

	harvester := crawler.NewHarvesterEngine(crawler.Config{
		Interval:   time.Duration(crawlerInterval) * time.Minute,
		MaxPerFeed: crawlerMax,
		Timeout:    30 * time.Second,
		Enabled:    crawlerEnabled,
		Sources:    crawler.DefaultSources(),
	}, svc, repo)

	if crawlerEnabled {
		go harvester.Start(appCtx)
	}

	// Initialize Telegram Bot Service (Two-Way Interactive Chat & Push Notifications)
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramEnabled := os.Getenv("TELEGRAM_BOT_ENABLED")
	var telegramBot *telegram.BotService
	if strings.EqualFold(telegramEnabled, "true") && telegramToken != "" {
		for attempt := 1; attempt <= 3; attempt++ {
			bot, err := telegram.NewBotService(telegramToken, svc, repo)
			if err == nil && bot.IsEnabled() {
				telegramBot = bot
				telegramBot.WithCrawler(harvester)
				slog.Info("Telegram Bot successfully authorized", slog.Int("attempt", attempt))
				break
			}
			slog.Warn("Telegram Bot auth attempt failed, retrying...", slog.Int("attempt", attempt), slog.Any("error", err))
			time.Sleep(1 * time.Second)
		}
	}

	// Initialize Daily Briefing & Push Notification Scheduler
	briefingEnabled := strings.EqualFold(os.Getenv("BRIEFING_ENABLED"), "true")
	briefingInterval, _ := strconv.Atoi(os.Getenv("BRIEFING_INTERVAL_MINUTES"))
	briefingHoursEnv := os.Getenv("BRIEFING_HOURS")
	var briefingHours []string
	if briefingHoursEnv != "" {
		for _, h := range strings.Split(briefingHoursEnv, ",") {
			trimmed := strings.TrimSpace(h)
			if trimmed != "" {
				briefingHours = append(briefingHours, trimmed)
			}
		}
	}
	if len(briefingHours) == 0 {
		briefingHours = []string{"07:00", "17:00"}
	}

	rateLimitPerSec := 25
	if rLimit, err := strconv.Atoi(os.Getenv("TELEGRAM_RATE_LIMIT_PER_SEC")); err == nil && rLimit > 0 && rLimit <= 30 {
		rateLimitPerSec = rLimit
	} else if rLimit, err := strconv.Atoi(os.Getenv("BROADCAST_RATE_LIMIT_PER_SEC")); err == nil && rLimit > 0 && rLimit <= 30 {
		rateLimitPerSec = rLimit
	}

	briefingTZ := strings.TrimSpace(os.Getenv("BRIEFING_TIMEZONE"))
	if briefingTZ == "" {
		briefingTZ = strings.TrimSpace(os.Getenv("TIMEZONE"))
	}
	if briefingTZ == "" {
		briefingTZ = "Asia/Jakarta"
	}

	briefingCfg := scheduler.Config{
		Enabled:                  briefingEnabled,
		DailyHours:               briefingHours,
		IntervalMinutes:          briefingInterval,
		MaxArticlesPerDigest:     5,
		BroadcastRateLimitPerSec: rateLimitPerSec,
		Timezone:                 briefingTZ,
	}

	briefingEngine := scheduler.NewBriefingEngine(briefingCfg, repo, aiService, telegramBot)
	if briefingEnabled {
		go briefingEngine.Start(appCtx)
	}

	// Initialize Topic Clustering & Trending Topics Radar Engine
	clusterCfg := clustering.DefaultConfig()
	if th, err := strconv.ParseFloat(os.Getenv("CLUSTER_SIMILARITY_THRESHOLD"), 32); err == nil && th > 0 {
		clusterCfg.SimilarityThreshold = float32(th)
	}
	if win, _ := strconv.Atoi(os.Getenv("CLUSTER_HOURS_WINDOW")); win > 0 {
		clusterCfg.HoursWindow = win
	}
	if interval, _ := strconv.Atoi(os.Getenv("CLUSTER_INTERVAL_MINUTES")); interval > 0 {
		clusterCfg.RecomputeIntervalMinutes = interval
	}
	if workers, _ := strconv.Atoi(os.Getenv("CLUSTER_SYNTHESIS_WORKERS")); workers > 0 {
		clusterCfg.SynthesisWorkers = workers
	}

	clusterEngine := clustering.NewClusterEngine(clusterCfg, repo, aiService, embeddingSvc)
	go clusterEngine.Start(appCtx)

	if telegramBot != nil && telegramBot.IsEnabled() {
		telegramBot.WithScheduler(briefingEngine)
		telegramBot.WithClustering(clusterEngine)
		go telegramBot.Start(appCtx)
	}

	// Initialize Handler with DB, Redis, Crawler, Scheduler, Clustering, and Repo references
	handler := deliveryHttp.NewNewsHandler(svc, db).
		WithCache(redisCache).
		WithCrawler(harvester).
		WithScheduler(briefingEngine).
		WithClustering(clusterEngine).
		WithRepository(repo)

	// Gin Engine with Structured Logging, Request-ID, CORS, and Body Limit Middleware
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.MaxBodySize(2 * 1024 * 1024)) // 2 MB limit for API request payloads (anti-DoS)
	r.Use(middleware.CORS())
	r.Use(middleware.RequestIDAndLogger())

	// Distributed Rate Limiting (Default: 60 requests per minute)
	rpm, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_RPM"))
	r.Use(middleware.RateLimit(redisCache, rpm))

	// Swagger Interactive Documentation UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.Health)
		api.POST("/auth/register", handler.RegisterUser)
		api.GET("/news/search", handler.SemanticSearch)
		api.GET("/news/trending", handler.GetTrending)
		api.GET("/news/clusters/:id", handler.GetClusterDetail)
		api.GET("/news/recent", handler.GetRecentArticles)
		api.GET("/crawler/status", handler.GetCrawlerStatus)

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.APIKeyAuth(svc))
		{
			protected.GET("/user/me", handler.GetMe)
			protected.POST("/user/rotate-key", handler.RotateAPIKey)
			protected.POST("/summarize", handler.Summarize)
			protected.POST("/qa", handler.QA)
			protected.POST("/news/qa", handler.QA)
			protected.POST("/qa/stream", handler.QAStream)
			protected.POST("/news/qa/stream", handler.QAStream)
			protected.POST("/qa/global", handler.GlobalQA)
			protected.POST("/news/qa/global", handler.GlobalQA)
			protected.POST("/news/compare", handler.Compare)
			protected.POST("/compare", handler.Compare)
			protected.POST("/news/clusters/recompute", handler.RecomputeClusters)
			protected.POST("/crawler/trigger", handler.TriggerCrawler)
			protected.POST("/notifications/broadcast", handler.TriggerBroadcast)
			protected.GET("/notifications/subscriptions", handler.GetSubscriptions)
		}
	}

	// Single-Binary Web UI Serving (Static Assets & SPA Fallback)
	distSubFS, err := fs.Sub(web.DistFS, "dist")
	if err == nil {
		fileServer := http.FileServer(http.FS(distSubFS))
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/swagger") {
				c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
				return
			}

			// Try to open static file
			trimmedPath := strings.TrimPrefix(path, "/")
			if trimmedPath == "" {
				trimmedPath = "index.html"
			}
			if f, openErr := distSubFS.Open(trimmedPath); openErr == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}

			// SPA Fallback
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start HTTP server in a non-blocking background goroutine
	go func() {
		slog.Info("Server starting...",
			slog.String("port", port),
			slog.String("swagger_docs", "http://localhost:"+port+"/swagger/index.html"),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed to listen", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Block until an OS termination signal is captured (SIGINT / SIGTERM)
	<-appCtx.Done()
	stopSignals() // Restore standard OS signal handling so subsequent Ctrl+C forces exit
	slog.Info("Termination signal received, initiating graceful shutdown...")

	// Configurable timeout context for draining in-flight requests (Default: 15 seconds)
	shutdownTimeout := 15 * time.Second
	if val := os.Getenv("SHUTDOWN_TIMEOUT_SECONDS"); val != "" {
		if sec, err := strconv.Atoi(val); err == nil && sec > 0 {
			shutdownTimeout = time.Duration(sec) * time.Second
		}
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown forced due to timeout", slog.String("error", err.Error()))
	} else {
		slog.Info("HTTP server gracefully stopped, all in-flight requests drained")
	}

	// Close database connection pool cleanly
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
		slog.Info("Database connection pool cleanly closed")
	}

	slog.Info("Server shutdown complete, process exiting safely")
}
