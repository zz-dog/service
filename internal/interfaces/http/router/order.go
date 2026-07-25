package router

import (
	"github.com/gin-gonic/gin"
)

func registerProductRoutes(api *gin.RouterGroup) {
	orderApi := api.Group("/Product")
	{
		orderApi.POST("create")
	}
}
