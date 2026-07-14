// graph.ts 关联图谱 API 调用。
//
// 后端 GET /api/graph?keywordId=1 返回 GraphData（节点+边）

import client from './client'

// 后端 GraphNode 结构
export interface GraphNode {
  id: string         // 后端拼好的全局 ID 如 "keyword:AI" / "entity:5" / "hot:12"
  type: 'keyword' | 'entity' | 'hot'
  label: string
  count: number
  kind?: string       // entity 类型: person/org/project/tech/concept/other
  source?: string     // hot 的来源
}

// 后端 GraphEdge 结构
export interface GraphEdge {
  id: string
  source: string
  target: string
  relation: 'tracks' | 'contains' | 'cooccur'
  weight: number
}

// 后端 GraphData
export interface GraphData {
  nodes: GraphNode[]
  edges: GraphEdge[]
}

// 拉图谱
//
// 因为响应拦截器已经剥 .data，所以 client.get 的返回直接是后端 ApiResponse
export async function getGraph(params: { keywordId?: number; keyword?: string }): Promise<GraphData> {
  type ApiResponse = { data: GraphData; meta: Record<string, unknown> }
  const res = await client.get<unknown, ApiResponse>('/graph', { params })
  return res.data
}