package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// reviewApplicationReq 审核请求体。
// approve 用 *bool:required 拦不住 false(bool 零值),指针才能区分"没传"和"传 false"——
// JSON 里不传 approve 和传 false 不是一回事,前者是参数缺失,后者是明确的"驳回"。
type reviewApplicationReq struct {
	Approve *bool  `json:"approve" binding:"required"`
	Reason  string `json:"reason"`
}

// ListApplications 申请单列表(管理端)
//
//	@Summary		申请单列表
//	@Description	全部入驻申请,时间倒序,供管理员审核。教学版全量,分页进阶。需要管理员(JWT+AdminAuth)
//	@Tags			商户管理
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.Response{data=[]merchantapp.ApplicationDTO}
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		403	{object}	response.Response	"需要管理员权限"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/admin/applications [get]
func (h *MerchantHandler) ListApplications(c *gin.Context) {
	dtos, err := h.merchantSvc.ListApplications(c.Request.Context())
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "applications", dtos)
}

// Review 审核
//
//	@Summary		审核入驻申请
//	@Description	approve=true 走通过三连(同事务:申请单迁移+建商户+回填 merchant_id),
//	                false 为驳回(reason 必填)。并发下两个管理员同时审,只有一人生效,
//	                另一个收到"当前状态不允许该操作"。
//	@Tags			商户管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		uint					true	"申请单ID"
//	@Param			request	body		reviewApplicationReq	true	"审核决定"
//	@Success		200		{object}	response.Response{data=string}
//	@Failure		400		{object}	response.Response	"参数缺失/驳回理由为空/当前状态不允许该操作(并发败北)"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		403		{object}	response.Response	"需要管理员权限"
//	@Failure		404		{object}	response.Response	"申请单不存在"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/admin/applications/{id}/review [post]
func (h *MerchantHandler) Review(c *gin.Context) {
	var req reviewApplicationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	appID := parseUintParam(c, "id")
	if err := h.merchantSvc.Review(c.Request.Context(), 0, appID, *req.Approve, req.Reason); err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "review", "审核完成")
}

// Ban 封禁商户
//
//	@Summary		封禁商户
//	@Description	营业中|已停业 → 已封禁(平台处罚,不可自行恢复)。需要管理员(JWT+AdminAuth)
//	@Tags			商户管理
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		uint	true	"商户ID"
//	@Success		200	{object}	response.Response{data=string}
//	@Failure		400	{object}	response.Response	"当前状态不允许封禁(如已封禁)"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		403	{object}	response.Response	"需要管理员权限"
//	@Failure		404	{object}	response.Response	"店铺不存在"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/admin/merchants/{id}/ban [post]
func (h *MerchantHandler) Ban(c *gin.Context) {
	merchantID := parseUintParam(c, "id")
	if err := h.merchantSvc.Ban(c.Request.Context(), merchantID); err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "ban", "商户已封禁")
}
