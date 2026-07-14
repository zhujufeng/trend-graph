// GraphPage.tsx 关联图谱可视化页面
//
// 用 React Flow 把后端返回的 GraphData 渲染成可交互网络图：
//   - 节点类型用不同颜色：keyword(主色)/entity(紫)/hot(青)
//   - 节点大小随 count 变化
//   - 边粗细随 weight 变化
//   - 点击节点高亮相邻节点 + 边
//   - 双击节点下钻到该节点相关热点列表（暂用 toast 提示）
//
// React Flow 基础概念：
//   - <ReactFlow> 是容器组件
//   - nodes/edges 是状态数组，要 setNodes/setEdges 改变渲染
//   - nodeTypes 自定义节点渲染器
//   - onNodeClick 等事件回调
//   - <Background> 背景网格
//   - <Controls> 缩放按钮
//   - <MiniMap> 缩略图

import { useEffect, useState, useCallback, useMemo } from 'react'
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  type Node,
  type Edge,
  type NodeProps,
  Handle,
  Position,
  useNodesState,
  useEdgesState,
} from 'reactflow'
import 'reactflow/dist/style.css'
import { ArrowLeft, Loader2, Network as NetworkIcon } from 'lucide-react'

import { getGraph } from '../api/graph'
import type { GraphData } from '../api/graph'
import { listKeywords, type Keyword } from '../api/keywords'

// ===== 自定义节点组件 =====

// 自定义节点用 React Flow 的 NodeProps 类型
// data 字段就是后端 GraphNode + 我们加的 selected 状态
interface CustomNodeData {
  label: string
  type: 'keyword' | 'entity' | 'hot'
  count: number
  kind?: string
  source?: string
  highlighted?: boolean
  dimmed?: boolean
}

// 用 React.memo 包一层，避免不必要重渲染
const CustomNode = ({ data }: NodeProps<CustomNodeData>) => {
  // 每种节点类型不同颜色
  const colorByType: Record<string, string> = {
    keyword: '#06b6d4', // cyan
    entity: '#8b5cf6',  // purple
    hot: '#f59e0b',     // amber
  }
  const color = colorByType[data.type] || '#6b7280'

  return (
    <div
      style={{
        background: data.dimmed ? 'rgba(255,255,255,0.02)' : `${color}22`,
        border: `2px solid ${data.highlighted ? color : `${color}55`}`,
        borderRadius: '8px',
        padding: '4px 10px',
        fontSize: '12px',
        color: data.dimmed ? '#4b5563' : '#e5e7eb',
        boxShadow: data.highlighted ? `0 0 12px ${color}` : 'none',
        cursor: 'pointer',
        transition: 'all 0.2s',
        maxWidth: '160px',
        textAlign: 'center',
      }}
    >
      {/* React Flow 要求 Handle 才能连边 */}
      <Handle type="target" position={Position.Top} style={{ background: 'transparent' }} />
      <div style={{ fontWeight: data.type === 'keyword' ? 600 : 400 }}>
        {data.label.length > 24 ? data.label.slice(0, 24) + '…' : data.label}
      </div>
      {data.count > 0 && (
        <div style={{ fontSize: '10px', color: '#6b7280' }}>×{data.count}</div>
      )}
      <Handle type="source" position={Position.Bottom} style={{ background: 'transparent' }} />
    </div>
  )
}

// nodeTypes 必须在组件外定义，避免每次渲染重建
const nodeTypes = { custom: CustomNode }

// ===== 主组件 =====

interface GraphPageProps {
  // 不用 router 也可以，这里简单从 query 拿 keyword
  initialKeyword?: string
  onBack?: () => void
}

export function GraphPage({ initialKeyword, onBack }: GraphPageProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [keywords, setKeywords] = useState<Keyword[]>([])
  const [selectedKeyword, setSelectedKeyword] = useState<string>(initialKeyword || '')
  const [graphData, setGraphData] = useState<GraphData | null>(null)

  // React Flow hooks: 自动管理 nodes/edges 状态
  // setNodes 用回调形式才能拿到最新值
  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  // 选中的节点（用于高亮相邻）
  // const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  void selectedNodeId // 保留状态供未来下钻用

  // ===== 加载关键词列表（用于顶部下拉） =====
  useEffect(() => {
    listKeywords()
      .then(setKeywords)
      .catch((e) => console.error('listKeywords failed', e))
  }, [])

  // ===== 拉图谱数据 =====
  const fetchGraph = useCallback(async (keyword: string) => {
    if (!keyword) return
    setLoading(true)
    setError('')
    try {
      const data = await getGraph({ keyword })
      setGraphData(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (selectedKeyword) {
      fetchGraph(selectedKeyword)
    }
  }, [selectedKeyword, fetchGraph])

  // ===== 把后端 GraphData 转成 React Flow nodes/edges =====
  // 自动布局：圆形排列关键词为中心，其他节点按圈层散开
  useEffect(() => {
    if (!graphData) return

    // 找中心节点（keyword 类型）
    const centerNode = graphData.nodes.find((n) => n.type === 'keyword')
    const centerX = 0
    const centerY = 0

    // 给每个节点分配位置
    const positionedNodes: Node<CustomNodeData>[] = graphData.nodes.map((n, i) => {
      if (n.id === centerNode?.id) {
        return {
          id: n.id,
          type: 'custom',
          position: { x: centerX, y: centerY },
          data: {
            label: n.label,
            type: n.type,
            count: n.count,
            kind: n.kind,
            source: n.source,
          },
        }
      }
      // 其他节点按圈层排列
      const angle = (i / Math.max(graphData.nodes.length - 1, 1)) * 2 * Math.PI
      const radius = 200 + Math.random() * 100
      return {
        id: n.id,
        type: 'custom',
        position: {
          x: centerX + Math.cos(angle) * radius,
          y: centerY + Math.sin(angle) * radius,
        },
        data: {
          label: n.label,
          type: n.type,
          count: n.count,
          kind: n.kind,
          source: n.source,
        },
      }
    })

    // 边
    const rfEdges: Edge[] = graphData.edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
      animated: e.relation === 'tracks',
      style: {
        stroke: e.relation === 'tracks' ? '#06b6d4' : e.relation === 'contains' ? '#f59e0b' : '#8b5cf6',
        strokeWidth: Math.min(4, Math.max(1, e.weight || 1)),
      },
      label: e.relation,
      labelStyle: { fontSize: 10, fill: '#9ca3af' },
      labelBgStyle: { fill: 'rgba(255,255,255,0.05)' },
    }))

    setNodes(positionedNodes)
    setEdges(rfEdges)
  }, [graphData, setNodes, setEdges])

  // ===== 节点点击：高亮相邻节点 =====
  const onNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      setSelectedNodeId(node.id)
      // 计算与该节点相邻的节点 ID 集合
      const connectedIds = new Set<string>([node.id])
      edges.forEach((e: Edge) => {
        if (e.source === node.id) connectedIds.add(e.target)
        if (e.target === node.id) connectedIds.add(e.source)
      })
      // 更新节点 data
      setNodes((nds: Node<CustomNodeData>[]) =>
        nds.map((n: Node<CustomNodeData>) => ({
          ...n,
          data: {
            ...n.data,
            highlighted: connectedIds.has(n.id),
            dimmed: !connectedIds.has(n.id),
          },
        })),
      )
      setEdges((eds: Edge[]) =>
        eds.map((e: Edge) => ({
          ...e,
          animated: e.source === node.id || e.target === node.id,
        })),
      )
    },
    [edges, setNodes, setEdges],
  )

  // ===== 取消选中 =====
  const onPaneClick = useCallback(() => {
    setSelectedNodeId(null)
    setNodes((nds: Node<CustomNodeData>[]) =>
      nds.map((n: Node<CustomNodeData>) => ({ ...n, data: { ...n.data, highlighted: false, dimmed: false } })),
    )
  }, [setNodes])

  // 节点数量 / 边数量统计
  const stats = useMemo(() => {
    return {
      nodes: graphData?.nodes.length ?? 0,
      edges: graphData?.edges.length ?? 0,
      keywords: graphData?.nodes.filter((n) => n.type === 'keyword').length ?? 0,
      entities: graphData?.nodes.filter((n) => n.type === 'entity').length ?? 0,
      hots: graphData?.nodes.filter((n) => n.type === 'hot').length ?? 0,
    }
  }, [graphData])

  return (
    <div className="h-full flex flex-col">
      {/* ====== 顶栏 ====== */}
      <header className="border-b border-border bg-surface/50 backdrop-blur px-4 py-3 flex items-center gap-3">
        <button
          onClick={onBack}
          className="flex items-center gap-1 text-sm text-text-secondary hover:text-accent transition"
        >
          <ArrowLeft className="w-4 h-4" />
          返回列表
        </button>
        <div className="flex items-center gap-2">
          <NetworkIcon className="w-5 h-5 text-accent" />
          <h1 className="text-base font-medium">
            <span className="text-gradient">关联图谱</span>
          </h1>
        </div>

        {/* 关键词选择 */}
        <div className="ml-auto flex items-center gap-2">
          <select
            value={selectedKeyword}
            onChange={(e) => setSelectedKeyword(e.target.value)}
            className="px-3 py-1.5 bg-base-bg border border-border rounded-md text-sm focus:border-accent focus:outline-none"
          >
            <option value="">选择关键词…</option>
            {keywords.map((k) => (
              <option key={k.id} value={k.word}>
                {k.word} ({k.intervalMin}m, {k.active ? '✓' : '⏸'})
              </option>
            ))}
          </select>
        </div>
      </header>

      {/* ====== 错误提示 ====== */}
      {error && (
        <div className="px-4 py-2 bg-red-500/10 border-b border-red-500/30 text-sm text-red-400">
          {error}
        </div>
      )}

      {/* ====== 统计信息 ====== */}
      {graphData && !loading && (
        <div className="px-4 py-2 border-b border-border text-xs text-text-muted flex items-center gap-4">
          <span>节点: <span className="text-text-primary">{stats.nodes}</span></span>
          <span>边: <span className="text-text-primary">{stats.edges}</span></span>
          <span>关键词: <span className="text-cyan-400">{stats.keywords}</span></span>
          <span>实体: <span className="text-purple-400">{stats.entities}</span></span>
          <span>热点: <span className="text-amber-400">{stats.hots}</span></span>
          <span className="ml-auto">点击节点高亮相邻关系 · 拖动节点重新布局</span>
        </div>
      )}

      {/* ====== React Flow 画布 ====== */}
      <div className="flex-1 relative">
        {loading ? (
          <div className="absolute inset-0 flex items-center justify-center">
            <Loader2 className="w-6 h-6 animate-spin text-text-muted" />
          </div>
        ) : graphData && graphData.nodes.length > 0 ? (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.2 }}
            minZoom={0.2}
            maxZoom={2}
          >
            <Background color="#1f2937" gap={20} />
            <Controls className="!bg-surface !border-border" />
            <MiniMap
              className="!bg-surface !border-border"
              nodeColor={(n: { data?: CustomNodeData }) => {
                const t = n.data?.type
                if (t === 'keyword') return '#06b6d4'
                if (t === 'entity') return '#8b5cf6'
                if (t === 'hot') return '#f59e0b'
                return '#6b7280'
              }}
            />
          </ReactFlow>
        ) : (
          <div className="absolute inset-0 flex items-center justify-center text-text-muted text-sm">
            {selectedKeyword
              ? '该关键词还没有图谱数据。请先在后端跑一次抓取+AI 分析生成实体。'
              : '请从顶部下拉选择一个监控关键词'}
          </div>
        )}
      </div>
    </div>
  )
}