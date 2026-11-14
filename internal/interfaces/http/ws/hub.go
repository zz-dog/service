package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	appim "github.com/wsc-zz/service/internal/application/im"
	domainim "github.com/wsc-zz/service/internal/domain/im"
)

// Hub 在线连接表:key 是 "u:7"/"m:45",value 是该身份当前唯一的连接。
// 单端单连接:同一身份再连一条,顶掉旧的(单设备假设;多设备是进阶项)。
// 内存态,重启即空——持久化是兜底,推送只是加速送达(关键决定 8)
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client

	im *appim.Service // dispatch 落库用;组合根在 Service 构造后注入(SetService)
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]*Client)}
}

// SetService 注入应用服务:组合根装配顺序是 仓储 → Hub → Service(Notifier=Hub)→
// SetService 回填——Hub 实现 Notifier 不需要 Service,dispatch 才需要,
// 这样绕开"Service 构造要 Hub、Hub 构造要 Service"的死结,依赖仍是单向的
func (h *Hub) SetService(svc *appim.Service) { h.im = svc }

// clientKey 连接表的 key:侧别前缀 + 身份 ID。
// 同一个 userId 可以同时有买家连接(u:7)和卖家连接(m:45)——C/B 双身份双连接
func clientKey(side domainim.Side, id uint64) string {
	if side == domainim.SideMerchant {
		return fmt.Sprintf("m:%d", id)
	}
	return fmt.Sprintf("u:%d", id)
}

func (c *Client) key() string { return clientKey(c.actor.Side, c.actor.SelfID()) }

// Register 注册连接。同一身份旧连接还在,直接关掉顶出——
// 旧连接的 readPump 因读错误退出,自己走 Unregister 清理,这里不用管它的收尾
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	if old, ok := h.clients[c.key()]; ok && old != c {
		old.conn.Close()
	}
	h.clients[c.key()] = c
	h.mu.Unlock()
}

// Unregister 注销连接。只删"表里挂的还是我"的情况:旧连接被顶后的
// 清理协程会迟到——不做这个判断,迟到的 Unregister 会把新连接从表里误删
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if cur, ok := h.clients[c.key()]; ok && cur == c {
		delete(h.clients, c.key())
	}
	h.mu.Unlock()
}

// PushToPeer 推给对侧在线连接。不在线直接返回——
// 未读计数已落库,推送只是加速送达,不是送达的必要条件
func (h *Hub) PushToPeer(side domainim.Side, peerID uint64, payload []byte) {
	h.mu.RLock()
	c := h.clients[clientKey(side, peerID)]
	h.mu.RUnlock()
	if c == nil {
		return
	}
	select {
	case c.send <- payload:
	default: // 发送缓冲满 = 连接假死:丢这帧不断连,真死了心跳会清理
	}
}

// NotifyMessage 实现 application/im.Notifier:组装推送信封,投给对侧。
// 应用层只认识 Notifier 接口,websocket 的存在感被压进这一个方法
func (h *Hub) NotifyMessage(conv *domainim.Conversation, msg *domainim.Message) {
	peerID, err := conv.PeerOf(msg.SenderSide) // 会话知道自己两端,推送目标不用 if/else
	if err != nil {
		return
	}
	payload, _ := json.Marshal(Envelope{
		Type: "chat.push",
		Data: mustJSON(chatPushData{
			ConversationID: conv.ID,
			MsgID:          msg.ID,
			SenderSide:     int8(msg.SenderSide),
			SenderID:       msg.SenderID,
			Content:        msg.Content,
			SentAt:         msg.CreatedAt,
		}),
	})
	h.PushToPeer(sideToPeerSide(msg.SenderSide), peerID, payload)
}

// dispatch 挂在 readPump 的循环体里:拆信封 → 按 type 路由。
// 业务的复杂性全在应用服务里,连接层只做信封拆装——
// 校验/落库/未读/推送都在 SendText,这里五行胶水
func (h *Hub) dispatch(c *Client, raw []byte) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		h.reply(c, "error", errorData{Code: "bad_frame", Message: "帧格式非法"})
		return
	}
	switch env.Type {
	case "chat.send":
		var d chatSendData
		if err := json.Unmarshal(env.Data, &d); err != nil || d.Content == "" {
			h.reply(c, "error", errorData{Code: "bad_send", Message: "参数非法"})
			return
		}
		// ctx 用 Background 而不是连接的请求 ctx:readPump 是长生命周期协程,
		// 握手的请求早已返回,那个 ctx 已取消——拿着已取消的 ctx 调落库,MySQL 调用直接失败
		conv, msg, err := h.im.SendText(context.Background(), c.actor, d.PeerID, d.Content)
		if err != nil {
			h.reply(c, "error", errorData{Code: "send_failed", Message: err.Error()})
			return
		}
		h.reply(c, "chat.ack", chatAckData{ConversationID: conv.ID, MsgID: msg.ID, SentAt: msg.CreatedAt})
		// 对侧推送不用这里管:SendText 内部已经 Notifier→NotifyMessage→PushToPeer 走完
	default:
		h.reply(c, "error", errorData{Code: "unknown_type", Message: "未知帧类型"})
	}
}

func (h *Hub) reply(c *Client, typ string, data any) {
	payload, err := json.Marshal(Envelope{Type: typ, Data: mustJSON(data)})
	if err != nil {
		log.Printf("im ws: 组帧失败 type=%s err=%v", typ, err)
		return
	}
	select {
	case c.send <- payload:
	default: // 缓冲满丢帧:回执类帧丢了客户端有超时兜底,不值得为它堵死连接
	}
}
