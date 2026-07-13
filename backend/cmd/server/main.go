// package main 是程序入口
package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/ai"
	"trend-graph/internal/analyzer"
	"trend-graph/internal/api"
	"trend-graph/internal/config"
	"trend-graph/internal/crawler"
	"trend-graph/internal/notify"
	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

func main() {
	// 1. 横幅
	fmt.Println("============================================")
	fmt.Println("  trend-graph backend")
	fmt.Println("  AI 热点监控 + 关联图谱工具")
	fmt.Println("  技术栈: Go + TypeScript + PostgreSQL")
	fmt.Println("============================================")

	// 2. 配置
	cfg := config.Load()
	fmt.Printf("配置加载完成: 端口=%s, 模型=%s\n", cfg.Port, cfg.DeepSeekModel)

	// 3. 数据库
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 4. AI（可选）
	var an *analyzer.Analyzer
	if cfg.DeepSeekAPIKey != "" {
		deepseek := ai.NewDeepSeekClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL)
		an = analyzer.NewAnalyzer(deepseek, cfg.DeepSeekModel)
		fmt.Println("AI 初始化完成: DeepSeek 已就绪")
	} else {
		fmt.Println("警告: 未配置 DEEPSEEK_API_KEY，AI 相关接口将返回 503")
	}

	// 5. WebSocket Hub（新）
	//    阶段 6 加入实时推送能力
	wsHub := notify.NewWebSocketHub()
	// 在后台启动 Hub 主循环
	go wsHub.Run()
	fmt.Println("WebSocket Hub 已启动")

	// 6. 装配 9 个爬虫 + multi 调度器
	var crawlers []types.Crawler = []types.Crawler{
		crawler.NewHackerNewsCrawler(),
		crawler.NewGitHubCrawler(),
		crawler.NewRedditCrawler(),
		crawler.NewBingCrawler(),
		crawler.NewBilibiliCrawler(),
		crawler.NewZhihuCrawler(),
		crawler.NewWeiboCrawler(),
		crawler.NewLinuxDoCrawler(),
		crawler.NewTwitterCrawler(),
	}
	multi := crawler.NewMultiCrawler(crawlers...)

	// 7. 装配 handler，新增 wsHub 依赖
	hotRepo := store.NewHotItemRepo(db)
	handler := api.NewHandler(hotRepo, an, wsHub, append(crawlers, multi)...)

	// 8. 启动 Gin
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "trend-graph",
			"version": "0.6.0",
			"sources": len(crawlers),
			"ws":      true,
		})
	})
	handler.Register(r)

	// 阶段 6：直接注册 /ws 路由（让 wsHub 处理）
	// WebSocket 路由不用走 API handler，直接挂到 root
	r.GET("/ws", func(c *gin.Context) {
		wsHub.HandleWS(c.Writer, c.Request)
	})

	// 9. 启动信息
	addr := ":" + cfg.Port
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Printf("已注册 %d 个信息源（source=all 可一次抓全部）\n", len(crawlers))
	fmt.Println("接口:")
	fmt.Println("  触发抓取: POST http://localhost:" + cfg.Port + "/api/crawl?source=all&limit=10&analyze=true&keyword=AI")
	fmt.Println("  读热点: GET  http://localhost:" + cfg.Port + "/api/hots?source=hn&since=7d")
	fmt.Println("  WebSocket: ws://localhost:" + cfg.Port + "/ws")
	fmt.Println("按 Ctrl+C 退出")

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}