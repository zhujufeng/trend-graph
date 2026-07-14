# 阶段 6：WebSocket 实时推送

> 对应 commit：`feat: stage 6 - WebSocket 实时推送`

## 🎯 目标

- 后端 WebSocket Hub 管理所有连接
- 抓取/分析完成后实时推送给所有在线客户端
- 前端自动接收推送，无需手动刷新

## 📚 学到的概念

### 1. WebSocket vs HTTP

| | HTTP | WebSocket |
|---|---|---|
| 连接 | 短连接（请求完就关） | 长连接（一直保持） |
| 方向 | 客户端→服务端单向 | 双向 |
| 推送 | 服务端不能主动推 | 服务端可主动推 |
| 握手 | 普通 HTTP | HTTP Upgrade 升级 |

实时通知类需求（聊天、推送、行情）必须用 WebSocket。

### 2. Hub 模式（gorilla 官方推荐架构）

```
                ┌─────── Hub（中心管理） ───────┐
                │  clients map[*Client]bool      │
                │  register / unregister / broadcast chan │
                └────────────┬─────────────────┘
        ┌─────────┬─────────┼─────────┬──────────┐
        │         │         │         │          │
   Client 1   Client 2   Client 3  Client N
   每个一对 readPump + writePump goroutine
```

一个 Hub 管理所有 Client，Client 各自有读写 goroutine。

### 3. select 多路复用

```go
for {
    select {
    case client := <-h.register:    // 新连接
    case client := <-h.unregister:  // 断开
    case message := <-h.broadcast:  // 广播
    }
}
```

`select` 同时监听多个 chan，某个 chan 有数据时自动唤醒对应分支。这是 Go 事件循环模式的核心。

### 4. 非阻塞 chan 写（防慢客户端）

```go
for client := range h.clients {
    select {
    case client.send <- message:  // 写得进就写
    default:                       // 写不进（缓冲满）踢掉这个客户端
        close(client.send)
        delete(h.clients, client)
    }
}
```

不让一个慢客户端拖死整个 Hub 广播。

### 5. Ping/Pong 心跳保活

```go
ticker := time.NewTicker(pingPeriod)  // 每 54 秒发一次 Ping
for {
    select {
    case <-ticker.C:
        conn.WriteMessage(websocket.PingMessage, nil)
    }
}
```

- 服务端发 Ping → 浏览器自动回 Pong
- 60 秒没收到 Pong → 认为客户端失联，踢掉
- 防止 NAT/防火墙长时间无数据导致连接断开

### 6. 统一事件封装（多态分发）

```go
type WSMessage struct {
    Type string `json:"type"`  // "hot_new"/"crawl_done"/"analyze_done"
    Data any    `json:"data"`
}
```

前端按 `type` 字段 `switch` 处理，加新事件类型只加常量。

### 7. React 自定义 Hook

```ts
function useWebSocket(options: UseWebSocketOptions): UseWebSocketReturn {
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  // ...
  return { connected, lastMessage, ... }
}
```

约定：函数名以 `use` 开头，内部用其他 Hook。把 WebSocket 逻辑封装成一个 Hook 复用。

### 8. useRef 持久引用

```ts
const wsRef = useRef<WebSocket | null>(null)
wsRef.current = new WebSocket(url)
```

`useRef` 类似 class 实例字段：值在多次渲染间持久存在，但修改**不触发 rerender**。用来持有 WebSocket 实例避免每次渲染重建。

### 9. 闭包陷阱（React 版）

```ts
// 错：onMessage 永远是首次渲染时的版本
ws.onmessage = (e) => onMessage(JSON.parse(e.data))

// 对：用 ref 持有最新 callback
const cbRef = useRef({ onMessage })
useEffect(() => { cbRef.current = { onMessage } }, [onMessage])
ws.onmessage = (e) => cbRef.current.onMessage?.(JSON.parse(e.data))
```

React useEffect 闭包陷阱和 Go 循环变量陷阱是同一类问题。

### 10. 自动重连 + unmount 清理

```ts
useEffect(() => {
  isMounted.current = true
  connect()
  return () => {  // cleanup
    isMounted.current = false
    clearTimeout(reconnectTimer.current)
    wsRef.current?.close()
  }
}, [connect])
```

组件卸载时必须清理 timer 和连接，否则内存泄漏。

## 🔍 关键代码

| 概念 | 文件 |
|---|---|
| Hub + Client | `backend/internal/notify/ws.go` |
| 事件类型常量 | `backend/internal/notify/ws.go` |
| 后端推送点 | `backend/internal/api/handler.go:TriggerCrawl` |
| 前端 Hook | `frontend/src/hooks/useWebSocket.ts` |
| 前端接入 | `frontend/src/pages/HotListPage.tsx` |
| Vite WS 代理 | `frontend/vite.config.ts` |

## 🧪 测试

```bash
go test -v -run TestHubBroadcast ./internal/notify/
```

## 🐛 踩坑

1. **`context.Background` 不是类型**：`_ = context.Background{}` 错，应该是 `context.Background()`（调用函数）
2. **前端 `useEffect` 死循环**：依赖数组写错，把 `onMessage` 函数放进 deps 会无限重建
3. **Vite 不代理 ws://**：要在 `vite.config.ts` 单独加 `/ws` 项 + `ws: true`
4. **nginx 反代 WebSocket**：必须加 `Upgrade`/`Connection` 头，否则连不上

## 📝 一句话总结

Hub 模式 = 一个中心 Hub + N 个 Client，每个 Client 独立读写 goroutine，是 Go WebSocket 项目的标准架构。