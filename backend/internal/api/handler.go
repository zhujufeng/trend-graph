// Package api 定义 HTTP 路由和 Handler。
//
// Go Web 项目常见分层：
//   cmd/server/main.go        → 程序入口，组装依赖并启动
//   internal/api/             → 路由 + Handler（接受请求、返回 JSON）
//   internal/crawler/         → 业务逻辑（抓取）
//   internal/types/           → 共享数据结构
//
// 这种分层让 main.go 保持极简，业务改动不影响启动代码。
package api

// 本文件依赖：
// - net/http: 用 http.StatusOK 这种状态常量
// - gin-gonic/gin: HTTP 框架，提供路由和 JSON 响应简写
// - trend-graph/internal/crawler: 注入爬虫依赖
// - trend-graph/internal/types: HotItem 类型
import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/types"
)

// Handler 是 API 层的"控制器"。
//
// 它持有所有依赖（爬虫、未来的 AI、数据库、通知器...），
// 通过方法（如 GetHots）暴露成 Gin Handler。
//
// 为什么用 struct 而不是全局变量？
// - 全局变量让测试和替换很困难
// - struct + 依赖注入更灵活，比如阶段 5 加新爬虫只改一处
type Handler struct {
	// crawlers 是所有已注册的爬虫列表。
	// 阶段 1 只有一个 HackerNews，阶段 5 会变成 9 个。
	// 用 map 是为按 source 名快速查找，比如 GET /api/hots?source=hn
	crawlers map[string]types.Crawler
}

// NewHandler 构造 Handler，注入爬虫依赖。
//
// 这里参数用可变参数 ...types.Crawler，
// 调用方可传 1 个或多个爬虫，main.go 里会清晰看到组装过程。
func NewHandler(crawlers ...types.Crawler) *Handler {
	m := make(map[string]types.Crawler, len(crawlers))
	for _, c := range crawlers {
		m[c.Source()] = c
	}
	return &Handler{crawlers: m}
}

// Register 把所有路由挂到 Gin engine 上。
//
// 为什么用方法而不是直接写 init()?
// - init() 是 Go 包级别的自动调用，时机不可控
// - 显式 Register 让 main.go 看清楚注册了哪些路由
func (h *Handler) Register(r *gin.Engine) {
	// 路由分组：所有 API 都在 /api 前缀下
	api := r.Group("/api")
	{
		// GET /api/hots 取热点列表
		// 支持两个 query 参数:
		//   - source: 信息源（默认 "hn"，阶段 5 改成 "all" 表示所有）
		//   - keyword: 监控关键词（可选）
		//   - limit: 抓取条数（默认 20）
		api.GET("/hots", h.GetHots)
	}
}

// GetHots 处理 GET /api/hots。
//
// Gin Handler 签名固定为 func(*gin.Context)。
// c.Query 读 query 参数，c.JSON 返回 JSON 响应。
func (h *Handler) GetHots(c *gin.Context) {
	// 1. 读 query 参数，给默认值
	source := c.DefaultQuery("source", "hn")
	keyword := c.Query("keyword") // 没传就是空字符串

	// limit 是字符串参数，需要转成 int。
	// strconv.Atoi 是 Go 把字符串转 int 的标准做法。
	// 转失败就用默认值 20（不报错给用户，宽容一点）。
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	// 2. 查找对应爬虫
	crawler, ok := h.crawlers[source]
	if !ok {
		// source 不存在就返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "unknown source",
			"source":  source,
			"sources": h.listSources(),
		})
		return
	}

	// 3. 调用爬虫抓取
	items, err := crawler.Fetch(keyword, limit)
	if err != nil {
		// 爬虫失败返回 502，类似网关错误
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "fetch failed",
			"detail": err.Error(),
		})
		return
	}

	// 4. 成功返回标准结构
	// 项目约定统一响应格式: { "data": ..., "meta": {...} }
	// 写 API 时不要随意换结构，前端会依赖
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{
			"source":  source,
			"keyword": keyword,
			"limit":   limit,
			"count":   len(items),
		},
	})
}

// listSources 返回所有已注册的 source 名，便于前端做下拉框
func (h *Handler) listSources() []string {
	out := make([]string, 0, len(h.crawlers))
	for name := range h.crawlers {
		out = append(out, name)
	}
	return out
}