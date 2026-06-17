package routes

import (
	"swapngo-backend/internal/handlers"
	"swapngo-backend/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter(engine *gin.Engine,
	authHandler handlers.AuthHandler,
	priceHandler handlers.PriceHandler,
	walletHandler handlers.WalletHandler,
	depositHandler handlers.DepositHandler,
	transferHandler handlers.TransferHandler,
	withdrawHandler handlers.WithdrawHandler,
	swapHandler handlers.SwapHandler,
	kycHandler handlers.KYCHandler,
	cryptoHandler handlers.CryptoHandler,
) {
	api := engine.Group("/api/v1")
	{
		AuthRoutes(api, authHandler)
		WalletRoutes(api, walletHandler)
		DepositRoutes(api, depositHandler)
		TransferRoutes(api, transferHandler)
		WithdrawRoutes(api, withdrawHandler)
		SwapRoutes(api, swapHandler)
		KYCRoutes(api, kycHandler)
		CryptoRoutes(api, cryptoHandler)
	}

	wsGroup := engine.Group("/ws")
	wsGroup.Use(middlewares.AuthMiddleware())
	{
		wsGroup.GET("/prices", priceHandler.HandleWS)
	}
}
