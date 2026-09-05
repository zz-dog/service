package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	specapp "github.com/wsc-zz/service/internal/application/spec"
	specpo "github.com/wsc-zz/service/internal/infrastructure/persistence/spec"

	"github.com/wsc-zz/service/internal/interfaces/http/handler"
)

func RegisterSpecRoutes(r *gin.RouterGroup) {
	specRepo := specpo.NewSpecRepository(global.DB)
	specSvc := specapp.NewService(specRepo)

	h := handler.NewSpecHandler(specSvc)
	specRouter := r.Group("/spec")

	specRouter.POST("/register", h.Create)
	specRouter.PUT("/update/:id", h.Update)
	specRouter.DELETE("/delete/:id", h.Delete)
	specRouter.GET("/findById/:id", h.FindByID)
	specRouter.GET("/findByName/:name", h.FindByName)
	specRouter.GET("/list", h.List)

}
