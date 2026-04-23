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
		privateWithdraw.GET("/all", utils.Handle[struct{}]("Fetched all withdrawals successfully", withdrawHandler.ViewAllWithdraws))
		privateWithdraw.GET("/:id", utils.Handle[struct{}]("Fetched withdrawal successfully", withdrawHandler.ViewWithdraw))
	}

}
