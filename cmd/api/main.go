package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	_ "news-aggregator/docs"
	"news-aggregator/internal/ai"
	"news-aggregator/internal/cache"
	"news-aggregator/internal/crawler"
	deliveryHttp "news-aggregator/internal/delivery/http"
	"news-aggregator/internal/middleware"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"news-aggregator/internal/service"
	"news-aggregator/internal/telegram"

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

	// Auto-Migrate Core Schemas
	if err := db.AutoMigrate(
		&models.User{},
		&models.Article{},
		&models.TelegramSubscription{},
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

	// Initialize Service with L1 Cache
	svc := service.NewNewsService(repo, aiService).WithCache(redisCache)

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
		go harvester.Start(context.Background())
	}

	// Initialize Telegram Bot Service (Two-Way Interactive Chat & Push Notifications)
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramEnabled := os.Getenv("TELEGRAM_BOT_ENABLED")
	if strings.EqualFold(telegramEnabled, "true") && telegramToken != "" {
		go func() {
			for attempt := 1; attempt <= 10; attempt++ {
				telegramBot, err := telegram.NewBotService(telegramToken, svc, repo)
				if err == nil && telegramBot.IsEnabled() {
					telegramBot.WithCrawler(harvester)
					telegramBot.Start(context.Background())
					slog.Info("Telegram Bot active and listening for messages", slog.Int("attempt", attempt))
					return
				}
				slog.Warn("Telegram Bot initialization attempt failed, retrying in 2s...",
					slog.Int("attempt", attempt),
					slog.Any("error", err),
				)
				time.Sleep(2 * time.Second)
			}
			slog.Error("Failed to start Telegram Bot after 10 attempts")
		}()
	}

	// Initialize Handler with DB and Redis references for live health checks
	handler := deliveryHttp.NewNewsHandler(svc, db).WithCache(redisCache).WithCrawler(harvester)

	// Gin Engine with Structured Logging & Request-ID Middleware
	r := gin.New()
	r.Use(gin.Recovery())
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
		api.GET("/news/search", handler.SemanticSearch)
		api.GET("/crawler/status", handler.GetCrawlerStatus)

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.APIKeyAuth(svc))
		{
			protected.POST("/summarize", handler.Summarize)
			protected.POST("/qa", handler.QA)
			protected.POST("/news/qa", handler.QA)
			protected.POST("/qa/stream", handler.QAStream)
			protected.POST("/news/qa/stream", handler.QAStream)
			protected.POST("/qa/global", handler.GlobalQA)
			protected.POST("/news/qa/global", handler.GlobalQA)
			protected.POST("/news/compare", handler.Compare)
			protected.POST("/compare", handler.Compare)
			protected.POST("/crawler/trigger", handler.TriggerCrawler)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Server starting...", slog.String("port", port), slog.String("swagger_docs", "http://localhost:"+port+"/swagger/index.html"))
	if err := r.Run(":" + port); err != nil {
		slog.Error("Server failed to run", slog.String("error", err.Error()))
	}
}
