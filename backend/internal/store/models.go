// Package store 是数据持久化层。
//
// Go 项目常见的分层：
//   - types/    → 业务实体（HotItem 这种）
//   - store/    → 数据库模型 + CRUD（GORM 模型定义在这里）
//   - api/      → HTTP Handler（接收请求，调 store 读写）
//
// 为什么把"模型"放在 store 包而不是 types 包？
//   - types.HotItem 是"业务实体"，没有任何 DB 概念，可以独立用
//   - store.HotItem 是"数据库行"，带 GORM tag、主键、时间戳
//   - 两者字段大致一致，但有 DB 特有字段（CreatedAt 等）
//   - 分开能让 types 包不依赖 GORM，更纯粹
package store

// 导入讲解：
// - time: 处理时间戳
// - gorm.io/gorm: GORM 核心，提供 Model 基类、字段类型
// - trend-graph/internal/types: 业务实体 HotItem（用做 crawler 输出转 store 模型的载体）
import (
	"time"

	"gorm.io/gorm"

	"trend-graph/internal/types"
)

// HotItem 是 hot_items 表对应的 GORM 模型。
//
// Go struct 的 gorm tag 用来告诉 GORM：
//   - column: 数据库列名（蛇形小写，PG 习惯）
//   - primaryKey: 主键
//   - index: 加索引（加速查询）
//   - not null: 非空约束
//
// GORM 用反射读这些 tag，自动建表、生成 SQL。
//
// 还有约定：
//   - 字段 ID → 主键自增（不用写 tag）
//   - 字段 CreatedAt/UpdatedAt → GORM 自动管理时间
//   - 字段 DeletedAt → 软删除（删=设时间戳，不真删行）
type HotItem struct {
	// 主键 ID，GORM 默认会字段当作主键并自增
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 标题
	Title string `gorm:"type:varchar(500);not null;index" json:"title"`

	// 原文链接
	URL string `gorm:"type:varchar(1000)" json:"url"`

	// 摘要（AI 生成）
	Summary string `gorm:"type:text" json:"summary"`

	// 来源平台简称：hn / weibo / bilibili ...
	// index 复合索引（和 PublishedAt 一起加，加速按来源+时间筛选查询）
	Source string `gorm:"type:varchar(32);not null;index:idx_source_published" json:"source"`

	// 热度分数
	Hot int `gorm:"default:0" json:"hot"`

	// 作者
	Author string `gorm:"type:varchar(128)" json:"author"`

	// 原始发布时间（Unix 秒）
	PublishedAt int64 `gorm:"index:idx_source_published" json:"publishedAt"`

	// 抓取时间（Unix 秒）
	FetchedAt int64 `gorm:"index" json:"fetchedAt"`

	// 关联的监控关键词 ID（可选，0 表示通用抓取）
	KeywordID *int64 `gorm:"index" json:"keywordId,omitempty"`

	// AI 相关性分析结果（阶段 3 才用）
	//   - 0~1 浮点数，1 = 高度相关，0 = 无关
	//   - 用 *float64 而不是 float64，方便 nil 表示"还没分析过"
	Relevance *float64 `gorm:"type:numeric(3,2)" json:"relevance,omitempty"`

	// AI 判断的真假（true/false），nil 表示还没分析过
	IsAuthentic *bool `gorm:"type:boolean" json:"isAuthentic,omitempty"`

	// 抽取的实体 JSON（阶段 8 关联图谱用）
	// 用 JSON 字符串存最简单，等阶段 8 再上实体表
	Entities string `gorm:"type:text" json:"entities,omitempty"`

	// GORM 标准三件套：自动管理时间戳
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // json:"-" 表示不返回给前端
}

// TableName 自定义表名。
// GORM 默认会把 HotItem 转成蛇形 hot_items，这里显式声明让代码更自解释。
func (HotItem) TableName() string { return "hot_items" }

// Keyword 是 keywords 表对应的模型，表示一个监控关键词配置。
//
// 用户在前端加个"AI"，激活后定时任务会拉取相关热点。
type Keyword struct {
	ID    int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Word  string `gorm:"type:varchar(128);not null;uniqueIndex" json:"word"`
	// 激活状态：true 代表定时抓，false 代表暂停
	Active bool `gorm:"default:true" json:"active"`
	// 抓取间隔（分钟），默认 30
	IntervalMin int `gorm:"default:30" json:"intervalMin"`
	// 上次抓取时间
	LastFetchedAt *time.Time `json:"lastFetchedAt,omitempty"`
	// 备注
	Note string `gorm:"type:varchar(500)" json:"note"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Keyword) TableName() string { return "keywords" }

// CrawlRun 是 crawl_runs 表对应模型，记录每次抓取任务执行情况。
//
// 用途：审计、监控爬虫健康状况、避免短时间内重复跑。
type CrawlRun struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	KeywordID  *int64    `gorm:"index" json:"keywordId,omitempty"`
	Source     string    `gorm:"type:varchar(32);not null;index" json:"source"`
	Keyword    string    `gorm:"type:varchar(128)" json:"keyword"`
	Status     string    `gorm:"type:varchar(16);not null" json:"status"` // success/failed/running
	ItemCount  int       `gorm:"default:0" json:"itemCount"`
	ErrorMsg   string    `gorm:"type:text" json:"errorMsg,omitempty"`
	StartedAt  time.Time `gorm:"not null" json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CrawlRun) TableName() string { return "crawl_runs" }

// FromBiz 把一个 types.HotItem（业务实体）转成 store.HotItem（DB 模型）。
//
// 这是项目里"业务实体 ↔ DB 模型"转换的辅助函数。
// 命名约定 FromXxx/ToXxx 让转换方向一目了然。
//
// 注意：类型转换不能直接赋值（Go struct 不能直接 =），
// 但字段完全相同时一个字段一个字段写也行；
// 这里函数式写法，更清晰。
func FromBiz(b types.HotItem, keywordID *int64) HotItem {
	return HotItem{
		Title:       b.Title,
		URL:         b.URL,
		Summary:     b.Summary,
		Source:      b.Source,
		Hot:         b.Hot,
		Author:      b.Author,
		PublishedAt: b.PublishedAt,
		FetchedAt:   b.FetchedAt,
		KeywordID:   keywordID,
	}
}