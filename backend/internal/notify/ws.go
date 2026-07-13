// Package notify 实现所有通知渠道：WebSocket / 邮件 / 飞书 / 钉钉。
//
// 阶段 6 只做 WebSocket。阶段 7 会把邮件和 Webhook 也加进来。
//
// 设计目标：
//   - 一个 Notifier 接口统一所有渠道
//   - WebSocket 是同步广播（在线才推送）；邮件/Webhook 是异步推送
//   - 互相不影响，单独失败不影响其他渠道
package notify

// 导入：
// - log: 打印推送错误，但不影响主流程
// - sync: RWMutex 保护 Hub 的 connections 集合
// - time: 心跳
// - gorilla/websocket: WebSocket 库
import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// ===== 0. Notifier 接口 =====

// Notifier 是统一通知接口。
// 所有渠道（WS / 邮件 / 飞书 / 钉钉）都实现它，业务代码只依赖接口。
//
// 设计哲学：
//   - 业务层（Handler / Scheduler）只关心"通知有新热点"这件事
//   - 不关心用 WS 还是邮件
//   - 改渠道只改注入的具体实现，不改业务代码（开闭原则）
type Notifier interface {
	// Notify 同步通知：把 payload 推出去
	// ctx 用来传超时/取消
	Notify(ctx context.Context, payload any) error
}

// ===== 1. WebSocket Hub =====

// WebSocketHub 是 WebSocket 连接管理中心。
//
// 设计参考：gorilla/websocket 官方 chat 模式
//   - 一个 Hub 维护所有在线连接
//   - 每个连接是独立的 Client goroutine
//   - 用 chan 做无锁消息传递（Go 推荐方式）
//
// 客户端连进来：Register chan；断开：Unregister chan； broadcast 推所有人。
type WebSocketHub struct {
	// clients 当前所有连接
	clients map[*WSClient]bool

	// register 新连接注册 chan
	register chan *WSClient

	// unregister 断开注销 chan
	unregister chan *WSClient

	// broadcast 广播 chan（任意消息塞进来，Hub 转给所有连接）
	broadcast chan []byte

	// mu 保护 clients map（读写分离用 RWMutex）
	mu sync.RWMutex
}

// NewWebSocketHub 构造
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:   make(map[*WSClient]bool),
		register:   make(chan *WSClient, 8),
		unregister: make(chan *WSClient, 8),
		broadcast:  make(chan []byte, 64),
	}
}

// Run 是 Hub 的主循环 goroutine。
// 应该在 main.go 里用 `go hub.Run()` 启动一次。
//
// 这是典型的 Go 事件循环模式：
// 用 select 同时监听多个 chan，收到事件就处理。
// select 会在某个 chan 有数据时自动唤醒对应分支。
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			// 加写锁修改 map
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WS Hub] 新连接 +1，当前共 %d\n", len(h.clients))

		case client := <-h.unregister:
			// 客户端断开，清理
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) // 关闭 send chan 让 client 写时发现关闭后退出
			}
			h.mu.Unlock()
			log.Printf("[WS Hub] 断开 -1，当前共 %d\n", len(h.clients))

		case message := <-h.broadcast:
			// 广播：拷给每个 client 的 send chan
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
					// 默认非阻塞写：满了就跳过这个客户端（防止慢客户端拖累全局）
				default:
					// 客户端缓冲满，认为它已卡死，踢掉
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast 是外部调用的广播入口：把任意 payload 推给所有在线客户端。
//
// 内部做 JSON 序列化、扔进 broadcast chan、Run 循环会处理。
func (h *WebSocketHub) Broadcast(payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	select {
	case h.broadcast <- data:
		return nil
	default:
		// broadcast chan 满了也返回成功（让调用方不为 WS 慢失败）
		// 这里选择丢弃：实时推送本来就是 best-effort
		return nil
	}
}

// Notify 实现 Notifier 接口（让 Hub 也能像邮件/飞书一样被注入业务层）
func (h *WebSocketHub) Notify(ctx context.Context, payload any) error {
	return h.Broadcast(payload)
}

// HandleWS 是 Gin/i.Handle 的 WebSocket 升级处理函数。
//
// 流程：
//   1) Upgrade 把 HTTP 连接升级为 WebSocket
//   2) 创建 WSClient 并 register 到 Hub
//   3) 启动两个 goroutine：readPump + writePump
//   4) readPump 持续读客户端发来的消息（我们不关心内容，但要监听关闭信号）
//   5) writePump 持续把 Hub 广播给这条连接的消息写出去
//
// Upgrader 配置允许跨域（开发期）
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 开发环境全放行，生产要校验 Origin
		return true
	},
}

// HandleWS 是给 gin 用 handler 函数
func (h *WebSocketHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	// Upgrade 升级 HTTP → WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade 失败: %v\n", err)
		return
	}

	// 创建 client 并注册
	client := &WSClient{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 32),
	}
	h.register <- client

	// 启动读写 goroutine
	go client.writePump()
	go client.readPump()
}

// ===== 2. WSClient =====

// WSClient 单条 WebSocket 连接
type WSClient struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	// send 是 Hub → client 的消息 chan
	// 有缓冲避免慢客户端阻塞 Hub 广播
	send chan []byte
}

// readPump 持续读客户端消息（哪怕只是把它扔掉），主要目的是感知断开
//
// 为什么要专门 goroutine 一直读？
//   - WebSocket 是双向的，不读的话客户端关闭事件接收不到
//   - 读会让 conn 底层感知对端关闭，conn.SetReadDeadline 设置心跳超时
const (
	// 心跳间隔
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10 // ping 比 pong timeout 短一点
)

func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	// 设置读超时：超过 pongWait 没收到任何消息（含 pong），认为客户端失联
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		// 客户端回 pong 时刷新读超时
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// 客户端关闭或网络断
			break
		}
		// 阶段 6 不处理客户端主动发来的消息，直接丢弃
	}
}

// writePump 持续把来自 Hub 的消息写到 WebSocket
//
// 关键点：定时 ping 客户端保活（每 pingPeriod 一次）
//   - WebSocket 用 Ping/Pong 帧做心跳
//   - 服务端发 Ping → 客户端浏览器自动回 Pong
//   - 没收到 Pong 就靠上面的 pongWait 超时清理
func (c *WSClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			// Hub 给我塞了新消息（或关闭信号）
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// send chan 已关，意味着 Hub 让我们关闭这条连接
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// 写消息（文本帧）
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}

		case <-ticker.C:
			// 周期性发 Ping 心跳
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ===== 3. 推送消息的统一格式 =====

// WSMessage 是 Hub 广播给客户端的标准消息结构。
// 前端根据 type 字段分发处理。
//
// 这一种"统一事件结构"让前端能扩展（新事件类型只加 type 常量）
type WSMessage struct {
	Type      string `json:"type"`      // "hot_new" / "crawl_done" / "analyze_done"
	Timestamp int64  `json:"timestamp"` // 服务器发送时间（秒）
	Data      any    `json:"data"`      // 携带的业务数据
}

// 事件类型常量
const (
	// EventHotNew 单条新热点入库
	EventHotNew = "hot_new"
	// EventCrawlDone 一次抓取任务完成
	EventCrawlDone = "crawl_done"
	// EventAnalyzeDone 一条热点 AI 分析完成
	EventAnalyzeDone = "analyze_done"
)

// SendHotNew 推一条"新热点"事件
func (h *WebSocketHub) SendHotNew(item any) error {
	return h.Broadcast(WSMessage{
		Type:      EventHotNew,
		Timestamp: time.Now().Unix(),
		Data:      item,
	})
}

// SendCrawlDone 推"抓取完成"事件
func (h *WebSocketHub) SendCrawlDone(meta any) error {
	return h.Broadcast(WSMessage{
		Type:      EventCrawlDone,
		Timestamp: time.Now().Unix(),
		Data:      meta,
	})
}

// SendAnalyzeDone 推"分析完成"事件
func (h *WebSocketHub) SendAnalyzeDone(data any) error {
	return h.Broadcast(WSMessage{
		Type:      EventAnalyzeDone,
		Timestamp: time.Now().Unix(),
		Data:      data,
	})
}