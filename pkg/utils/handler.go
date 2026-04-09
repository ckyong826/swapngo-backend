package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 1. 定义全站统一的 JSON 格式
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// 2. 核心魔法：泛型 Handler 模板
// T 代表你的 Request 结构体类型
func Handle[T any](
	successMsg string,
	execute func(ctx *gin.Context, req *T) (any, error),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T

		// --- 阶段 1: ParamCheck (参数校验) ---
		// GIN 会自动根据结构体里的 binding:"required" 等标签进行严格校验
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Parameter validation failed: " + err.Error(),
			})
			return
		}

		// --- 阶段 2: Execute (执行核心业务) ---
		data, err := execute(c, &req)
		if err != nil {
			// 这里默认返回 500，你可以根据具体的 err 类型进一步细化状态码 (如 400, 403)
			c.JSON(http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		// --- 阶段 3: 返回成功信息 ---
		c.JSON(http.StatusOK, APIResponse{
			Success: true,
			Message: successMsg,
			Data:    data,
		})
	}
}