package router

import (
	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	ossclient "github.com/wsc-zz/service/pkg/oss"
)

// registerMediaRoutes 媒体上传/签名路由:挂在 admin 组下,路径 /api/catalog/admin/media/**。
// 由 admin.go 的 JWTAuth+AdminAuth 组统一收口——上传签名是管理端(和将来的商户端)能力,
// 放开到 C 端用户等于谁都能往 bucket 塞文件。
// 存进 DB/返回给前端的是 OSS URL,与挂载路径无关,handler 与装配一行不改;
// 前端调用方需要同步改路径(阶段四 Swagger 分组后文档里一目了然)
func registerMediaRoutes(r *gin.RouterGroup) {
	client := ossclient.NewClientClient(ossclient.ClientConfig{
		Base:      global.Conf.Oss.PublicBaseUrl,
		Bucket:    global.Conf.Oss.Bucket,
		KeyId:     global.Conf.Oss.AccessKeyId,
		KeySecret: global.Conf.Oss.AccessAccessKeySecret,
		Region:    global.Conf.Oss.Region,
	})
	h := handler.NewMidiaHandler(client)
	mediaApi := r.Group("/media")
	{
		mediaApi.POST("/upload", h.Upload)
		mediaApi.GET("/uploadURL", h.UploadURL)
		mediaApi.GET("/signURL", h.SignURL)
	}
}
