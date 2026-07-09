package router

import (
	"demo/internal/handle"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/user/:id")
		apiGroup.POST("/user/create", handle.CreateUser)
		apiGroup.POST("/user/login", handle.LoginWithUsername)
	}
	return r
}
