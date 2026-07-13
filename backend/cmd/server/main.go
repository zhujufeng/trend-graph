// package main 表示这是一个可执行程序。
// main.go 是程序入口，约定放在 cmd/server/ 下。
package main

// 本阶段引入了 Gin 框架，所以多了几个 import：
// - fmt: 打印启动信息
// - log: 错误日志
// - os: 读取环境变量（端口）
// - gin-gonic/gin: Web 框架
// - trend-graph/internal/api: 我们自己写的路由层
// - trend-graph/internal/crawler: 爬虫实现
//
// 注意 import 顺序的分组写法：
//   1) 标准库在前
//   2) 第三方库在中
//   3) 项目内部包在后
// 这是 Go 社区的共识写法，gofmt 会自动按字母排序但不会分组，
// 分组靠人手维护，有助于快速看出依赖来源。
import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/api"
	"trend-graph/internal/crawler"
)

// main 是程序入口函数，Go 程序从这里开始执行。
func main() {
	// 1. 启动横幅，让用户看到服务起来了
	fmt.Println("============================================")
	fmt.Println("  trend-graph backend")
	fmt.Println("  AI 热点监控 + 关联图谱工具")
	fmt.Println("  技术栈: Go + TypeScript")
	fmt.Println("============================================")

	// 2. 读端口配置，没设置就用 8080
	// 这个模式（默认值兜底）在 Go 项目里到处都是
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 3. 装配依赖（依赖注入）
	// 这一步在项目变大后会变成一个 init 文件，本阶段先简单写在 main 里。
	// 阶段 1 只有 HackerNews 一个爬虫，阶段 5 会变成 9 个。
	hnCrawler := crawler.NewHackerNewsCrawler()
	handler := api.NewHandler(hnCrawler)

	// 4. 创建 Gin 实例并注册路由
	// gin.Default() 自带 Logger 和 Recovery 两个中间件：
	// - Logger: 每个请求打印一行日志（方法、路径、状态码、耗时）
	// - Recovery: panic 自动恢复，不让进程挂掉
	r := gin.Default()

	// 健康检查接口（阶段 0 已有，这里迁移成 Gin 风格）
	// 用来给 Docker / k8s 做存活探针，或者给前端确认后端活着
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "trend-graph",
			"version": "0.1.0",
		})
	})

	// 注册业务路由（/api/hots 等）
	handler.Register(r)

	// 5. 启动 HTTP 服务
	// gin.Engine 实现了 http.Handler 接口，可以直接传给 http.ListenAndServe
	// 这里用 r.Run 是 Gin 的简写，内部就是 http.ListenAndServe
	addr := ":" + port
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Println("接口:")
	fmt.Println("  健康检查: GET http://localhost:" + port + "/health")
	fmt.Println("  热点列表: GET http://localhost:" + port + "/api/hots?source=hn&limit=20")
	fmt.Println("按 Ctrl+C 退出")

	// r.Run 阻塞主 goroutine，直到服务停止或出错
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}