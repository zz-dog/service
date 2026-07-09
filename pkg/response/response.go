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

// Fail 通用错误，由调用方指定 HTTP 状态码和业务码
func Fail(c *gin.Context, httpStatus, code int, msg string) {
	JSON(c, httpStatus, code, msg, nil)
}

// BadRequest 参数校验、业务逻辑等客户端错误
func BadRequest(c *gin.Context, code int, msg string) {
	JSON(c, http.StatusBadRequest, code, msg, nil)
}

// Unauthorized 未登录或 token 失效
func Unauthorized(c *gin.Context, code int, msg string) {
	JSON(c, http.StatusUnauthorized, code, msg, nil)
}

// Forbidden 无权限访问
func Forbidden(c *gin.Context, code int, msg string) {
	JSON(c, http.StatusForbidden, code, msg, nil)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, code int, msg string) {
	JSON(c, http.StatusNotFound, code, msg, nil)
}

// ServerError 服务端内部错误
func ServerError(c *gin.Context, code int, msg string) {
	JSON(c, http.StatusInternalServerError, code, msg, nil)
}
