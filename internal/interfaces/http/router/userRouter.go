package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	addressapp "github.com/wsc-zz/service/internal/application/address"
	userapp "github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/infrastructure/auth"
	addresspo "github.com/wsc-zz/service/internal/infrastructure/persistence/address"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/transaction"
	userpo "github.com/wsc-zz/service/internal/infrastructure/persistence/user"
	"github.com/wsc-zz/service/internal/infrastructure/security"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	"github.com/wsc-zz/service/internal/interfaces/http/middleware"
	"github.com/wsc-zz/service/pkg/response"
)

func registerUserRoutes(r *gin.RouterGroup) {
	userRepo := userpo.NewUserRepository(global.DB)
	hasher := security.NewBcryptHasher()
	tokenIssuer := auth.NewJWTTokenIssuer()
	tokenBlacklist := NewTokenBlacklist()
	userSvc := userapp.NewService(userRepo, hasher, tokenIssuer, tokenBlacklist)
	h := handler.NewHandler(userSvc)
	apiGroup := r.Group("/user")

	// 健康检查：供部署流水线 / 负载均衡探活使用，不校验 JWT
	apiGroup.GET("/health", healthCheck)

	apiGroup.POST("/register", h.Register)
	apiGroup.POST("/login", h.Login)

	// 用户资料：需要登录（JWT 中间件校验 token 并写入 userId）
	apiGroup.PUT("/user/profile", middleware.JWTAuth(tokenBlacklist), h.UpdateUser) // 更新当前用户资料

	// 收货地址：全部走 JWT，userID 取自 token，不信任请求参数
	addressRepo := addresspo.NewAddressRepository(global.DB)
	addressSvc := addressapp.NewService(addressRepo, transaction.NewRunner(global.DB))
	ah := handler.NewAddressHandler(addressSvc)
	addr := apiGroup.Group("/address", middleware.JWTAuth(tokenBlacklist))
	{
		addr.POST("", ah.Create)        // /api/user/address
		addr.GET("", ah.List)           // /api/user/address
		addr.GET("/:id", ah.GetAddress) // /api/user/address/:id
		addr.PUT("/:id", ah.Update)     // /api/user/address/:id
		addr.DELETE("/:id", ah.Delete)  // /api/user/address/:id
		addr.PUT("/:id/default", ah.SetDefault)
	}

	// 用户管理:管理端接口,/admin 段收口。identity 只有一个域要收口,管理端组
	// 就近写在域路由文件里,不单独开 admin.go。
	// JWTAuth 与 AdminAuth 显式串联:与 catalog 的 admin 组同款两层结构。
	// 用户应用服务只装配一份,管理端/C 端共用同一个 handler——拆的是路由,不是服务
	admin := apiGroup.Group("/admin", middleware.JWTAuth(tokenBlacklist), middleware.AdminAuth())
	{
		admin.GET("/list", h.GetUserList)    // /api/user/admin/list
		admin.PUT("/status", h.ChangeStatus) // /api/user/admin/status
	}
}

// healthCheck 健康检查
//
//	@Summary		健康检查
//	@Description	供部署流水线 / 负载均衡探活使用，不校验 JWT
//	@Tags			用户
//	@Produce		json
//	@Success		200	{object}	response.Response
//	@Router			/user/health [get]
func healthCheck(c *gin.Context) {
	response.SuccessMsg(c, "ok", nil)
}
