// package main 是程序入口
package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/ai"
	"trend-graph/internal/analyzer"
	"trend-graph/internal/api"
	"trend-graph/internal/config"
	"trend-graph/internal/crawler"
	"trend-graph/internal/notify"
	"trend-graph/internal/scheduler"
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
		cli := ai.NewDeepSeekClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL)
		an = analyzer.NewAnalyzer(cli, cfg.DeepSeekModel)
		fmt.Println("AI 初始化完成: DeepSeek 已就绪")
	}

	// 5. WebSocket Hub
	wsHub := notify.NewWebSocketHub()
	go wsHub.Run()
	fmt.Println("WebSocket Hub 已启动")

	// 6. 装配 9 个爬虫 + multi
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

	// 7. 装配 repo
	hotRepo := store.NewHotItemRepo(db)
	keywordRepo := store.NewKeywordRepo(db)

	// 8. 装配通知渠道（阶段 7）
	//    把 SMTP / 飞书 / 钉钉任一配置了就启用，组成 MultiChannelNotifier
	var notifiers []notify.Notifier
	if cfg.SMTPUser != "" && cfg.SMTPTo != "" {
		port, _ := strconv.Atoi(cfg.SMTPPort)
		notifiers = append(notifiers, notify.NewEmailNotifier(cfg.SMTPHost, port, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.SMTPTo))
		fmt.Println("邮件通知已启用")
	}
	if cfg.FeishuWebhook != "" {
		notifiers = append(notifiers, notify.NewFeishuNotifier(cfg.FeishuWebhook))
		fmt.Println("飞书 Webhook 已启用")
	}
	if cfg.DingTalkWebhook != "" {
		notifiers = append(notifiers, notify.NewDingTalkNotifier(cfg.DingTalkWebhook, cfg.DingTalkSecret))
		fmt.Println("钉钉 Webhook 已启用")
	}
	// wsHub 也作为一个 channel（实时在线推送）
	notifiers = append(notifiers, wsHub)
	multiNotifier := notify.NewMultiChannelNotifier(notifiers...)

	// 9. 启动调度器（阶段 7）
	//    把爬虫/AI/通知全串起来，按 keywords 表里的间隔自动跑
	sched := scheduler.NewScheduler(multi, hotRepo, keywordRepo, an, multiNotifier, wsHub)
	sched.Start()

	// 10. handler 装配
	handler := api.NewHandler(hotRepo, keywordRepo, an, wsHub, append(crawlers, multi)...)

	// 11. 启动 Gin
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":      "ok",
			"service":     "trend-graph",
			"version":     "0.7.0",
			"sources":     len(crawlers),
			"ws":          true,
			"schedulers":  len(sched.Entries()),
			"notifiers":   len(notifiers),
		})
	})
	handler.Register(r)

	// WS 路由（不走 /api 前缀）
	r.GET("/ws", func(c *gin.Context) {
		wsHub.HandleWS(c.Writer, c.Request)
	})

	// 12. 启动信息
	addr := ":" + cfg.Port
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Printf("已注册 %d 个信息源 + %d 个通知渠道\n", len(crawlers), len(notifiers))
	fmt.Println("接口:")
	fmt.Println("  抓取:      POST http://localhost:" + cfg.Port + "/api/crawl?source=all&limit=10&analyze=true&keyword=AI")
	fmt.Println("  读热点:    GET  http://localhost:" + cfg.Port + "/api/hots")
	fmt.Println("  关键词:    GET  http://localhost:" + cfg.Port + "/api/keywords")
	fmt.Println("  WebSocket: ws://localhost:" + cfg.Port + "/ws")
	fmt.Println("按 Ctrl+C 退出")

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}