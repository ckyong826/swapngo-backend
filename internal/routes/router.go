package routes

import (
	"swapngo-backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter(engine *gin.Engine, authHandler handlers.AuthHandler) {
	api := engine.Group("/api/v1")
	{
		AuthRoutes(api, authHandler)
	}
}
