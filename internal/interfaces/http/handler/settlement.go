package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	settlementapp "github.com/wsc-zz/service/internal/application/settlement"
	domainsettlement "github.com/wsc-zz/service/internal/domain/settlement"
	"github.com/wsc-zz/service/pkg/response"
	"github.com/wsc-zz/service/pkg/validator"
)

// settlementHandler 分账管理端 handler。只服务管理员:查账、干预卡死的单子。
// 用户侧"我的分账"是进阶 2(要做 merchantId 服务端推导,防伪造越权)。
type settlementHandler struct {
	settleSvc *settlementapp.Service
}

func NewSettlementHandler(settleSvc *settlementapp.Service) *settlementHandler {
	return &settlementHandler{settleSvc: settleSvc}
}

type listSettlementsRequest struct {
	Status   int `form:"status" binding:"omitempty,min=1,max=4"` // 0/不传 = 不过滤
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// List 分账单列表(含明细)
//
//	@Summary		分账单列表
//	@Description	全部分账单,时间倒序含明细,可按状态过滤。对账入口:核对 settle_amount + commission_amount == total_amount。需要管理员(JWT+AdminAuth)
//	@Tags			分账管理
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			status		query		int	false	"状态:不传或 0 查全部,1-待结算,2-结算中,3-已结算,4-结算失败"
//	@Param			page		query		int	false	"页码,默认 1"
//	@Param			pageSize	query		int	false	"每页数量,默认 10,最大 100"
//	@Success		200			{object}	response.Response{data=settlementapp.SettlementListDto}
//	@Failure		400			{object}	response.Response	"参数校验失败"
//	@Failure		401			{object}	response.Response	"未登录或 token 失效"
//	@Failure		403			{object}	response.Response	"需要管理员权限"
//	@Failure		500			{object}	response.Response	"服务器内部错误"
//	@Router			/payment/admin/settlements [get]
func (h *settlementHandler) List(c *gin.Context) {
	var req listSettlementsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, validator.ErrorMsg(err))
		return
	}
	// 翻页默认值:不传给个合理起点(binding 是 omitempty,零值放行到这)
	page, pageSize := req.Page, req.PageSize
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}
	result, err := h.settleSvc.List(c.Request.Context(), domainsettlement.ListQuery{
		Status:   req.Status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.SuccessMsg(c, "settlements", result)
}

// Execute 立即执行
//
//	@Summary		立即执行分账
//	@Description	不等 job,立刻执行一张待结算分账单。与 job 共用抢锁语义(条件更新 1→2):
//	                并发调用只有一个成功,另一个收到"分账单正在结算中"。需要管理员(JWT+AdminAuth)
//	@Tags			分账管理
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			settleNo	path		string				true	"结算单号"
//	@Success		200			{object}	response.Response	"结算完成"
//	@Failure		400			{object}	response.Response	"当前状态不允许该操作(非待结算)"
//	@Failure		401			{object}	response.Response	"未登录或 token 失效"
//	@Failure		403			{object}	response.Response	"需要管理员权限"
//	@Failure		404			{object}	response.Response	"分账单不存在"
//	@Failure		423			{object}	response.Response	"分账单正在结算中(别人在跑,不是错误)"
//	@Failure		500			{object}	response.Response	"服务器内部错误"
//	@Router			/payment/admin/settlements/{settleNo}/execute [post]
func (h *settlementHandler) Execute(c *gin.Context) {
	if err := h.settleSvc.Execute(c.Request.Context(), c.Param("settleNo")); err != nil {
		h.mapError(c, err)
		return
	}
	response.SuccessMsg(c, "execute", "结算完成")
}

// Retry 失败单重置
//
//	@Summary		重试失败的分账单
//	@Description	结算失败(终态,重试超限)→ 待结算,单子回到 job 的扫描范围。
//	                前提是先解决失败原因(如 mock_fail 开关),否则只会再失败一轮。需要管理员(JWT+AdminAuth)
//	@Tags			分账管理
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			settleNo	path		string				true	"结算单号"
//	@Success		200			{object}	response.Response	"已重置回待结算"
//	@Failure		400			{object}	response.Response	"当前状态不允许该操作(非结算失败终态)"
//	@Failure		401			{object}	response.Response	"未登录或 token 失效"
//	@Failure		403			{object}	response.Response	"需要管理员权限"
//	@Failure		404			{object}	response.Response	"分账单不存在"
//	@Failure		500			{object}	response.Response	"服务器内部错误"
//	@Router			/payment/admin/settlements/{settleNo}/retry [post]
func (h *settlementHandler) Retry(c *gin.Context) {
	if err := h.settleSvc.Retry(c.Request.Context(), c.Param("settleNo")); err != nil {
		h.mapError(c, err)
		return
	}
	response.SuccessMsg(c, "retry", "已重置回待结算,等待 job 执行")
}

// mapError 分账领域错误 → HTTP 语义。抢占失败(ErrSettleExecuting)是并发竞争的正常
// 结果,用 423 Locked 而不是 500——重试有意义,500 会误导调用方以为系统坏了
func (h *settlementHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainsettlement.ErrSettlementNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, domainsettlement.ErrInvalidStatusTransition):
		response.BadRequest(c, err.Error())
	case errors.Is(err, domainsettlement.ErrSettleExecuting):
		response.JSON(c, http.StatusLocked, err.Error(), nil) // 非标准状态码,走底层 JSON
	default:
		response.ServerError(c, err.Error())
	}
}
