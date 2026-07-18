// package main 是程序入口
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/ai"
	"trend-graph/internal/analyzer"
	"trend-graph/internal/api"
	"trend-graph/internal/auth"
	"trend-graph/internal/config"
	"trend-graph/internal/notify"
	"trend-graph/internal/radar"
	"trend-graph/internal/store"
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

	// 7. 装配 repo
	hotRepo := store.NewHotItemRepo(db)
	keywordRepo := store.NewKeywordRepo(db)
	if err := keywordRepo.EnsureDefault(); err != nil {
		log.Fatalf("初始化关注主题失败: %v", err)
	}
	graphRepo := store.NewGraphRepo(db) // 阶段 8 新增
	signalRepo := store.NewSignalRepo(db)
	sourceConfigRepo := store.NewSourceConfigRepo(db)
	deliveryRepo := store.NewDeliveryRepo(db)
	if err := sourceConfigRepo.EnsureDefaults(); err != nil {
		log.Fatalf("初始化来源配置失败: %v", err)
	}
	sessionRepo := store.NewAdminSessionRepo(db)
	authService := auth.NewService(
		cfg.AdminPassword,
		sessionRepo,
		time.Duration(cfg.AdminSessionHours)*time.Hour,
		cfg.SessionCookieSecure,
	)

	// 8. 装配通知渠道（阶段 7）
	//    把 SMTP / 飞书 / 钉钉任一配置了就启用，组成 MultiChannelNotifier
	var notifiers []notify.Notifier
	var feishuNotifier *notify.FeishuNotifier
	if cfg.SMTPUser != "" && cfg.SMTPTo != "" {
		port, _ := strconv.Atoi(cfg.SMTPPort)
		notifiers = append(notifiers, notify.NewEmailNotifier(cfg.SMTPHost, port, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.SMTPTo))
		fmt.Println("邮件通知已启用")
	}
	if cfg.FeishuWebhook != "" {
		feishuNotifier = notify.NewFeishuNotifier(cfg.FeishuWebhook)
		notifiers = append(notifiers, feishuNotifier)
		fmt.Println("飞书 Webhook 已启用")
	}
	if cfg.DingTalkWebhook != "" {
		notifiers = append(notifiers, notify.NewDingTalkNotifier(cfg.DingTalkWebhook, cfg.DingTalkSecret))
		fmt.Println("钉钉 Webhook 已启用")
	}
	// wsHub 也作为一个 channel（实时在线推送）
	notifiers = append(notifiers, wsHub)
	// 旧关键词调度器会绕过来源配置并抓取 Reddit r/all，因此不再启动。
	// 新采集器经受鉴权的内部接口写入；后续调度只读取 source_configs。
	handler := api.NewHandler(hotRepo, keywordRepo, graphRepo, an, wsHub)

	// 11. 启动 Gin
	r := gin.Default()
	if cfg.InternalIngestSecret != "" {
		api.NewIngestionHandler(cfg.InternalIngestSecret, signalRepo).Register(r)
		fmt.Println("内部采集写入接口已启用")
	}

	var analysisRunner *radar.AnalysisRunner
	if an != nil {
		analysisRunner = radar.NewAnalysisRunner(signalRepo, keywordRepo, an, cfg.DeepSeekModel)
	}
	var deliveryService *radar.DeliveryService
	if feishuNotifier != nil {
		deliveryService = radar.NewDeliveryService(deliveryRepo, signalRepo, feishuNotifier)
	}
	runAnalysisAndAlerts := func() {
		if analysisRunner == nil {
			return
		}
		result, err := analysisRunner.Run(context.Background(), time.Now())
		if err != nil {
			log.Printf("信号分析失败: %v", err)
			return
		}
		log.Printf("信号分析完成: 已分析=%d 已拒绝=%d 今日余量=%d", result.Analyzed, result.Rejected, result.QuotaRemaining)
		if deliveryService != nil && cfg.MajorAlertsEnabled {
			if err := deliveryService.SendMajorAlerts(context.Background(), time.Now()); err != nil {
				log.Printf("重磅信号提醒失败: %v", err)
			}
		}
	}

	schedulerCount := 0
	var collectionRunner *radar.CollectionRunner
	if cfg.BackgroundJobsEnabled && cfg.InternalIngestSecret != "" {
		collectionRunner = radar.NewCollectionRunner(
			sourceConfigRepo,
			keywordRepo,
			cfg.CollectorDir,
			"http://127.0.0.1:"+cfg.Port,
			cfg.InternalIngestSecret,
		)
		collectionCron, err := radar.NewCollectionCron(func() {
			if err := collectionRunner.Run(context.Background()); err != nil {
				log.Printf("采集任务部分或全部失败: %v", err)
			}
			runAnalysisAndAlerts()
		})
		if err != nil {
			log.Fatalf("初始化采集调度失败: %v", err)
		}
		collectionCron.Start()
		defer collectionCron.Stop()
		schedulerCount = len(collectionCron.Entries())
		fmt.Printf("采集调度已启用: %d 条计划\n", schedulerCount)
	}
	if cfg.BackgroundJobsEnabled && deliveryService != nil && cfg.DigestEnabled {
		digestCron, err := radar.NewDigestCron(func() {
			if err := deliveryService.SendDigest(context.Background(), time.Now()); err != nil {
				log.Printf("飞书摘要发送失败: %v", err)
			}
		})
		if err != nil {
			log.Fatalf("初始化摘要调度失败: %v", err)
		}
		digestCron.Start()
		defer digestCron.Stop()
		schedulerCount += len(digestCron.Entries())
		fmt.Println("飞书摘要调度已启用: 每日 08:00 / 18:00")
	}
	r.GET("/health", func(c *gin.Context) {
		sourceCount := 0
		configs, err := sourceConfigRepo.List()
		if err == nil {
			for _, config := range configs {
				if config.Enabled {
					sourceCount++
				}
			}
		}
		c.JSON(200, gin.H{
			"status":     "ok",
			"service":    "trend-graph",
			"version":    "0.7.0",
			"sources":    sourceCount,
			"ws":         true,
			"schedulers": schedulerCount,
			"notifiers":  len(notifiers),
		})
	})
	r.POST("/api/auth/login", authService.Login)
	r.POST("/api/auth/logout", authService.Require(), authService.Logout)
	privateAPI := r.Group("/api")
	privateAPI.Use(authService.Require())
	handler.Register(privateAPI)
	api.NewSourceConfigHandler(sourceConfigRepo).Register(privateAPI)
	api.NewRadarHandler(signalRepo).Register(privateAPI)
	contentHandler := api.NewContentPackageHandler(signalRepo, nil)
	if an != nil {
		contentHandler = api.NewContentPackageHandler(signalRepo, an)
	}
	contentHandler.Register(privateAPI)

	// WS 路由（不走 /api 前缀）
	r.GET("/ws", authService.Require(), func(c *gin.Context) {
		wsHub.HandleWS(c.Writer, c.Request)
	})

	// 12. 启动信息
	addr := ":" + cfg.Port
	fmt.Printf("服务启动于 http://localhost%s\n", addr)
	fmt.Printf("已注册 %d 个通知渠道\n", len(notifiers))
	fmt.Println("接口:")
	fmt.Println("  读热点:    GET  http://localhost:" + cfg.Port + "/api/hots")
	fmt.Println("  关键词:    GET  http://localhost:" + cfg.Port + "/api/keywords")
	fmt.Println("  WebSocket: ws://localhost:" + cfg.Port + "/ws")
	fmt.Println("按 Ctrl+C 退出")

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}
	if collectionRunner != nil {
		go func() {
			log.Println("首次采集已启动")
			if err := collectionRunner.Run(context.Background()); err != nil {
				log.Printf("首次采集部分或全部失败: %v", err)
			}
			runAnalysisAndAlerts()
			log.Println("首次采集与分析完成")
		}()
	} else if cfg.BackgroundJobsEnabled && analysisRunner != nil {
		go runAnalysisAndAlerts()
	}
	if err := r.RunListener(listener); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
