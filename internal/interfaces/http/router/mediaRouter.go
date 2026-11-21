package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	"github.com/wsc-zz/service/internal/interfaces/rpc/merchantrpc"
	ossclient "github.com/wsc-zz/service/pkg/oss"
)

// newOSSClient 媒体上传/签名共用一份 OSS 客户端:管理端与商户端只差命名空间策略,不差存储
func newOSSClient() *ossclient.Client {
	return ossclient.NewClientClient(ossclient.ClientConfig{
		Base:      global.Conf.Oss.PublicBaseUrl,
		Bucket:    global.Conf.Oss.Bucket,
		KeyId:     global.Conf.Oss.AccessKeyId,
		KeySecret: global.Conf.Oss.AccessAccessKeySecret,
		Region:    global.Conf.Oss.Region,
	})
}

// registerMediaRoutes 管理端媒体上传/签名路由:挂在 admin 组下,路径 /api/catalog/admin/media/**。
// 由 admin.go 的 JWTAuth+AdminAuth 组统一收口——上传签名是管理端能力,
// 放开到 C 端用户等于谁都能往 bucket 塞文件;商户端走 registerMerchantMediaRoutes,
// 同一组 handler 由命名空间策略区分行为
func registerMediaRoutes(r *gin.RouterGroup) {
	h := handler.NewMidiaHandler(newOSSClient())
	mediaApi := r.Group("/media")
	{
		mediaApi.POST("/upload", h.Upload)
		mediaApi.GET("/uploadURL", h.UploadURL)
		mediaApi.GET("/signURL", h.SignURL)
		mediaApi.GET("/url", h.URL) // Bucket 公共读时的永久访问 URL;私有内容仍走 signURL
	}
}

// registerMerchantMediaRoutes 商户端媒体上传/签名:挂在 /merchant 组(JWTAuth)下,
// 路径 /api/catalog/merchant/media/**。鉴权模型与商户商品接口一致:路由层只管认证,
// merchantID 由 handler 经 merchant RPC 实时推导——对象强制落在 merchant/<id>/ 命名空间,
// 商户 A 无法覆盖/签名商户 B 的对象;管理端(平台命名空间)不受此限制
func registerMerchantMediaRoutes(r *gin.RouterGroup) {
	merchantGw, err := merchantrpc.NewClient(global.Conf.Merchant.RpcAddr, 3*time.Second)
	if err != nil {
		panic(err) // 装配失败服务起不来,快速失败(与 newProductHandler 同语义)
	}
	h := handler.NewMerchantMidiaHandler(newOSSClient(), merchantGw)
	mediaApi := r.Group("/media")
	{
		// 走 Merchant* 包装方法:与 admin 共用实现,但 Swagger 文档各出各的路径
		mediaApi.POST("/upload", h.MerchantUpload)      // 上传(≤10MB 图片),对象落 merchant/<id>/
		mediaApi.GET("/uploadURL", h.MerchantUploadURL) // 直传签名:按本店前缀生成 key+PUT URL
		mediaApi.GET("/signURL", h.MerchantSignURL)     // 临时读 URL(1h):仅限本店对象
		mediaApi.GET("/url", h.MerchantURL)             // 永久读 URL:仅限本店对象
	}
}
