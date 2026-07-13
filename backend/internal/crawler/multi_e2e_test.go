// 多源并发抓取端到端测试。
// 这是阶段 5 的"毕业测试"：并发调度 9 个源，看每个源各自抓多少条。
//
// 注意：sandbox 反爬可能让部分源 0 结果或犯 error；
// 你本机正常环境应该大部分源都有数据。
// 测试用例只要求"至少有一个源成功"就算 PASS。
//
// 运行: go test -v -run TestMultiE2E ./internal/crawler/
package crawler

import (
	"testing"
)

func TestMultiE2E_ConcurrentFetch(t *testing.T) {
	// 组装所有 9 源（同 main.go 的顺序）
	all := []interface{ Fetch(string, int) ([]interface{}, error) }{} // 占位类型断言不行
	_ = all

	// 直接构造实例切片
	var multi = NewMultiCrawler(
		NewHackerNewsCrawler(),
		NewGitHubCrawler(),
		NewRedditCrawler(),
		NewBingCrawler(),
		NewBilibiliCrawler(),
		NewZhihuCrawler(),
		NewWeiboCrawler(),
		NewLinuxDoCrawler(),
		NewTwitterCrawler(),
	)

	// keyword=AI，limit=5 每源
	results, errs := multi.FetchAll("AI", 5)

	// 打印汇总
	t.Logf("==== 抓取结果汇总 ====")
	successCount := 0
	totalItems := 0
	for source, items := range results {
		t.Logf("✓ %-10s: %d 条", source, len(items))
		successCount++
		totalItems += len(items)
		for i, it := range items {
			if i >= 2 {
				t.Logf("    ... 还有 %d 条", len(items)-i)
				break
			}
			t.Logf("    #%d %s → %s", i+1, it.Title, it.URL)
		}
	}
	for source, err := range errs {
		t.Logf("✗ %-10s: %v", source, err)
	}
	t.Logf("==== %d 个源成功，共 %d 条 ====", successCount, totalItems)

	if successCount == 0 {
		t.Fatal("所有源都失败了，请检查网络或反爬策略")
	}
}
