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
		if err := c.ShouldBindJSON(&req); err != nil {
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

