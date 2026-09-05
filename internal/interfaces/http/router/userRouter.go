package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	userapp "github.com/wsc-zz/service/internal/application/user"
	"github.com/wsc-zz/service/internal/infrastructure/auth"
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

	userSvc := userapp.NewService(userRepo, hasher, tokenIssuer)
	h := handler.NewHandler(userSvc)
	apiGroup := r.Group("/user")
	// 健康检查：供部署流水线 / 负载均衡探活使用，不校验 JWT
	apiGroup.GET("/health", func(c *gin.Context) {
		response.SuccessMsg(c, "ok", nil)
	})

	apiGroup.POST("/register", h.Register)
	apiGroup.POST("/login", h.Login)
	apiGroup.GET("/list", middleware.JWTAuth(), h.GetUserList)
	apiGroup.PUT("/status", middleware.JWTAuth(), h.ChangeStatus)
	// 用户资料：需要登录（JWT 中间件校验 token 并写入 userId）
	apiGroup.PUT("/user/profile", middleware.JWTAuth(), h.UpdateUser) // 更新当前用户资料
}
