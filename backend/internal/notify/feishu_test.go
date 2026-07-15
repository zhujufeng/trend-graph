package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestFeishuNotifierSendsRichPostWithSourceLink(t *testing.T) {
	var payload map[string]any
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		defer request.Body.Close()
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}

	err := (&FeishuNotifier{webhook: "https://example.invalid/webhook", client: client}).Notify(context.Background(), FeishuPost{
		Title: "AI 信号雷达 · 早报 2026-07-15",
		Sections: []FeishuSection{{
			Text:     "MCP Inspector\n事实：新增检查流程\n行动：本地复现",
			LinkText: "查看原始来源",
			LinkURL:  "https://github.com/owner/repo",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload["msg_type"] != "post" {
		t.Fatalf("payload = %#v", payload)
	}
	content := payload["content"].(map[string]any)
	post := content["post"].(map[string]any)
	zh := post["zh_cn"].(map[string]any)
	if zh["title"] != "AI 信号雷达 · 早报 2026-07-15" {
		t.Fatalf("zh_cn = %#v", zh)
	}
	rows := zh["content"].([]any)
	link := rows[0].([]any)[1].(map[string]any)
	if link["href"] != "https://github.com/owner/repo" {
		t.Fatalf("link = %#v", link)
	}
}
