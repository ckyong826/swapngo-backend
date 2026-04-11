package routes

import (
	"swapngo-backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter(engine *gin.Engine, authHandler handlers.AuthHandler, priceHandler handlers.PriceHandler, walletHandler handlers.WalletHandler, depositHandler handlers.DepositHandler) {
	api := engine.Group("/api/v1")
	{
		AuthRoutes(api, authHandler)
		WalletRoutes(api, walletHandler)
		DepositRoutes(api, depositHandler)
	}

	wsGroup := engine.Group("/ws")
	{
		wsGroup.GET("/prices", priceHandler.HandleWS)
	}
}
