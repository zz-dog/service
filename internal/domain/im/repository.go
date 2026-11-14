package im

import (
	"context"
	"time"
)

type ConversationRepository interface {
	// 不存在返回 ErrConversationNotFound;"没有会话"是正常业务态
	FindByUserAndMerchant(ctx context.Context, userID, merchantID uint64) (*Conversation, error)
	Create(ctx context.Context, c *Conversation) error
	// UpdateOnNewMessage 新消息到达时的会话侧更新:刷新快照 + 对侧未读自增。
	// 领域层只声明"要做什么",SQL 原子自增是持久层的翻译(阶段三)
	UpdateOnNewMessage(ctx context.Context, convID uint64, senderSide Side, content string, at time.Time) error
	ListByUser(ctx context.Context, userID uint64, limit int) ([]*Conversation, error)
	ListByMerchant(ctx context.Context, merchantID uint64, limit int) ([]*Conversation, error)
	FindByID(ctx context.Context, convID uint64) (*Conversation, error)
	// ClearUnread 我侧未读置零:身份列进 WHERE,越权校验和幂等一次解决
	ClearUnread(ctx context.Context, convID, selfID uint64, side Side) error
}

type MessageRepository interface {
	Insert(ctx context.Context, m *Message) (uint64, error) // 返回自增 ID,ack 要用
	// 游标分页:倒序取 limit 条,返回前反转成时间正序(聊天窗口按时间正序渲染)
	ListByConversation(ctx context.Context, convID uint64, beforeID uint64, limit int) ([]*Message, error)
}
