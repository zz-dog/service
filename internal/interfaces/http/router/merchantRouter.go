package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	merchantapp "github.com/wsc-zz/service/internal/application/merchant"
	merchantpo "github.com/wsc-zz/service/internal/infrastructure/persistence/merchant"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/transaction"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	"github.com/wsc-zz/service/internal/interfaces/http/middleware"
)

// InitMerchantRouter merchant 服务的路由入口:入驻申请/店铺管理挂在 /api/merchant 前缀下。
// 网关按最长前缀把 /api/merchant/** 转发到 merchant 服务,因此路径必须带此前缀。
func InitMerchantRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	merchantGroup := r.Group("/api/merchant")

	// 应用服务只装配一份,用户端/管理端共用同一个 handler——拆的是路由,不是服务
	appRepo := merchantpo.NewMerchantApplicationRepository(global.DB)
	mRepo := merchantpo.NewMerchantRepository(global.DB)
	merchantSvc := merchantapp.NewService(appRepo, mRepo, transaction.NewRunner(global.DB))
	h := handler.NewMerchantHandler(merchantSvc)
	blacklist := NewTokenBlacklist()

	registerMerchantRoutes(merchantGroup, h, blacklist)
	registerMerchantAdminRoutes(merchantGroup, h, blacklist)
	return r
}

// registerMerchantRoutes 用户侧:提交申请/查进度/撤回/店铺管理,全部 JWT。
// userID 一律取自 token,不信任请求参数。
func registerMerchantRoutes(api *gin.RouterGroup, h *handler.MerchantHandler, blacklist middleware.TokenBlacklistChecker) {
	user := api.Group("", middleware.JWTAuth(blacklist))
	{
		user.POST("/application", h.SubmitApplication)                // 提交入驻申请
		user.GET("/application/my", h.MyApplications)                 // 我的申请历史
		user.POST("/application/:id/withdraw", h.WithdrawApplication) // 撤回(仅待审核)
		user.GET("/my", h.MyShop)                                     // 我的店铺
		user.PUT("/my", h.UpdateShop)                                 // 更新店铺资料
		user.POST("/my/suspend", h.Suspend)                           // 停业
		user.POST("/my/resume", h.Resume)                             // 恢复营业
	}
}

// registerMerchantAdminRoutes 管理端:审核/封禁。
// JWTAuth 在前(认证:你是谁),AdminAuth 在后(授权:你够不够格),链式组合;
// 鉴权在路由层一次做齐,handler 里不再人人自觉(与 catalog/user 的 admin 组同款)。
func registerMerchantAdminRoutes(api *gin.RouterGroup, h *handler.MerchantHandler, blacklist middleware.TokenBlacklistChecker) {
	admin := api.Group("/admin", middleware.JWTAuth(blacklist), middleware.AdminAuth())
	{
		admin.GET("/applications", h.ListApplications)   // 申请列表(教学版全量)
		admin.POST("/applications/:id/review", h.Review) // 审核 {approve, reason}
		admin.POST("/merchants/:id/ban", h.Ban)          // 封禁商户
	}
}
