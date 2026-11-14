package im

import "errors"

var (
	// ErrConversationNotFound 会话不存在
	ErrConversationNotFound = errors.New("会话不存在")
	// ErrNotParticipant 越权拉别人会话
	ErrNotParticipant = errors.New("越权拉别人会话")
	// ErrMessageNotFound 消息不存在
	ErrMessageNotFound = errors.New("消息不存在")
	// ErrContentTooLong 消息内容过长
	ErrContentTooLong = errors.New("消息内容过长")
	// ErrContentEmpty 消息内容为空
	ErrContentEmpty = errors.New("消息内容为空")
	// ErrInvalidSide 发送方身份不合法
	ErrInvalidSide = errors.New("发送方身份不合法")
)
