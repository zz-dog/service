package handler

import (
	"github.com/gin-gonic/gin"
	domainproduct "github.com/wsc-zz/service/internal/domain/product"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// 管理端商品审核接口:挂在 catalog 的 /admin 路由组(JWTAuth + AdminAuth 一处收口)。
// 管理端既有 CRUD 保留兜底(自营+善后),本文件只放审核新增:队列/审核/违规下架。

type pendingListReq struct {
	Name     string `form:"name"`
	Page     int    `form:"page" binding:"required"`
	PageSize int    `form:"pageSize" binding:"required"`
}

// AdminPendingList 待审核列表
//
//	@Summary		待审核列表(管理端)
//	@Description	商户提交的待审核商品,审核队列;自营商品不走审核,不在此列
//	@Tags			管理端-商品审核
//	@Produce		json
//	@Param			name		query		string	false	"商品名称(模糊匹配)"
//	@Param			page		query		int		true	"页码"
//	@Param			pageSize	query		int		true	"每页数量"
//	@Success		200			{object}	response.Response{data=productapp.ProductListResult}
//	@Failure		400			{object}	response.Response	"查询参数校验失败"
//	@Failure		500			{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/admin/product/pending [get]
func (h *ProductHandler) AdminPendingList(c *gin.Context) {
	var req pendingListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	result, err := h.productSvc.PendingList(c.Request.Context(), domainproduct.ListQuery{
		Name:     req.Name,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, result)
}

type reviewReq struct {
	// 指针:JSON 不传 approve 和传 false 必须区分开——
	// binding:"required" 对 bool 零值会误伤语义正确的 false,指针判 nil 才严谨
	Approve *bool  `json:"approve" binding:"required"`
	Reason  string `json:"reason"`
}

// AdminReview 审核
//
//	@Summary		审核商品(管理端)
//	@Description	对待审核商品做出通过/驳回决定;驳回理由必填,商户在"我的商品"可见
//	@Tags			管理端-商品审核
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"商品ID"
//	@Param			request	body		reviewReq	true	"审核决定"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response	"商品不存在/当前状态不允许审核/驳回理由不能为空"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/admin/product/{id}/review [post]
func (h *ProductHandler) AdminReview(c *gin.Context) {
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	var req reviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	err := h.productSvc.Review(c.Request.Context(), id, *req.Approve, req.Reason)
	if err != nil {
		if isBadRequest(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

type takedownReq struct {
	Reason string `json:"reason" binding:"required"`
}

// AdminTakedown 违规下架
//
//	@Summary		违规下架(管理端)
//	@Description	对已上架商品做违规处置(理由必填);违规下架是终态,商户不可自行恢复
//	@Tags			管理端-商品审核
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"商品ID"
//	@Param			request	body		takedownReq	true	"处置理由"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response	"商品不存在/仅已上架可处置/理由不能为空"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/catalog/admin/product/{id}/takedown [post]
func (h *ProductHandler) AdminTakedown(c *gin.Context) {
	id := parseUintParam(c, "id")
	if id == 0 {
		response.BadRequest(c, "商品ID格式错误")
		return
	}
	var req takedownReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	err := h.productSvc.Takedown(c.Request.Context(), id, req.Reason)
	if err != nil {
		if isBadRequest(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.ServerError(c, err.Error())
		return
	}
	response.Success(c, nil)
}
