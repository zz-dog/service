package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	productapp "github.com/wsc-zz/service/internal/application/product"
	"github.com/wsc-zz/service/pkg/response"

	ossclient "github.com/wsc-zz/service/pkg/oss"
)

// MerchantIdentityResolver 商户成员关系校验的窄接口(与 middleware.TokenBlacklistChecker 同款纪律):
// 媒体 handler 只关心"这个登录用户是不是这家店的成员(且能做日常经营)",
// 由 router 组合根注入 merchant RPC 实现。身份不进 token:店可能随时被封禁/停业/
// 成员被移除,(userId, merchantId) 必须实时校验,不采信请求参数
type MerchantIdentityResolver interface {
	AuthorizeMember(ctx context.Context, userID, merchantID uint) (productapp.MemberInfo, error)
}

// 商户端媒体操作的统一拒绝语
var (
	errNotLoggedIn   = errors.New("未登录或登录已过期")
	errNoShop        = errors.New("缺少店铺上下文或无权访问该店铺")
	errKeyOutOfScope = errors.New("无权访问其他店铺的媒体资源")
)

type MidiaHandler struct {
	client *ossclient.Client
	// merchantScope 商户端命名空间解析器:nil = 管理端(平台命名空间,key 不限)。
	// 商户端返回 "merchant/<merchantID>" 前缀——上传强制落本店名下,
	// 读接口校验归属,商户 A 无法覆盖/签名商户 B 的对象
	merchantScope func(c *gin.Context) (string, error)
}

func NewMidiaHandler(client *ossclient.Client) *MidiaHandler {
	return &MidiaHandler{client: client}
}

// NewMerchantMidiaHandler 商户端媒体 handler:X-Merchant-Id 头声明店铺,
// (userId, merchantId) 经商户服务实时校验成员关系(任意角色可经营),
// 缺头/非成员拒绝。与商户商品接口同一鉴权哲学:路由层只管认证,归属服务端收口
func NewMerchantMidiaHandler(client *ossclient.Client, resolver MerchantIdentityResolver) *MidiaHandler {
	return &MidiaHandler{
		client: client,
		merchantScope: func(c *gin.Context) (string, error) {
			userID, ok := getCurrentUserID(c)
			if !ok {
				return "", errNotLoggedIn
			}
			merchantID, ok := getCurrentMerchantID(c)
			if !ok {
				return "", errNoShop
			}
			m, err := resolver.AuthorizeMember(c.Request.Context(), userID, merchantID)
			if err != nil {
				return "", err
			}
			if m.MerchantID == 0 || !m.Role.CanOperateProduct() {
				return "", errNoShop
			}
			return fmt.Sprintf("merchant/%d", m.MerchantID), nil
		},
	}
}

// uploadKey 决定本次上传的对象 key:商户端一律服务端生成并强制本店前缀
// (不采信表单 key,否则商户可把文件写进别人命名空间);管理端沿用自定义前缀能力
func (h *MidiaHandler) uploadKey(c *gin.Context, ext string) (string, error) {
	if h.merchantScope == nil {
		if key := c.PostForm("key"); key != "" {
			return key, nil
		}
		return ossclient.GenKey("", ext), nil
	}
	scope, err := h.merchantScope(c)
	if err != nil {
		return "", err
	}
	return ossclient.GenKey(scope, ext), nil
}

// ensureKeyScope 商户端读接口的归属校验:key 必须落在自己店铺前缀下——
// 永久 URL 一旦签出等于把对方资源挂进自己页面,归属必须在服务端卡住
func (h *MidiaHandler) ensureKeyScope(c *gin.Context, key string) error {
	if h.merchantScope == nil {
		return nil
	}
	scope, err := h.merchantScope(c)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(key, scope+"/") {
		return errKeyOutOfScope
	}
	return nil
}

// 允许的图片类型:嗅探结果(真身) → 扩展名(key/Content-Type 用)。
// 收口在这里:要支持新格式加一行,其余代码不动。
var allowTypes = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// allowExts 是 allowTypes 的反向映射:扩展名(ext 参数/key 用) → MIME(签名 URL 的 Content-Type 用)。
var allowExts = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"webp": "image/webp",
	"gif":  "image/gif",
}

const maxUploadSize = 10 << 20 // 10MB:图片上限。视频走阶段五直传,别在这里放宽

// Upload 上传媒体文件
//
//	@Summary		上传媒体文件
//	@Description	上传图片到 OSS,返回访问 URL。multipart/form-data:file 必传(≤10MB,
//	             	仅 png/jpeg/webp/gif,类型按请求嗅探的真身校验);key 可选,不传则
//	             	由服务端生成。需要管理端权限（JWT + AdminAuth）
//	@Tags			管理端-媒体资源
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			file	formData	file							true	"图片文件（≤10MB）"
//	@Param			key		formData	string							false	"对象 key 前缀，不传自动生成"
//	@Success		200		{object}	response.Response{data=string}	"图片访问 URL"
//	@Failure		400		{object}	response.Response				"缺少 file 字段"
//	@Failure		401		{object}	response.Response				"未登录或 token 失效"
//	@Failure		413		{object}	response.Response				"文件过大（上限 10MB）"
//	@Failure		415		{object}	response.Response				"不支持的媒体类型"
//	@Failure		500		{object}	response.Response				"上传 OSS 失败"
//	@Router			/catalog/admin/media/upload [post]
func (h *MidiaHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if header.Size > maxUploadSize {
		response.JSON(c, http.StatusRequestEntityTooLarge, "文件过大", nil)
		return
	}
	contentType := header.Header.Get("Content-Type")
	ext, ok := allowTypes[contentType]
	if !ok {
		response.JSON(c, http.StatusUnsupportedMediaType, "不支持的媒体类型", nil)
		return
	}
	// key 由命名空间策略决定:管理端可自定义前缀;商户端服务端生成并强制本店前缀
	key, err := h.uploadKey(c, ext)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	url, err := h.client.Put(c, ossclient.PutInput{Ext: ext, Reader: file}, key)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, url)
}

// UploadURLResp 直传签名响应:前端 PUT 到 URL,上传后凭 Key 换读 URL
type UploadURLResp struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// UploadURL 获取 OSS 直传签名 URL
//
//	@Summary		获取 OSS 直传签名 URL
//	@Description	按扩展名生成对象 key 与预签名 PUT URL,前端直接 PUT 文件本体到该 URL
//	             	(Content-Type 须与扩展名对应的 MIME 一致),有效期 1 小时。
//	             	上传后凭返回的 key 调 /admin/media/url 换永久 URL（Bucket 公共读时），或 /admin/media/signURL 换 1 小时临时读 URL。需要管理端权限（JWT + AdminAuth）
//	@Tags			管理端-媒体资源
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			ext	query		string											true	"扩展名（png/jpg/webp/gif）"
//	@Success		200	{object}	response.Response{data=handler.UploadURLResp}	"key 与签名 URL"
//	@Failure		400	{object}	response.Response								"缺少 ext"
//	@Failure		401	{object}	response.Response								"未登录或 token 失效"
//	@Failure		415	{object}	response.Response								"不支持的扩展名"
//	@Failure		500	{object}	response.Response								"生成签名 URL 失败"
//	@Router			/catalog/admin/media/uploadURL [get]
func (h *MidiaHandler) UploadURL(c *gin.Context) {
	ext := c.Query("ext")
	if ext == "" {
		response.BadRequest(c, "缺少 ext")
		return
	}
	// allowExts 的 value(对应 MIME)仅作文档参考,校验只看 key
	if _, ok := allowExts[ext]; !ok {
		response.JSON(c, http.StatusUnsupportedMediaType, "不支持的扩展名", nil)
		return
	}
	// key 由命名空间策略决定:管理端平台前缀;商户端强制本店前缀
	key, err := h.uploadKey(c, ext)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	url, err := h.client.Presign(c, key, ext, time.Hour)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, UploadURLResp{Key: key, URL: url})
}

// SignURL 获取媒体文件的签名访问 URL
//
//	@Summary		获取媒体签名 URL
//	@Description	按 key 生成 OSS 预签名读 URL(私有 Bucket 读取用),有效期 1 小时。需要管理端权限（JWT + AdminAuth）
//	@Tags			管理端-媒体资源
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			key	query		string							true	"对象 key"
//	@Success		200	{object}	response.Response{data=string}	"签名 URL"
//	@Failure		400	{object}	response.Response				"缺少 key"
//	@Failure		401	{object}	response.Response				"未登录或 token 失效"
//	@Failure		500	{object}	response.Response				"生成签名 URL 失败"
//	@Router			/catalog/admin/media/signURL [get]
func (h *MidiaHandler) SignURL(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.BadRequest(c, "缺少 key")
		return
	}
	if err := h.ensureKeyScope(c, key); err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	url, err := h.client.SignURL(c, key, time.Hour)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, url)
}

// URL 获取对象的永久访问 URL
//
//	@Summary		获取对象永久访问 URL
//	@Description	按 key 返回 base+key 拼接的永久 URL，供 C 端长期展示商品图等场景。
//	             	前提：Bucket 读写权限为公共读，否则匿名访问 403。
//	             	真正私有的内容不要放公共读 bucket，改用 /admin/media/signURL 换临时 URL。
//	             	需要管理端权限（JWT + AdminAuth）
//	@Tags			管理端-媒体资源
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			key	query		string							true	"对象 key"
//	@Success		200	{object}	response.Response{data=string}	"永久访问 URL"
//	@Failure		400	{object}	response.Response				"缺少 key"
//	@Failure		401	{object}	response.Response				"未登录或 token 失效"
//	@Router			/catalog/admin/media/url [get]
func (h *MidiaHandler) URL(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		response.BadRequest(c, "缺少 key")
		return
	}
	if err := h.ensureKeyScope(c, key); err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	response.Success(c, h.client.URL(key))
}
