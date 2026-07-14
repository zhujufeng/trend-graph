# 阶段 8：🎯 关联图谱差异化亮点

> 对应 commit：`feat: stage 8 - 关联图谱差异化亮点`

## 🎯 目标

- 数据层：实体表 + 关系表
- AI 抽取带类型的实体（人/组织/项目/技术）
- 三重网络：关键词 ↔ 实体 ↔ 热点
- 前端用 React Flow 可交互可视化

## 📚 学到的概念

### 1. 图数据建模

无向图三要素：
- **节点（Node）**：实体本身
- **边（Edge）**：节点间的关系
- **权重（Weight）**：边的强度

本项目三重图：

```
keyword ──tracks──→ entity ←──contains── hot
                       │
                       └──cooccur──→ entity
```

- `keyword → entity`：因为抓到含该实体的热点（tracks 边）
- `hot → entity`：热点包含实体（contains 边）
- `entity ↔ entity`：实体在同一热点共现（cooccur 边，权重=共现次数）

### 2. PG ON CONFLICT UPSERT

```go
r.db.Clauses(clause.OnConflict{
    Columns: []clause.Column{{Name: "name"}},  // 触发唯一索引
    DoUpdates: clause.Assignments(map[string]interface{}{
        "count":     gorm.Expr("entities.count + 1"),  // 命中冲突时 count+1
        "last_seen": time.Now(),
    }),
}).Create(&e)
```

等价 SQL：
```sql
INSERT INTO entities (name, kind, count, ...)
VALUES (...)
ON CONFLICT (name) DO UPDATE
SET count = entities.count + 1, last_seen = NOW();
```

UPSERT = UPDATE or INSERT，避免"先 SELECT 再判断 INSERT/UPDATE"的两次查询。

### 3. 复合唯一索引

```go
type EntityRelation struct {
    TypeFrom string `gorm:"uniqueIndex:idx_rel_pair,priority:1"`
    IDFrom   int64  `gorm:"uniqueIndex:idx_rel_pair,priority:2"`
    Relation string `gorm:"uniqueIndex:idx_rel_pair,priority:3"`
    TypeTo   string `gorm:"uniqueIndex:idx_rel_pair,priority:4"`
    IDTo     int64  `gorm:"uniqueIndex:idx_rel_pair,priority:5"`
}
```

同名 `idx_rel_pair` = 同一复合唯一索引，`priority` 决定列顺序。

ON CONFLICT 的 `Columns` 列表必须和复合唯一索引的列完全一致（顺序也要对）。

### 4. 多态外键（type+id 复合标识）

```go
TypeFrom string  // "keyword" / "entity" / "hot"
IDFrom   int64   // 对应表的 ID
```

三种节点（keyword/entity/hot）的 ID 各自从自己表来，无法用单一外键约束。用 `(type, id)` 复合标识是图数据库常见设计。

### 5. GORM clause.OnConflict

```go
clause.OnConflict{
    Columns:    []clause.Column{...},     // 触发冲突的列
    DoUpdates:  clause.Assignments(map),  // 命中冲突时更新什么
}
```

`clause.Assignments` 用 map 表达"列名=值"，比 `clause.Assignments(columns)` 灵活。

### 6. AI Prompt 工程扩展

从阶段 3 的简单 entities 扩展到带类型的 typedEntities：

```
- typedEntities: 数组，每项 {"name":"...","kind":"..."}
  kind 取值 person/org/project/tech/concept/other
```

让 AI 不只抽实体还分类，这是图谱节点着色的依据。

### 7. 三重图查询算法

```go
func GetGraph(keywordID int64, word string) (*GraphData, error) {
    // 1. 起点节点：keyword
    // 2. keyword tracks 的 entity（边 + 节点）
    // 3. 这些 entity 在哪些 hot 里（反向查 contains）
    // 4. entity 之间的 cooccur 边
    // 5. 限制 100 节点 / 200 边避免图过大
}
```

多段 SQL 拼装出完整图，前端直接渲染。

### 8. React Flow 基础

```tsx
<ReactFlow
  nodes={nodes}      // 节点数组
  edges={edges}      // 边数组
  onNodesChange={onNodesChange}
  onNodeClick={onNodeClick}
  nodeTypes={nodeTypes}  // 自定义节点组件
  fitView            // 自动适配视口
>
  <Background />     // 背景网格
  <Controls />       // 缩放按钮
  <MiniMap />        // 缩略图
</ReactFlow>
```

### 9. 自定义节点组件

```tsx
const CustomNode = ({ data }: NodeProps<CustomNodeData>) => {
  return <div style={{ ... }}>{data.label}</div>
}

const nodeTypes = { custom: CustomNode }  // 必须在组件外定义
```

`nodeTypes` 对象必须定义在组件**外部**，避免每次渲染重建（重建会导致 React Flow 性能问题）。

### 10. 节点点击高亮相邻

```ts
const onNodeClick = (_event, node) => {
  const connectedIds = new Set<string>([node.id])
  edges.forEach(e => {
    if (e.source === node.id) connectedIds.add(e.target)
    if (e.target === node.id) connectedIds.add(e.source)
  })
  setNodes(nds => nds.map(n => ({
    ...n,
    data: {
      ...n.data,
      highlighted: connectedIds.has(n.id),
      dimmed: !connectedIds.has(n.id),
    },
  })))
}
```

用 Set 做邻接集合，O(1) 查找。

## 🔍 关键代码

| 概念 | 文件 |
|---|---|
| 实体表 + 关系表 | `backend/internal/store/graph_models.go` |
| 图谱 CRUD + 查询 | `backend/internal/store/graph_repo.go` |
| AI 抽取 typedEntities | `backend/internal/analyzer/analyzer.go` |
| 图谱 API | `backend/internal/api/handler.go:GetGraph` |
| 自动抓取写图谱 | `backend/internal/scheduler/scheduler.go` |
| 前端图谱页 | `frontend/src/pages/GraphPage.tsx` |
| 前端 API | `frontend/src/api/graph.ts` |

## 🧪 测试

```bash
go test -v -run TestE2E_Graph ./internal/store/
# 真打 HN → AI 抽实体 → 入图谱 → 查图
```

## 🐛 踩坑

1. **`no unique or exclusion constraint matching ON CONFLICT`**：复合唯一索引的 GORM tag 用 `priority:N` 排序，ON CONFLICT 的 Columns 列表要和索引列顺序一致
2. **AutoMigrate 不删旧索引**：改了 uniqueIndex 后要手动 `DROP TABLE` 重建，GORM 不会自动清理旧约束
3. **`assignment mismatch: 1 variable but ... returns 2 values`**：Go 函数返回多值就用多变量接
4. **reactflow 类型找不到**：装包后 tsc 缓存可能没刷新，重跑 `npm run build`
5. **`nodeTypes` 必须组件外定义**：定义在组件内会导致每次渲染重建，React Flow 警告
6. **`useNodesState`/`useEdgesState` 回调的 `n` 隐式 any**：要显式标注 `n: Node<CustomNodeData>[]`

## 📝 一句话总结

关联图谱是这个项目的**差异化亮点**：把分散的热点用 AI 抽取实体串成网络，让用户一眼看出关联关系，简历上能写出彩。