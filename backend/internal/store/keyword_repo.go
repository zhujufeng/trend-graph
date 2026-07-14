// keyword_repo.go 封装 keywords 表的 CRUD。
//
// 这是阶段 7 新增：让用户能管理监控关键词，
// 后续定时任务会按 keywords 表里 active=true 的条目自动跑抓取。
package store

import (
	"time"

	"gorm.io/gorm"
)

// KeywordRepo 监控关键词数据访问对象
type KeywordRepo struct {
	db *gorm.DB
}

// NewKeywordRepo 构造函数
func NewKeywordRepo(db *gorm.DB) *KeywordRepo {
	return &KeywordRepo{db: db}
}

// Create 增加一个监控关键词
func (r *KeywordRepo) Create(word, note string, intervalMin int) (*Keyword, error) {
	if intervalMin <= 0 {
		intervalMin = 30
	}
	k := &Keyword{
		Word:        word,
		Note:        note,
		Active:      true,
		IntervalMin: intervalMin,
	}
	if err := r.db.Create(k).Error; err != nil {
		return nil, err
	}
	return k, nil
}

// List 列出所有关键词（支持只看激活的）
func (r *KeywordRepo) List(activeOnly bool) ([]Keyword, error) {
	var ks []Keyword
	tx := r.db.Order("created_at DESC")
	if activeOnly {
		tx = tx.Where("active = ?", true)
	}
	err := tx.Find(&ks).Error
	return ks, err
}

// Get 按主键查
func (r *KeywordRepo) Get(id int64) (*Keyword, error) {
	var k Keyword
	if err := r.db.First(&k, id).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

// UpdateActive 切换激活/暂停状态
func (r *KeywordRepo) UpdateActive(id int64, active bool) error {
	return r.db.Model(&Keyword{}).Where("id = ?", id).Update("active", active).Error
}

// UpdateInterval 调间隔
func (r *KeywordRepo) UpdateInterval(id int64, intervalMin int) error {
	return r.db.Model(&Keyword{}).Where("id = ?", id).Update("interval_min", intervalMin).Error
}

// UpdateLastFetched 记录上次抓取时间（调度器调用）
func (r *KeywordRepo) UpdateLastFetched(id int64, t time.Time) error {
	return r.db.Model(&Keyword{}).Where("id = ?", id).Update("last_fetched_at", t).Error
}

// Delete 删除关键词（软删）
func (r *KeywordRepo) Delete(id int64) error {
	return r.db.Delete(&Keyword{}, id).Error
}