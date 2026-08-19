package router

import (
	"github.com/gin-gonic/gin"
	specapp "github.com/wsc-zz/service/internal/application/spec"

	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

func RegisterSpecRoutes(r *gin.Engine, specSvc *specapp.Service) {
	h := handler.NewSpecHandler(specSvc)
	specRouter := r.Group("/spec")

	specRouter.POST("/register", h.Create)
	// 规格维度路由组

}
