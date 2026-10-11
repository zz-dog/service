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

func (h *MidiaHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, 400, err.Error())
	}
	key := c.PostForm("key")

	if header.Size > maxUploadSize {
		response.Fail(c, http.StatusRequestEntityTooLarge, 413, "文件过大")
	}
	contentType := header.Header.Get("Content-Type")
	ext, ok := allowTypes[contentType]
	if !ok {
		response.Fail(c, http.StatusUnsupportedMediaType, 415, "不支持的媒体类型")
	}
	if key == "" {
		key = ossclient.GenKey("", ext)
	}
	url, err := h.client.Put(c, ossclient.PutInput{Ext: ext, Reader: file}, "test")
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
	}
	response.Success(c, url)
}

func (h *MidiaHandler) SignURL(c *gin.Context) {
	key := c.Query("key")
	ext := c.Query("ext")
	key = ossclient.GenKey(key, ext)
	url, err := h.client.Presign(c, key, ext, time.Hour)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, 500, err.Error())
	}
	response.Success(c, url)
}
