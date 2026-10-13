package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/pkg/response"

	ossclient "github.com/wsc-zz/service/pkg/oss"
)

type MidiaHandler struct {
	client *ossclient.Client
}

// 允许的图片类型:嗅探结果(真身) → 扩展名(key/Content-Type 用)。
// 收口在这里:要支持新格式加一行,其余代码不动。
var allowTypes = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
	"image/gif":  "gif",
}

const maxUploadSize = 10 << 20 // 10MB:图片上限。视频走阶段五直传,别在这里放宽

func NewMidiaHandler(client *ossclient.Client) *MidiaHandler {
	return &MidiaHandler{client}
}

// Upload 上传媒体文件
//
//	@Summary		上传媒体文件
//	@Description	上传图片到 OSS,返回访问 URL。multipart/form-data:file 必传(≤10MB,
//	             	仅 png/jpeg/webp/gif,类型按请求嗅探的真身校验);key 可选,不传则
//	             	由服务端生成。需要登录（JWT）
//	@Tags			媒体资源
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			file	formData	file	true	"图片文件（≤10MB）"
//	@Param			key		formData	string	false	"对象 key 前缀，不传自动生成"
//	@Success		200		{object}	response.Response{data=string}	"图片访问 URL"
//	@Failure		400		{object}	response.Response	"缺少 file 字段"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		413		{object}	response.Response	"文件过大（上限 10MB）"
//	@Failure		415		{object}	response.Response	"不支持的媒体类型"
//	@Failure		500		{object}	response.Response	"上传 OSS 失败"
//	@Router			/catalog/public/media/upload [post]
func (h *MidiaHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	key := c.PostForm("key")

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
	if key == "" {
		key = ossclient.GenKey("", ext)
	}
	url, err := h.client.Put(c, ossclient.PutInput{Ext: ext, Reader: file}, "test")
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, url)
}

// SignURL 获取媒体文件的签名访问 URL
//
//	@Summary		获取媒体签名 URL
//	@Description	按 key 与扩展名生成 OSS 预签名 URL,有效期 1 小时。需要登录（JWT）
//	@Tags			媒体资源
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			key	query	string	true	"对象 key"
//	@Param			ext	query	string	true	"扩展名（png/jpg/webp/gif）"
//	@Success		200	{object}	response.Response{data=string}	"签名 URL"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		500	{object}	response.Response	"生成签名 URL 失败"
//	@Router			/catalog/public/media/signURL [get]
func (h *MidiaHandler) SignURL(c *gin.Context) {
	key := c.Query("key")
	ext := c.Query("ext")
	key = ossclient.GenKey(key, ext)
	url, err := h.client.Presign(c, key, ext, time.Hour)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, url)
}
