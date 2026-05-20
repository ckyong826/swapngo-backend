package routes

import (
	"swapngo-backend/internal/handlers"
	requests "swapngo-backend/pkg/requests/deposit"
	"swapngo-backend/pkg/utils"

	"swapngo-backend/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func DepositRoutes(router *gin.RouterGroup, depositHandler handlers.DepositHandler) {
	privateDeposit := router.Group("/private/deposit")
	privateDeposit.Use(middlewares.AuthMiddleware())
	{
		privateDeposit.POST("/initiate", utils.Handle[requests.InitiateDepositReq]("Deposit initiated successfully", depositHandler.DepositMYRC))
		privateDeposit.GET("", utils.Handle[struct{}]("Fetched all deposits successfully", depositHandler.ViewAllDeposits))
		privateDeposit.GET("/:id", utils.Handle[struct{}]("Fetched deposit successfully", depositHandler.ViewDeposit))
		privateDeposit.POST("/:id/simulate-paid", utils.Handle[struct{}]("Payment simulated", depositHandler.SimulatePayment))
	}

	publicDeposit := router.Group("/public/deposit")
	{
		publicDeposit.POST("/webhook", depositHandler.HandleWebhook)
	}
}
