// Package analyzer 封装 AI 分析业务逻辑。
//
// 这里有两层抽象：
//   - ai.Client：底层大模型调用（DeepSeek 实现）
//   - analyzer.Analyzer：业务层，定义"扩展查询"和"分析热点"等具体任务
//
// 业务代码（api.Handler / scheduler）只依赖 analyzer，不直接碰 ai.Client。
// 这样：
//   - 换底层模型只动一处
//   - 测试 Analyzer 时可以注入 mock 的 ai.Client
package analyzer

// 导入：
// - context: 超时与取消
// - encoding/json: 让 AI 返回的 JSON 文本再解析成 struct
// - fmt: 拼错误
// - strings: trim 等
// - trend-graph/internal/ai: 大模型客户端
// - trend-graph/internal/types: HotItem 类型
import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"trend-graph/internal/ai"
	"trend-graph/internal/types"
)

// Analyzer 是 AI 业务分析器。
type Analyzer struct {
	// ai 是大模型客户端（依赖注入，方便测试时换 mock）
	ai ai.Client
	// model 是要调的模型，例如 "deepseek-chat"
	model string
}

// NewAnalyzer 构造函数。
// 接受接口 ai.Client 而不是具体 *DeepSeekClient，遵循"依赖倒置"原则。
func NewAnalyzer(aiClient ai.Client, model string) *Analyzer {
	return &Analyzer{ai: aiClient, model: model}
}

// ===== 1. 查询扩展 =====

// expandResult 是 AI 返回 JSON 的结构。
type expandResult struct {
	Keywords []string `json:"keywords"`
}

// ExpandQuery 输入一个原始关键词，让 AI 扩展成多个同义词/相关词。
//
// 比如输入 "AI"，AI 会扩展成 ["AI","人工智能","大模型","LLM","AGI","ChatGPT"]。
// 这一步是为了让爬虫阶段能搜到更多相关内容（提高召回率）。
//
// 关键技巧：
//   - system 提示词要明确角色和输出格式
//   - 用 response_format=json_object 强制 AI 返回 JSON
//   - 在 system prompt 里给一个示例 JSON，AI 会学着模仿
func (a *Analyzer) ExpandQuery(ctx context.Context, keyword string) ([]string, error) {
	systemPrompt := `你是一个关键词扩展助手。
用户会给你一个监控关键词，你需要扩展出 3-7 个相关的搜索词，提升信息检索召回率。

规则：
1. 输出语言匹配关键词本身（中文给中文，英文给英文）
2. 包含原词
3. 包含常见的同义词、缩写、相关术语
4. 不要解释，只返回 JSON

输出 JSON 格式：
{"keywords": ["AI", "人工智能", "大模型", "LLM", "AGI"]}`

	userPrompt := fmt.Sprintf("请扩展关键词：%s", keyword)

	req := ai.ChatRequest{
		Model: a.model,
		Messages: []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		// Temperature：低温度让输出更稳定，扩展任务用 0.3
		Temperature: 0.3,
		MaxTokens:   300,
		// 强制返回 JSON（DeepSeek 兼容 OpenAI 这一项）
		ResponseFormat: &ai.ResponseFormat{Type: "json_object"},
	}

	content, _, err := a.ai.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}

	// AI 返回的字符串里是 JSON，要再解析一层
	content = strings.TrimSpace(content)
	var res expandResult
	if err := json.Unmarshal([]byte(content), &res); err != nil {
		// 兜底：至少返回原关键词
		return []string{keyword}, nil
	}
	if len(res.Keywords) == 0 {
		return []string{keyword}, nil
	}
	return res.Keywords, nil
}

// ===== 2. 综合分析（摘要+相关性+真假+实体） =====

// AnalysisResult 是对一条热点的 AI 综合分析结果。
//
// 这个 struct 既是 AI 输出 JSON 解析的目标，也是 store 层的字段来源。
type AnalysisResult struct {
	Summary     string       `json:"summary"`      // 一句话摘要
	Relevance   float64      `json:"relevance"`    // 0~1 相关性
	IsAuthentic bool         `json:"isAuthentic"`  // 是否可信（排除明显谣言）
	Entities    []string     `json:"entities"`     // 兼容老结构：实体名列表
	// 阶段 8 新增：实体带类型，用于关联图谱
	TypedEntities []TypedEntity `json:"typedEntities"`
	Reason      string        `json:"reason"`        // 判断理由（便于调试）
}

// TypedEntity 带类型的实体（阶段 8 关联图谱用）
type TypedEntity struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // person/org/project/tech/concept/other
}

// AnalyzeHot 对一条热点做综合 AI 分析。
//
// keyword 是监控关键词（用来判断相关性）。
// item 是热点详情。
//
// 通过一次 AI 调用拿到多个分析结果，省 token 又快。
func (a *Analyzer) AnalyzeHot(ctx context.Context, keyword string, item types.HotItem) (*AnalysisResult, error) {
	// systemPrompt 是 DeepSeek 的系统提示词
	systemPrompt := `你是热点内容分析助手。对一条热点做综合分析，输出 JSON。

字段说明：
- summary: 30 字以内一句话摘要
- relevance: 0~1 浮点数，关键词相关性（0=无关，1=强相关）
- isAuthentic: true/false，true=看起来真实可信，false=疑似夸大/谣言/标题党
- entities: 字符串数组，提取其中的人名/公司/项目/技术名词（兼容字段，与 typedEntities 一致）
- typedEntities: 数组，每项 {"name":"...","kind":"..."}，kind 取值 person/org/project/tech/concept/other
- reason: 50 字以内，说明你判断的依据

只返回 JSON，不要其他文本。`

	// 拼用户输入：把关键词 + 热点信息一起送过去
	// HotItem 的 Author 字段也带上，有助于 AI 判断
	userPrompt := fmt.Sprintf(`监控关键词：%s
热点标题：%s
来源链接：%s
作者：%s
热度：%d

请分析，返回 JSON。`,
		keyword,
		item.Title,
		item.URL,
		item.Author,
		item.Hot,
	)

	req := ai.ChatRequest{
		Model: a.model,
		Messages: []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature:    0.2,
		MaxTokens:       400,
		ResponseFormat:  &ai.ResponseFormat{Type: "json_object"},
	}

	content, _, err := a.ai.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %w", err)
	}
	content = strings.TrimSpace(content)

	var res AnalysisResult
	if err := json.Unmarshal([]byte(content), &res); err != nil {
		// AI 偶尔不严格返回 JSON，兜底给一个"未分析"结果
		return &AnalysisResult{
			Summary:   item.Title, // 至少把标题当摘要
			Relevance: 0,
			Entities:  []string{},
			Reason:    "AI 返回非 JSON，无法解析",
		}, nil
	}

	// 防御：Relevance 越界就 clamp 进 [0,1]
	if res.Relevance < 0 {
		res.Relevance = 0
	}
	if res.Relevance > 1 {
		res.Relevance = 1
	}
	if res.Entities == nil {
		res.Entities = []string{}
	}
	// 阶段 8：如果 typedEntities 为空但 entities 有内容，自动从 entities 推类型（默认 other）
	if len(res.TypedEntities) == 0 && len(res.Entities) > 0 {
		res.TypedEntities = make([]TypedEntity, 0, len(res.Entities))
		for _, n := range res.Entities {
			res.TypedEntities = append(res.TypedEntities, TypedEntity{Name: n, Kind: "other"})
		}
	}
	return &res, nil
}