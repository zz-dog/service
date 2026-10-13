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

// JSON 底层输出，code 跟随 HTTP 状态码；非标准状态码或需要自定义时直接使用
func JSON(c *gin.Context, httpStatus int, msg string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:    httpStatus,
		Message: msg,
		Data:    data,
	})
}

func Success(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, "success", data)
}

func SuccessMsg(c *gin.Context, msg string, data interface{}) {
	JSON(c, http.StatusOK, msg, data)
}

// BadRequest 参数校验、业务逻辑等客户端错误
func BadRequest(c *gin.Context, msg string) {
	JSON(c, http.StatusBadRequest, msg, nil)
}

// Unauthorized 未登录或 token 失效
func Unauthorized(c *gin.Context, msg string) {
	JSON(c, http.StatusUnauthorized, msg, nil)
}

// Forbidden 无权限访问
func Forbidden(c *gin.Context, msg string) {
	JSON(c, http.StatusForbidden, msg, nil)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, msg string) {
	JSON(c, http.StatusNotFound, msg, nil)
}

// ServerError 服务端内部错误
func ServerError(c *gin.Context, msg string) {
	JSON(c, http.StatusInternalServerError, msg, nil)
}
