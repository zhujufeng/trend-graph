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
	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

func main() {
	// 1. 启动横幅
	fmt.Println("============================================")
	fmt.Println("  trend-graph backend")
	fmt.Println("  AI 热点监控 + 关联图谱工具")
	fmt.Println("  技术栈: Go + TypeScript + PostgreSQL")
	fmt.Println("============================================")

	// 2. 加载配置
	cfg := config.Load()
	fmt.Printf("配置加载完成: 端口=%s, 模型=%s\n", cfg.Port, cfg.DeepSeekModel)

	// 3. 初始化数据库
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 4. 初始化 AI（可选）
	var an *analyzer.Analyzer
	if cfg.DeepSeekAPIKey != "" {
		deepseek := ai.NewDeepSeekClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL)
		an = analyzer.NewAnalyzer(deepseek, cfg.DeepSeekModel)
		fmt.Println("AI 初始化完成: DeepSeek 已就绪")
	} else {
		fmt.Println("警告: 未配置 DEEPSEEK_API_KEY，AI 相关接口将返回 503")
	}

	// 5. 装配所有爬虫（阶段 5）
	//    这里把 9 个源的构造函数挨个调用，
	//    以后加新源只需要在这里加一行。
	//
	// 用 builder 风格组装：
	// 类型是 []types.Crawler（接口切片）
	var crawlers []types.Crawler = []types.Crawler{
		crawler.NewHackerNewsCrawler(),
		crawler.NewGitHubCrawler(),
		crawler.NewRedditCrawler(),
		crawler.NewBingCrawler(),
		crawler.NewBilibiliCrawler(),
		crawler.NewZhihuCrawler(),
		crawler.NewWeiboCrawler(),
		crawler.NewLinuxDoCrawler(),
		crawler.NewTwitterCrawler(), // 没 token 的 Fetch 会返回 ErrTwitterNoToken，会被上层当失败源跳过
	}
	// 多源调度器（让 source=all 能一次抓全部）
	multi := crawler.NewMultiCrawler(crawlers...)

	// 6. 装配依赖
	hotRepo := store.NewHotItemRepo(db)
	// 把 multi + 9 个单源都注册进 handler
	// 这样前端 source=hn / weibo / all 都能用
	handler := api.NewHandler(hotRepo, an, append(crawlers, multi)...)

	// 7. 启动 Gin
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "trend-graph",
			"version": "0.5.0",
			"sources": len(crawlers),
		})
	})
	handler.Register(r)

	// 8. 启动信息
	addr := ":" + cfg.Port
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Printf("已注册 %d 个信息源（source=all 可一次抓全部）:\n", len(crawlers))
	for _, c := range crawlers {
		fmt.Printf("  - %s\n", c.Source())
	}
	fmt.Println("")
	fmt.Println("接口:")
	fmt.Println("  触发抓取: POST http://localhost:" + cfg.Port + "/api/crawl?source=all&limit=10&analyze=true&keyword=AI")
	fmt.Println("  读热点:   GET  http://localhost:" + cfg.Port + "/api/hots?source=hn&since=7d")
	fmt.Println("  列信息源: GET  http://localhost:" + cfg.Port + "/api/sources")
	fmt.Println("按 Ctrl+C 退出")

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}