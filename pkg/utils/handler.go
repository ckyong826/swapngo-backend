package utils

import (
	"net/http"
	"swapngo-backend/pkg/responses"

	"github.com/gin-gonic/gin"
)


func Handle[T any](
	successMsg string,
	execute func(ctx *gin.Context, req *T) (any, error),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T

		// 1. ParamCheck
		var err error
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete {
			err = c.ShouldBindQuery(&req) 
		} else {
			err = c.ShouldBindJSON(&req)
		}

		if err != nil && err.Error() != "EOF" { // 允许 GET 请求完全没有参数
			c.JSON(http.StatusBadRequest, responses.APIResponse{Success: false, Error: "Invalid parameters: " + err.Error()})
			return
		}

		// 2. Execute
		data, err := execute(c, &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, responses.APIResponse{Success: false, Error: err.Error()})
			return
		}

		// 3. Response
		c.JSON(http.StatusOK, responses.APIResponse{Success: true, Message: successMsg, Data: data})
	}
}

