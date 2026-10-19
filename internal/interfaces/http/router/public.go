package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"
	ossclient "github.com/wsc-zz/service/pkg/oss"
)

func registerPublicRoutes(r *gin.RouterGroup) {
	client := ossclient.NewClientClient(ossclient.ClientConfig{
		Base:      global.Conf.Oss.PublicBaseUrl,
		Bucket:    global.Conf.Oss.Bucket,
		KeyId:     global.Conf.Oss.AccessKeyId,
		KeySecret: global.Conf.Oss.AccessAccessKeySecret,
		Region:    global.Conf.Oss.Region,
	})
	blacklist := NewTokenBlacklist()
	h := handler.NewMidiaHandler(client)
	publicApi := r.Group("/public", middleware.JWTAuth(blacklist))
	{
		publicApi.POST("/media/upload", h.Upload)
		publicApi.GET("/media/uploadURL", h.UploadURL)
		publicApi.GET("/media/signURL", h.SignURL)
	}
}
