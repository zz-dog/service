package handler

import (
	"github.com/gin-gonic/gin"

	merchantapp "github.com/wsc-zz/service/internal/application/merchant"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// 商户端店铺接口:N:N 下店铺是显式资源,路径 /merchant/shops/:id 直接声明"操作哪家店"。
// userID 永远取自 JWT;归属与角色在应用层校验(非成员与店不存在同错误,不泄露存在性),
// 权限不够(是成员但不是 owner)→ 403。

type updateShopReq struct {
	Name string `json:"name" binding:"required,max=64"`
	Logo string `json:"logo" binding:"omitempty,max=512"`
	Desc string `json:"desc" binding:"omitempty,max=512"`
}

// MyShops 我的店铺列表
//
//	@Summary		我的店铺列表
//	@Description	当前登录用户归属的全部店铺,每家带 myRole(1=店主 2=管理员 3=员工)。
//	                一个用户可以加入多家店,前端按 myRole 决定展示哪些操作。
//	@Tags			商户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.Response{data=[]merchantapp.ShopDTO}
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/shops [get]
func (h *MerchantHandler) MyShops(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	dtos, err := h.merchantSvc.MyShops(c.Request.Context(), userID)
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "myShops", dtos)
}

// UpdateShop 更新店铺资料
//
//	@Summary		更新店铺资料(店主)
//	@Description	更新店铺名称/Logo/描述(全字段替换)。仅店主可操作。
//	@Tags			商户
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		uint			true	"店铺ID"
//	@Param			request	body		updateShopReq	true	"店铺资料"
//	@Success		200		{object}	response.Response{data=merchantapp.ShopDTO}
//	@Failure		400		{object}	response.Response	"参数校验失败/店铺信息不完整"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		403		{object}	response.Response	"是成员但不是店主"
//	@Failure		404		{object}	response.Response	"店铺不存在(非成员同此错误,不泄露存在性)"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/shops/{id} [put]
func (h *MerchantHandler) UpdateShop(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	merchantID := parseUintParam(c, "id")
	var req updateShopReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	dto, err := h.merchantSvc.UpdateShop(c.Request.Context(), userID, merchantID, merchantapp.UpdateShopInput{
		Name: req.Name,
		Logo: req.Logo,
		Desc: req.Desc,
	})
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "updateShop", dto)
}

// Suspend 停业
//
//	@Summary		停业(店主)
//	@Description	营业中 → 已停业,仅店主可操作。
//	@Tags			商户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		uint	true	"店铺ID"
//	@Success		200	{object}	response.Response{data=string}
//	@Failure		400	{object}	response.Response	"当前状态不允许该操作(如已封禁)"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		403	{object}	response.Response	"是成员但不是店主"
//	@Failure		404	{object}	response.Response	"店铺不存在(非成员同此错误,不泄露存在性)"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/shops/{id}/suspend [post]
func (h *MerchantHandler) Suspend(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	if err := h.merchantSvc.Suspend(c.Request.Context(), userID, parseUintParam(c, "id")); err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "suspend", "店铺已停业")
}

// Resume 恢复营业
//
//	@Summary		恢复营业(店主)
//	@Description	已停业 → 营业中,仅店主可操作;封禁的店不在此列。
//	@Tags			商户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		uint	true	"店铺ID"
//	@Success		200	{object}	response.Response{data=string}
//	@Failure		400	{object}	response.Response	"当前状态不允许该操作(如已封禁)"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		403	{object}	response.Response	"是成员但不是店主"
//	@Failure		404	{object}	response.Response	"店铺不存在(非成员同此错误,不泄露存在性)"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/shops/{id}/resume [post]
func (h *MerchantHandler) Resume(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	if err := h.merchantSvc.Resume(c.Request.Context(), userID, parseUintParam(c, "id")); err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "resume", "店铺已恢复营业")
}
