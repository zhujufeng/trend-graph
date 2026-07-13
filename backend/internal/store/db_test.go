// 集成测试：真打 PostgreSQL 验证 AutoMigrate + CRUD 全链路。
// 运行: go test -v ./internal/store/
package store

import (
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testDB 测试用一启动就建好的全局 db 实例
func testDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5432 user=tguser password=tgpass dbname=trend_graph sslmode=disable timezone=Asia/Shanghai"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		t.Fatalf("连接数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&HotItem{}, &Keyword{}, &CrawlRun{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// TestHotItemCRUD 验证：插入、查询、过滤、AI 结果更新
func TestHotItemCRUD(t *testing.T) {
	db := testDB(t)
	repo := NewHotItemRepo(db)

	// 清表：用 WHERE 1=1 删全部（受 DeletedAt 软删影响用 Unlink 模式）
	// 真实业务不要这么干，测试为方便才直接清
	db.Unscoped().Where("1 = 1").Delete(&HotItem{})

	// 1. 批量插入
	now := time.Now().Unix()
	items := []HotItem{
		{Title: "测试1", URL: "https://test1", Source: "hn", Hot: 100, PublishedAt: now, FetchedAt: now},
		{Title: "测试2", URL: "https://test2", Source: "hn", Hot: 200, PublishedAt: now - 100, FetchedAt: now},
		{Title: "测试3", URL: "https://test3", Source: "weibo", Hot: 50, PublishedAt: now - 200, FetchedAt: now},
	}
	n, err := repo.BatchCreate(items)
	if err != nil {
		t.Fatalf("插入失败: %v", err)
	}
	t.Logf("成功插入 %d 行", n)

	// 2. 查全部（source 过滤 hn）
	list, total, err := repo.List("hn", 0, time.Time{}, 100, 0)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	t.Logf("查 source=hn: total=%d, 返回 %d 条", total, len(list))
	if len(list) != 2 {
		t.Errorf("应有 2 条 hn 数据，实际 %d", len(list))
	}

	// 3. 时间过滤（最近 5 分钟，对应 published_at 是 now / now-100 / now-200 秒）
	since := time.Now().Add(-5 * time.Minute)
	recent, _, err := repo.List("", 0, since, 100, 0)
	if err != nil {
		t.Fatalf("List 时间过滤失败: %v", err)
	}
	t.Logf("查最近5分钟: %d 条", len(recent))
	if len(recent) < 2 {
		t.Errorf("最近5分钟应有 >=2 条，实际 %d", len(recent))
	}

	// 4. 更新 AI 结果
	if len(list) > 0 {
		id := list[0].ID
		rel := 0.85
		_ = repo.UpdateAIResult(id, "这是 AI 摘要", rel, true, `["AI","GPU"]`)
		got, _ := repo.GetByID(id)
		if got.Summary == "" || got.Relevance == nil || *got.Relevance != 0.85 {
			t.Errorf("AI 结果更新异常: %+v", got)
		}
		t.Logf("AI 结果更新成功: 文章 %d 摘要=%s", id, got.Summary)
	}
}
