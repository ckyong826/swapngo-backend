package middlewares

import (
	"net/http"

	"swapngo-backend/pkg/responses"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware rejects requests whose JWT role is not "ADMIN".
// Must be placed after AuthMiddleware so that "role" is already in the context.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != "ADMIN" {
			c.JSON(http.StatusForbidden, responses.APIResponse{
				Success: false,
				Message: "Forbidden: admin access required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
