// Package config 负责加载和管理应用配置。
//
// Go 项目的配置来源通常有几种：
//  1. 命令行参数（flag）
//  2. 环境变量（os.Getenv）
//  3. 配置文件（.env / .yaml / .json）
//  4. 远程配置中心（etcd / Apollo）
//
// 本阶段用最简单的 .env 文件 + 环境变量覆盖。
// 用 github.com/joho/godotenv 把 .env 文件里的键值对注入到 os.Environ。
// 真实生产环境会把配置加载抽成 viper/koanf 等更强大的库，
// 但阶段 2 我们专注学数据库，配置保持最简。
package config

// 导入讲解：
// - fmt: 拼接错误信息
// - os: 读取环境变量
// - github.com/joho/godotenv: 加载 .env 文件
import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 是应用的全局配置结构。
//
// 把所有配置统一装到一个 struct 里，而不是到处 os.Getenv，
// 好处是：
//  1. main.go 只需要 cfg := config.Load() 一行
//  2. 函数签名可以传 cfg 让依赖一目了然
//  3. 测试时可以构造不同 cfg 测不同行为
type Config struct {
	// HTTP 服务监听端口
	Port string

	// PostgreSQL 连接串。GORM 用这个串 Dial 数据库。
	// 标准 PG URL 格式: host=... port=... user=... password=... dbname=... sslmode=...
	DatabaseURL string

	// DeepSeek AI 配置（阶段 3 才用，先占位）
	DeepSeekAPIKey  string
	DeepSeekModel   string
	DeepSeekBaseURL string

	// 单管理员私有访问配置。密码只保存在服务端环境变量。
	AdminPassword         string
	AdminSessionHours     int
	SessionCookieSecure   bool
	InternalIngestSecret  string
	GitHubToken           string
	RedditClientID        string
	RedditClientSecret    string
	CollectorDir          string
	BackgroundJobsEnabled bool

	// 阶段 7 通知渠道配置
	// 任何一项留空就跳过对应渠道
	SMTPHost string
	SMTPPort string // string 方便处理
	SMTPUser string
	SMTPPass string
	SMTPFrom string
	SMTPTo   string // 逗号分隔多收件人

	FeishuWebhook      string
	DigestEnabled      bool
	MajorAlertsEnabled bool
	DingTalkWebhook    string
	DingTalkSecret     string
}

// Load 加载配置。
//
// 调用顺序：
//  1. 尝试从 .env 文件加载（如果存在）
//  2. 从环境变量读取并填充 Config struct
//  3. 必填项缺失就 panic 让你立刻知道
//
// 注意：Go 里 panic 会让进程退出，只能在启动时用。
// 运行时错误要用 error，不要 panic。
func Load() *Config {
	// 尝试加载 .env，找不到也不报错（生产环境可能不用 .env）
	// godotenv.Load() 默认加载当前工作目录的 .env
	_ = godotenv.Load()

	cfg := &Config{
		Port:                  getEnv("PORT", "8080"),
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		DeepSeekAPIKey:        getEnv("DEEPSEEK_API_KEY", ""),
		DeepSeekModel:         getEnv("DEEPSEEK_MODEL", "deepseek-v4-pro"),
		DeepSeekBaseURL:       getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		AdminPassword:         getEnv("ADMIN_PASSWORD", ""),
		AdminSessionHours:     getEnvInt("ADMIN_SESSION_HOURS", 168),
		SessionCookieSecure:   getEnvBool("SESSION_COOKIE_SECURE", true),
		InternalIngestSecret:  getEnv("INTERNAL_INGEST_SECRET", ""),
		GitHubToken:           getEnv("GITHUB_TOKEN", ""),
		RedditClientID:        getEnv("REDDIT_CLIENT_ID", ""),
		RedditClientSecret:    getEnv("REDDIT_CLIENT_SECRET", ""),
		CollectorDir:          getEnv("COLLECTOR_DIR", "../services/collector"),
		BackgroundJobsEnabled: getEnvBool("BACKGROUND_JOBS_ENABLED", true),

		// 阶段 7 通知配置
		SMTPHost:           getEnv("SMTP_HOST", ""),
		SMTPPort:           getEnv("SMTP_PORT", "465"),
		SMTPUser:           getEnv("SMTP_USER", ""),
		SMTPPass:           getEnv("SMTP_PASS", ""),
		SMTPFrom:           getEnv("SMTP_FROM", ""),
		SMTPTo:             getEnv("SMTP_TO", ""),
		FeishuWebhook:      getEnv("FEISHU_WEBHOOK", ""),
		DigestEnabled:      getEnvBool("DIGEST_ENABLED", true),
		MajorAlertsEnabled: getEnvBool("MAJOR_ALERTS_ENABLED", true),
		DingTalkWebhook:    getEnv("DINGTALK_WEBHOOK", ""),
		DingTalkSecret:     getEnv("DINGTALK_SECRET", ""),
	}

	// 阶段 2 数据库是必需的，没配就报错退出
	if cfg.DatabaseURL == "" {
		panic("DATABASE_URL 环境变量未设置，请参考 backend/.env.example")
	}
	if cfg.AdminPassword == "" {
		panic("ADMIN_PASSWORD 环境变量未设置；私有仪表盘不能以匿名模式启动")
	}

	return cfg
}

// getEnv 读环境变量，没有就用默认值。
// 这是 Go 配置代码的标准 helper 写法。
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil || v <= 0 {
		return defaultVal
	}
	return v
}

func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return parsed
}
