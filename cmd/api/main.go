package main

import (
	"log"
	"os"

	"swapngo-backend/internal/bizs"
	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/handlers"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/routes"
	"swapngo-backend/internal/services"
	"swapngo-backend/pkg/database"
	"swapngo-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// 0. Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables or defaults")
	}

	// 1. Initialize Database
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

	// Auto Migrate
	err = db.AutoMigrate(&models.User{}, &models.Account{}, &models.Wallet{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed successfully!")

	// 2. Initialize Repositories
	userRepo := repositories.NewUserRepository(db)
	accountRepo := repositories.NewAccountRepository(db)
	walletRepo := repositories.NewWalletRepository(db)

	// 3. Initialize Clients
	walletClient := clients.NewWalletClient()

	// 4. Initialize Services
	userService := services.NewUserService(userRepo)
	accountService := services.NewAccountService(accountRepo)
	walletService := services.NewWalletService(walletRepo, walletClient)

	// 5. Initialize Biz
	authBiz := bizs.NewAuthBiz(db, userService, accountService, walletService)

	// 6. Initialize Handlers
	authHandler := handlers.NewAuthHandler(authBiz)

	// 7. Setup Router
	router := gin.Default()
	
	// Add health test
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, map[string]string{"status": "ok"})
	})
	
	// Register all routes
	routes.SetupRouter(router, authHandler)

	// 8. Start server
	log.Println("Server is starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}