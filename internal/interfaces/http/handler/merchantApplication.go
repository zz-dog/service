package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/pkg/response"
)

// MyApplications 我的申请历史
//
//	@Summary		我的申请历史
//	@Description	当前用户的全部入驻申请,时间倒序。归属即过滤条件(userID 取自 JWT)。
//	@Tags			商户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.Response{data=[]merchantapp.ApplicationDTO}
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/application/my [get]
func (h *MerchantHandler) MyApplications(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	dtos, err := h.merchantSvc.MyApplications(c.Request.Context(), userID)
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "myApplications", dtos)
}

// WithdrawApplication 撤回申请
//
//	@Summary		撤回申请
//	@Description	撤回自己的待审核申请。只能撤本人的单(别人的单与不存在同错误),
//	                且仅待审核状态可撤;刚被人审核过的撤回请求会被状态机拒绝。
//	@Tags			商户
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		uint	true	"申请单ID"
//	@Success		200	{object}	response.Response{data=string}
//	@Failure		400	{object}	response.Response	"当前状态不允许撤回"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		404	{object}	response.Response	"申请单不存在(别人的单同此错误,不泄露存在性)"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/application/{id}/withdraw [post]
func (h *MerchantHandler) WithdrawApplication(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	appID := parseUintParam(c, "id")
	err := h.merchantSvc.Withdraw(c.Request.Context(), userID, appID)
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "withdraw", "申请已撤回")
}
