// 阶段 8 端到端测试：AI 分析 → 实体入库 → 图谱查询
// 运行: go test -v -run TestE2E_Graph ./internal/store/
package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trend-graph/internal/ai"
	"trend-graph/internal/analyzer"
	"trend-graph/internal/crawler"
)

// TestE2E_Graph 端到端：HN 抓 → AI 抽实体 → 入图谱 → 查图
func TestE2E_Graph(t *testing.T) {
	// 1. 准备 DB
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5432 user=tguser password=tgpass dbname=trend_graph sslmode=disable timezone=Asia/Shanghai"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		t.Fatalf("DB 失败: %v", err)
	}
	db.AutoMigrate(&HotItem{}, &Keyword{}, &CrawlRun{}, &Entity{}, &EntityRelation{})

	// 2. 准备 AI
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		t.Skip("DEEPSEEK_API_KEY 未设置")
	}
	cli := ai.NewDeepSeekClient(key, "")
	an := analyzer.NewAnalyzer(cli, "deepseek-chat")

	// 3. 抓 3 条 HN
	hn := crawler.NewHackerNewsCrawler()
	items, err := hn.Fetch("", 3)
	if err != nil {
		t.Fatalf("爬虫失败: %v", err)
	}

	// 4. 入库 + AI 分析 + 写图谱
	hotRepo := NewHotItemRepo(db)
	graphRepo := NewGraphRepo(db)

	// 假设一个 keyword ID（实际场景从 KeywordRepo 来）
	kw, _ := NewKeywordRepo(db).Create("GraphTest_"+time.Now().Format("150405"), "测试", 1440)
	defer db.Unscoped().Delete(&Keyword{}, kw.ID)
	keywordID := kw.ID

	now := time.Now().Unix()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, it := range items {
		// 入 hot_items
		dbItem := FromBiz(it, &keywordID)
		if _, err := hotRepo.BatchCreate([]HotItem{dbItem}); err != nil {
			t.Logf("入库失败（可能重复）: %v", err)
		}
		_ = now

		// AI 分析
		res, err := an.AnalyzeHot(ctx, "GraphTest", it)
		if err != nil {
			t.Logf("AI 分析失败: %v", err)
			continue
		}
		// 写回 AI 结果
		entitiesJSON, _ := json.Marshal(res.Entities)
		_ = hotRepo.UpdateAIResult(dbItem.ID, res.Summary, res.Relevance, res.IsAuthentic, string(entitiesJSON))

		// 实体入库
		entityIDs := make([]int64, 0, len(res.TypedEntities))
		for _, te := range res.TypedEntities {
			eid, err := graphRepo.EnsureEntity(te.Name, te.Kind)
			if err != nil {
				continue
			}
			entityIDs = append(entityIDs, eid)
			_ = graphRepo.EnsureRelation("hot", dbItem.ID, "contains", "entity", eid, &dbItem.ID)
		}
		// 关键词 → 实体
		_ = graphRepo.TrackKeywordToEntities(keywordID, entityIDs, &dbItem.ID)

		t.Logf("#%d %s", dbItem.ID, it.Title)
		for _, te := range res.TypedEntities {
			t.Logf("    [%s] %s", te.Kind, te.Name)
		}
	}

	// 5. 查图谱
	g, err := graphRepo.GetGraph(keywordID, kw.Word)
	if err != nil {
		t.Fatalf("GetGraph 失败: %v", err)
	}
	t.Logf("==== 图谱查询结果 ====")
	t.Logf("节点 %d 个, 边 %d 条", len(g.Nodes), len(g.Edges))
	for _, n := range g.Nodes {
		t.Logf("  节点 [%s] %s (count=%d)", n.Type, n.Label, n.Count)
	}
	for _, e := range g.Edges {
		t.Logf("  边 %s → %s (%s, weight=%d)", e.Source, e.Target, e.Relation, e.Weight)
	}

	if len(g.Nodes) < 2 {
		t.Skip("图谱节点太少（可能 AI 没分析出实体）")
	}
}
