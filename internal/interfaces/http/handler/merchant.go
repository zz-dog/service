package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	merchantapp "github.com/wsc-zz/service/internal/application/merchant"
	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// MerchantHandler 是商户入驻/店铺管理的 HTTP 处理器。
// 店铺管理在 merchantShop.go、管理端在 merchantAdmin.go——拆的是文件,handler/服务共用一份。
type MerchantHandler struct {
	merchantSvc *merchantapp.Service
}

func NewMerchantHandler(merchantSvc *merchantapp.Service) *MerchantHandler {
	return &MerchantHandler{merchantSvc: merchantSvc}
}

type submitApplicationReq struct {
	ShopName     string `json:"shopName" binding:"required,max=64"`
	ContactName  string `json:"contactName" binding:"required,max=32"`
	ContactPhone string `json:"contactPhone" binding:"required,max=16"`
	LicenseURL   string `json:"licenseUrl" binding:"required,max=512"`
	IDCardURL    string `json:"idCardUrl" binding:"required,max=512"`
}

// writeMerchantError 商户域错误 → HTTP 状态码。
// 不存在类 → 404;规则/状态类 → 400;其余 → 500。
func writeMerchantError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainmerchant.ErrApplicationNotFound),
		errors.Is(err, domainmerchant.ErrMerchantNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, domainmerchant.ErrAlreadyPending),
		errors.Is(err, domainmerchant.ErrAlreadyMerchant),
		errors.Is(err, domainmerchant.ErrShopNameTaken),
		errors.Is(err, domainmerchant.ErrInvalidApplication),
		errors.Is(err, domainmerchant.ErrInvalidMerchantInfo),
		errors.Is(err, domainmerchant.ErrInvalidRejectReason),
		errors.Is(err, domainmerchant.ErrInvalidStatusTransition):
		response.BadRequest(c, err.Error())
	default:
		response.ServerError(c, err.Error())
	}
}

// SubmitApplication 提交入驻申请
//
//	@Summary		提交入驻申请
//	@Description	提交入驻申请,状态直接是待审核。材料 URL 来自 OSS 上传接口。
//	@Tags			商户
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			request	body		submitApplicationReq	true	"申请信息"
//	@Success		200		{object}	response.Response{data=merchantapp.ApplicationDTO}
//	@Failure		400		{object}	response.Response	"参数校验失败/已有待审核申请/已拥有店铺/申请信息不完整"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/application [post]
func (h *MerchantHandler) SubmitApplication(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	var req submitApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	dto, err := h.merchantSvc.Submit(c.Request.Context(), userID, merchantapp.SubmitInput{
		ShopName:     req.ShopName,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		LicenseURL:   req.LicenseURL,
		IDCardURL:    req.IDCardURL,
	})
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "submitApplication", dto)
}
