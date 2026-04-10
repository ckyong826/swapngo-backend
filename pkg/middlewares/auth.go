package middlewares

import (
	"net/http"
	"strings"

	"swapngo-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: No token provided"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: Invalid header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		
		// 🌟 Call your actual ParseJWT function
		userID, err := utils.ParseJWT(tokenString) 
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Unauthorized: Invalid or expired token"})
			c.Abort()
			return
		}
		
		// Inject the real userID into the context
		c.Set("userID", userID)
		c.Next()
	}
}