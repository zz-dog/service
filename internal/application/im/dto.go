package imapp

import (
	"time"

	domainim "github.com/wsc-zz/service/internal/domain/im"
)

// ConversationDTO 会话视图。只回身份 ID + 快照,不带对端昵称——
// 对端名字在 identity/merchant 服务的表里,跨服务聚合查询是另一个课题,不混进列表接口。
// PeerID/Unread 是"我这一侧"的视角值,由应用层按 Actor 归一化,前端拿到直接渲染
type ConversationDTO struct {
	ID             uint64    `json:"id"`
	UserID         uint64    `json:"userId"` // 买家侧身份
	MerchantID     uint64    `json:"merchantId"`
	PeerID         uint64    `json:"peerId"` // 对端身份:买家视角=merchantId,卖家视角=userId
	LastMsgContent string    `json:"lastMsgContent"`
	LastMsgAt      time.Time `json:"lastMsgAt"`
	Unread         int32     `json:"unread"` // 我这一侧的未读数
	CreatedAt      time.Time `json:"createdAt"`
}

// MessageDTO 消息视图。SenderSide 原样回枚举(1 买家 2 卖家),
// 客户端拿它判断气泡在左还是右,不用再和服务端身份对表
type MessageDTO struct {
	ID             uint64    `json:"id"`
	ConversationID uint64    `json:"conversationId"`
	SenderSide     int8      `json:"senderSide"`
	SenderID       uint64    `json:"senderId"`
	ContentType    int8      `json:"contentType"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}

func toConversationDTO(c *domainim.Conversation, a Actor) ConversationDTO {
	peerID, _ := c.PeerOf(a.Side) // side 合法性由 Actor 构造侧保证,err 不可能
	dto := ConversationDTO{
		ID:             c.ID,
		UserID:         c.UserID,
		MerchantID:     c.MerchantID,
		PeerID:         peerID,
		LastMsgContent: c.LastMsgContent,
		LastMsgAt:      c.LastMsgAt,
		CreatedAt:      c.CreatedAt,
	}
	switch a.Side {
	case domainim.SideUser:
		dto.Unread = c.UserUnread
	case domainim.SideMerchant:
		dto.Unread = c.MerchantUnread
	}
	return dto
}

func toMessageDTO(m *domainim.Message) MessageDTO {
	return MessageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderSide:     int8(m.SenderSide),
		SenderID:       m.SenderID,
		ContentType:    int8(m.ContentType),
		Content:        m.Content,
		CreatedAt:      m.CreatedAt,
	}
}

func toMessageDTOs(msgs []*domainim.Message) []MessageDTO {
	dtos := make([]MessageDTO, 0, len(msgs))
	for _, m := range msgs {
		dtos = append(dtos, toMessageDTO(m))
	}
	return dtos
}
