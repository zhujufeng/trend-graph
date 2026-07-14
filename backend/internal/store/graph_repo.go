// graph_repo.go: 实体节点 + 关系的 CRUD
package store

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GraphRepo 封装实体和关系的操作
type GraphRepo struct {
	db *gorm.DB
}

// NewGraphRepo 构造
func NewGraphRepo(db *gorm.DB) *GraphRepo {
	return &GraphRepo{db: db}
}

// EnsureEntity 入库一个实体（不存在创建，存在则 count+1 和 last_seen 更新）
//
// 用 OnConflict 等价 INSERT ... ON CONFLICT DO UPDATE（PG 特有）
//
// 返回实体的主键
func (r *GraphRepo) EnsureEntity(name, kind string) (int64, error) {
	e := Entity{
		Name: name,
		Kind: firstNonEmpty(kind, "other"),
		FirstSeen: time.Now(),
		LastSeen: time.Now(),
		Count: 1,
	}
	// OnConflict 命中 UniqueIndex(name) 时执行 UPDATE
	res := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}}, // 触发唯一索引 name
		DoUpdates: clause.Assignments(map[string]interface{}{
			"count":     gorm.Expr("entities.count + 1"), // 计数+1
			"last_seen": time.Now(),
			"updated_at": time.Now(),
		}),
	}).Create(&e)
	if res.Error != nil {
		return 0, res.Error
	}
	// OnConflict 后 e.ID 可能是 0（如果命中冲突就没回填），
	// 重新查询拿稳定 ID
	if e.ID == 0 {
		var found Entity
		if err := r.db.Where("name = ?", name).First(&found).Error; err != nil {
			return 0, err
		}
		return found.ID, nil
	}
	return e.ID, nil
}

// EnsureRelation 入库一条关系（已存在则 weight + 1）
//
// 三元组 (typeFrom,idFrom,relation,typeTo,idTo) 用唯一索引去重
func (r *GraphRepo) EnsureRelation(typeFrom string, idFrom int64, relation, typeTo string, idTo int64, hotID *int64) error {
	rel := EntityRelation{
		TypeFrom: typeFrom,
		IDFrom:   idFrom,
		Relation: relation,
		TypeTo:   typeTo,
		IDTo:     idTo,
		Weight:   1,
		HotID:    hotID,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "type_from"}, {Name: "id_from"},
			{Name: "relation"},
			{Name: "type_to"}, {Name: "id_to"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"weight":     gorm.Expr("entity_relations.weight + 1"),
			"updated_at": time.Now(),
		}),
	}).Create(&rel).Error
}

// IngestHotEntities 把一条热点的 entities 全部入库 + 建关系
//
// 步骤：
//   1) 每个 entity 字符串 → EnsureEntity 拿 ID
//   2) 建 hot → entity 的 contains 关系
//   3) 两两实体建 cooccur 共现关系
func (r *GraphRepo) IngestHotEntities(hotID int64, entities []string) error {
	if len(entities) == 0 {
		return nil
	}
	// 1. 实体入库 + 建 hot→entity 边
	entityIDs := make([]int64, 0, len(entities))
	for _, name := range entities {
		id, err := r.EnsureEntity(name, "other") // kind 由 AI 提取时已经存在 hotItem.entities JSON 里，简化统一 other
		if err != nil {
			continue
		}
		entityIDs = append(entityIDs, id)
		// hot → entity
		_ = r.EnsureRelation("hot", hotID, "contains", "entity", id, &hotID)
	}
	// 2. 两两共现
	for i := 0; i < len(entityIDs); i++ {
		for j := i + 1; j < len(entityIDs); j++ {
			// 按 ID 小→大写去重（实际有唯一索引）
			a, b := entityIDs[i], entityIDs[j]
			if a > b {
				a, b = b, a
			}
			_ = r.EnsureRelation("entity", a, "cooccur", "entity", b, &hotID)
		}
	}
	return nil
}

// TrackKeywordToEntities 给某关键词建立与它"涉及"的实体关系
// 含义：因为这次抓取_KEYWORD_K 触发抓到了 hot_H，hot_H 含 entity_E → keyword_K track entity_E
func (r *GraphRepo) TrackKeywordToEntities(keywordID int64, entityIDs []int64, hotID *int64) error {
	for _, eid := range entityIDs {
		_ = r.EnsureRelation("keyword", keywordID, "tracks", "entity", eid, hotID)
	}
	return nil
}

// ===== 查询：返回前端能直接渲染的网络图数据 =====

// GraphNode 节点（前端 React Flow 节点）
type GraphNode struct {
	ID      string `json:"id"`         // 全局唯一 ID：用 type+id 组合
	Type    string `json:"type"`       // "keyword"/"entity"/"hot"
	Label   string `json:"label"`      // 显示文字
	Count   int    `json:"count"`      // 节点大小用
	Kind    string `json:"kind,omitempty"` // entity 类型
	Source  string `json:"source,omitempty"` // hot 节点的来源
	// 节点位置（前端可二次排布，这里先给个随机初值）
	X       *float64 `json:"x,omitempty"`
	Y       *float64 `json:"y,omitempty"`
}

// GraphEdge 边
type GraphEdge struct {
	ID       string `json:"id"`        // "from-to-relation"
	Source   string `json:"source"`    // 起点节点 ID
	Target   string `json:"target"`    // 终点节点 ID
	Relation string `json:"relation"`  // 包含/涉及/共现
	Weight   int    `json:"weight"`     // 边粗细
}

// GraphData 是给前端的完整图
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GetGraph 按一个关键词查它的关联图
//
// 算法：
//   1) 找关键词本身（起点节点）
//   2) 关键词 tracks 的实体（track 边 + entity 节点）
//   3) 这些实体存在于哪些热点（contains 反向边 + hot 节点）
//   4) 实体之间的 cooccur 边
//
// 限制最多 100 节点 + 200 边，避免返回超大图
func (r *GraphRepo) GetGraph(keywordID int64, keywordWord string) (*GraphData, error) {
	g := &GraphData{Nodes: []GraphNode{}, Edges: []GraphEdge{}}

	// 节点 ID 生成函数（统一格式 type+":"+id）
	nodeID := func(t string, id int64) string {
		// Sprintf 一次，统一约定
		// 不用 strconv 是因为拼接方便
		return t + ":" + itoa(id)
	}

	// 1) 起点节点：关键词
	g.Nodes = append(g.Nodes, GraphNode{
		ID:    "keyword:" + keywordWord,
		Type:  "keyword",
		Label: keywordWord,
		Count: 1,
	})

	// 2) 关键词 track 的实体（含权重）
	type relRow struct {
		IDTo    int64  `gorm:"column:id_to"`
		Weight  int    `gorm:"column:weight"`
	}
	var trackRels []relRow
	err := r.db.Table("entity_relations").
		Select("id_to, weight").
		Where("type_from = 'keyword' AND id_from = ? AND relation = 'tracks'", keywordID).
		Scan(&trackRels).Error
	if err != nil {
		return nil, err
	}

	entityIDs := make(map[int64]bool, len(trackRels))
	for _, tr := range trackRels {
		entityIDs[tr.IDTo] = true
		// fake edge id
		g.Edges = append(g.Edges, GraphEdge{
			ID:       "keyword:" + keywordWord + "->" + nodeID("entity", tr.IDTo) + "-tracks",
			Source:   "keyword:" + keywordWord,
			Target:   nodeID("entity", tr.IDTo),
			Relation: "tracks",
			Weight:   tr.Weight,
		})
	}

	if len(entityIDs) == 0 {
		return g, nil
	}

	// 3) 把实体本身的信息查出来
	idList := make([]int64, 0, len(entityIDs))
	for id := range entityIDs {
		idList = append(idList, id)
	}
	var entities []Entity
	err = r.db.Where("id IN ?", idList).Find(&entities).Error
	if err != nil {
		return nil, err
	}
	entityName := make(map[int64]string, len(entities))
	for _, e := range entities {
		entityName[e.ID] = e.Name
		g.Nodes = append(g.Nodes, GraphNode{
			ID:    nodeID("entity", e.ID),
			Type:  "entity",
			Label: e.Name,
			Count: e.Count,
			Kind:  e.Kind,
		})
	}

	// 4) 实体被哪些热点 contains
	var hotRels []struct {
		IDFrom  int64 `gorm:"column:id_from"`
		IDTo    int64 `gorm:"column:id_to"`
		Weight  int   `gorm:"column:weight"`
	}
	err = r.db.Table("entity_relations").
		Select("id_from, id_to, weight").
		Where("type_from = 'hot' AND type_to = 'entity' AND relation = 'contains' AND id_to IN ?", idList).
		Scan(&hotRels).Error
	if err != nil {
		return nil, err
	}

	// 整理：每个 hot_id 关联的实体列表
	hotToEntities := make(map[int64][]int64, 32)
	hotIDsSet := make(map[int64]bool, 32)
	for _, hr := range hotRels {
		hotToEntities[hr.IDFrom] = append(hotToEntities[hr.IDFrom], hr.IDTo)
		hotIDsSet[hr.IDFrom] = true
	}

	// 5) 把热点节点加进来
	hotIDs := make([]int64, 0, len(hotIDsSet))
	for id := range hotIDsSet {
		hotIDs = append(hotIDs, id)
	}
	var hotItems []HotItem
	if len(hotIDs) > 0 {
		err = r.db.Where("id IN ?", hotIDs).Find(&hotItems).Error
		if err != nil {
			return nil, err
		}
	}
	for _, h := range hotItems {
		// 限制最多 100 节点，避免图过大
		if len(g.Nodes) >= 100 {
			break
		}
		g.Nodes = append(g.Nodes, GraphNode{
			ID:    nodeID("hot", h.ID),
			Type:  "hot",
			Label: h.Title,
			Count: h.Hot,
			Source: h.Source,
		})
		ents := hotToEntities[h.ID]
		for _, eid := range ents {
			g.Edges = append(g.Edges, GraphEdge{
				ID:       nodeID("hot", h.ID) + "->" + nodeID("entity", eid) + "-contains",
				Source:   nodeID("hot", h.ID),
				Target:   nodeID("entity", eid),
				Relation: "contains",
			})
		}
	}

	// 6) 实体-实体共现边
	var cooccurRels []struct {
		IDFrom  int64 `gorm:"column:id_from"`
		IDTo    int64 `gorm:"column:id_to"`
		Weight  int   `gorm:"column:weight"`
	}
	err = r.db.Table("entity_relations").
		Select("id_from, id_to, weight").
		Where("type_from = 'entity' AND type_to = 'entity' AND relation = 'cooccur'").
		Where("id_from IN ? AND id_to IN ?", idList, idList).
		Scan(&cooccurRels).Error
	if err != nil {
		return nil, err
	}
	for _, cr := range cooccurRels {
		if len(g.Edges) >= 200 {
			break
		}
		g.Edges = append(g.Edges, GraphEdge{
			ID:       nodeID("entity", cr.IDFrom) + "-" + nodeID("entity", cr.IDTo) + "-cooccur",
			Source:   nodeID("entity", cr.IDFrom),
			Target:   nodeID("entity", cr.IDTo),
			Relation: "cooccur",
			Weight:   cr.Weight,
		})
	}

	return g, nil
}

// firstNonEmpty 返回第一个非空字符串
func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

// itoa 简化的 int→string，避免 strconv 引入
func itoa(n int64) string {
	// 简化版：直接用 string(rune(n)) 不对，这里用 fmt.Sprint
	// 但又不愿 import fmt 增加依赖，所以手动实现
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}