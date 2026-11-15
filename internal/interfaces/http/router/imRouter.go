package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wsc-zz/service/global"

	imapp "github.com/wsc-zz/service/internal/application/im"
	impo "github.com/wsc-zz/service/internal/infrastructure/persistence/im"
	"github.com/wsc-zz/service/internal/infrastructure/persistence/transaction"
	"github.com/wsc-zz/service/internal/interfaces/http/handler"
	middleware "github.com/wsc-zz/service/internal/interfaces/http/middleware"
	"github.com/wsc-zz/service/internal/interfaces/http/ws"
	"github.com/wsc-zz/service/internal/interfaces/rpc/merchantrpc"
)

// InitImRouter im 服务的路由入口:聊天挂在 /api/im 前缀下。
// 网关按最长前缀把 /api/im/** 转发到 im 服务(标准库 ReverseProxy 原生透传
// WebSocket 升级),因此路径必须带此前缀;WS 握手也走 9000 网关端口
func InitImRouter() *gin.Engine {
	r := gin.Default()
	r.Use(CORSMiddleware())
	api := r.Group("/api/im")

	// ---- 装配:依赖单向,无循环 ----
	// 仓储 → Hub → Service(Notifier=Hub,Hub 实现了端口)→ SetService 回填:
	// Hub.dispatch 要调 Service,Service 要调 Notifier(=Hub),
	// 接口定义在 application 包,两者互不 import 对方的具体类型
	convRepo := impo.NewConversationRepository(global.DB)
	msgRepo := impo.NewMessageRepository(global.DB)
	runner := transaction.NewRunner(global.DB)
	resolver := newMerchantResolver()
	blacklist := NewTokenBlacklist()

	hub := ws.NewHub()
	svc := imapp.NewService(convRepo, msgRepo, runner, resolver, hub)
	hub.SetService(svc)

	h := handler.NewImHandler(svc)
	wsh := ws.NewWSHandler(hub, svc, blacklist)

	// ---- WS 握手:鉴权在握手 handler 内联(token 走 query,JWTAuth 中间件
	// 只认 header),所以注册在 JWT 组外;路径与 REST 组不冲突 ----
	api.GET("/client/ws", wsh.UserWS)       // 买家端(/client 段与 /merchant 对称)
	api.GET("/merchant/ws", wsh.MerchantWS) // 商家工作台

	// ---- C 端(买家):列表/历史/已读,全部 JWT;挂 /client 段,与 /merchant 对称 ----
	user := api.Group("/client", middleware.JWTAuth(blacklist))
	{
		user.GET("/conversations", h.MyConversations)
		user.GET("/conversations/:id/messages", h.MyMessages)
		user.POST("/conversations/:id/read", h.MyRead)
	}

	// ---- B 端(商家工作台):同构三接口,视角换商户侧。
	// merchantId 在 handler 里由 token 推导,路由层只管认证——
	// 拆的是路由视角,不是服务,C/B 共用同一个 Service(与 catalog 同款) ----
	merchant := api.Group("/merchant", middleware.JWTAuth(blacklist))
	{
		merchant.GET("/conversations", h.MerchantConversations)
		merchant.GET("/conversations/:id/messages", h.MerchantMessages)
		merchant.POST("/conversations/:id/read", h.MerchantRead)
	}
	return r
}

// merchantResolver 把 merchantrpc 客户端适配成 im 应用的 MerchantResolver 端口:
// 协议翻译,不做策略——merchant_id=0 原样返回,"不是商户"的裁决在应用层/握手层
// (依赖倒置,application 不 import rpc 客户端的具体类型)
type merchantResolver struct{ gw *merchantrpc.Client }

func newMerchantResolver() merchantResolver {
	// 商户身份推导低频且握手/请求期一次调用,3s 超时足够;装配失败快速 panic
	gw, err := merchantrpc.NewClient(global.Conf.Merchant.RpcAddr, 3*time.Second)
	if err != nil {
		panic(err)
	}
	return merchantResolver{gw: gw}
}

func (r merchantResolver) MerchantIDByUserID(ctx context.Context, userID uint64) (uint64, error) {
	m, err := r.gw.GetMerchantByUserID(ctx, uint(userID))
	if err != nil {
		return 0, err
	}
	return uint64(m.MerchantID), nil
}
