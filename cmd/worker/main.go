package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"swapngo-backend/internal/bizs"
	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/fsm"
	"swapngo-backend/internal/kafka"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/services"
	"swapngo-backend/internal/ws"
	config "swapngo-backend/pkg/configs"
	"swapngo-backend/pkg/database"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// 1. Load ENVs
	config.Load()

	// 2. Init DB
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")

	db, err := database.InitDB(dbHost, dbUser, dbPass, dbName, dbPort)
	if err != nil {
		log.Fatalf("Worker DB init failed: %v", err)
	}

	// 3. Init Sui Client
	suiClient := chains.NewSuiClient(config.Env.SUIChainURL)

	// 4. Init Repositories and Services
	walletRepo := repositories.NewWalletRepository(db)
	accountRepo := repositories.NewAccountRepository(db)
	swapRepo := repositories.NewSwapRepository(db)
	depositRepo := repositories.NewDepositRepository(db)
	withdrawRepo := repositories.NewWithdrawRepository(db)
	transferRepo := repositories.NewTransferRepository(db)

	walletClient := clients.NewWalletClient()
	paymentClient := clients.NewBillplzClient(
		getenv("BILLPLZ_API_URL", "https://www.billplz-sandbox.com/api/v3"),
		getenv("BILLPLZ_API_KEY", ""),
	)

	walletService := services.NewWalletService(walletRepo, accountRepo, walletClient)
	tokenService := services.NewTokenService(walletRepo, swapRepo, accountRepo, suiClient)
	depositService := services.NewDepositService(depositRepo)

	swapFsm := fsm.BuildSwapFSM()
	depositFsm := fsm.BuildDepositFSM()
	withdrawFsm := fsm.BuildWithdrawFSM()
	transferFsm := fsm.BuildTransferFSM()

	hub := ws.NewHub() // standalone hub for the worker

	swapBiz := bizs.NewSwapBiz(db, swapRepo, accountRepo, tokenService, hub, swapFsm)
	depositBiz := bizs.NewDepositBiz(db, depositRepo, tokenService, accountRepo, hub, depositFsm, paymentClient, depositService)
	withdrawBiz := bizs.NewWithdrawBiz(db, withdrawRepo, accountRepo, tokenService, walletService, paymentClient, hub, withdrawFsm)
	transferBiz := bizs.NewTransferBiz(db, transferRepo, walletRepo, accountRepo, tokenService, hub, transferFsm)

	// 5. Init Handlers
	swapHandler := kafka.NewSwapHandler(swapBiz)
	depositHandler := kafka.NewDepositHandler(depositBiz)
	withdrawHandler := kafka.NewWithdrawHandler(withdrawBiz)
	transferHandler := kafka.NewTransferHandler(transferBiz)

	// 6. Init Kafka AntigravityWorker
	brokers := strings.Split(config.Env.KafkaBrokers, ",")
	topics := []string{"swap_events_topic", "deposit_events_topic", "withdraw_events_topic", "transfer_events_topic"}
	groupID := "swap-worker-group"

	worker := kafka.NewWorker(brokers, groupID, topics)
	
	// Ensure graceful shutdown
	defer func() {
		if err := worker.Close(); err != nil {
			log.Printf("Worker shutdown error: %v", err)
		}
	}()

	// Register Routes
	worker.Register("swap_events_topic", swapHandler.HandleSwapInitiatedEvent)
	worker.Register("deposit_events_topic", depositHandler.HandleDepositEvent)
	worker.Register("withdraw_events_topic", withdrawHandler.HandleWithdrawEvent)
	worker.Register("transfer_events_topic", transferHandler.HandleTransferEvent)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Handle SIGINT/SIGTERM
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopChan
		log.Println("\nReceived shutdown signal. Stopping worker...")
		cancel() // Signal worker.Start to stop fetching
	}()

	// Start consuming
	worker.Start(ctx)

	log.Println("Worker exited gracefully.")
}
