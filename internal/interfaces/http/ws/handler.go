package ws

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/wsc-zz/service/global"
	appim "github.com/wsc-zz/service/internal/application/im"
	domainim "github.com/wsc-zz/service/internal/domain/im"
	domainmerchant "github.com/wsc-zz/service/internal/domain/merchant"
	"github.com/wsc-zz/service/internal/infrastructure/auth"
	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"
	"github.com/wsc-zz/service/pkg/response"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // 开发期从宽;生产要校验 Origin 防跨站 WS 劫持
}

// WSHandler WS 握手层:鉴权全部发生在 Upgrade 之前——
// 拒绝必须发生在升级之前,一旦 Hijack 升级,HTTP 语义就没了,想拒都拒不成。
// 不复用 JWTAuth 中间件:它只认 Authorization 头,而浏览器 WS 带不了 header,
// token 只能放 query;不为 WS 去改通用中间件,在握手 handler 里内联鉴权
type WSHandler struct {
	hub       *Hub
	im        *appim.Service
	blacklist middleware.TokenBlacklistChecker
}

func NewWSHandler(hub *Hub, svc *appim.Service, blacklist middleware.TokenBlacklistChecker) *WSHandler {
	return &WSHandler{hub: hub, im: svc, blacklist: blacklist}
}

// UserWS 买家端握手 GET /api/im/client/ws?token=xxx
func (h *WSHandler) UserWS(c *gin.Context) {
	actor, ok := h.userActor(c.Request.Context(), c.Query("token"))
	if !ok {
		// 升级前的拒绝是普通 HTTP:401 照常回 JSON
		response.Unauthorized(c, "token无效")
		return
	}
	h.serve(c, actor)
}

// MerchantWS 商家工作台握手 GET /api/im/merchant/ws?token=xxx&merchantId=123。
// N:N 下"以哪家店的身份工作"必须显式声明;浏览器 WS 带不了自定义 header,
// 店铺上下文只能走 query——query 同样只是"你想用哪家店",
// 能不能用由 merchant RPC 实时校验 (userId, merchantId) 成员关系,采信的只是声明本身
func (h *WSHandler) MerchantWS(c *gin.Context) {
	actor, status, msg := h.merchantActor(c.Request.Context(), c.Query("token"), c.Query("merchantId"))
	if status != http.StatusOK {
		c.JSON(status, gin.H{"code": status, "message": msg})
		return
	}
	h.serve(c, actor)
}

// serve 鉴权通过后的公共路径:升级 → 注册连接表 → 双泵起飞
func (h *WSHandler) serve(c *gin.Context, actor appim.Actor) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // Upgrade 失败时 gorilla 已自带 HTTP 错误响应
	}
	client := newClient(h.hub, conn, actor)
	h.hub.Register(client)
	go client.writePump() // 写泵独立协程
	client.readPump()     // 读泵占住当前请求协程:连接断开它才返回
	global.Logger.Info("im 连接关闭",
		zap.Int8("side", int8(actor.Side)), zap.Uint64("self_id", actor.SelfID()))
}

// userActor 买家端身份:JWT 解析 + 黑名单检查,通过则组装 Actor{SideUser, userID}。
// 手动做而不用 JWTAuth:token 在 query 里,没有 "Bearer " 前缀那套 header 格式
func (h *WSHandler) userActor(ctx context.Context, token string) (appim.Actor, bool) {
	claims, err := auth.ParseToken(token)
	if err != nil {
		return appim.Actor{}, false
	}
	if h.revoked(ctx, claims) {
		return appim.Actor{}, false
	}
	uid, err := strconv.ParseUint(claims.UserID, 10, 64)
	if err != nil || uid == 0 {
		return appim.Actor{}, false
	}
	return appim.Actor{Side: domainim.SideUser, UserID: uid}, true
}

// merchantActor 卖家端身份:JWT + 黑名单之后,再实时校验 (userId, merchantId) 成员关系;
// 缺 merchantId → 400,非成员/店不存在 → 403(统一口径,不泄露存在性)。返回 (actor, httpStatus, msg)
func (h *WSHandler) merchantActor(ctx context.Context, token, merchantIDQuery string) (appim.Actor, int, string) {
	claims, err := auth.ParseToken(token)
	if err != nil {
		return appim.Actor{}, http.StatusUnauthorized, "token无效"
	}
	if h.revoked(ctx, claims) {
		return appim.Actor{}, http.StatusUnauthorized, "token已失效，请重新登录"
	}
	uid, err := strconv.ParseUint(claims.UserID, 10, 64)
	if err != nil || uid == 0 {
		return appim.Actor{}, http.StatusUnauthorized, "token无效"
	}
	mid, err := strconv.ParseUint(merchantIDQuery, 10, 64)
	if err != nil || mid == 0 {
		return appim.Actor{}, http.StatusBadRequest, "缺少 merchantId 店铺上下文"
	}
	merchantID, err := h.im.ResolveMerchant(ctx, uid, mid)
	if err != nil {
		if errors.Is(err, domainmerchant.ErrMerchantNotFound) {
			return appim.Actor{}, http.StatusForbidden, "还不是商户或无权访问该店铺"
		}
		global.Logger.Error("im 握手校验商户身份失败", zap.Uint64("user_id", uid), zap.Uint64("merchant_id", mid), zap.Error(err))
		return appim.Actor{}, http.StatusInternalServerError, "商户身份校验失败"
	}
	return appim.Actor{Side: domainim.SideMerchant, UserID: uid, MerchantID: merchantID}, http.StatusOK, ""
}

// revoked 黑名单检查:握手期请求还活着,用它的 ctx;Redis 故障 fail-open
// (与 JWTAuth 中间件同款取舍,黑名单是安全加速,不是可用性闸门)
func (h *WSHandler) revoked(ctx context.Context, claims *auth.UserClaims) bool {
	if claims.ID == "" || h.blacklist == nil {
		return false
	}
	r, err := h.blacklist.IsRevoked(ctx, claims.ID)
	if err != nil {
		global.Logger.Warn("im 握手查询token黑名单失败,降级放行", zap.Error(err))
		return false
	}
	return r
}
