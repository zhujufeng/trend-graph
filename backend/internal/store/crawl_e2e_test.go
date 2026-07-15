// 端到端测试：爬虫抓 HN → 转 store.HotItem → 入库 → 查出来
// 模拟 POST /api/crawl?source=hn 的完整流程（跳过 HTTP 层）
//
// 运行: go test -v -run TestEndToEnd_HNToDB ./internal/store/
package store

import (
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trend-graph/internal/crawler"
	"trend-graph/internal/types"
)

// TestEndToEnd_HNToDB 真打 HN API + 真 INSERT PostgreSQL
// 这个测试是阶段 2 的"毕业测试"，通过即意味着阶段 2 主流程完成。
func TestEndToEnd_HNToDB(t *testing.T) {
	if os.Getenv("RUN_LIVE_TESTS") != "1" {
		t.Skip("set RUN_LIVE_TESTS=1 to run networked integration tests")
	}
	// 1. 准备 DB
	dsn := testDatabaseURL(t)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	db.AutoMigrate(&HotItem{}, &Keyword{}, &CrawlRun{})

	// 2. 清表
	db.Unscoped().Where("source = ?", "hn").Delete(&HotItem{})

	// 3. 用 crawler 抓
	hn := crawler.NewHackerNewsCrawler()
	items, err := hn.Fetch("", 5)
	if err != nil {
		t.Fatalf("爬虫失败: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("爬虫抓到 0 条")
	}
	t.Logf("爬虫抓到 %d 条 HN 数据", len(items))

	// 4. 转换 + 批量入库
	repo := NewHotItemRepo(db)
	dbItems := make([]HotItem, 0, len(items))
	for _, it := range items {
		// 用 store.FromBiz 转换
		dbItems = append(dbItems, FromBiz(it, nil))
	}
	n, err := repo.BatchCreate(dbItems)
	if err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	t.Logf("成功入库 %d 条", n)

	// 5. 从数据库再读出来验证
	list, total, err := repo.List("hn", 0, time.Time{}, 100, 0)
	if err != nil {
		t.Fatalf("回查失败: %v", err)
	}
	t.Logf("从数据库回查 source=hn: total=%d, 返回 %d 条", total, len(list))
	if total == 0 {
		t.Error("回查为空，入库失败或回查出错")
	}

	// 用 types.Crawler 接口验证多态（顺带学习 interface 用法）
	var c types.Crawler = hn
	if c.Source() != "hn" {
		t.Errorf("Source() = %q, want 'hn'", c.Source())
	}
}
