package im

import "time"

// Conversation 会话:买家和卖家之间的一对一聊天线程。
type Conversation struct {
	ID             uint64
	UserID         uint64 // 买家侧身份
	MerchantID     uint64 // 卖家侧身份
	LastMsgContent string // 最近一条消息的快照:列表页不 JOIN 消息表就能渲染
	LastMsgAt      time.Time
	UserUnread     int32 // 买家未读
	MerchantUnread int32 // 卖家未读
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewConversation(userID, merchantID uint64) *Conversation {
	return &Conversation{
		UserID:     userID,
		MerchantID: merchantID,
	}
}

// PeerOf 我这一侧对面是谁:买家会话对面是 merchantId,卖家会话对面是 userId。
// 发消息的 peerId 归一化、推送目标定位都靠它——会话知道自己的两端,
// 外面不用到处 if side==user,这是把"两端"这个知识收口进聚合
func (c *Conversation) PeerOf(side Side) (uint64, error) {
	switch side {
	case SideUser:
		return c.MerchantID, nil
	case SideMerchant:
		return c.UserID, nil
	default:
		return 0, ErrInvalidSide
	}
}

// BelongsTo 成员校验:这一侧的这个人是不是本会话的参与者。
// 历史、已读都必须过它——水平越权的生命线,admin-user-split 的同款纪律
func (c *Conversation) BelongsTo(side Side, selfID uint64) bool {
	switch side {
	case SideUser:
		return c.UserID == selfID
	case SideMerchant:
		return c.MerchantID == selfID
	default:
		return false
	}
}
