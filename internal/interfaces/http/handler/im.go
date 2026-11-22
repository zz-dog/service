package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	imapp "github.com/wsc-zz/service/internal/application/im"
	domainim "github.com/wsc-zz/service/internal/domain/im"
	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
	"github.com/wsc-zz/service/pkg/response"
)

// ImHandler im 的 REST 入口:列表/历史/已读。
// 发消息和收推送走 WebSocket(见 internal/interfaces/http/ws),长短连接各司其职——
// 查询走 REST 无状态、curl 好调试,收发走 WS 服务端才能主动推
type ImHandler struct {
	svc *imapp.Service
}

func NewImHandler(svc *imapp.Service) *ImHandler {
	return &ImHandler{svc: svc}
}

// MyConversations 买家会话列表
//
//	@Summary		我的会话列表
//	@Description	买家侧会话,按最近消息排序。peerId=对面商户,unread=我的未读数。
//	                只回身份 ID 和快照,对端昵称前端自行聚合。
//	@Tags			消息
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			limit	query		int	false	"条数(默认20,上限50)"
//	@Success		200		{object}	response.Response{data=[]imapp.ConversationDTO}
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/im/client/conversations [get]
func (h *ImHandler) MyConversations(c *gin.Context) {
	actor := imapp.Actor{Side: domainim.SideUser, UserID: currentUint64(c)}
	h.listConversations(c, actor)
}

// MerchantConversations 卖家会话列表
//
//	@Summary		商家会话列表
//	@Description	商家工作台侧会话。"当前店铺"由 X-Merchant-Id 头声明,
//	                (userId, merchantId) 成员关系由 merchant RPC 实时校验,永不采信 body。
//	@Tags			消息
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Merchant-Id	header		int	true	"当前店铺ID"
//	@Param			limit	query		int	false	"条数(默认20,上限50)"
//	@Success		200		{object}	response.Response{data=[]imapp.ConversationDTO}
//	@Failure		400		{object}	response.Response	"缺少 X-Merchant-Id 店铺上下文头"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		403		{object}	response.Response	"还不是商户或无权访问该店铺"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/im/merchant/conversations [get]
func (h *ImHandler) MerchantConversations(c *gin.Context) {
	actor, ok := h.merchantActor(c)
	if !ok {
		return
	}
	h.listConversations(c, actor)
}

// MyMessages 买家拉历史消息
//
//	@Summary		历史消息(买家)
//	@Description	游标分页:before 传上一页最后一条的 ID,不传/传0=第一页;
//	                返回按时间正序。非会话成员返回 403。
//	@Tags			消息
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		path		int	true	"会话ID"
//	@Param			before	query		int	false	"翻页游标(上一页最后一条的消息ID)"
//	@Param			limit	query		int	false	"条数(默认20,上限50)"
//	@Success		200		{object}	response.Response{data=[]imapp.MessageDTO}
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		403		{object}	response.Response	"不是会话成员"
//	@Failure		404		{object}	response.Response	"会话不存在"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/im/client/conversations/{id}/messages [get]
func (h *ImHandler) MyMessages(c *gin.Context) {
	actor := imapp.Actor{Side: domainim.SideUser, UserID: currentUint64(c)}
	h.history(c, actor)
}

// MerchantMessages 卖家拉历史消息
//
//	@Summary		历史消息(商家)
//	@Description	同买家侧,身份视角换商户侧;"当前店铺"由 X-Merchant-Id 头声明并实时校验。
//	@Tags			消息
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Merchant-Id	header		int	true	"当前店铺ID"
//	@Param			id		path		int	true	"会话ID"
//	@Param			before	query		int	false	"翻页游标(上一页最后一条的消息ID)"
//	@Param			limit	query		int	false	"条数(默认20,上限50)"
//	@Success		200		{object}	response.Response{data=[]imapp.MessageDTO}
//	@Failure		400		{object}	response.Response	"缺少 X-Merchant-Id 店铺上下文头"
//	@Failure		401		{object}	response.Response	"未登录或 token 失效"
//	@Failure		403		{object}	response.Response	"不是会话成员/无权访问该店铺"
//	@Failure		404		{object}	response.Response	"会话不存在"
//	@Failure		500		{object}	response.Response	"服务器内部错误"
//	@Router			/im/merchant/conversations/{id}/messages [get]
func (h *ImHandler) MerchantMessages(c *gin.Context) {
	actor, ok := h.merchantActor(c)
	if !ok {
		return
	}
	h.history(c, actor)
}

// MyRead 买家标记已读
//
//	@Summary		标记已读(买家)
//	@Description	我侧未读清零;条件 UPDATE,重复调用幂等仍 200。
//	@Tags			消息
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		int	true	"会话ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		403	{object}	response.Response	"不是会话成员"
//	@Failure		404	{object}	response.Response	"会话不存在"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/im/client/conversations/{id}/read [post]
func (h *ImHandler) MyRead(c *gin.Context) {
	actor := imapp.Actor{Side: domainim.SideUser, UserID: currentUint64(c)}
	h.markRead(c, actor)
}

// MerchantRead 卖家标记已读
//
//	@Summary		标记已读(商家)
//	@Description	同买家侧,身份视角换商户侧;"当前店铺"由 X-Merchant-Id 头声明并实时校验。
//	@Tags			消息
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			X-Merchant-Id	header		int	true	"当前店铺ID"
//	@Param			id	path		int	true	"会话ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response	"缺少 X-Merchant-Id 店铺上下文头"
//	@Failure		401	{object}	response.Response	"未登录或 token 失效"
//	@Failure		403	{object}	response.Response	"不是会话成员/无权访问该店铺"
//	@Failure		404	{object}	response.Response	"会话不存在"
//	@Failure		500	{object}	response.Response	"服务器内部错误"
//	@Router			/im/merchant/conversations/{id}/read [post]
func (h *ImHandler) MerchantRead(c *gin.Context) {
	actor, ok := h.merchantActor(c)
	if !ok {
		return
	}
	h.markRead(c, actor)
}

// ---- 公共胶水:两个端点的差异只有 Actor 的组装,收发逻辑一份 ----

func (h *ImHandler) listConversations(c *gin.Context, actor imapp.Actor) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	dtos, err := h.svc.ListConversations(c.Request.Context(), actor, limit)
	if err != nil {
		writeImError(c, err)
		return
	}
	response.SuccessMsg(c, "conversations", dtos)
}

func (h *ImHandler) history(c *gin.Context, actor imapp.Actor) {
	convID := parseUint64Param(c, "id")
	before, _ := strconv.ParseUint(c.Query("before"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	dtos, err := h.svc.History(c.Request.Context(), actor, convID, before, limit)
	if err != nil {
		writeImError(c, err)
		return
	}
	response.SuccessMsg(c, "messages", dtos)
}

func (h *ImHandler) markRead(c *gin.Context, actor imapp.Actor) {
	convID := parseUint64Param(c, "id")
	if err := h.svc.MarkRead(c.Request.Context(), actor, convID); err != nil {
		writeImError(c, err)
		return
	}
	response.SuccessMsg(c, "read", nil)
}

// merchantActor B 端身份组装:X-Merchant-Id 头声明"以哪家店的身份工作",
// (userId, merchantId) 成员关系由 ResolveMerchant 经 merchant RPC 实时校验——
// 铁律第 6 条的 N:N 版:身份/归属不进 token 也不信 body,伪造 merchantId 毫无意义。
// 缺头 → 400;非成员/店不存在 → 403(统一口径,不泄露存在性)
func (h *ImHandler) merchantActor(c *gin.Context) (imapp.Actor, bool) {
	uid := currentUint64(c)
	mid, ok := getCurrentMerchantID(c)
	if !ok {
		response.BadRequest(c, "缺少 " + HeaderMerchantID + " 店铺上下文头")
		return imapp.Actor{}, false
	}
	merchantID, err := h.svc.ResolveMerchant(c.Request.Context(), uid, uint64(mid))
	if err != nil {
		if errors.Is(err, domainmerchant.ErrMerchantNotFound) {
			response.Forbidden(c, "还不是商户或无权访问该店铺")
			return imapp.Actor{}, false
		}
		response.ServerError(c, "商户身份校验失败")
		return imapp.Actor{}, false
	}
	return imapp.Actor{Side: domainim.SideMerchant, UserID: uid, MerchantID: merchantID}, true
}

// writeImError 领域错误 → HTTP 状态码:
// 越权 403(不藏进 404,前端要能区分"被拦"和"没有"),
// 不存在 404,内容/参数非法 400,其余一律 500
func writeImError(c *gin.Context, err error) {
	switch err {
	case domainim.ErrNotParticipant:
		response.Forbidden(c, err.Error())
	case domainim.ErrConversationNotFound:
		response.NotFound(c, err.Error())
	case domainim.ErrContentEmpty, domainim.ErrContentTooLong, domainim.ErrInvalidSide:
		response.BadRequest(c, err.Error())
	default:
		response.ServerError(c, err.Error())
	}
}

// currentUint64 取 JWT 中间件写入的 userID(上下文里是 string,转 uint64)
func currentUint64(c *gin.Context) uint64 {
	v, exists := c.Get("userId")
	if !exists {
		return 0
	}
	s, _ := v.(string)
	id, _ := strconv.ParseUint(s, 10, 64)
	return id
}

// parseUint64Param 解析路径段数字 ID,非法返回 0——0 查不到记录,自然落到 404
func parseUint64Param(c *gin.Context, name string) uint64 {
	id, _ := strconv.ParseUint(c.Param(name), 10, 64)
	return id
}
