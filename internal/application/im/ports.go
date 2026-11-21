package imapp

import (
	"context"

	domainim "github.com/wsc-zz/service/internal/domain/im"
)

// MerchantResolver 商户成员关系校验端口(N:N):
// B 端请求/握手显式声明 merchantId(REST 走 X-Merchant-Id 头,WS 走 query),
// 这里实时校验 (userID, merchantID) 的成员关系——归属不进 token,也永不采信 body。
type MerchantResolver interface {
	// Authorize 校验成员关系;非成员/店不存在返回 domainmerchant.ErrMerchantNotFound
	Authorize(ctx context.Context, userID, merchantID uint64) error
}

// Notifier 推送端口:落库成功后通知"对侧在线的连接"。
// 应用层只依赖这个接口,websocket 的一切(hub、连接、信封)关在 interfaces 层——
// 依赖倒置,和 JWTAuth 的 TokenBlacklistChecker 同款
type Notifier interface {
	NotifyMessage(conv *domainim.Conversation, msg *domainim.Message)
}

// TxRunner 事务端口(应用层定义契约,infrastructure/persistence/transaction 提供):
// SendText 的三步(建会话/插消息/刷会话)必须同生共死,普通读和单步写不走它。
// 事务是资源,按需用——与 merchant 应用的 TxRunner 同款
type TxRunner interface {
	RunIm(ctx context.Context, fn func(domainim.ConversationRepository, domainim.MessageRepository) error) error
}
