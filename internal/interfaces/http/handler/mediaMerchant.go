package handler

import (
	"github.com/gin-gonic/gin"
)

// 商户端媒体接口:与 /admin/media/** 共用同一 handler 与同一份 OSS 存储,
// 差异只有两点——命名空间策略(对象强制落 merchant/<id>/)与鉴权组(JWTAuth,无 AdminAuth)。
// N:N 鉴权模型:"当前店铺"由 X-Merchant-Id 头声明,(userId, merchantId) 成员关系与角色
// 由 merchant RPC 实时校验,对象强制落本店命名空间。
// 独立的方法只为给 Swagger 出独立的路径文档,方法体一行业务逻辑都不重复;
// 路由挂载见 router/mediaRouter.go 的 registerMerchantMediaRoutes

// MerchantUpload 上传媒体文件(商户)
//
//	@Summary		上传媒体文件(商户)
//	@Description	商户上传图片到 OSS,返回永久访问 URL(Bucket 公共读时匿名可访问)。
//	             	multipart/form-data:file 必传(≤10MB,仅 png/jpeg/webp/gif,类型按请求嗅探的真身校验)。
//	             	对象 key 服务端生成并强制落在 merchant/<merchantID>/ 前缀下;
//	             	"当前店铺"由 X-Merchant-Id 头声明并实时校验成员关系,表单不收 key。
//	             	需要商户登录(JWT;缺店铺上下文/非成员返回 400)
//	@Tags			商户端-媒体资源
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Merchant-Id	header		int		true	"当前店铺ID"
//	@Param			file	formData	file							true	"图片文件（≤10MB）"
//	@Success		200		{object}	response.Response{data=string}	"图片永久访问 URL"
//	@Failure		400		{object}	response.Response				"参数校验失败/缺少店铺上下文/非店铺成员"
//	@Failure		401		{object}	response.Response				"未登录或 token 失效"
//	@Failure		413		{object}	response.Response				"文件过大（上限 10MB）"
//	@Failure		415		{object}	response.Response				"不支持的媒体类型"
//	@Failure		500		{object}	response.Response				"上传 OSS 失败"
//	@Router			/catalog/merchant/media/upload [post]
func (h *MidiaHandler) MerchantUpload(c *gin.Context) { h.Upload(c) }

// MerchantUploadURL 获取 OSS 直传签名 URL(商户)
//
//	@Summary		获取 OSS 直传签名 URL(商户)
//	@Description	按扩展名生成对象 key(强制 merchant/<merchantID>/ 前缀)与预签名 PUT URL,
//	             	前端直接 PUT 文件本体到该 URL(Content-Type 须与扩展名对应的 MIME 一致),有效期 1 小时。
//	             	上传后凭返回的 key 调 /catalog/merchant/media/url 换永久 URL(公共读 bucket),
//	             	或 /catalog/merchant/media/signURL 换 1 小时临时读 URL。
//	             	需要商户登录(JWT + X-Merchant-Id;缺店铺上下文/非成员返回 400)
//	@Tags			商户端-媒体资源
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Merchant-Id	header		int												true	"当前店铺ID"
//	@Param			ext	query		string											true	"扩展名（png/jpg/webp/gif）"
//	@Success		200	{object}	response.Response{data=handler.UploadURLResp}	"key 与签名 PUT URL"
//	@Failure		400	{object}	response.Response								"缺少 ext/缺少店铺上下文/非店铺成员"
//	@Failure		401	{object}	response.Response								"未登录或 token 失效"
//	@Failure		415	{object}	response.Response								"不支持的扩展名"
//	@Failure		500	{object}	response.Response								"生成签名 URL 失败"
//	@Router			/catalog/merchant/media/uploadURL [get]
func (h *MidiaHandler) MerchantUploadURL(c *gin.Context) { h.UploadURL(c) }

// MerchantSignURL 获取媒体文件的签名访问 URL(商户)
//
//	@Summary		获取媒体签名 URL(商户)
//	@Description	按 key 生成 OSS 预签名读 URL(私有内容临时读取用),有效期 1 小时。
//	             	仅限本店对象:key 不在 merchant/<merchantID>/ 前缀下返回 403。
//	             	需要商户登录(JWT + X-Merchant-Id)
//	@Tags			商户端-媒体资源
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Merchant-Id	header		int		true	"当前店铺ID"
//	@Param			key	query		string							true	"对象 key"
//	@Success		200	{object}	response.Response{data=string}	"签名 URL"
//	@Failure		400	{object}	response.Response				"缺少 key/缺少店铺上下文/非店铺成员"
//	@Failure		401	{object}	response.Response				"未登录或 token 失效"
//	@Failure		403	{object}	response.Response				"无权访问其他店铺的媒体资源"
//	@Failure		500	{object}	response.Response				"生成签名 URL 失败"
//	@Router			/catalog/merchant/media/signURL [get]
func (h *MidiaHandler) MerchantSignURL(c *gin.Context) { h.SignURL(c) }

// MerchantURL 获取对象的永久访问 URL(商户)
//
//	@Summary		获取对象永久访问 URL(商户)
//	@Description	按 key 返回本店对象的永久 URL(base+key 拼接,Bucket 公共读时匿名可访问),
//	             	供商品图等 C 端长期展示场景。仅限本店对象:key 不在 merchant/<merchantID>/
//	             	前缀下返回 403。需要商户登录(JWT + X-Merchant-Id)
//	@Tags			商户端-媒体资源
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Merchant-Id	header		int		true	"当前店铺ID"
//	@Param			key	query		string							true	"对象 key"
//	@Success		200	{object}	response.Response{data=string}	"永久访问 URL"
//	@Failure		400	{object}	response.Response				"缺少 key/缺少店铺上下文/非店铺成员"
//	@Failure		401	{object}	response.Response				"未登录或 token 失效"
//	@Failure		403	{object}	response.Response				"无权访问其他店铺的媒体资源"
//	@Failure		500	{object}	response.Response				"服务器内部错误"
//	@Router			/catalog/merchant/media/url [get]
func (h *MidiaHandler) MerchantURL(c *gin.Context) { h.URL(c) }
