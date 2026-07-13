// 端到端测试：HN爬虫 → 入库 → 对每条做 AI 分析 → 验证写回 DB
// 这个是阶段 3 的"毕业测试"。需要 DEEPSEEK_API_KEY 和 PG 都在线。
//
// 运行: go test -v -run TestE2E_CrawlToAnalyze ./internal/analyzer/
package analyzer

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trend-graph/internal/ai"
	"trend-graph/internal/crawler"
	"trend-graph/internal/store"
)

func TestE2E_CrawlToAnalyze(t *testing.T) {
	// 1. 准备 AI
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		t.Skip("DEEPSEEK_API_KEY 未设置")
	}
	base := os.Getenv("DEEPSEEK_BASE_URL")
	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}
	cli := ai.NewDeepSeekClient(key, base)
	an := NewAnalyzer(cli, model)

	// 2. 准备 DB
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5432 user=tguser password=tgpass dbname=trend_graph sslmode=disable timezone=Asia/Shanghai"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		t.Fatalf("连接 DB 失败: %v", err)
	}
	db.AutoMigrate(&store.HotItem{}, &store.Keyword{}, &store.CrawlRun{})
	db.Unscoped().Where("source = ?", "hn").Delete(&store.HotItem{})

	repo := store.NewHotItemRepo(db)
	hn := crawler.NewHackerNewsCrawler()

	// 3. 抓 3 条 HN
	items, err := hn.Fetch("", 3)
	if err != nil {
		t.Fatalf("爬虫失败: %v", err)
	}
	t.Logf("爬虫抓到 %d 条", len(items))

	// 4. 入库
	dbItems := make([]store.HotItem, 0, len(items))
	for _, it := range items {
		dbItems = append(dbItems, store.FromBiz(it, nil))
	}
	n, err := repo.BatchCreate(dbItems)
	if err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	t.Logf("入库 %d 条", n)

	// 5. 对每条用 AI 分析，写回 DB
	keyword := "AI"
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	analyzed := 0
	for i, dbItem := range dbItems {
		// 注：BatchCreate 之后 dbItems[i].ID 已经被 GORM 回填
		res, err := an.AnalyzeHot(ctx, keyword, items[i])
		if err != nil {
			t.Logf("AI 分析第 %d 条失败: %v", i+1, err)
			continue
		}
		// 把实体列表转 JSON
		// 注意：用 encoding/json 但为了简化这里直接拼字符串
		// 真实项目用 json.Marshal
		ents := "["
		for j, e := range res.Entities {
			if j > 0 {
				ents += ","
			}
			ents += "\"" + e + "\""
		}
		ents += "]"
		_ = repo.UpdateAIResult(dbItem.ID, res.Summary, res.Relevance, res.IsAuthentic, ents)
		analyzed++

		t.Logf("#%d [%s] 相关性=%.2f 真假=%v 摘要=%s 实体=%v",
			i+1, items[i].Title, res.Relevance, res.IsAuthentic, res.Summary, res.Entities)
	}
	t.Logf("成功分析 %d 条", analyzed)
	if analyzed == 0 {
		t.Fatal("全部 AI 分析失败，请检查网络或 API key")
	}

	// 6. 回查 DB 验证 summary 已写入
	list, total, err := repo.List("hn", 0, time.Time{}, 100, 0)
	if err != nil {
		t.Fatalf("回查失败: %v", err)
	}
	hasSummaryCount := 0
	for _, it := range list {
		if it.Summary != "" && it.Relevance != nil {
			hasSummaryCount++
		}
	}
	t.Logf("数据库回查: total=%d, 有摘要+相关性=%d", total, hasSummaryCount)
	if hasSummaryCount == 0 {
		t.Error("数据库里没有任何 AI 分析结果，写回失败")
	}
}
