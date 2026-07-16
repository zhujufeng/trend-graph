// 多源爬虫集成测试。
// 注意：某些源（Reddit/Bing）可能因为反爬或网络环境失败，这是正常现象，
// 你本机浏览器环境正常应该都能通过。CI/sandbox 上失败可以 t.Skip 不算挂。
//
// 运行: go test -v -run TestMulti ./internal/crawler/
package crawler

import (
	"testing"
)

func TestMulti_GitHub(t *testing.T) {
	requireLiveTest(t)
	c := NewGitHubCrawler()
	items, err := c.Fetch("", 5)
	if err != nil {
		t.Fatalf("Fetch 失败: %v", err)
	}
	t.Logf("GitHub Trending 抓到 %d 条", len(items))
	for i, it := range items {
		t.Logf("#%d %s → %s (hot=%d)", i+1, it.Title, it.URL, it.Hot)
	}
	if len(items) == 0 {
		t.Error("抓到 0 条")
	}
}

func TestMulti_Reddit(t *testing.T) {
	requireLiveTest(t)
	c := NewRedditCrawler()
	items, err := c.Fetch("go", 5)
	if err != nil {
		t.Logf("Reddit 失败（可能反爬/IP 被限）: %v", err)
		t.Skip("Reddit 在 sandbox 可能被反爬 / 在本机正常环境重试")
	}
	t.Logf("Reddit 抓到 %d 条", len(items))
	for i, it := range items {
		t.Logf("#%d %s → %s (hot=%d)", i+1, it.Title, it.URL, it.Hot)
	}
}

func TestMulti_Bing(t *testing.T) {
	requireLiveTest(t)
	c := NewBingCrawler()
	items, err := c.Fetch("AI Agent", 5)
	if err != nil {
		t.Logf("Bing 失败（可能反爬/IP 被限）: %v", err)
		t.Skip("Bing 在 sandbox 可能被反爬 / 在本机正常环境重试")
	}
	t.Logf("Bing 抓到 %d 条", len(items))
	for i, it := range items {
		t.Logf("#%d %s → %s", i+1, it.Title, it.URL)
	}
}
