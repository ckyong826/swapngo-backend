package routes

import (
	"swapngo-backend/internal/handlers"
	requests "swapngo-backend/pkg/requests/transfer"
	"swapngo-backend/pkg/utils"

	"swapngo-backend/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func TransferRoutes(router *gin.RouterGroup, TransferHandler handlers.TransferHandler) {
	privateTransfer := router.Group("/private/transfer")
	privateTransfer.Use(middlewares.AuthMiddleware())
	{
		privateTransfer.POST("/initiate", utils.Handle[requests.InitiateTransferReq]("Transfer initiated successfully", TransferHandler.TransferMYRC))
		privateTransfer.GET("/all", utils.Handle[struct{}]("Fetched all transfers successfully", TransferHandler.ViewAllTransfers))
		privateTransfer.GET("/:id", utils.Handle[struct{}]("Fetched transfer successfully", TransferHandler.ViewTransfer))
	}

}
