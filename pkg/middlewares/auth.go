package middlewares

import (
	"net/http"
	"strings"

	"swapngo-backend/pkg/responses"
	"swapngo-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 1. 尝试从 Header 提取 (标准 REST 方式)
		// 格式: Authorization: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// 2. 如果 Header 没有，尝试从 URL Query 提取 (为了兼容 WebSocket)
		// 格式: /ws?token=<token>
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		// 3. 校验 Token 是否存在
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, responses.APIResponse{
				Success: false, 
				Error: "Unauthorized: No token provided",
			})
			c.Abort()
			return
		}

		// 4. 调用你的 utils.ParseJWT 进行解析
		userID, err := utils.ParseJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, responses.APIResponse{
				Success: false, 
				Error: "Unauthorized: " + err.Error(),
			})
			c.Abort()
			return
		}

		// 5. 将解析出的 userID 注入上下文
		c.Set("user_id", userID)
		
		c.Next()
	}
}