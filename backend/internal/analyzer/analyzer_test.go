// analyzer 真打 DeepSeek 的集成测试。
// 运行: go test -v ./internal/analyzer/
package analyzer

import (
	"context"
	"os"
	"testing"
	"time"

	"trend-graph/internal/ai"
	"trend-graph/internal/types"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func newTestAnalyzer(t *testing.T) *Analyzer {
	if os.Getenv("RUN_LIVE_TESTS") != "1" {
		t.Skip("set RUN_LIVE_TESTS=1 to call DeepSeek")
	}
	key := os.Getenv("TEST_DEEPSEEK_API_KEY")
	if key == "" {
		t.Skip("TEST_DEEPSEEK_API_KEY 未设置，跳过集成测试")
	}
	base := os.Getenv("TEST_DEEPSEEK_BASE_URL")
	model := os.Getenv("TEST_DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-v4-pro"
	}
	cli := ai.NewDeepSeekClient(key, base)
	return NewAnalyzer(cli, model)
}

// TestExpandQuery 真打 DeepSeek 验证查询扩展
func TestExpandQuery(t *testing.T) {
	a := newTestAnalyzer(t)
	// 5 秒超时，避免 CI 慢
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keywords, err := a.ExpandQuery(ctx, "AI")
	if err != nil {
		t.Fatalf("ExpandQuery 失败: %v", err)
	}
	t.Logf("扩展结果: %v", keywords)
	if len(keywords) < 2 {
		t.Errorf("扩展条数太少: %d", len(keywords))
	}
	// 至少应该包含原词附近的概念
	hasAI := false
	for _, k := range keywords {
		if k == "AI" {
			hasAI = true
			break
		}
	}
	if !hasAI {
		t.Logf("注意：扩展结果里没有原词 'AI'（这是 AI 自由发挥，不强制）")
	}
}

// TestAnalyzeHot 真打 DeepSeek 验证单条热点综合分析
func TestAnalyzeHot(t *testing.T) {
	a := newTestAnalyzer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	item := types.HotItem{
		Title:  "GhostLock, a stack-UAF that has existed in all Linux distributions for 15 years",
		URL:    "https://nebusec.ai/research/ionstack-part-2/",
		Source: "hn",
		Author: "ranger_danger",
		Hot:    251,
	}
	keyword := "AI"

	res, err := a.AnalyzeHot(ctx, keyword, item)
	if err != nil {
		t.Fatalf("AnalyzeHot 失败: %v", err)
	}
	t.Logf("摘要:    %s", res.Summary)
	t.Logf("相关性:  %.2f", res.Relevance)
	t.Logf("真假:    %v", res.IsAuthentic)
	t.Logf("实体:    %v", res.Entities)
	t.Logf("理由:    %s", res.Reason)

	if res.Summary == "" {
		t.Error("摘要不能为空")
	}
	if res.Relevance < 0 || res.Relevance > 1 {
		t.Errorf("相关性越界: %f", res.Relevance)
	}
}
