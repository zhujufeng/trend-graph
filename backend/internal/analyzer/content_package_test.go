package analyzer

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"trend-graph/internal/ai"
)

func TestGenerateContentPackagePreservesSourcesAndRejectsFakeFirstPersonClaims(t *testing.T) {
	valid := `{
		"strategy":{"angle":"三步复现","audience":"AI 工具用户","evidenceNote":"第三方文档"},
		"xiaohongshu":{"title":"可复现工作流","body":"按文档完成三步配置","tags":["AI"]},
		"wechat":{"title":"完整教程","body":"本文依据官方文档整理","tags":["AI"]},
		"x":{"chinese":"三步配置流程","english":"A three-step setup workflow"},
		"visualPlan":[{"purpose":"封面","aspectRatio":"3:4","prompt":"简洁中文信息图，三步流程"}]
	}`
	an := NewAnalyzer(&contentAI{content: valid}, "deepseek-v4-pro")
	draft, err := an.GenerateContentPackage(context.Background(),
		SignalInput{OriginalTitle: "Agent Workflow", OriginalURL: "https://example.com/release"},
		EvidenceInput{SourceURL: "https://example.com/docs", EvidenceClass: "original_documentation", Excerpt: "Install then run."},
		json.RawMessage(`{"action":"复现"}`),
	)
	if err != nil {
		t.Fatalf("GenerateContentPackage: %v", err)
	}
	if len(draft.X.SourceLinks) != 2 || draft.X.SourceLinks[0] != "https://example.com/release" {
		t.Fatalf("source links = %#v", draft.X.SourceLinks)
	}

	an = NewAnalyzer(&contentAI{content: strings.Replace(valid, "按文档完成三步配置", "我实测提升了十倍", 1)}, "deepseek-v4-pro")
	if _, err := an.GenerateContentPackage(context.Background(),
		SignalInput{OriginalTitle: "Agent Workflow", OriginalURL: "https://example.com/release"},
		EvidenceInput{SourceURL: "https://example.com/docs", EvidenceClass: "documented_third_party_practice", Excerpt: "A user reported results."},
		json.RawMessage(`{"action":"复现"}`),
	); err == nil {
		t.Fatal("expected invented first-person claim to be rejected")
	}
}

type contentAI struct{ content string }

func (f *contentAI) Chat(context.Context, ai.ChatRequest) (string, *ai.ChatResponse, error) {
	return f.content, nil, nil
}
