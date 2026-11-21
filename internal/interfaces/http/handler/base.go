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

// HeaderMerchantID 商户端"当前店铺"上下文头:N:N 下一个用户可归属多家店,
// 请求必须显式声明操作哪家店;服务端实时校验成员关系,永不采信 body 里的 merchantId。
// 放 header 而不是塞进每个路径,是因为"店"对这些服务是上下文不是资源
// (merchant 服务里店铺本身是资源,走路径 /merchant/shops/:id)。
const HeaderMerchantID = "X-Merchant-Id"

// getCurrentMerchantID 取当前请求的店铺上下文;缺失/非法返回 (0,false) → 调用方 400。
// 只解析不校验:这个 ID 是否真实、请求者是否是它的成员,由应用层经 merchant RPC 裁决。
func getCurrentMerchantID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.GetHeader(HeaderMerchantID), 10, 64)
	if err != nil || id == 0 {
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
