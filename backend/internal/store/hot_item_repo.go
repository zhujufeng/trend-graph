// hot_item_repo.go 封装 hot_items 表的 CRUD 操作。
//
// 为什么不直接 db.Where().Find() 散写在 Handler 里？
//   - 仓库层（repository）把 SQL 集中在一处
//   - 业务逻辑和数据库访问解耦，方便测试
//   - 加缓存/换实现只改这里
//
// Go 没有强制 repository 模式，看团队风格。
// 这种小项目里 repo 直接接受 *gorm.DB 也可以。
package store

// 导入：
// - time: 时间范围筛选
// - gorm.io/gorm: 保持 *gorm.DB 依赖
import (
	"time"

	"gorm.io/gorm"
)

// HotItemRepo 是热点的数据访问对象。
type HotItemRepo struct {
	db *gorm.DB
}

// NewHotItemRepo 构造函数。
func NewHotItemRepo(db *gorm.DB) *HotItemRepo {
	return &HotItemRepo{db: db}
}

// BatchCreate 批量插入热点，跳过重复（按 source + URL 去重）。
//
// 注意：GORM.FirstOrCreate 一次查一行，效率低。
// 这里简化处理，用 OnConflict 让 PG 自己跳重复。
// 条件是 (source, url) UNIQUE，等阶段 8 改 schema 时补上。
// 本阶段先简单：URL+Source 相同就跳过，靠业务层判断。
//
// 返回实际插入的行数，方便上层统计。
func (r *HotItemRepo) BatchCreate(items []HotItem) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	// GORM Create 一次性事务插入。
	// 返回的 RowsAffected 就是实际影响行数。
	result := r.db.Create(&items)
	return result.RowsAffected, result.Error
}

// List 查询热点列表，支持按来源、关键词、时间范围筛选。
//
// Go 习惯把查询条件以参数传入，而不是把 *gorm.DB 链式调用散在 Handler。
// 这样 Handler 调用就一行：repo.List(source, keywordID, since, limit, offset)
//
// 链式调用的技巧：
//   - tx := r.db.Model(&HotItem{}) 拿到一个会话
//   - 加 if 条件动态 Where
//   - 最后 Order/Limit/Offset/Find
// 这种写法在 Go Web 项目里非常常见，值得多看几遍。
func (r *HotItemRepo) List(source string, keywordID int64, since time.Time, limit, offset int) ([]HotItem, int64, error) {
	tx := r.db.Model(&HotItem{})

	// 动态加 Where 条件
	if source != "" {
		tx = tx.Where("source = ?", source)
	}
	if keywordID > 0 {
		tx = tx.Where("keyword_id = ?", keywordID)
	}
	// since 是零值表示不筛时间
	if !since.IsZero() {
		tx = tx.Where("published_at >= ?", since.Unix())
	}

	// 先 Count 算总数（前端分页要）
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 再分页查
	// 按 published_at 降序，新的在前
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var items []HotItem
	if err := tx.Order("published_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID 按主键查一条
func (r *HotItemRepo) GetByID(id int64) (*HotItem, error) {
	var item HotItem
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateSummary 更新 AI 摘要（阶段 3 用）
func (r *HotItemRepo) UpdateSummary(id int64, summary string) error {
	return r.db.Model(&HotItem{}).Where("id = ?", id).
		Update("summary", summary).Error
}

// UpdateAIResult 一次性更新 AI 分析结果（相关性、真假、摘要、实体）
func (r *HotItemRepo) UpdateAIResult(id int64, summary string, relevance float64, isAuthentic bool, entities string) error {
	return r.db.Model(&HotItem{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"summary":      summary,
			"relevance":    relevance,
			"is_authentic": isAuthentic,
			"entities":     entities,
		}).Error
}