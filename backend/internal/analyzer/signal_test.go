package analyzer

import (
	"context"
	"strings"
	"testing"

	"trend-graph/internal/ai"
)

func TestAnalyzeSignalUsesEvidenceAndReturnsStructuredOutput(t *testing.T) {
	client := &captureAIClient{content: `{
		"matchedTopics":["AI"],
		"valueScore":4,
		"evidenceClass":"original_documentation",
		"facts":[{"claim":"提供本地 MCP 检查流程","sourceUrl":"https://github.com/owner/repo/SKILL.md"}],
		"whatChanged":"新增可复现检查流程",
		"audience":"MCP 开发者",
		"practicalUse":"定位协议错误",
		"prerequisites":"本地测试服务器",
		"toolType":"mcp",
		"compatibleClients":["Codex","Claude Code"],
		"installation":"按 SKILL.md 配置 MCP 服务器",
		"painPoint":"调试反馈不清晰",
		"action":"按 SKILL.md 在本地复现",
		"contentOpportunity":"制作 MCP 排错清单",
		"uncertainty":"尚未由本人验证",
		"alertEligible":false,
		"alertCategory":"",
		"alertReason":"常规工具更新"
	}`}
	client.response.Usage.PromptTokens = 120
	client.response.Usage.CompletionTokens = 80
	analyzer := NewAnalyzer(client, "deepseek-v4-pro")

	output, err := analyzer.AnalyzeSignal(context.Background(),
		SignalInput{Source: "github", OriginalTitle: "MCP Inspector", OriginalURL: "https://github.com/owner/repo", Topics: []string{"AI", "机器人"}},
		EvidenceInput{SourceURL: "https://github.com/owner/repo/SKILL.md", EvidenceClass: "original_documentation", Excerpt: "Install and run against a local server."},
	)
	if err != nil {
		t.Fatal(err)
	}
	if client.request.Model != "deepseek-v4-pro" || client.request.ResponseFormat == nil || client.request.ResponseFormat.Type != "json_object" {
		t.Fatalf("request = %#v", client.request)
	}
	userPrompt := client.request.Messages[len(client.request.Messages)-1].Content
	if !strings.Contains(userPrompt, "Install and run against a local server.") || !strings.Contains(userPrompt, "original_documentation") || !strings.Contains(userPrompt, "AI、机器人") {
		t.Fatalf("user prompt = %s", userPrompt)
	}
	if !strings.Contains(string(output.JSON), `"action":"按 SKILL.md 在本地复现"`) || output.InputTokens != 120 || output.OutputTokens != 80 {
		t.Fatalf("output = %#v", output)
	}
}

func TestAnalyzeSignalRejectsAlertWithoutExplicitCategory(t *testing.T) {
	client := &captureAIClient{content: `{
		"matchedTopics":["AI"],"valueScore":5,
		"evidenceClass":"original_documentation",
		"facts":[{"claim":"正式发布","sourceUrl":"https://example.com/docs"}],
		"whatChanged":"新模型发布","audience":"开发者","practicalUse":"迁移模型","action":"阅读迁移文档",
		"alertEligible":true,"alertReason":"影响重大"
	}`}
	analyzer := NewAnalyzer(client, "deepseek-v4-pro")
	_, err := analyzer.AnalyzeSignal(context.Background(),
		SignalInput{OriginalTitle: "Model", OriginalURL: "https://example.com/release", Topics: []string{"AI"}},
		EvidenceInput{SourceURL: "https://example.com/docs", EvidenceClass: "original_documentation", Excerpt: "Released."},
	)
	if err == nil {
		t.Fatal("expected missing alert category to be rejected")
	}
}

func TestAnalyzeSignalRejectsUnknownTopicAndInvalidValueScore(t *testing.T) {
	base := `{
		"matchedTopics":["unknown"],"valueScore":4,
		"evidenceClass":"publisher_feed",
		"facts":[{"claim":"正式发布","sourceUrl":"https://example.com/feed"}],
		"whatChanged":"发布更新","audience":"开发者","practicalUse":"了解变化","action":"阅读原文",
		"alertEligible":false,"alertCategory":"","alertReason":""
	}`
	client := &captureAIClient{content: base}
	analyzer := NewAnalyzer(client, "deepseek-v4-pro")
	_, err := analyzer.AnalyzeSignal(context.Background(),
		SignalInput{OriginalTitle: "Update", OriginalURL: "https://example.com/post", Topics: []string{"AI"}},
		EvidenceInput{SourceURL: "https://example.com/feed", EvidenceClass: "publisher_feed", Excerpt: "Released."},
	)
	if err == nil || !strings.Contains(err.Error(), "unknown topic") {
		t.Fatalf("error = %v", err)
	}
}

func TestAnalyzeSignalAllowsExplicitSubscriptionWithoutTopicMatch(t *testing.T) {
	client := &captureAIClient{content: `{
		"matchedTopics":[],"valueScore":3,"evidenceClass":"original_documentation",
		"facts":[{"claim":"发布 v2","sourceUrl":"https://github.com/acme/tool/releases/tag/v2"}],
		"whatChanged":"发布 v2","audience":"订阅者","practicalUse":"评估升级","action":"阅读说明",
		"alertEligible":false,"alertCategory":"","alertReason":""
	}`}
	analyzer := NewAnalyzer(client, "deepseek-v4-pro")
	_, err := analyzer.AnalyzeSignal(context.Background(),
		SignalInput{OriginalTitle: "v2", OriginalURL: "https://github.com/acme/tool/releases/tag/v2", Topics: []string{"机器人"}, AllowUnmatched: true},
		EvidenceInput{SourceURL: "https://github.com/acme/tool/releases/tag/v2", EvidenceClass: "original_documentation", Excerpt: "Version 2."},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzeSignalBoundsEvidenceAndRejectsTruncatedOutput(t *testing.T) {
	client := &captureAIClient{content: `{"evidenceClass":"original_documentation"`}
	client.response.Choices = append(client.response.Choices, struct {
		Message      ai.Message `json:"message"`
		FinishReason string     `json:"finish_reason"`
	}{FinishReason: "length"})
	analyzer := NewAnalyzer(client, "deepseek-v4-pro")
	excerpt := "BEGIN\n" + strings.Repeat("中", 20_000) + "\nEND"

	_, err := analyzer.AnalyzeSignal(context.Background(),
		SignalInput{OriginalTitle: "Large evidence", OriginalURL: "https://example.com/release"},
		EvidenceInput{SourceURL: "https://example.com/docs", EvidenceClass: "original_documentation", Excerpt: excerpt},
	)
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("error = %v", err)
	}
	userPrompt := client.request.Messages[len(client.request.Messages)-1].Content
	if !strings.Contains(userPrompt, "BEGIN") || !strings.Contains(userPrompt, "END") || !strings.Contains(userPrompt, "证据正文已截断") {
		t.Fatalf("bounded prompt did not preserve both ends")
	}
	if client.request.MaxTokens < 2_000 {
		t.Fatalf("max tokens = %d", client.request.MaxTokens)
	}
}

type captureAIClient struct {
	content  string
	request  ai.ChatRequest
	response ai.ChatResponse
}

func (c *captureAIClient) Chat(_ context.Context, request ai.ChatRequest) (string, *ai.ChatResponse, error) {
	c.request = request
	return c.content, &c.response, nil
}
