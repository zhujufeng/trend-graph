package analyzer

import (
	"context"
	"strings"
	"testing"

	"trend-graph/internal/ai"
)

func TestAnalyzeSignalUsesEvidenceAndReturnsStructuredOutput(t *testing.T) {
	client := &captureAIClient{content: `{
		"evidenceClass":"original_documentation",
		"facts":[{"claim":"提供本地 MCP 检查流程","sourceUrl":"https://github.com/owner/repo/SKILL.md"}],
		"whatChanged":"新增可复现检查流程",
		"audience":"MCP 开发者",
		"practicalUse":"定位协议错误",
		"prerequisites":"本地测试服务器",
		"painPoint":"调试反馈不清晰",
		"action":"按 SKILL.md 在本地复现",
		"contentOpportunity":"制作 MCP 排错清单",
		"uncertainty":"尚未由本人验证",
		"alertEligible":false,
		"alertReason":"常规工具更新"
	}`}
	client.response.Usage.PromptTokens = 120
	client.response.Usage.CompletionTokens = 80
	analyzer := NewAnalyzer(client, "deepseek-v4-pro")

	output, err := analyzer.AnalyzeSignal(context.Background(),
		SignalInput{OriginalTitle: "MCP Inspector", OriginalURL: "https://github.com/owner/repo"},
		EvidenceInput{SourceURL: "https://github.com/owner/repo/SKILL.md", EvidenceClass: "original_documentation", Excerpt: "Install and run against a local server."},
	)
	if err != nil {
		t.Fatal(err)
	}
	if client.request.Model != "deepseek-v4-pro" || client.request.ResponseFormat == nil || client.request.ResponseFormat.Type != "json_object" {
		t.Fatalf("request = %#v", client.request)
	}
	userPrompt := client.request.Messages[len(client.request.Messages)-1].Content
	if !strings.Contains(userPrompt, "Install and run against a local server.") || !strings.Contains(userPrompt, "original_documentation") {
		t.Fatalf("user prompt = %s", userPrompt)
	}
	if !strings.Contains(string(output.JSON), `"action":"按 SKILL.md 在本地复现"`) || output.InputTokens != 120 || output.OutputTokens != 80 {
		t.Fatalf("output = %#v", output)
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
