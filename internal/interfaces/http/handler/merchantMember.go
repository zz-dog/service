package handler

import (
	"github.com/gin-gonic/gin"

	merchantapp "github.com/wsc-zz/service/internal/application/merchant"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// 商户端成员管理接口(全部仅店主):人事权收在 owner 一处。
// 邀请直接指定目标 userID——教学版不校验目标用户存在(identity 未对 merchant 开放 RPC),
// 真实系统应换成邀请链接或工单,并把"查无此人"拦在邀请入口。

type inviteMemberReq struct {
	UserID uint `json:"userId" binding:"required"`
	Role   int  `json:"role" binding:"required"`
}

type changeMemberRoleReq struct {
	Role int `json:"role" binding:"required"`
}

// ListMembers 成员列表
//
//	@Summary		店铺成员列表(店主)
//	@Description	店铺全部成员与角色(1=店主 2=管理员 3=员工),店主在前。仅店主可查。
//	@Tags			商户-成员管理
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		uint	true	"店铺ID"
//	@Success		200	{object}	response.Response{data=[]merchantapp.MemberDTO}
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		403	{object}	response.Response	"是成员但不是店主"
//	@Failure		404	{object}	response.Response	"店铺不存在(非成员同此错误,不泄露存在性)"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/shops/{id}/members [get]
func (h *MerchantHandler) ListMembers(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	dtos, err := h.merchantSvc.ListMembers(c.Request.Context(), userID, parseUintParam(c, "id"))
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "members", dtos)
}

// InviteMember 邀请成员
//
//	@Summary		邀请成员(店主)
//	@Description	把指定用户加进店铺,角色只能 admin(管理员)或 staff(员工)。
//	                每店成员数有上限;目标已在店里 → 400"该用户已是店铺成员"。
//	@Tags			商户-成员管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		uint				true	"店铺ID"
//	@Param			request	body		inviteMemberReq		true	"目标用户与角色"
//	@Success		200		{object}	response.Response{data=merchantapp.MemberDTO}
//	@Failure		400		{object}	response.Response	"参数校验失败/角色不合法/已是成员/成员数达上限"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		403		{object}	response.Response	"是成员但不是店主"
//	@Failure		404		{object}	response.Response	"店铺不存在(非成员同此错误,不泄露存在性)"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/shops/{id}/members [post]
func (h *MerchantHandler) InviteMember(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	var req inviteMemberReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	dto, err := h.merchantSvc.InviteMember(c.Request.Context(), userID, parseUintParam(c, "id"), merchantapp.InviteMemberInput{
		TargetUserID: req.UserID,
		Role:         req.Role,
	})
	if err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "inviteMember", dto)
}

// RemoveMember 移除成员
//
//	@Summary		移除成员(店主)
//	@Description	把指定用户移出店铺。店主不可被移除(包括店主移除自己)。
//	@Tags			商户-成员管理
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path	uint	true	"店铺ID"
//	@Param			userId	path	uint	true	"目标成员的用户ID"
//	@Success		200		{object}	response.Response{data=string}
//	@Failure		400		{object}	response.Response	"不能移除店主"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		403		{object}	response.Response	"是成员但不是店主"
//	@Failure		404		{object}	response.Response	"店铺不存在或目标不是店铺成员"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/shops/{id}/members/{userId} [delete]
func (h *MerchantHandler) RemoveMember(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	if err := h.merchantSvc.RemoveMember(c.Request.Context(), userID, parseUintParam(c, "id"), parseUintParam(c, "userId")); err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "removeMember", "成员已移除")
}

// ChangeMemberRole 调整成员角色
//
//	@Summary		调整成员角色(店主)
//	@Description	管理员↔员工互换(2↔3)。店主角色不可调整,也不能授出新的店主。
//	@Tags			商户-成员管理
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path	uint					true	"店铺ID"
//	@Param			userId	path	uint					true	"目标成员的用户ID"
//	@Param			request	body	changeMemberRoleReq		true	"新角色"
//	@Success		200		{object}	response.Response{data=string}
//	@Failure		400		{object}	response.Response	"角色不合法/不能调整店主"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		403		{object}	response.Response	"是成员但不是店主"
//	@Failure		404		{object}	response.Response	"店铺不存在或目标不是店铺成员"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/merchant/shops/{id}/members/{userId}/role [put]
func (h *MerchantHandler) ChangeMemberRole(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c, "未登录或登录已过期")
		return
	}
	var req changeMemberRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	if err := h.merchantSvc.ChangeMemberRole(c.Request.Context(), userID, parseUintParam(c, "id"), parseUintParam(c, "userId"), merchantapp.ChangeMemberRoleInput{
		Role: req.Role,
	}); err != nil {
		writeMerchantError(c, err)
		return
	}
	response.SuccessMsg(c, "changeMemberRole", "角色已更新")
}
