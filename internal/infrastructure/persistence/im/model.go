package im

import (
	"time"

	domainim "github.com/wsc-zz/service/internal/domain/im"
)

// ConversationPO 会话:买家和卖家之间的一对一聊天线程。
// 唯一键 (user_id, merchant_id) 是懒创建的并发守门员——
// 两人同时发首条消息,谁后插入谁撞唯一索引,撞了说明会话已在,重查即可。
type ConversationPO struct {
	ID uint64 `gorm:"primaryKey"`
	// uniqueIndex 建唯一键兜底懒创建并发;两个二级索引分别服务
	// C/B 两端会话列表的"按最近消息排序"扫描,列序 = 等值列在前、排序列在后
	UserID         uint64    `gorm:"uniqueIndex:idx_conv_pair;index:idx_user_last,priority:1"`
	MerchantID     uint64    `gorm:"uniqueIndex:idx_conv_pair;index:idx_merchant_last,priority:1"`
	LastMsgContent string    `gorm:"type:varchar(255)"` // 最近一条消息的快照,列表页不 JOIN 消息表
	LastMsgAt      time.Time `gorm:"index:idx_user_last,priority:2;index:idx_merchant_last,priority:2"`
	UserUnread     int32     // 买家未读
	MerchantUnread int32     // 卖家未读
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (ConversationPO) TableName() string { return "im_conversations" }

// MessagePO 消息:不可变日志,永不 UPDATE——每条消息 = 追加一行。
// (conversation_id, id) 复合索引是游标分页的完美形状:
// WHERE conversation_id=? AND id<? ORDER BY id DESC 直接沿索引倒序扫描。
type MessagePO struct {
	ID uint64 `gorm:"primaryKey;index:idx_conv_msg,priority:2"`
	// ConversationID 会话外键,逻辑关联(im 服务独占这两张表,不建物理外键)
	ConversationID uint64 `gorm:"index:idx_conv_msg,priority:1"`
	SenderSide     int8   // 谁发的:1 买家 2 卖家(领域枚举 Side 的落库形态)
	SenderID       uint64 // 发送者的 userId(两侧都是 userId)
	ContentType    int8   // 1 文本 2 图片
	Content        string `gorm:"type:varchar(512)"` // 文本;图片消息时是 OSS URL
	CreatedAt      time.Time
}

func (MessagePO) TableName() string { return "im_messages" }

func toConversationPO(c *domainim.Conversation) ConversationPO {
	return ConversationPO{
		ID:             c.ID,
		UserID:         c.UserID,
		MerchantID:     c.MerchantID,
		LastMsgContent: c.LastMsgContent,
		LastMsgAt:      c.LastMsgAt,
		UserUnread:     c.UserUnread,
		MerchantUnread: c.MerchantUnread,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

func toConversation(po ConversationPO) *domainim.Conversation {
	return &domainim.Conversation{
		ID:             po.ID,
		UserID:         po.UserID,
		MerchantID:     po.MerchantID,
		LastMsgContent: po.LastMsgContent,
		LastMsgAt:      po.LastMsgAt,
		UserUnread:     po.UserUnread,
		MerchantUnread: po.MerchantUnread,
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

func toMessagePO(m *domainim.Message) MessagePO {
	return MessagePO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderSide:     int8(m.SenderSide),
		SenderID:       m.SenderID,
		ContentType:    int8(m.ContentType),
		Content:        m.Content,
		CreatedAt:      m.CreatedAt,
	}
}

func toMessage(po MessagePO) *domainim.Message {
	return &domainim.Message{
		ID:             po.ID,
		ConversationID: po.ConversationID,
		SenderSide:     domainim.Side(po.SenderSide),
		SenderID:       po.SenderID,
		ContentType:    domainim.ContentType(po.ContentType),
		Content:        po.Content,
		CreatedAt:      po.CreatedAt,
	}
}
