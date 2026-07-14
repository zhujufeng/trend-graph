# 阶段 3：接入 DeepSeek（查询扩展 + 综合分析）

> 对应 commit：`feat: stage 3 - DeepSeek AI 接入`

## 🎯 目标

- 接入 DeepSeek 大模型 API
- 实现**查询扩展**：输入"AI"扩展成 7 个相关词
- 实现**综合分析**：一次 AI 调用拿到摘要+相关性+真假+实体

## 📚 学到的概念

### 1. 依赖倒置原则

```go
type Client interface {
    Chat(ctx context.Context, req ChatRequest) (string, *ChatResponse, error)
}

type Analyzer struct {
    ai ai.Client  // 接口，不是具体实现
}
```

业务层依赖 `ai.Client` 接口而不是 `*DeepSeekClient`，未来换 OpenAI/通义只换底层实现。

### 2. OpenAI 兼容协议

DeepSeek 用 OpenAI 兼容的 `/v1/chat/completions` 协议：

```go
type ChatRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
    ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}
```

`messages` 是 `[system, user]` 数组：system 设定人格/规则，user 是问题。

### 3. 让 LLM 返回结构化 JSON 的技巧

```go
ResponseFormat: &ResponseFormat{Type: "json_object"}
```

加这个字段强制 AI 输出 JSON。还要在 prompt 里给一个示例 JSON：

```
输出 JSON 格式：
{"keywords": ["AI", "人工智能", "大模型"]}
```

AI 会学着模仿这个格式输出。

### 4. Temperature 调控随机性

```go
Temperature: 0.3  // 低 = 稳定（适合扩展任务）
Temperature: 0.7  // 中 = 平衡
Temperature: 1.0  // 高 = 发散（适合创意）
```

分析类任务用 0.2~0.3 最稳。

### 5. context.Context 跨调用传超时

```go
ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
defer cancel()
res, err := an.AnalyzeHot(ctx, keyword, item)
```

ctx 贯穿整条调用链，超时自动 cancel 所有下游 HTTP 请求，避免 goroutine 泄漏。

### 6. 兜底容错（防 AI 返回不规范 JSON）

```go
if err := json.Unmarshal([]byte(content), &res); err != nil {
    return &AnalysisResult{
        Summary: item.Title,  // 至少用标题当摘要
        Reason: "AI 返回非 JSON，无法解析",
    }, nil  // 不报错让主流程继续
}
```

真实项目里 AI 输出不 100% 稳定，必须兜底。

### 7. 可选依赖（AI 没配置时基础功能仍能跑）

```go
var an *analyzer.Analyzer
if cfg.DeepSeekAPIKey != "" {
    an = analyzer.NewAnalyzer(...)
}
// an 可能为 nil，Handler 调用时判空
```

## 🔍 关键代码

| 概念 | 文件 |
|---|---|
| DeepSeek 客户端 | `backend/internal/ai/deepseek.go` |
| 查询扩展 + 综合分析 | `backend/internal/analyzer/analyzer.go` |
| API 接入 | `backend/internal/api/handler.go:ExpandQuery/AnalyzeHot` |
| 测试用 TestMain 加载 .env | `backend/internal/analyzer/analyzer_test.go` |

## 🧪 测试

```bash
go test -v -run TestExpandQuery ./internal/analyzer/    # 看 'AI' 扩展成 7 词
go test -v -run TestAnalyzeHot ./internal/analyzer/      # 单条综合分析
go test -v -run TestE2E ./internal/analyzer/             # 端到端：HN→AI→DB
```

## 🐛 踩坑

1. **`go test` 不自动读 .env**：要在测试包加 `TestMain` 调 `godotenv.Load("../../.env")`，工作目录是测试源码所在目录
2. **DeepSeek 实际模型名可能是 deepseek-v4-flash**：返回 JSON 里 model 字段会变，但请求时仍用 `deepseek-chat`
3. **AI 返回 JSON 偶尔多/少引号**：必加 `ResponseFormat=json_object`，不然容易翻车

## 📝 一句话总结

让 AI 返回结构化 JSON 的关键是 `response_format=json_object` + prompt 里给示例 + 兜底解析失败。