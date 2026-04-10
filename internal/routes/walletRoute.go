package routes

import (
	"swapngo-backend/internal/handlers"
	"swapngo-backend/pkg/utils"

	"swapngo-backend/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func WalletRoutes(router *gin.RouterGroup, walletHandler handlers.WalletHandler) {
	privateWallet := router.Group("/private/wallet")
	privateWallet.Use(middlewares.AuthMiddleware())
	{
		privateWallet.GET("/get-balance", utils.Handle[any]("Successfully registered", walletHandler.GetTotalBalanceByUserID))
	}
}
