// Hub 单元测试：注册 fake client，发广播，验证消息是否到 client.send chan
// 运行: go test -v ./internal/notify/
package notify

import (
	"context"
	"testing"
	"time"
)

// TestHubBroadcast 测试 Hub 把消息广播给已注册的客户端
func TestHubBroadcast(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// 装个假 client（send chan 替代真实连接）
	client := &WSClient{
		hub:  hub,
		send: make(chan []byte, 8),
	}
	hub.register <- client

	// 给 hub 一点时间处理 register 事件
	time.Sleep(20 * time.Millisecond)

	// 发广播
	msg := WSMessage{Type: EventHotNew, Timestamp: time.Now().Unix(), Data: "test"}
	err := hub.Notify(context.Background(), msg)
	if err != nil {
		t.Fatalf("Notify 失败: %v", err)
	}

	// 等 client.send 收到消息
	select {
	case got := <-client.send:
		t.Logf("client 收到 %d bytes: %s", len(got), string(got))
		// 解析回 struct 验证
		// 这里简单验证字段存不存在
		s := string(got)
		if !contains(s, "hot_new") {
			t.Errorf("消息里没有 type=hot_new: %s", s)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("500ms 内没收到消息")
	}

	// 注销
	hub.unregister <- client
	time.Sleep(20 * time.Millisecond)
	hub.mu.RLock()
	count := len(hub.clients)
	hub.mu.RUnlock()
	if count != 0 {
		t.Errorf("注销后 clients 应为 0，实际 %d", count)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
