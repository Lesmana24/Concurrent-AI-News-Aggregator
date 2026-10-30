package main

import (
	"fmt"
	"log"
	"news-aggregator/internal/models"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	host := os.Getenv("DB_HOST")
	if host == "localhost" {
		host = "127.0.0.1"
	}
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable connect_timeout=5",
		host, user, password, dbname, port)

	fmt.Printf("Connecting to PostgreSQL at %s:%s (database: %s)...\n", host, port, dbname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		// If target database doesn't exist yet, connect to 'postgres' to create it
		rootDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable connect_timeout=5",
			host, user, password, port)
		rootDB, rootErr := gorm.Open(postgres.Open(rootDSN), &gorm.Config{})
		if rootErr != nil {
			log.Fatalf("Cannot connect to PostgreSQL server: %v", rootErr)
		}
		fmt.Printf("Creating database '%s'...\n", dbname)
		rootDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("Still cannot connect to '%s': %v", dbname, err)
		}
	}

	fmt.Println("Connected successfully to PostgreSQL!")

	// AutoMigrate tables
	if err := db.AutoMigrate(&models.User{}, &models.Article{}); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}
	fmt.Println("AutoMigrate completed: tables 'users' and 'articles' are ready.")

	// Seed test user as specified in Techspec 5.1
	testKey := "test-key-123"
	var existing models.User
	result := db.Where("api_key = ?", testKey).First(&existing)

	if result.Error != nil {
		newUser := models.User{
			Email:  "developer@example.com",
			APIKey: testKey,
			Quota:  50,
		}
		if err := db.Create(&newUser).Error; err != nil {
			log.Fatalf("Failed to create seed user: %v", err)
		}
		fmt.Printf("Seed user created successfully!\n  - ID: %d\n  - Email: %s\n  - APIKey: %s\n  - Quota: %d\n", newUser.ID, newUser.Email, newUser.APIKey, newUser.Quota)
	} else {
		existing.Quota = 50
		db.Save(&existing)
		fmt.Printf("Seed user already exists (Quota refreshed to 50):\n  - ID: %d\n  - Email: %s\n  - APIKey: %s\n  - Quota: %d\n", existing.ID, existing.Email, existing.APIKey, existing.Quota)
	}

	var count int64
	db.Model(&models.User{}).Count(&count)
	fmt.Printf("Total users in database: %d\n", count)
}
