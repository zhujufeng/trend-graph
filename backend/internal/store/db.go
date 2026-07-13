// db.go 提供 GORM 数据库实例的初始化与全局访问。
//
// 为什么单独一个文件管 DB？
//   - main.go 里只要一句 db := store.New(cfg.DatabaseURL)
//   - 其他业务文件拿到 db 后做 CRUD，不用关心连接细节
//   - 切换数据库（开发 SQLite / 生产 PG）只改这一个文件
package store

// 导入讲解：
// - fmt/time: 错误和时间配置
// - gorm.io/gorm: ORM 核心
// - gorm.io/driver/postgres: PostgreSQL 驱动（GORM 用它生成 PG 方言 SQL）
import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New 创建并返回 *gorm.DB。
//
// 步骤：
//   1) 用配置 dsn 拨号到 PostgreSQL
//   2) 配置连接池（生产环境重要！）
//   3) 自动迁移（AutoMigrate）把模型同步到表结构
//   4) 返回可用的 db 实例
//
// 参数 dsn 是 PostgreSQL 连接串，
//   格式: host=... port=... user=... password=... dbname=... sslmode=...
func New(dsn string) (*gorm.DB, error) {
	// 1. 打开连接。gorm.Open 不会真的连，只是初始化实例，
	//    真正的连接池在第一次查询时建立。
	//    &gorm.Config{} 里可以改日志级别、命名策略等。
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// GORM 日志：开发用 Info（打印 SQL），生产用 Silent 或 Error
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 2. 取出底层 *sql.DB 调连接池
	//    高并发场景必调，否则默认连接数太少会成瓶颈
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}
	// 最大空闲连接数。一般 10 够用，生产可以根据 QPS 调大。
	sqlDB.SetMaxIdleConns(10)
	// 最大打开连接数。PostgreSQL 默认 max_connections=100，
	// 这里设 50 留余量给 psql / 后台任务
	sqlDB.SetMaxOpenConns(50)
	// 连接最大存活时间。避免长时间复用同一连接导致 PG 端超时
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 3. 自动迁移
	//    AutoMigrate 会：
	//      - 表不存在 → 建表
	//      - 表存在但少字段 → ALTER TABLE 加字段
	//      - 不会删字段、不会改类型（保护数据）
	//    开发期超方便，生产建议用 migration 工具（goose/atlas）
	if err := db.AutoMigrate(&HotItem{}, &Keyword{}, &CrawlRun{}); err != nil {
		return nil, fmt.Errorf("自动迁移失败: %w", err)
	}
	log.Println("数据库初始化 + AutoMigrate 完成")

	return db, nil
}