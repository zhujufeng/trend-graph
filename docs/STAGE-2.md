# 阶段 2：数据库设计（GORM + PostgreSQL）

> 对应 commit：`feat: stage 2 - GORM + PostgreSQL 持久化`

## 🎯 目标

把抓到的热点存进数据库，支持按时间/来源筛选查询。

## 📚 学到的概念

### 1. ORM 与 GORM

ORM = Object-Relational Mapping，把 Go struct 和数据库表对应起来，不用手写 SQL。

```go
type HotItem struct {
    ID    int64  `gorm:"primaryKey;autoIncrement"`
    Title string `gorm:"type:varchar(500);not null;index"`
}

db.AutoMigrate(&HotItem{})  // 自动建表
db.Create(&item)            // INSERT
db.First(&item, 1)          // SELECT WHERE id=1
db.Where("source = ?", "hn").Find(&items)  // SELECT WHERE
```

### 2. struct tag：gorm

```go
Title string `gorm:"type:varchar(500);not null;index:idx_source_published"`
```

- `type:varchar(500)` 列类型
- `not null` 非空约束
- `index` 单字段索引
- `index:idx_xxx` 复合索引（同名即同一复合索引）

### 3. GORM 模型约定三件套

```go
CreatedAt time.Time      `gorm:"autoCreateTime"`  // 自动管理创建时间
UpdatedAt time.Time      `gorm:"autoUpdateTime"`  // 自动管理更新时间
DeletedAt gorm.DeletedAt `gorm:"index"`           // 软删除（删 = 设时间戳，不真删行）
```

写 GORM 模型几乎必带这三个字段。

### 4. 连接池配置

```go
sqlDB, _ := db.DB()
sqlDB.SetMaxIdleConns(10)     // 最大空闲连接
sqlDB.SetMaxOpenConns(50)    // 最大打开连接
sqlDB.SetConnMaxLifetime(time.Hour)  // 连接最大存活
```

高并发场景必调，否则默认连接数太少成瓶颈。

### 5. 软删除

GORM 用 `gorm.DeletedAt` 字段实现软删除：调 `db.Delete()` 不会真删行，而是设 `deleted_at` 时间戳；查询自动加 `WHERE deleted_at IS NULL`。

`Unscoped()` 可以绕过软删过滤：
```go
db.Unscoped().Where("1=1").Delete(&HotItem{})  // 真删
```

### 6. 链式 Where + 分页

```go
tx := r.db.Model(&HotItem{})
if source != "" {
    tx = tx.Where("source = ?", source)
}
if !since.IsZero() {
    tx = tx.Where("published_at >= ?", since.Unix())
}
tx.Count(&total)         // 总数（分页用）
tx.Order("published_at DESC").Limit(limit).Offset(offset).Find(&items)
```

这是 Go Web 项目里非常常见的"动态条件查询"模式。

### 7. 业务实体 vs DB 模型

- `types.HotItem` 是**业务实体**：纯数据结构，不依赖 GORM，可在任何层用
- `store.HotItem` 是**DB 模型**：带 GORM tag、时间戳字段
- 用 `store.FromBiz(bizItem, &keywordID)` 转换

分开让 types 包不依赖 GORM，更纯粹。

### 8. Repository 模式

把 SQL 集中在 `*Repo` 类里：

```go
type HotItemRepo struct { db *gorm.DB }
func (r *HotItemRepo) List(...) ([]HotItem, int64, error)
```

好处：业务和 SQL 解耦，方便换实现/加缓存/单测。

## 🔍 关键代码

| 概念 | 文件 |
|---|---|
| 三张表模型 | `backend/internal/store/models.go` |
| DB 初始化 + 连接池 | `backend/internal/store/db.go` |
| HotItemRepo CRUD | `backend/internal/store/hot_item_repo.go` |
| 配置加载 | `backend/internal/config/config.go` |
| API 入库 + 查询 | `backend/internal/api/handler.go` |

## 🧪 测试

```bash
# 跑 CRUD 单元测试
go test -v -run TestHotItemCRUD ./internal/store/

# 跑端到端：HN→入库→回查
go test -v -run TestEndToEnd_HNToDB ./internal/store/
```

## 🐛 踩坑

1. **GORM v2 用 `gorm.io/gorm`，v1 是 `github.com/jinzhu/gorm`**：导入路径别搞错
2. **`BatchCreate(&items)` 后 items[i].ID 被回填**：可以继续拿到 ID 用
3. **复合索引名要一致**：`gorm:"index:idx_source_published"` 同名 = 同一复合索引

## 📝 一句话总结

ORM 让数据库操作从写 SQL 变成调方法，但底层原理（索引、连接池、软删）还是要懂。