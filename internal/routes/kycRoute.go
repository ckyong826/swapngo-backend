package routes

import (
	"swapngo-backend/internal/handlers"
	"swapngo-backend/pkg/middlewares"
	kycReq "swapngo-backend/pkg/requests/kyc"
	"swapngo-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

func KYCRoutes(router *gin.RouterGroup, kycHandler handlers.KYCHandler) {
	// User routes — any authenticated user
	private := router.Group("/private/kyc")
	private.Use(middlewares.AuthMiddleware())
	{
		private.POST("/submit", utils.Handle[kycReq.SubmitKYCRequest]("KYC submitted successfully", kycHandler.SubmitKYC))
		private.GET("/status", utils.Handle[struct{}]("KYC status fetched successfully", kycHandler.GetKYCStatus))
	}

	// Admin routes — authenticated + ADMIN role required
	admin := router.Group("/private/admin/kyc")
	admin.Use(middlewares.AuthMiddleware(), middlewares.AdminMiddleware())
	{
		admin.GET("/pending", kycHandler.ListPendingKYC)
		admin.PUT("/:id/approve", kycHandler.ApproveKYC)
		admin.PUT("/:id/reject", kycHandler.RejectKYC)
	}
}
