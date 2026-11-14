package im

import "time"

// Side 发送方在会话的哪一侧。存库、走信封都是这个枚举——
// 别用 bool:sender_side=1/2 落在数据库和日志里,可读性完全不同
type Side int8

const (
	SideUser     Side = iota + 1 // 1 买家
	SideMerchant                 // 2 卖家
)

type ContentType int8

const (
	ContentText  ContentType = iota + 1 // 1 文本
	ContentImage                        // 2 图片(URL,扩展小练习)
)

// Message 消息:会话的从属实体,不可变——落库后没有任何字段会被修改
type Message struct {
	ID             uint64
	ConversationID uint64 // 所属会话
	SenderSide     Side   // 谁发的:1 买家 2 卖家
	SenderID       uint64 // 发送者的 userId(两侧都是 userId)
	ContentType    ContentType
	Content        string // 文本;图片消息时是 OSS URL
	CreatedAt      time.Time
}

// NewTextMessage 文本消息构造:领域规则收口在构造函数——
// 空内容、超长(教学版 500 字)在这里拒绝,服务层不用散落 if
func NewTextMessage(convID uint64, side Side, senderID uint64, content string) (*Message, error) {
	if len(content) == 0 {
		return nil, ErrContentEmpty
	}
	if len(content) > 500 {
		return nil, ErrContentTooLong
	}
	return &Message{
		ConversationID: convID,
		SenderSide:     side,
		SenderID:       senderID,
		ContentType:    ContentText,
		Content:        content,
	}, nil
}
