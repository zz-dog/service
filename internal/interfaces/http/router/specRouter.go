package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	specapp "github.com/wsc-zz/service/internal/application/spec"
	specpo "github.com/wsc-zz/service/internal/infrastructure/persistence/spec"

	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"
)

// newSpecHandler 规格应用服务的装配:管理端/C 端两个注册函数共用一套
func newSpecHandler() *handler.SpecHandler {
	specRepo := specpo.NewSpecRepository(global.DB)
	specSvc := specapp.NewService(specRepo)
	return handler.NewSpecHandler(specSvc)
}

// registerSpecAdminRoutes 规格管理端写接口:挂在 /api/catalog/admin/spec/**,
// 由 admin.go 的 JWTAuth+AdminAuth 组统一收口
func registerSpecAdminRoutes(r *gin.RouterGroup) {
	h := newSpecHandler()

	specRouter := r.Group("/spec")

	specRouter.POST("/register", h.Create)
	specRouter.PUT("/update/:id", h.Update)
	specRouter.DELETE("/delete/:id", h.Delete)
}

// registerSpecRoutes 规格 C 端读接口:保持原路径原鉴权
func registerSpecRoutes(r *gin.RouterGroup) {
	h := newSpecHandler()
	blacklist := NewTokenBlacklist()
	specRouter := r.Group("/spec", middleware.JWTAuth(blacklist))

	specRouter.GET("/findById/:id", h.FindByID)
	specRouter.GET("/findByName/:name", h.FindByName)
	specRouter.GET("/list", h.List)
}
