package routes

import (
	"swapngo-backend/internal/handlers"
	requests "swapngo-backend/pkg/requests/swap"
	"swapngo-backend/pkg/utils"

	"swapngo-backend/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func SwapRoutes(router *gin.RouterGroup, swapHandler handlers.SwapHandler) {
	privateSwap := router.Group("/private/swap")
	privateSwap.Use(middlewares.AuthMiddleware())
	{
		privateSwap.POST("/initiate", utils.Handle[requests.InitiateSwapReq]("Swap initiated successfully", swapHandler.InitiateExecute))
	}
}
