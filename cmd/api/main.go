package main

import (
	"fmt"
	"log/slog"
	"os"

	_ "news-aggregator/docs"
	"news-aggregator/internal/ai"
	deliveryHttp "news-aggregator/internal/delivery/http"
	"news-aggregator/internal/middleware"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"news-aggregator/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title           Concurrent AI News Aggregator API
// @version         1.0
// @description     High-performance Concurrent AI News Aggregator backend in Go with web scraping, parallel LLM summarization, and strict guardrail News Q&A.
// @termsOfService  https://github.com/Lesmana24/Concurrent-AI-News-Aggregator

// @contact.name    API Support
// @contact.url     https://github.com/Lesmana24/Concurrent-AI-News-Aggregator

// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT

// @host            localhost:8080
// @BasePath        /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-KEY
// @description Personal API key for authentication and quota management
func main() {
	// Initialize Structured JSON Logger (slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load .env
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found, using system environment variables")
	}

	// DB Connection
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// AutoMigrate
	err = db.AutoMigrate(&models.User{}, &models.Article{})
	if err != nil {
		slog.Error("Database migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize Repository
	repo := repository.NewPostgresRepository(db)

	// Initialize AI Service (Groq -> NVIDIA NIM -> Google AI Studio -> Local-Fallback)
	aiService := ai.NewDefaultAIService()

	// Initialize Service
	svc := service.NewNewsService(repo, aiService)

	// Initialize Handler with DB reference for live health checks
	handler := deliveryHttp.NewNewsHandler(svc, db)

	// Gin Engine with Structured Logging & Request-ID Middleware
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestIDAndLogger())

	// Swagger Interactive Documentation UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.Health)

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.APIKeyAuth(svc))
		{
			protected.POST("/summarize", handler.Summarize)
			protected.POST("/qa", handler.QA)
			protected.POST("/news/qa", handler.QA)
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
