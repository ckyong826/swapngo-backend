package main

import (
	"log"
	"os"

	"swapngo-backend/internal/bizs"
	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/fsm"
	"swapngo-backend/internal/handlers"
	"swapngo-backend/internal/kafka"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/routes"
	"swapngo-backend/internal/services"
	"swapngo-backend/internal/ws"
	config "swapngo-backend/pkg/configs"
	"swapngo-backend/pkg/database"
	"swapngo-backend/pkg/utils"

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

	// Load config
	config.Load()

	// Initialize Kafka producer early so publish errors surface at startup
	kafka.InitProducer()

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
	err = db.AutoMigrate(&models.User{}, &models.Account{}, &models.Wallet{}, &models.Deposit{}, &models.Withdrawal{}, &models.Transfer{}, &models.Swap{}, &models.KYC{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed successfully!")

	// 2. Initialize Repositories
	kycRepo := repositories.NewKYCRepository(db)
	userRepo := repositories.NewUserRepository(db)
	accountRepo := repositories.NewAccountRepository(db)
	walletRepo := repositories.NewWalletRepository(db)
	depositRepo := repositories.NewDepositRepository(db)
	withdrawRepo := repositories.NewWithdrawRepository(db)
	transferRepo := repositories.NewTransferRepository(db)
	swapRepo := repositories.NewSwapRepository(db)

	// 3. Initialize Clients
	walletClient := clients.NewWalletClient()
	hub := ws.NewHub()
	// clients.StartPriceWorker()
	
	suiClient := chains.NewSuiClient(config.Env.SUIChainURL)
	paymentClient := clients.NewBillplzClient(
		getenv("BILLPLZ_API_URL", "https://www.billplz-sandbox.com/api/v3"),
		getenv("BILLPLZ_API_KEY", ""),
	)

	// 4. Initialize Services
	userService := services.NewUserService(userRepo)
	accountService := services.NewAccountService(accountRepo)
	walletService := services.NewWalletService(walletRepo, accountRepo, walletClient)
	tokenService := services.NewTokenService(walletRepo, swapRepo, accountRepo, suiClient)
	depositService := services.NewDepositService(depositRepo)

	// 5. Initialize Biz
	authBiz := bizs.NewAuthBiz(db, userService, accountService, walletService)
	priceBiz := bizs.NewPriceBiz(hub)
	depositFsm := fsm.BuildDepositFSM()
	depositBiz := bizs.NewDepositBiz(db, depositRepo, tokenService, accountRepo, hub, depositFsm, paymentClient, depositService)
	withdrawFsm := fsm.BuildWithdrawFSM()
	withdrawBiz := bizs.NewWithdrawBiz(db, withdrawRepo,accountRepo, tokenService,walletService,paymentClient, hub, withdrawFsm)
	transferFsm := fsm.BuildTransferFSM()
	transferBiz := bizs.NewTransferBiz(db, transferRepo, walletRepo, accountRepo, tokenService, hub, transferFsm)
	swapFsm := fsm.BuildSwapFSM()
	swapBiz := bizs.NewSwapBiz(db, swapRepo, accountRepo, tokenService, hub, swapFsm)
	kycEncryptKey := utils.DeriveKey(config.Env.KYCEncryptKey)
	kycBiz := bizs.NewKYCBiz(db, kycRepo, userRepo, kycEncryptKey)

	// 6. Initialize Handlers
	authHandler := handlers.NewAuthHandler(authBiz)
	priceHandler := handlers.NewPriceHandler(priceBiz, hub)
	walletHandler := handlers.NewWalletHandler(walletService)
	depositHandler := handlers.NewDepositHandler(depositBiz)
	withdrawHandler := handlers.NewWithdrawHandler(withdrawBiz)
	transferHandler := handlers.NewTransferHandler(transferBiz)
	swapHandler := handlers.NewSwapHandler(swapBiz)
	kycHandler := handlers.NewKYCHandler(kycBiz)

	// 7. Setup Router
	router := gin.Default()
	
	// Register all routes
	routes.SetupRouter(router, authHandler, priceHandler, walletHandler, depositHandler, transferHandler, withdrawHandler, swapHandler, kycHandler)

	// 8. Start server
	log.Println("Server is starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}