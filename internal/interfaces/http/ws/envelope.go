package ws

import (
	"encoding/json"
	"time"

	domainim "github.com/wsc-zz/service/internal/domain/im"
)

// Envelope 统一帧格式;Data 用 RawMessage 延迟解析——
// dispatch 先看 type 再解对应结构,未知 type 直接回 error 帧,不做全量反序列化。
// WS 部分不进 swagger:swag 只认 REST 路由语义,信封协议靠教程的表格维护
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// chatSendData C→S 发文本;peerId = 对端身份(买家填 merchantId,卖家填 userId)
type chatSendData struct {
	PeerID  uint64 `json:"peerId"`
	Content string `json:"content"`
}

// chatAckData S→C 落库成功的回执,客户端凭它把"发送中"变"已送达"
type chatAckData struct {
	ConversationID uint64    `json:"conversationId"`
	MsgID          uint64    `json:"msgId"`
	SentAt         time.Time `json:"sentAt"`
}

// chatPushData S→C 推给在线对端;senderSide 让客户端直接判断气泡方向
type chatPushData struct {
	ConversationID uint64    `json:"conversationId"`
	MsgID          uint64    `json:"msgId"`
	SenderSide     int8      `json:"senderSide"`
	SenderID       uint64    `json:"senderId"`
	Content        string    `json:"content"`
	SentAt         time.Time `json:"sentAt"`
}

// errorData 非法帧/落库失败/未知 type
type errorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null") // 结构体全是可序列化字段,这里到不了;兜底成合法 JSON
	}
	return b
}

// sideToPeerSide 推送目标是发送方的对侧:买家发 → 卖家收,反之亦然
func sideToPeerSide(side domainim.Side) domainim.Side {
	if side == domainim.SideUser {
		return domainim.SideMerchant
	}
	return domainim.SideUser
}
