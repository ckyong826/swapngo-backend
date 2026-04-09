package routes

import (
	"swapngo-backend/internal/handlers"
	"swapngo-backend/pkg/requests"
	"swapngo-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.RouterGroup, authHandler handlers.AuthHandler) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", utils.Handle[requests.RegisterRequest]("Successfully registered", authHandler.Register))
	}
}
