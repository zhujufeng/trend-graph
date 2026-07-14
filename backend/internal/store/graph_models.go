// models.go 定义图谱相关的数据库模型：Entity 和 EntityRelation
//
// 三重图模型设计：
//   - HotItem（已在 store 包定义）：外围节点，代表"热点"
//   - Keyword（已在 store 包定义）：核心节点，代表"监控关键词"
//   - Entity（本文件新增）：中间节点，代表 AI 从热点中抽取的实体（人/公司/项目等）
//
// 关系（边）有两种：
//   - 热点包含实体：通过 hot_item.entities JSON 字段已存了一份
//                  阶段 8 进一步用 EntityRelation 表存查询友好版本
//   - 关键词涉及实体：因为某关键词抓到的热点中含某实体，因此关键词—实体产生边
//   - 实体共现：同一热点里多个实体组成"共现边"，权重 = 共现次数
//
// 这样查询时就能得到一张可交互的网络：关键词 → 实体 → 热点
package store

import (
	"time"

	"gorm.io/gorm"
)

// Entity 实体表
//
// 实体是从多条热点中由 AI 抽取出来的、去重后的"概念"：
//   - "OpenAI"、"GPT-4"、"Sam Altman"、"Claude"、"Anthropic"...
// 阶段 8 重点：把热点 entities JSON 里的字符串实体落到这张表，
// 后续可以查询"实体→热点→关键词"反向链路。
type Entity struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"type:varchar(128);not null;uniqueIndex" json:"name"`
	// 实体类型：person（人）/ org（组织）/ project（项目）/ tech（技术）/ concept（概念）/ other
	Kind      string         `gorm:"type:varchar(32);index;default:'other'" json:"kind"`
	// 出现次数（每被抓到一次 +1）——做节点大小
	Count     int            `gorm:"default:1" json:"count"`
	FirstSeen time.Time      `gorm:"index" json:"firstSeen"`
	LastSeen  time.Time      `gorm:"index" json:"lastSeen"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Entity) TableName() string { return "entities" }

// EntityRelation 实体关系表：节点之间的"边"
//
// 一行就是一条边，类型由 Relation 字段区分。
// 不限制方向（图论无向图），按 from_id < to_id 写入去重即可。
//
// uniqueIndex:idx_rel_pair 是 PG 唯一约束，用于 ON CONFLICT 去重
type EntityRelation struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	// From / To 是两个端点的 ID
	// 注意三种端点类型用 (TypeFrom, IDFrom) 复合标识，避免主键冲突：
	//   keyword_id, entity_id, hot_id 各自从自己的表来
	TypeFrom  string         `gorm:"type:varchar(16);not null;uniqueIndex:idx_rel_pair,priority:1" json:"typeFrom"`
	IDFrom    int64          `gorm:"not null;uniqueIndex:idx_rel_pair,priority:2" json:"idFrom"`
	Relation  string         `gorm:"type:varchar(16);not null;default:'cooccur';uniqueIndex:idx_rel_pair,priority:3" json:"relation"`
	TypeTo    string         `gorm:"type:varchar(16);not null;uniqueIndex:idx_rel_pair,priority:4" json:"typeTo"`
	IDTo      int64          `gorm:"not null;uniqueIndex:idx_rel_pair,priority:5" json:"idTo"`

	// Weight 边权重：出现次数越多越大
	Weight    int            `gorm:"default:1" json:"weight"`

	// HotID（可选）记录"共现于哪条热点"
	HotID     *int64         `gorm:"index" json:"hotId,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (EntityRelation) TableName() string { return "entity_relations" }