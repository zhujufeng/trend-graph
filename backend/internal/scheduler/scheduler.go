// Package scheduler 用 robfig/cron 实现关键词定时监控任务。
//
// 流程：
//   - 启动时查 keywords 表中 active=true 的关键词
//   - 为每个关键词注册一个 cron 任务（间隔由 IntervalMin 决定）
//   - 每次触发：调 MultiCrawler.Fetch → 入库 → AI 分析 → 通知渠道群发
//   - 周期性重载（每 1 分钟检查 keywords 表是否有变动，对应增删 cron）
//
// 这是阶段 7 的核心组件，把"爬虫/AI/通知"全部串起来。
package scheduler

// 导入：
// - log/context/fmt/time: 小工具
// - sync.Mutex: 保护内部状态
// - robfig/cron/v3: 定时任务库
// - 项目内的爬虫/AI/Store/notify
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"trend-graph/internal/analyzer"
	"trend-graph/internal/crawler"
	"trend-graph/internal/notify"
	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

// Scheduler 是关键词监控任务调度器
type Scheduler struct {
	cron *cron.Cron

	multiCrawler *crawler.MultiCrawler
	hotRepo      *store.HotItemRepo
	keywordRepo  *store.KeywordRepo
	analyzer     *analyzer.Analyzer
	notifier     notify.Notifier
	wsHub        *notify.WebSocketHub
	// 阶段 8 新增：图谱存储
	graphRepo *store.GraphRepo

	mu      sync.Mutex
	entries map[int64]cron.EntryID
}

// NewScheduler 构造
func NewScheduler(
	mc *crawler.MultiCrawler,
	hotRepo *store.HotItemRepo,
	keywordRepo *store.KeywordRepo,
	an *analyzer.Analyzer,
	notifier notify.Notifier,
	wsHub *notify.WebSocketHub,
	graphRepo *store.GraphRepo,
) *Scheduler {
	c := cron.New()
	return &Scheduler{
		cron:         c,
		multiCrawler: mc,
		hotRepo:      hotRepo,
		keywordRepo:  keywordRepo,
		analyzer:     an,
		notifier:     notifier,
		wsHub:        wsHub,
		graphRepo:    graphRepo,
		entries:      make(map[int64]cron.EntryID),
	}
}

// Start 启动调度器
//
// 内部步骤：
//   1) 启动底层 cron 库
//   2) 加载所有 active 关键词，注册任务
//   3) 启动定时重载（每 1 分钟）让关键字增删生效
func (s *Scheduler) Start() {
	s.cron.Start()
	log.Println("[Scheduler] 已启动")

	// 首次加载
	if err := s.Reload(); err != nil {
		log.Printf("[Scheduler] 首次加载失败: %v\n", err)
	}

	// 每 1 分钟重新加载关键词表，让用户改的关键词生效
	// 用 @every 简写，不用记 cron 语法
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := s.Reload(); err != nil {
				log.Printf("[Scheduler] 重载失败: %v\n", err)
			}
		}
	}()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("[Scheduler] 已停止")
}

// Entries 返回当前已注册的 cron 任务数量（给 main 做 health 查询用）
func (s *Scheduler) Entries() []cron.Entry {
	return s.cron.Entries()
}

// Reload 重新加载关键词并同步 cron 任务
//
// 算法（简单增量）：
//   1) 查 active 关键词列表
//   2) 已注册但当前不在列表的 → 移除
//   3) 列表里有但没注册的 → 添加
//   4) 同 ID 但间隔变了的不处理（一期功能简化）
func (s *Scheduler) Reload() error {
	keywords, err := s.keywordRepo.List(true)
	if err != nil {
		return fmt.Errorf("查询 keywords 失败: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	active := make(map[int64]bool, len(keywords))
	for _, k := range keywords {
		active[k.ID] = true
		// 不存在就注册
		if _, has := s.entries[k.ID]; !has {
			entryID, err := s.addKeywordJob(k)
			if err != nil {
				log.Printf("[Scheduler] 注册关键词 %s 失败: %v\n", k.Word, err)
				continue
			}
			s.entries[k.ID] = entryID
			log.Printf("[Scheduler] 注册关键词 %q (每 %d 分钟) → cron#%d\n", k.Word, k.IntervalMin, entryID)
		}
	}

	// 删除已失效的关键词任务
	for id, entryID := range s.entries {
		if !active[id] {
			s.cron.Remove(entryID)
			delete(s.entries, id)
			log.Printf("[Scheduler] 移除关键词 cron#%d\n", entryID)
		}
	}

	return nil
}

// addKeywordJob 为一个关键词注册 cron 任务
//
// 间隔用 @every Nm 写法（cron 库支持）：
//   @every 30m 表示每 30 分钟跑一次
// 对监控这种"间隔重于定点"的场景最合适，比 5 段 cron 更不容易写错。
func (s *Scheduler) addKeywordJob(k store.Keyword) (cron.EntryID, error) {
	spec := fmt.Sprintf("@every %dm", k.IntervalMin)
	return s.cron.AddFunc(spec, func() {
		// 每次触发都跑这个闭包
		s.runKeywordJob(k.ID, k.Word, k.IntervalMin)
	})
}

// runKeywordJob 单个关键词的一次执行
//
// 步骤：
//   1) MultiCrawler 并发抓所有源
//   2) 把成功的源结果入库
//   3) 对每条热点做 AI 分析
//   4) 通过 WebSocket 实时推送
//   5) 把命中关键词的热点汇总成简报，发给通知渠道
//   6) 更新 keyword.last_fetched_at
func (s *Scheduler) runKeywordJob(keywordID int64, keyword string, intervalMin int) {
	startTime := time.Now()
	log.Printf("[Scheduler] 触发关键词 %q (每 %d 分钟)\n", keyword, intervalMin)

	// 限每源 10 条，免得太重带宽
	results, errs := s.multiCrawler.FetchAll(keyword, 10)

	// 入库
	// dbItems 存放 store.HotItem（含数据库 ID），bizItems 保留 types.HotItem 给 AI 分析
	dbItems := make([]store.HotItem, 0, 50)
	bizItems := make([]types.HotItem, 0, 50)
	for _, items := range results {
		for _, it := range items {
			dbItems = append(dbItems, store.FromBiz(it, &keywordID))
			bizItems = append(bizItems, it)
		}
	}
	if len(dbItems) > 0 {
		if _, err := s.hotRepo.BatchCreate(dbItems); err != nil {
			log.Printf("[Scheduler] 入库失败: %v\n", err)
		}
	}

	// 把错误源汇总成简报
	errSources := make([]string, 0, len(errs))
	for src := range errs {
		errSources = append(errSources, src)
	}

	// AI 分析（可选）
	analyzedCount := 0
	if s.analyzer != nil && len(dbItems) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		for i := range dbItems {
			res, err := s.analyzer.AnalyzeHot(ctx, keyword, bizItems[i])
			if err != nil {
				continue
			}
			entitiesJSON, _ := json.Marshal(res.Entities)
			_ = s.hotRepo.UpdateAIResult(dbItems[i].ID, res.Summary, res.Relevance, res.IsAuthentic, string(entitiesJSON))
			dbItems[i].Summary = res.Summary
			dbItems[i].Relevance = &res.Relevance
			dbItems[i].IsAuthentic = &res.IsAuthentic
			analyzedCount++

			// 阶段 8：实体写入图谱
			// 调度器有 keywordID，能建立 keyword→entity 的 tracks 边
			if s.graphRepo != nil {
				s.ingestEntitiesToGraph(dbItems[i].ID, &keywordID, res)
			}

			if s.wsHub != nil {
				_ = s.wsHub.SendAnalyzeDone(map[string]interface{}{
					"id":          dbItems[i].ID,
					"title":       dbItems[i].Title,
					"summary":     res.Summary,
					"relevance":   res.Relevance,
					"isAuthentic": res.IsAuthentic,
					"entities":    res.Entities,
				})
			}
		}
	}

	// 通知（只在高相关命中才推，避免刷屏）
	// 阈值：至少 1 条 relevance >= 0.5 才通知
	shouldNotify := false
	for i := range dbItems {
		if dbItems[i].Relevance != nil && *dbItems[i].Relevance >= 0.5 {
			shouldNotify = true
			break
		}
	}

	if shouldNotify && s.notifier != nil {
		// 拼简报
		report := buildReport(keyword, intervalMin, len(dbItems), analyzedCount, errSources, startTime)
		_ = s.notifier.Notify(context.Background(), report)
		log.Printf("[Scheduler] 已推送通知: 关键词 %q\n", keyword)
	}

	// 更新 last_fetched_at
	_ = s.keywordRepo.UpdateLastFetched(keywordID, time.Now())

	log.Printf("[Scheduler] 关键词 %q 完成: 入库 %d 条, 分析 %d 条, 错误源 %d 个\n",
		keyword, len(dbItems), analyzedCount, len(errSources))
}

// ingestEntitiesToGraph 是 scheduler 用的辅助函数（与 handler 同名同实现）
// 让自动抓取产生的 AI 实体也进入关联图谱
func (s *Scheduler) ingestEntitiesToGraph(hotID int64, keywordID *int64, res *analyzer.AnalysisResult) {
	if s.graphRepo == nil {
		return
	}
	entityIDs := make([]int64, 0, len(res.TypedEntities))
	for _, te := range res.TypedEntities {
		id, err := s.graphRepo.EnsureEntity(te.Name, te.Kind)
		if err != nil {
			continue
		}
		entityIDs = append(entityIDs, id)
	}
	if len(entityIDs) == 0 {
		return
	}
	for _, eid := range entityIDs {
		_ = s.graphRepo.EnsureRelation("hot", hotID, "contains", "entity", eid, &hotID)
	}
	if keywordID != nil {
		_ = s.graphRepo.TrackKeywordToEntities(*keywordID, entityIDs, &hotID)
	}
	for i := 0; i < len(entityIDs); i++ {
		for j := i + 1; j < len(entityIDs); j++ {
			a, b := entityIDs[i], entityIDs[j]
			if a > b {
				a, b = b, a
			}
			_ = s.graphRepo.EnsureRelation("entity", a, "cooccur", "entity", b, &hotID)
		}
	}
}

// buildReport 拼简报文本
//
// 这一段纯粹是拼字符串，给邮件/飞书/钉钉用
func buildReport(keyword string, intervalMin int, totalN, analyzedN int, errSources []string, start time.Time) string {
	var b []string
	b = append(b, fmt.Sprintf("trend-graph 监控简报"))
	b = append(b, fmt.Sprintf("关键词: %s", keyword))
	b = append(b, fmt.Sprintf("执行时间: %s", start.Format("2006-01-02 15:04:05")))
	b = append(b, fmt.Sprintf("间隔: 每 %d 分钟", intervalMin))
	b = append(b, fmt.Sprintf("入库: %d 条", totalN))
	b = append(b, fmt.Sprintf("AI 分析: %d 条", analyzedN))
	b = append(b, fmt.Sprintf("耗时: %v", time.Since(start).Round(time.Millisecond)))
	if len(errSources) > 0 {
		b = append(b, fmt.Sprintf("失败源: %v", errSources))
	}
	return strings.Join(b, "\n")
}

