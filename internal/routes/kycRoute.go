package routes

import (
	"swapngo-backend/internal/handlers"
	"swapngo-backend/pkg/middlewares"
	kycReq "swapngo-backend/pkg/requests/kyc"
	"swapngo-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

func KYCRoutes(router *gin.RouterGroup, kycHandler handlers.KYCHandler) {
	private := router.Group("/private/kyc")
	private.Use(middlewares.AuthMiddleware())
	{
		private.POST("/submit", utils.Handle[kycReq.SubmitKYCRequest]("KYC submitted successfully", kycHandler.SubmitKYC))
		private.GET("/status", utils.Handle[struct{}]("KYC status fetched successfully", kycHandler.GetKYCStatus))
	}
}
