// package main 表示这是一个可执行程序。
package main

// 阶段 3 多了 analyzer 包
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
)

// main 是程序入口。
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
	//    如果没配 DEEPSEEK_API_KEY 就跳过，让基础抓取仍然能跑
	var an *analyzer.Analyzer
	if cfg.DeepSeekAPIKey != "" {
		deepseek := ai.NewDeepSeekClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL)
		an = analyzer.NewAnalyzer(deepseek, cfg.DeepSeekModel)
		fmt.Println("AI 初始化完成: DeepSeek 已就绪")
	} else {
		fmt.Println("警告: 未配置 DEEPSEEK_API_KEY，AI 相关接口将返回 503")
	}

	// 5. 装配依赖
	//    现在依赖链:
	//      db → hotRepo
	//      hnCrawler
	//      deepseek → analyzer (可选)
	//      handler 注入 hotRepo + analyzer + crawler
	hnCrawler := crawler.NewHackerNewsCrawler()
	hotRepo := store.NewHotItemRepo(db)
	handler := api.NewHandler(hotRepo, an, hnCrawler)

	// 6. 启动 Gin
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "trend-graph",
			"version": "0.3.0",
		})
	})
	handler.Register(r)

	// 7. 启动
	addr := ":" + cfg.Port
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Println("接口:")
	fmt.Println("  健康检查:           GET  http://localhost:" + cfg.Port + "/health")
	fmt.Println("  触发抓取(+AI可选):  POST http://localhost:" + cfg.Port + "/api/crawl?source=hn&limit=5&analyze=true&keyword=AI")
	fmt.Println("  读热点列表:        GET  http://localhost:" + cfg.Port + "/api/hots?source=hn&since=7d")
	fmt.Println("  取单条热点:        GET  http://localhost:" + cfg.Port + "/api/hots/{id}")
	fmt.Println("  查询扩展(新):       POST http://localhost:" + cfg.Port + "/api/expand  body: {\"keyword\":\"AI\"}")
	fmt.Println("  分析热点(新):       POST http://localhost:" + cfg.Port + "/api/analyze/{id}?keyword=AI")
	fmt.Println("  列出信息源:         GET  http://localhost:" + cfg.Port + "/api/sources")
	fmt.Println("按 Ctrl+C 退出")

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}