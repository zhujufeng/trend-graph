// package main 表示这是一个可执行程序。
// main.go 是程序入口，约定放在 cmd/server/ 下。
package main

// 阶段 2 多了两个 import：
//   - trend-graph/internal/config: 加载配置
//   - trend-graph/internal/store: 数据库初始化和访问
import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/api"
	"trend-graph/internal/config"
	"trend-graph/internal/crawler"
	"trend-graph/internal/store"
)

// main 是程序入口函数。
func main() {
	// 1. 启动横幅
	fmt.Println("============================================")
	fmt.Println("  trend-graph backend")
	fmt.Println("  AI 热点监控 + 关联图谱工具")
	fmt.Println("  技术栈: Go + TypeScript + PostgreSQL")
	fmt.Println("============================================")

	// 2. 加载配置
	// config.Load() 内部会从 .env 和环境变量加载，并校验必需项
	cfg := config.Load()
	fmt.Printf("配置加载完成: 端口=%s, DeepSeek 模型=%s\n", cfg.Port, cfg.DeepSeekModel)

	// 3. 初始化数据库（新加）
	// 失败直接退出，让运维立刻知道配置问题
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 4. 装配依赖
	// 现在依赖链清晰：
	//   db (gorm.DB) → hotRepo (CRUD)
	//   crawler (HackerNews)
	//   handler 注入 hotRepo + crawler
	hnCrawler := crawler.NewHackerNewsCrawler()
	hotRepo := store.NewHotItemRepo(db)
	handler := api.NewHandler(hotRepo, hnCrawler)

	// 5. 创建 Gin 实例并注册路由
	r := gin.Default()

	// 健康检查（阶段 0 风格保留）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "trend-graph",
			"version": "0.2.0",
		})
	})

	// 业务路由
	handler.Register(r)

	// 6. 启动服务
	addr := ":" + cfg.Port
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Println("接口:")
	fmt.Println("  健康检查:        GET  http://localhost:" + cfg.Port + "/health")
	fmt.Println("  触发抓取+入库:   POST http://localhost:" + cfg.Port + "/api/crawl?source=hn&limit=10")
	fmt.Println("  读热点列表:      GET  http://localhost:" + cfg.Port + "/api/hots?source=hn&since=24h")
	fmt.Println("  取单条热点:      GET  http://localhost:" + cfg.Port + "/api/hots/{id}")
	fmt.Println("  列出信息源:      GET  http://localhost:" + cfg.Port + "/api/sources")
	fmt.Println("按 Ctrl+C 退出")

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}