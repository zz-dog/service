package handler

import (
	"github.com/gin-gonic/gin"

	merchantapp "github.com/wsc-zz/service/internal/application/merchant"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

type updateShopReq struct {
	Name string `json:"name" binding:"required,max=64"`
	Logo string `json:"logo" binding:"omitempty,max=512"`
	Desc string `json:"desc" binding:"omitempty,max=512"`
}

// MyShop 我的店铺
//
//	@Summary		我的店铺
//	@Description	当前登录用户的店铺。一人一店:FindByUserID 查出来的店就是你的,
//	                归属天然成立,不存在"查别人的店"这个入口。
//	@Tags			商户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.Response{data=merchantapp.MerchantDTO}
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		404	{object}	response.Response	"还没有店铺"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/my [get]
func (h *MerchantHandler) MyShop(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	dto, err := h.merchantSvc.MyShop(c.Request.Context(), userID)
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "myShop", dto)
}

// UpdateShop 更新店铺资料
//
//	@Summary		更新店铺资料
//	@Description	更新自己店铺的名称/Logo/描述(全字段替换)。改名次数限制是阶段五的进阶。
//	@Tags			商户
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			request	body	updateShopReq	true	"店铺资料"
//	@Success		200	{object}	response.Response{data=merchantapp.MerchantDTO}
//	@Failure		400	{object}	response.Response	"参数校验失败/店铺信息不完整"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		404	{object}	response.Response	"还没有店铺"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/my [put]
func (h *MerchantHandler) UpdateShop(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	var req updateShopReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	dto, err := h.merchantSvc.UpdateShop(c.Request.Context(), userID, merchantapp.UpdateShopInput{
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
//	@Summary		停业
//	@Description	营业中 → 已停业,只有店主本人可操作(userID 取自 JWT)。
//	@Tags			商户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.Response{data=string}
//	@Failure		400	{object}	response.Response	"当前状态不允许该操作(如已封禁)"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		404	{object}	response.Response	"还没有店铺"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/my/suspend [post]
func (h *MerchantHandler) Suspend(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	if err := h.merchantSvc.Suspend(c.Request.Context(), userID); err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "suspend", "店铺已停业")
}

// Resume 恢复营业
//
//	@Summary		恢复营业
//	@Description	已停业 → 营业中,只有店主本人可操作;封禁的店不在此列。
//	@Tags			商户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.Response{data=string}
//	@Failure		400	{object}	response.Response	"当前状态不允许该操作(如已封禁)"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		404	{object}	response.Response	"还没有店铺"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/my/resume [post]
func (h *MerchantHandler) Resume(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	if err := h.merchantSvc.Resume(c.Request.Context(), userID); err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "resume", "店铺已恢复营业")
}
