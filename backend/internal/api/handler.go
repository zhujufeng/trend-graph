// Package api 定义 HTTP 路由和 Handler。
//
// 阶段 3 改动：
//   - Handler 多依赖一个 *analyzer.Analyzer
//   - 路由新增:
//       POST /api/expand           查询扩展
//       POST /api/analyze/:id      对已入库的某一热点做 AI 分析
//       POST /api/crawl?analyze=true  抓取后自动分析
package api

// 导入：
// - context: 给 AI 调用设置超时（DeepSeek 慢的话会被取消）
// - encoding/json: 把 AI 分析的实体列表打包成 JSON 存库
// - net/http / strconv / time: 同阶段 2
// - gin / store / types
// - trend-graph/internal/analyzer 新加
import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/analyzer"
	"trend-graph/internal/notify"
	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

// Handler 是 API 控制器
type Handler struct {
	crawlers map[string]types.Crawler
	hotRepo  *store.HotItemRepo
	keywordRepo *store.KeywordRepo
	// 阶段 8 新增：图谱存储
	graphRepo *store.GraphRepo
	analyzer    *analyzer.Analyzer
	wsHub *notify.WebSocketHub
}

// NewHandler 构造函数（参数继续扩展）
func NewHandler(
	hotRepo *store.HotItemRepo,
	keywordRepo *store.KeywordRepo,
	graphRepo *store.GraphRepo,
	an *analyzer.Analyzer,
	wsHub *notify.WebSocketHub,
	crawlers ...types.Crawler,
) *Handler {
	m := make(map[string]types.Crawler, len(crawlers))
	for _, c := range crawlers {
		m[c.Source()] = c
	}
	return &Handler{
		crawlers:    m,
		hotRepo:     hotRepo,
		keywordRepo: keywordRepo,
		graphRepo:   graphRepo,
		analyzer:    an,
		wsHub:       wsHub,
	}
}

// Register 路由注册
func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/hots", h.ListHots)
		api.GET("/hots/:id", h.GetHot)
		api.POST("/crawl", h.TriggerCrawl)
		api.GET("/sources", h.ListSources)

		// 阶段 3 新增
		api.POST("/expand", h.ExpandQuery)
		api.POST("/analyze/:id", h.AnalyzeHot)

		// 阶段 7 新增：关键词管理
		api.GET("/keywords", h.ListKeywords)
		api.POST("/keywords", h.CreateKeyword)
		api.PATCH("/keywords/:id", h.UpdateKeyword)
		api.DELETE("/keywords/:id", h.DeleteKeyword)

		// 阶段 8 新增：关联图谱
		api.GET("/graph", h.GetGraph)
	}
}

// ===== 阶段 8：关联图谱 =====

// GetGraph GET /api/graph?keywordId=1 或 /api/graph?keyword=AI
//
// 返回 GraphData: 节点 + 边，前端用 React Flow 渲染。
func (h *Handler) GetGraph(c *gin.Context) {
	if h.graphRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "graphRepo not configured"})
		return
	}

	// 优先用 keywordId 直接查
	var keywordID int64
	if k := c.Query("keywordId"); k != "" {
		keywordID, _ = strconv.ParseInt(k, 10, 64)
	}
	keywordWord := c.Query("keyword")

	// 没传 keywordId 就用 keyword 文本反查
	if keywordID == 0 && keywordWord != "" {
		kw, err := h.keywordRepo.List(false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query keywords failed", "detail": err.Error()})
			return
		}
		for _, k := range kw {
			if k.Word == keywordWord {
				keywordID = k.ID
				break
			}
		}
		if keywordID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "keyword not found", "keyword": keywordWord})
			return
		}
	}
	if keywordID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "need keywordId or keyword"})
		return
	}

	g, err := h.graphRepo.GetGraph(keywordID, keywordWord)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "graph query failed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": g,
		"meta": gin.H{"keywordId": keywordID, "nodes": len(g.Nodes), "edges": len(g.Edges)},
	})
}

// ingestEntitiesToGraph 把 AI 抽取的实体写入图谱（关键词→实体→热点三方关系）
//
// 这个 helper 在 TriggerCrawl 和 AnalyzeHot 两处都会被调用，
// 因为图谱关系是 AI 分析后才能产生（需要 typedEntities）。
func (h *Handler) ingestEntitiesToGraph(hotID int64, keywordID *int64, res *analyzer.AnalysisResult) {
	if h.graphRepo == nil {
		return
	}

	// 1. 实体入库 → 拿 entityID
	entityIDs := make([]int64, 0, len(res.TypedEntities))
	for _, te := range res.TypedEntities {
		id, err := h.graphRepo.EnsureEntity(te.Name, te.Kind)
		if err != nil {
			continue
		}
		entityIDs = append(entityIDs, id)
	}
	if len(entityIDs) == 0 {
		return
	}

	// 2. 热点 → 实体 的 contains 边
	for _, eid := range entityIDs {
		_ = h.graphRepo.EnsureRelation("hot", hotID, "contains", "entity", eid, &hotID)
	}

	// 3. 关键词 → 实体 的 tracks 边（如果传了 keywordID）
	if keywordID != nil {
		_ = h.graphRepo.TrackKeywordToEntities(*keywordID, entityIDs, &hotID)
	}

	// 4. 实体之间两两共现边
	for i := 0; i < len(entityIDs); i++ {
		for j := i + 1; j < len(entityIDs); j++ {
			a, b := entityIDs[i], entityIDs[j]
			if a > b {
				a, b = b, a
			}
			_ = h.graphRepo.EnsureRelation("entity", a, "cooccur", "entity", b, &hotID)
		}
	}
}

// ===== 阶段 7：关键词管理 =====

// CreateKeyword POST /api/keywords
// Body: {"word":"AI","note":"监控大模型动态","intervalMin":30}
func (h *Handler) CreateKeyword(c *gin.Context) {
	if h.keywordRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "keywordRepo not configured"})
		return
	}
	var body struct {
		Word        string `json:"word"`
		Note        string `json:"note"`
		IntervalMin int    `json:"intervalMin"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Word == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "word 必填"})
		return
	}
	k, err := h.keywordRepo.Create(body.Word, body.Note, body.IntervalMin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": k})
}

// ListKeywords GET /api/keywords?activeOnly=true
func (h *Handler) ListKeywords(c *gin.Context) {
	if h.keywordRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "keywordRepo not configured"})
		return
	}
	activeOnly := c.Query("activeOnly") == "true"
	ks, err := h.keywordRepo.List(activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ks, "count": len(ks)})
}

// UpdateKeyword PATCH /api/keywords/:id
// Body: {"active":false} 或 {"intervalMin":60}
func (h *Handler) UpdateKeyword(c *gin.Context) {
	if h.keywordRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "keywordRepo not configured"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		Active      *bool `json:"active"`
		IntervalMin *int  `json:"intervalMin"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if body.Active != nil {
		if err := h.keywordRepo.UpdateActive(id, *body.Active); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed", "detail": err.Error()})
			return
		}
	}
	if body.IntervalMin != nil {
		if err := h.keywordRepo.UpdateInterval(id, *body.IntervalMin); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed", "detail": err.Error()})
			return
		}
	}
	k, _ := h.keywordRepo.Get(id)
	c.JSON(http.StatusOK, gin.H{"data": k})
}

// DeleteKeyword DELETE /api/keywords/:id
func (h *Handler) DeleteKeyword(c *gin.Context) {
	if h.keywordRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "keywordRepo not configured"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.keywordRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "deleted": true}})
}

// ExpandQuery POST /api/expand
//
// 请求体: {"keyword": "AI"}
// 响应体: {"data": ["AI","人工智能",...], "meta": {...}}
//
// 让 AI 把一个关键词扩展成同义词/相关词，提升后续抓取召回率。
//
// 注意 AI 是必需依赖，没配 analyzer 就返回 503。
func (h *Handler) ExpandQuery(c *gin.Context) {
	if h.analyzer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 未配置（analyzer=nil）"})
		return
	}
	var body struct {
		Keyword string `json:"keyword"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "keyword 必填"})
		return
	}

	// 给 AI 调用 30 秒超时，避免 DeepSeek 偶尔卡住整个请求
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	keywords, err := h.analyzer.ExpandQuery(ctx, body.Keyword)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "ai failed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": keywords,
		"meta": gin.H{"keyword": body.Keyword, "count": len(keywords)},
	})
}

// AnalyzeHot POST /api/analyze/:id?keyword=AI
//
// 对数据库中一条热点做 AI 综合分析，把摘要/相关性/真假/实体写回数据库。
//
// 流程：
//   1) 查数据库拿原始热点
//   2) 调 analyzer.AnalyzeHot 做 AI 分析
//   3) 用 hotRepo.UpdateAIResult 写回 4 个字段
//   4) 返回分析结果给前端
func (h *Handler) AnalyzeHot(c *gin.Context) {
	if h.analyzer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI 未配置"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// keyword 是监控关键词（用户传入），用于判断相关性
	keyword := c.Query("keyword")

	// 1. 查数据库
	item, err := h.hotRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// 2. AI 分析
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// store.HotItem → types.HotItem 才能 fit 进 analyzer 的签名
	bizItem := types.HotItem{
		Title:  item.Title,
		URL:    item.URL,
		Source: item.Source,
		Hot:    item.Hot,
		Author: item.Author,
	}
	res, err := h.analyzer.AnalyzeHot(ctx, keyword, bizItem)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "ai failed", "detail": err.Error()})
		return
	}

	// 3. 实体列表转 JSON 字符串存库
	entitiesJSON, _ := json.Marshal(res.Entities)

	// 4. 写回数据库
	if err := h.hotRepo.UpdateAIResult(id, res.Summary, res.Relevance, res.IsAuthentic, string(entitiesJSON)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed", "detail": err.Error()})
		return
	}

	// 阶段 8：实体写入图谱
	// 注意 AnalyzeHot 接口当前没有 keywordID，只能 ingest hot→entity，不建 keyword→entity 边
	h.ingestEntitiesToGraph(id, nil, res)

	// 5. 返回前端完整结果
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":          id,
			"title":       item.Title,
			"summary":     res.Summary,
			"relevance":   res.Relevance,
			"isAuthentic": res.IsAuthentic,
			"entities":    res.Entities,
			"reason":      res.Reason,
		},
		"meta": gin.H{"keyword": keyword},
	})
}

// TriggerCrawl POST /api/crawl?source=hn&keyword=AI&limit=5&analyze=true
//
// 阶段 2 已有；阶段 3 多加 ?analyze=true 触发自动 AI 分析。
//
// 注意这是阶段 3 的关键改动，把抓取与 AI 串成一条龙。
func (h *Handler) TriggerCrawl(c *gin.Context) {
	source := c.DefaultQuery("source", "hn")
	keyword := c.Query("keyword")

	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	// 新：是否触发 AI 分析。空 / "0" / "false" 都视为否
	analyze := c.Query("analyze") == "true" || c.Query("analyze") == "1"

	// 1. 查找爬虫
	crawler, ok := h.crawlers[source]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "unknown source",
			"sources": h.listSources(),
		})
		return
	}

	// 2. 抓取
	items, err := crawler.Fetch(keyword, limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "fetch failed", "detail": err.Error()})
		return
	}

	// 3. 入库
	dbItems := make([]store.HotItem, 0, len(items))
	for _, it := range items {
		dbItems = append(dbItems, store.FromBiz(it, nil))
	}
	inserted, err := h.hotRepo.BatchCreate(dbItems)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db insert failed", "detail": err.Error()})
		return
	}

	// 4. 可选 AI 分析
	//    analyze=true 且 analyzer 已配置才会执行
	//    每分析完一条就通过 WS 广播，前端能逐条冒出
	analyzed := 0
	if analyze && h.analyzer != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
		defer cancel()
		for i := range dbItems {
			res, err := h.analyzer.AnalyzeHot(ctx, keyword, items[i])
			if err != nil {
				continue
			}
			entitiesJSON, _ := json.Marshal(res.Entities)
			_ = h.hotRepo.UpdateAIResult(dbItems[i].ID, res.Summary, res.Relevance, res.IsAuthentic, string(entitiesJSON))
			// 同步更新内存对象，让响应里也带着 AI 结果
			dbItems[i].Summary = res.Summary
			dbItems[i].Relevance = &res.Relevance
			dbItems[i].IsAuthentic = &res.IsAuthentic
			dbItems[i].Entities = string(entitiesJSON)
			analyzed++

			// 阶段 8：把 AI 抽取的实体写入图谱
			// TriggerCrawl 时 keyword 不一定有 keywordID（手动抓取不绑定监控关键词），keywordID 传 nil
			h.ingestEntitiesToGraph(dbItems[i].ID, nil, res)

			// 阶段 6：单条分析完成 → WS 推给前端
			if h.wsHub != nil {
				_ = h.wsHub.SendAnalyzeDone(gin.H{
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

	// 5. 阶段 6：抓取完成 → WS 广播给所有在线客户端
	//    前端监听到这个事件会自动刷新列表
	if h.wsHub != nil {
		_ = h.wsHub.SendCrawlDone(gin.H{
			"source":   source,
			"keyword":  keyword,
			"fetched":  len(items),
			"inserted": inserted,
			"analyzed": analyzed,
		})
	}

	// 6. 返回 HTTP 响应
	c.JSON(http.StatusOK, gin.H{
		"data": dbItems,
		"meta": gin.H{
			"source":    source,
			"keyword":   keyword,
			"fetched":   len(items),
			"inserted":  inserted,
			"analyzed":  analyzed,
			"analyze":   analyze,
			"fetchedAt": time.Now().Unix(),
		},
	})
}

// ===== 以下是阶段 2 没改的代码（保持一致） =====

// ListHots GET /api/hots?source=hn&keywordId=0&since=24h&limit=20&offset=0
func (h *Handler) ListHots(c *gin.Context) {
	source := c.Query("source")

	keywordID := int64(0)
	if k := c.Query("keywordId"); k != "" {
		if n, err := strconv.ParseInt(k, 10, 64); err == nil {
			keywordID = n
		}
	}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed", "detail": err.Error()})
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

// ListSources 列出所有可用信息源
func (h *Handler) ListSources(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": h.listSources(), "count": len(h.crawlers)})
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