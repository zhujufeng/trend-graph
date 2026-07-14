// keywords.ts 封装关键词管理相关的 API 调用。
// 路径对应后端:
//   GET    /api/keywords
//   POST   /api/keywords
//   PATCH  /api/keywords/:id
//   DELETE /api/keywords/:id

// 注意：client 在响应拦截器里已经返回 response.data，
// 所以 client.get() 拿到的"返回值"就是后端 body 本身（即 {data, meta}）。
// 这里继续解一层 .data 拿到真正的业务 payload。

import client from './client'

// 后端 Keyword 模型
export interface Keyword {
  id: number
  word: string
  note: string
  active: boolean
  intervalMin: number
  lastFetchedAt?: string
  createdAt: string
}

// 后端统一响应 { data: ..., meta?: {...} }
interface ApiResponse<T> {
  data: T
  meta?: Record<string, unknown>
}

// 列出全部关键词
export async function listKeywords(): Promise<Keyword[]> {
  const res = await client.get<unknown, ApiResponse<Keyword[]>>('/keywords')
  return res.data
}

// 新建关键词
export async function createKeyword(word: string, note: string, intervalMin: number): Promise<Keyword> {
  const res = await client.post<unknown, ApiResponse<Keyword>>('/keywords', {
    word,
    note,
    intervalMin,
  })
  return res.data
}

// 更新关键词（激活/暂停或间隔）
// PATCH 部分更新，未传的字段不动
export async function updateKeyword(
  id: number,
  active?: boolean,
  intervalMin?: number,
): Promise<Keyword> {
  const body: Record<string, unknown> = {}
  if (active !== undefined) body.active = active
  if (intervalMin !== undefined) body.intervalMin = intervalMin
  const res = await client.patch<unknown, ApiResponse<Keyword>>(`/keywords/${id}`, body)
  return res.data
}

// 删除关键词
export async function deleteKeyword(id: number): Promise<void> {
  await client.delete(`/keywords/${id}`)
}