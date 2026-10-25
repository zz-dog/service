package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// getCurrentUserID 从 gin 上下文取 JWT 中间件写入的 userID。
// JWT 中间件写入的是 string(见 auth.UserClaims.UserID),这里转成 uint。
// 从 order.go 提取到公共文件:此后所有需要"当前用户"的 handler 共用这一份,
// userID 永远取自 token,不信任请求参数。
func getCurrentUserID(c *gin.Context) (uint, bool) {
	v, exists := c.Get("userId")
	if !exists {
		return 0, false
	}
	s, ok := v.(string)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// parseUintParam 解析 URL 路径段里的数字 ID,非法返回 0——
// 0 查不到记录,自然落到 404,不向客户端泄露"格式错"和"不存在"的区别。
func parseUintParam(c *gin.Context, name string) uint {
	id, _ := strconv.ParseUint(c.Param(name), 10, 64)
	return uint(id)
}
