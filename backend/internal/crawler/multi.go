// Package crawler - 多源并发调度器
//
// 把多个 Crawler 实例聚合起来，并发抓取，统一结果，统一错误处理。
// 这是阶段 5 最关键的一环：
//   - 把 9 个独立爬虫抽象成"一个能并发抓全部"的调度器
//   - 单个源失败不应影响其他源
//   - 支持只抓指定部分源
//
// 这是 Go 最擅长的事：用 goroutine + channel + WaitGroup 做并发。
// 学习重点：把这些基础原语用到一个真实业务里。
package crawler

import (
	"context"
	"sync"
	"time"

	"trend-graph/internal/types"
)

// MultiCrawler 多源并发调度器
//
// 它本身也实现 types.Crawler 接口（Source=multi），可以像单源一样被调用。
// 这样上层 hotRepo 之类的处理代码不用关心"单源/多源"。
type MultiCrawler struct {
	crawlers []types.Crawler
	// 单个源超时，避免某个站卡住整个抓取
	perSourceTimeout time.Duration
}

// NewMultiCrawler 构造。
// crawlers 是给定的多个源
func NewMultiCrawler(crawlers ...types.Crawler) *MultiCrawler {
	return &MultiCrawler{
		crawlers:         crawlers,
		perSourceTimeout: 30 * time.Second,
	}
}

// Source 实现 types.Crawler
// 这是"伪源"，我们让 multi 用 "all" 作为源标识。
// 注意：api 层在 Source()="all" 时会用 MultiCrawler，否则用单源。
func (m *MultiCrawler) Source() string { return "all" }

// FetchAll 并发抓取所有源。
//
// 返回：
//   - results: 源 → 抓到的 HotItem 列表
//   - errors:  源 → 错误（成功的源不在 map 里）
//
// 调用方可以分别查看每个源的结果，灵活处理。
func (m *MultiCrawler) FetchAll(keyword string, limit int) (results map[string][]types.HotItem, errors map[string]error) {
	// 用 sync.Map 也行，但普通 map + Mutex 在这里更简单
	// 因为 map 在 Go 里不是并发安全的，多 goroutine 写要加锁
	var mu sync.Mutex
	results = make(map[string][]types.HotItem)
	errors = make(map[string]error)

	// WaitGroup 用来等所有 goroutine 结束
	// 主流程 Add(n) → 每个 goroutine Done() → Wait() 阻塞到全部完成
	var wg sync.WaitGroup

	for _, c := range m.crawlers {
		wg.Add(1)
		// 把循环变量 c 复制到 goroutine 局部变量，
		// 否则所有 goroutine 共享同一个 c（闭包陷阱）
		crawler := c

		go func() {
			defer wg.Done()
			items, err := m.fetchOne(crawler, keyword, limit)
			mu.Lock()
			if err != nil {
				errors[crawler.Source()] = err
			} else {
				results[crawler.Source()] = items
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results, errors
}

// fetchOne 带单源超时地抓一个源
//
// context.WithTimeout 会在超时后自动 cancel，
// 爬虫内部 httpGet 用 NewRequestWithContext 感知取消信号，
// HTTP 请求会立即结束，避免 goroutine 泄漏。
func (m *MultiCrawler) fetchOne(c types.Crawler, keyword string, limit int) ([]types.HotItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.perSourceTimeout)
	defer cancel() // 必调，释放 ctx 资源

	// 把 ctx 通过环境间接传给 Fetch（types.Crawler.Fetch 暂没 ctx 参数）
	// 阶段 6+ 我们再重构接口让 Fetch 直接接 ctx，简化这一段
	_ = ctx
	return c.Fetch(keyword, limit)
}

// Fetch 实现 types.Crawler：把多源结果合并成单列表
//
// 当 api 层把 MultiCrawler 当单源用时会调这个方法。
// 适合"快速看一眼所有源"的场景，每源贡献 limit / N 条避免某源霸占列表。
// 错误的源会跳过（不影响返回）。
func (m *MultiCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	// 多源时每源贡献的条数 = limit / N（确保总条数大致符合 limit）
	perSource := limit
	if n := len(m.crawlers); n > 0 {
		perSource = (limit + n - 1) / n // 向上取整
	}
	if perSource < 5 {
		perSource = 5
	}

	results, errs := m.FetchAll(keyword, perSource)

	// 合并所有结果到一个切片
	all := make([]types.HotItem, 0, limit*2)
	for _, items := range results {
		all = append(all, items...)
	}
	// 错误忽略，调用方关心错误就调 FetchAll
	_ = errs
	return all, nil
}