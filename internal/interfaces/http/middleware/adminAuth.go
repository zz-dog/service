package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/internal/domain/user"
	"github.com/wsc-zz/service/pkg/response"
)

// AdminAuth 管理端授权中间件：必须挂在 JWTAuth 之后。
// 不解析 token——那是 JWTAuth 的职责；这里只读它写入上下文的 role 做判断。
// 认证回答"你是谁"（401），授权回答"你够不够格"（403），语义分开，
// 将来商户端/运营端都是"JWTAuth + XXAuth"的组合，不是重写。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 授权中间件的一切意外（取不到、类型不符）都落到"拒绝"：
		// fail to least privilege，宁可误拒、不能误放
		v, exists := c.Get("role")
		role, ok := v.(user.Role) // 断言成领域类型：接口动态类型不符时 ok=false，普通用户 role=0 走不进来
		if !exists || !ok || role != user.RoleAdmin {
			response.Forbidden(c, "需要管理员权限")
			// Abort 是必须的：只 return 不拦链，gin 会继续执行后面的业务 handler，
			// 出现"客户端看到 403、数据却已经改了"的两头骗
			c.Abort()
			return
		}
		c.Next()
	}
}
