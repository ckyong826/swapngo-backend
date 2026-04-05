package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 定义了统一的 API 返回结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // omitempty: 如果 data 为 nil，则在 JSON 中不显示此字段
}

// Success 处理请求成功的情况，并返回 200 OK
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK, // 200
		Message: "Success",
		Data:    data,
	})
}

// Error 处理请求失败的情况，允许自定义 HTTP 状态码和错误信息
func Error(c *gin.Context, httpCode int, message string) {
	c.JSON(httpCode, Response{
		Code:    httpCode,
		Message: message,
		Data:    nil,
	})
}