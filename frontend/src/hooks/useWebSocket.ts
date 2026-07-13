// useWebSocket.ts
//
// 封装 WebSocket 连接生命周期 + 自动重连 + 心跳感知 + 事件分发。
//
// Hook 是 React 函数组件复用逻辑的标准方式。
// 自定义 Hook 名必须以 use 开头（约定 + eslint 规则）。
//
// 用法：
//   const { lastMessage, connected } = useWebSocket({
//     url: 'ws://localhost:8080/ws',
//     onMessage: (msg) => console.log(msg),
//   })

import { useEffect, useRef, useState, useCallback } from 'react'

// WebSocket 后端推送消息结构（同 notify.WSMessage）
export interface WSMessage {
  type: 'hot_new' | 'crawl_done' | 'analyze_done'
  timestamp: number
  data: unknown
}

interface UseWebSocketOptions {
  // WebSocket URL，如 ws://localhost:8080/ws
  url: string
  // 收到消息的回调（按 type 分发由调用方处理）
  onMessage?: (msg: WSMessage) => void
  // 连接状态变化回调
  onConnect?: () => void
  onDisconnect?: () => void
  // 是否自动重连（默认 true）
  autoReconnect?: boolean
  // 重连间隔（ms），默认 3000
  reconnectInterval?: number
  // 最大重连次数，默认无限
  maxReconnectAttempts?: number
}

interface UseWebSocketReturn {
  // 当前是否连接中
  connected: boolean
  // 最新一条消息
  lastMessage: WSMessage | null
  // 手动发送消息（阶段 6 不常用，预留）
  sendMessage: (data: string) => void
  // 手动断开
  disconnect: () => void
  // 手动重连
  reconnect: () => void
  // 已尝试重连次数
  reconnectCount: number
}

// useWebSocket 是自定义 Hook
//
// 内部用 useRef 持有 WebSocket 实例（避免每次渲染重建）。
// useRef 类似 class 实例字段：值在多次渲染间持久存在，但修改不触发 rerender。
export function useWebSocket(options: UseWebSocketOptions): UseWebSocketReturn {
  const {
    url,
    onMessage,
    onConnect,
    onDisconnect,
    autoReconnect = true,
    reconnectInterval = 3000,
    maxReconnectAttempts = Infinity,
  } = options

  // 状态：是否连接（rerender 触发器）
  const [connected, setConnected] = useState(false)
  // 最新消息
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null)
  // 重连计数
  const [reconnectCount, setReconnectCount] = useState(0)

  // ref：引用对象，不触发渲染
  const wsRef = useRef<WebSocket | null>(null)
  // 重连计时器
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  // 防止组件卸载后还在重连
  const isMounted = useRef(true)
  // 当前重连次数（用 ref 避免闭包陷阱）
  const attemptCount = useRef(0)

  // 用 ref 持有最新回调，避免每次重渲染都重新绑定 onmessage
  const cbRef = useRef({ onMessage, onConnect, onDisconnect })
  useEffect(() => {
    cbRef.current = { onMessage, onConnect, onDisconnect }
  }, [onMessage, onConnect, onDisconnect])

  // 创建连接的函数（用 useCallback 缓存避免重建）
  const connect = useCallback(() => {
    if (!isMounted.current) return

    try {
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        if (!isMounted.current) return
        setConnected(true)
        attemptCount.current = 0
        setReconnectCount(0)
        cbRef.current.onConnect?.()
      }

      ws.onmessage = (event) => {
        if (!isMounted.current) return
        try {
          const msg = JSON.parse(event.data) as WSMessage
          setLastMessage(msg)
          cbRef.current.onMessage?.(msg)
        } catch (e) {
          console.warn('[WS] 解析消息失败', e, event.data)
        }
      }

      ws.onerror = (e) => {
        console.warn('[WS] 错误', e)
      }

      ws.onclose = () => {
        if (!isMounted.current) return
        setConnected(false)
        cbRef.current.onDisconnect?.()

        // 自动重连
        if (autoReconnect && attemptCount.current < maxReconnectAttempts) {
          attemptCount.current += 1
          setReconnectCount(attemptCount.current)
          reconnectTimer.current = setTimeout(() => {
            connect()
          }, reconnectInterval)
        }
      }
    } catch (e) {
      console.error('[WS] 连接异常', e)
    }
  }, [url, autoReconnect, reconnectInterval, maxReconnectAttempts])

  // 初始连接 + 清理
  useEffect(() => {
    isMounted.current = true
    connect()

    return () => {
      // 组件卸载时彻底清理
      isMounted.current = false
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [connect])

  // 手动发送
  const sendMessage = useCallback((data: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(data)
    }
  }, [])

  // 手动断开
  const disconnect = useCallback(() => {
    if (reconnectTimer.current) clearTimeout(reconnectTimer.current)
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
  }, [])

  // 手动重连
  const reconnect = useCallback(() => {
    attemptCount.current = 0
    disconnect()
    setTimeout(connect, 100)
  }, [connect, disconnect])

  return {
    connected,
    lastMessage,
    sendMessage,
    disconnect,
    reconnect,
    reconnectCount,
  }
}