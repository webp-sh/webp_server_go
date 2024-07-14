package main

import (
	"fmt"
	"log"
	"os"

	"webp_server_go/pkg/api_services/apis"
	"webp_server_go/pkg/api_services/database"
	"webp_server_go/pkg/api_services/models"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load() // load .env file
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	allowedKeys := os.Getenv("X_API_KEY")

	// Initialize database connection
	db, err := gorm.Open(sqlite.Open(os.Getenv("DB_PATH")), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Run auto migrations
	err = db.AutoMigrate(&models.Sites{})
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	app := fiber.New() // Create a new Fiber instance

	// route setup
	apis.SetupRoutes(app, &database.Database{DB: db}, allowedKeys)

	// Start the server
	fmt.Println("Server is running on http://localhost:3355")

	log.Fatal(app.Listen(":3355"))
}
