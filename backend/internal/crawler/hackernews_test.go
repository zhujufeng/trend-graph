// Go 测试文件约定以 _test.go 结尾。
// 用 `go test` 运行，运行期间测试代码可以访问同包内私有字段（小写标识符）。
package crawler

import (
	"encoding/json"
	"testing"
)

// TestHackerNewsFetch 真打 HN API，验证爬虫链路是否通畅。
// 这种依赖外部网络的叫"集成测试"，CI 里跑容易不稳，但阶段 1 学习用它最直观。
//
// 运行: cd backend && go test -v -run TestHackerNewsFetch ./internal/crawler/
func TestHackerNewsFetch(t *testing.T) {
	c := NewHackerNewsCrawler()
	items, err := c.Fetch("", 3)
	if err != nil {
		t.Fatalf("抓取失败: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("抓取到 0 条，疑似 HN API 或网络异常")
	}
	t.Logf("成功抓取 %d 条", len(items))
	for i, it := range items {
		b, _ := json.MarshalIndent(it, "", "  ")
		t.Logf("--- #%d ---\n%s", i+1, string(b))
	}
}

// TestSource 验证 Source() 返回固定字符串 "hn"。
// 这种不依赖网络的叫"单元测试"，运行快、稳定、CI 必跑。
func TestSource(t *testing.T) {
	c := NewHackerNewsCrawler()
	if got := c.Source(); got != "hn" {
		t.Errorf("Source() = %q, want %q", got, "hn")
	}
}
