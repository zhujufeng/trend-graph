// Package api 定义 HTTP 路由和 Handler。
//
// 阶段 2 改动：
//   - Handler 多依赖一个 HotItemRepo（数据库访问）
//   - 多依赖一个 Crawler（用来触发抓取并入库）
//   - 路由新增:
//       POST /api/crawl          主动触发一次抓取并入库
//       GET  /api/hots           从数据库读（替代阶段 1 的直接调爬虫版本）
//       GET  /api/hots/:id       取单条
package api

// 导入同阶段 1，多了 store 包
import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

// Handler 是 API 层的"控制器"。
//
// 阶段 2 多了两个依赖：
//   - hotRepo: 数据库访问
//   - 从 types.Crawler 改用 map[string]types.Crawler 不变
type Handler struct {
	crawlers map[string]types.Crawler
	hotRepo  *store.HotItemRepo
}

// NewHandler 构造函数，参数逐步扩展。
//
// 现在传三个：crawlers、hotRepo
// 阶段 3 之后会再加 aiClient、graphRepo...
func NewHandler(hotRepo *store.HotItemRepo, crawlers ...types.Crawler) *Handler {
	m := make(map[string]types.Crawler, len(crawlers))
	for _, c := range crawlers {
		m[c.Source()] = c
	}
	return &Handler{crawlers: m, hotRepo: hotRepo}
}

// Register 路由注册
func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/hots", h.ListHots)        // 新：从数据库读
		api.GET("/hots/:id", h.GetHot)      // 取单条
		api.POST("/crawl", h.TriggerCrawl)  // 手动触发一次抓取+入库
		api.GET("/sources", h.ListSources)  // 返回所有可用源
	}
}

// TriggerCrawl POST /api/crawl?source=hn&keyword=AI&limit=20
//
// 主动触发一次抓取，把结果插入数据库。
// 这是给前端"立即抓一下"按钮 / 后台定时任务调用的入口。
func (h *Handler) TriggerCrawl(c *gin.Context) {
	source := c.DefaultQuery("source", "hn")
	keyword := c.Query("keyword")

	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	crawler, ok := h.crawlers[source]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "unknown source",
			"sources": h.listSources(),
		})
		return
	}

	// 1. 调爬虫
	items, err := crawler.Fetch(keyword, limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "fetch failed",
			"detail": err.Error(),
		})
		return
	}

	// 2. 转成 store 模型并批量入库
	//    注意：阶段 2 暂不绑 keywordID（设为 nil）
	//    阶段 7 加关键词管理时改成传 keywordID
	dbItems := make([]store.HotItem, 0, len(items))
	for _, it := range items {
		dbItems = append(dbItems, store.FromBiz(it, nil))
	}
	inserted, err := h.hotRepo.BatchCreate(dbItems)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "db insert failed",
			"detail": err.Error(),
		})
		return
	}

	// 3. 返回入库结果
	c.JSON(http.StatusOK, gin.H{
		"data": dbItems,
		"meta": gin.H{
			"source":    source,
			"keyword":   keyword,
			"fetched":   len(items),
			"inserted":  inserted, // 实际入库行数（受去重影响）
			"fetchedAt": time.Now().Unix(),
		},
	})
}

// ListHots GET /api/hots?source=hn&keywordId=0&since=24h&limit=20&offset=0
//
// 从数据库读热点列表，支持：
//   - source: 来源过滤
//   - keywordId: 监控关键词 ID 过滤
//   - since: 时间范围，比如 1h / 24h / 7d，省略则不过滤
//   - limit/offset: 分页
func (h *Handler) ListHots(c *gin.Context) {
	source := c.Query("source")

	keywordID := int64(0)
	if k := c.Query("keywordId"); k != "" {
		if n, err := strconv.ParseInt(k, 10, 64); err == nil {
			keywordID = n
		}
	}

	// since 解析：1h / 24h / 7d / 30d
	// Go 标准库提供 time.ParseDuration，但只支持 h/m/s 不支持 d，
	// 所以 d（天）需要单独处理。
	var since time.Time
	if s := c.Query("since"); s != "" {
		d, err := parseSince(s)
		if err == nil {
			since = time.Now().Add(-d)
		}
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	items, total, err := h.hotRepo.List(source, keywordID, since, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "db query failed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{
			"source":    source,
			"keywordId": keywordID,
			"since":     c.Query("since"),
			"limit":     limit,
			"offset":    offset,
			"total":     total,
			"count":     len(items),
		},
	})
}

// GetHot 取单条热点
func (h *Handler) GetHot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	item, err := h.hotRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

// ListSources 返回所有可用信息源
func (h *Handler) ListSources(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data":  h.listSources(),
		"count": len(h.crawlers),
	})
}

// listSources 内部 helper
func (h *Handler) listSources() []string {
	out := make([]string, 0, len(h.crawlers))
	for name := range h.crawlers {
		out = append(out, name)
	}
	return out
}

// parseSince "1h" "24h" "7d" → time.Duration
//
// Go 标准库 time.ParseDuration 支持 ns/µs/ms/s/m/h，
// 不支持 d（天），所以手动处理。
func parseSince(s string) (time.Duration, error) {
	if len(s) > 1 && s[len(s)-1] == 'd' {
		days, err := strconv.Atoi(s[:len(s)-1])
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}