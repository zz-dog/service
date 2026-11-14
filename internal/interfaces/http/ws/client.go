package ws

import (
	"time"

	"github.com/gorilla/websocket"

	appim "github.com/wsc-zz/service/internal/application/im"
)

const (
	writeWait  = 10 * time.Second  // 单次写超时
	pongWait   = 60 * time.Second  // 多久没 pong 判死
	pingPeriod = pongWait * 9 / 10 // ping 间隔略短于判死线:丢一拍还有救
	sendBuf    = 16                // send 通道缓冲,避免写泵被阻塞时阻塞业务协程
	maxMsgSize = 4096              // readLimit:防恶意超大帧撑爆内存(文本消息上限 500 字,4KB 足够宽裕)
)

// Client 一条长连接:身份 + 连接 + 单写通道。
// gorilla 的黄金法则——conn 不支持并发写:读归 readPump,
// 所有写(chat.ack/chat.push/error 帧、ping)都走 send 通道汇进 writePump
type Client struct {
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte // 单写通道:writePump 独占 conn 的写权限
	actor appim.Actor
}

func newClient(hub *Hub, conn *websocket.Conn, actor appim.Actor) *Client {
	return &Client{hub: hub, conn: conn, send: make(chan []byte, sendBuf), actor: actor}
}

// readPump 读泵:占住握手请求的协程,连接的生命周期由它守护。
// 无论怎么退出(断开/超时/被顶),清理只有 defer 这一处
func (c *Client) readPump() {
	defer func() { c.hub.Unregister(c); c.conn.Close() }()
	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// 浏览器收到 ping 自动回 pong,pong 一到就续期读死线——判死计时器被心跳持续喂着。
	// 心跳是判死,不是保活:拔网线的死连接 TCP 层不报错,全靠这个死线兜住
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return // 断开/超时/被顶,统一从这退出
		}
		c.hub.dispatch(c, raw)
	}
}

// writePump 写泵:唯一有资格写 conn 的协程。两个来源汇进 select:
// send 通道的业务帧 + ticker 的 ping(心跳)
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() { ticker.Stop(); c.conn.Close() }()
	for {
		select {
		case payload, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				return // send 被关闭,连接作废
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
