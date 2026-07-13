// types.ts 定义前端用的类型，和后端返回的 JSON 字段对齐。
//
// TypeScript 的核心价值就在这里：编译期检查类型错误，
// 比如你 typo 把 hotItem.title 写成 hotItem.tile 会立即报错。
//
// 字段命名规则：和后端 JSON response 里的字段保持一致（驼峰命名）。
// 后端 struct 的 json tag 决定了字段名。

// 一条热点内容。对应后端 store.HotItem 的 JSON 序列化结果。
export interface HotItem {
  id: number
  title: string
  url: string
  summary: string
  source: string
  hot: number
  author: string
  publishedAt: number  // unix 秒
  fetchedAt: number     // unix 秒
  keywordId?: number
  // 可选字段：AI 分析后才有
  relevance?: number     // 0~1
  isAuthentic?: boolean
  entities?: string      // JSON 字符串，需要再解析
  createdAt: string      // ISO 时间
  updatedAt: string
}

// 后端统一响应格式 { data, meta }
export interface ApiResponse<T> {
  data: T
  meta?: Record<string, unknown>
}

// 后端统一错误格式 { error, detail? }
export interface ApiError {
  error: string
  detail?: string
}

// 触发抓取返回的 meta
export interface CrawlMeta {
  source: string
  keyword: string
  fetched: number
  inserted: number
  analyzed?: number
  analyze?: boolean
  fetchedAt: number
}

// 列表查询返回的 meta（含分页信息）
export interface ListMeta {
  source: string
  keywordId: number
  since: string
  limit: number
  offset: number
  total: number
  count: number
}

// 查询参数类型
export interface ListParams {
  source?: string
  keywordId?: number
  since?: string       // "1h" "24h" "7d"
  limit?: number
  offset?: number
}

// AI 查询扩展结果
export type ExpandResult = string[]