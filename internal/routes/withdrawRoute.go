package routes

import (
	"swapngo-backend/internal/handlers"
	requests "swapngo-backend/pkg/requests/withdrawal"
	"swapngo-backend/pkg/utils"

	"swapngo-backend/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func WithdrawRoutes(router *gin.RouterGroup, withdrawHandler handlers.WithdrawHandler) {
	privateWithdraw := router.Group("/private/withdraw")
	privateWithdraw.Use(middlewares.AuthMiddleware())
	{
		privateWithdraw.POST("/initiate", utils.Handle[requests.InitiateWithdrawReq]("Withdraw initiated successfully", withdrawHandler.WithdrawMYRC))
	}

}
