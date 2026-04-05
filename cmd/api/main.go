package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ckyong826/swapngo-backend/pkg/database"
	"github.com/ckyong826/swapngo-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/ckyong826/swapngo-backend/internal/models"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// initialize database connection: defaults match docker-compose.yml (Postgres published on host 5433).
	db, err := database.InitDB(
		getenv("DB_HOST", "localhost"),
		getenv("DB_USER", "root"),
		getenv("DB_PASSWORD", "secretpassword"),
		getenv("DB_NAME", "swapngo"),
		getenv("DB_PORT", "5433"),
	)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// auto migrate models
	err = db.AutoMigrate(&models.User{}, &models.Wallet{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed successfully!")

	// initialize gin engine
	router := gin.Default()

	// test our unified response package
	router.GET("/health", func(c *gin.Context) {
		// simulate some status data from database
		serverStatus := map[string]string{
			"db_status": "connected",
			"version":   "1.0.0",
		}
		
		// return data gracefully using pkg/response
		response.Success(c, serverStatus)
	})

	router.GET("/error-test", func(c *gin.Context) {
		// simulate an error return
		response.Error(c, http.StatusBadRequest, "Invalid request parameters")
	})

	// start server
	log.Println("Server is starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}