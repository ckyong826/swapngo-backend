package routes

import (
	"swapngo-backend/internal/handlers"

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
	) {
	api := engine.Group("/api/v1")
	{
		AuthRoutes(api, authHandler)
		WalletRoutes(api, walletHandler)
		DepositRoutes(api, depositHandler)
		TransferRoutes(api, transferHandler)
		WithdrawRoutes(api, withdrawHandler)
		SwapRoutes(api,swapHandler)
	}

	wsGroup := engine.Group("/ws")
	{
		wsGroup.GET("/prices", priceHandler.HandleWS)
	}
}
