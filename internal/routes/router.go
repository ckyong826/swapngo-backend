package routes

import (
	"swapngo-backend/internal/handlers"
	"swapngo-backend/pkg/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter(engine *gin.Engine, authHandler handlers.AuthHandler) {
	api := engine.Group("/api/v1")
	public := api.Group("/public")
	{
		AuthRoutes(public, authHandler)
	}
	private := api.Group("/private")
	// Private route need bearer token to access
	private.Use(middlewares.AuthMiddleware())
	{
		
	}
}
