package routes

import (
	"swapngo-backend/internal/handlers"
	requests "swapngo-backend/pkg/requests/crypto"
	"swapngo-backend/pkg/middlewares"
	"swapngo-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

func CryptoRoutes(router *gin.RouterGroup, cryptoHandler handlers.CryptoHandler) {
	privateCrypto := router.Group("/private/crypto")
	privateCrypto.Use(middlewares.AuthMiddleware())
	{
		privateCrypto.POST("/deposit/sui", utils.Handle[requests.DepositSUIReq]("SUI deposit credited", cryptoHandler.DepositSUI))
		privateCrypto.POST("/deposit/simulate", utils.Handle[requests.SimulateDepositReq]("Simulated deposit credited", cryptoHandler.SimulateDeposit))
		privateCrypto.POST("/withdraw", utils.Handle[requests.WithdrawReq]("Withdrawal processed", cryptoHandler.Withdraw))
		privateCrypto.GET("", utils.Handle[struct{}]("Fetched crypto transactions successfully", cryptoHandler.ViewAll))
	}
}
