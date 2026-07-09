package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func JSON(c *gin.Context, httpStatus, code int, msg string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: msg,
		Data:    data,
	})
}

func Success(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, 0, "success", data)
}

func SuccessMsg(c *gin.Context, msg string, data interface{}) {
	JSON(c, http.StatusOK, 0, msg, data)
}

func Fail(c *gin.Context, code int, msg string) {
	JSON(c, http.StatusBadRequest, code, msg, nil)
}

func FailWithStatus(c *gin.Context, httpStatus, code int, msg string) {
	JSON(c, httpStatus, code, msg, nil)
}
