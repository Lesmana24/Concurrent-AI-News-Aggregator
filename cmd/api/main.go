package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"news-aggregator/internal/ai"
	deliveryHttp "news-aggregator/internal/delivery/http"
	"news-aggregator/internal/middleware"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"news-aggregator/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// DB Connection
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// AutoMigrate
	err = db.AutoMigrate(&models.User{}, &models.Article{})
	if err != nil {
		log.Fatal("Migration failed: ", err)
	}

	// Initialize Repository
	repo := repository.NewPostgresRepository(db)

	// Initialize AI Service (Groq -> NVIDIA NIM -> Google AI Studio -> Local-Fallback)
	aiService := ai.NewDefaultAIService()

	// Initialize Service
	svc := service.NewNewsService(repo, aiService)

	// Initialize Handler
	handler := deliveryHttp.NewNewsHandler(svc)

	// Gin Server
	r := gin.Default()

	// Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

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

	log.Printf("Server starting on port %s...", port)
	r.Run(":" + port)
}
