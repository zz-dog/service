package imapp

import (
	"context"

	domainim "github.com/wsc-zz/service/internal/domain/im"
)

type MerchantResolver interface {
	// 找不到商户返回 (0, nil)——"不是商户"是正常业务态,不是错误
	MerchantIDByUserID(ctx context.Context, userID uint64) (uint64, error)
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
