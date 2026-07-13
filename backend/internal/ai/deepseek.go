// Package ai 封装大模型接入。
//
// 设计目标：
//   - 阶段 3 实现 DeepSeek 客户端
//   - 以后要换 OpenAI / 通义 / 智谱，只要写新的 struct 实现 Client 接口
//   - 业务层不直接依赖 DeepSeek，只依赖 Client 接口（依赖倒置）
package ai

// 导入讲解：
// - bytes: 把 JSON body 包成 io.Reader
// - context: 跨调用传超时/取消
// - encoding/json: 序列化请求 + 反序列化响应
// - fmt: 拼错误信息
// - io: 读响应体
// - net/http: 发 HTTP 请求
// - time: 设置超时
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message 是 OpenAI 兼容 chat 协议里的"单条消息"。
//
// role 三种：
//   - "system":    系统提示，给整个对话设定人格、规则
//   - "user":      用户输入
//   - "assistant": 模型上次的回复
//
// 阶段 3 我们主要用 system + user 一来一回，不做多轮对话。
type Message struct {
	Role    string `json:"role"`    // system / user / assistant
	Content string `json:"content"` // 消息内容
}

// ChatRequest 是给 DeepSeek 的请求体。
// 字段对齐 OpenAI /v1/chat/completions 协议。
type ChatRequest struct {
	Model       string    `json:"model"`              // 模型名，deepseek-chat / deepseek-reasoner
	Messages    []Message `json:"messages"`           // 消息列表
	Temperature float64   `json:"temperature,omitempty"` // 温度，0 最确定，1 最发散
	MaxTokens   int       `json:"max_tokens,omitempty"` // 回复最长 token 数
	// 让 AI 返回 JSON 用这个字段：
	//   - "json_object" 强制返回 JSON（OpenAI/DeepSeek 都支持）
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ResponseFormat 控制输出格式
type ResponseFormat struct {
	Type string `json:"type"` // "json_object" / "text"
}

// ChatResponse 是 DeepSeek 返回的响应体。
//
// 这里只关心两个字段就够：
//   - Choices[0].Message.Content  →  AI 的回复文本
//   - Usage                       →  token 消耗（计费/调优）
//
// 其它字段（id/object/created）不在定义里就自动丢弃，Go json 默认行为。
type ChatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"` // stop/length/content_filter
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Client 是大模型客户端的抽象接口。
//
// 接口只需一个方法 Chat，方便未来：
//   - 写假实现做单元测试（mock）
//   - 写 OpenAI / 通义实现只换底层
type Client interface {
	// Chat 发一次对话。
	// msgs 是消息列表（至少 1 条），返回 AI 的纯文本回复。
	Chat(ctx context.Context, req ChatRequest) (string, *ChatResponse, error)
}

// DeepSeekClient 是 DeepSeek 的具体实现。
//
// 字段：
//   - apiKey：密钥，从 Authorization Header 带过去
//   - baseURL：API 地址，默认 https://api.deepseek.com/v1
//   - httpClient：复用连接的 HTTP 客户端
type DeepSeekClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewDeepSeekClient 构造客户端。
//
// 默认值兜底：baseURL 留空就用官方地址。
func NewDeepSeekClient(apiKey, baseURL string) *DeepSeekClient {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	return &DeepSeekClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		// 60 秒超时：DeepSeek 有时回复慢，尤其 reasoner 模型
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Chat 实现 Client 接口。
//
// 流程：
//   1) 组装请求体 + 鉴权 Header
//   2) POST /chat/completions
//   3) 检查 HTTP 状态、解析 JSON
//   4) 返回回复文本
func (c *DeepSeekClient) Chat(ctx context.Context, req ChatRequest) (string, *ChatResponse, error) {
	// 1. 序列化请求体
	body, err := json.Marshal(req)
	if err != nil {
		return "", nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 2. 构造 HTTP 请求
	//    - 用 baseURL + /chat/completions
	//    - 用 context 让超时可以由调用方控制
	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("构造请求失败: %w", err)
	}

	// 3. 必须的 Header：
	//    - Content-Type: application/json
	//    - Authorization: Bearer <key>
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 4. 发请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", nil, fmt.Errorf("调用 DeepSeek 失败: %w", err)
	}
	defer resp.Body.Close()

	// 5. 读响应体
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 6. 非 200 就报错（响应体也带回去便于 debug）
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("DeepSeek 返回 %d: %s", resp.StatusCode, string(raw))
	}

	// 7. 解析 ChatResponse
	var chatResp ChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return "", nil, fmt.Errorf("解析响应 JSON 失败: %w, body=%s", err, string(raw))
	}

	// 8. 支持空回复（choices 为空）
	if len(chatResp.Choices) == 0 {
		return "", &chatResp, fmt.Errorf("AI 返回 0 个 choices: %s", string(raw))
	}

	return chatResp.Choices[0].Message.Content, &chatResp, nil
}