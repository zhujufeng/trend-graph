import client from './client'

export interface Topic {
  id: number
  word: string
  note: string
  active: boolean
  intervalMin: number
  lastFetchedAt?: string
  createdAt: string
}

interface ApiResponse<T> {
  data: T
  meta?: Record<string, unknown>
}

export async function listTopics(): Promise<Topic[]> {
  const res = await client.get<unknown, ApiResponse<Topic[]>>('/keywords')
  return res.data
}

export async function createTopic(word: string): Promise<Topic> {
  const res = await client.post<unknown, ApiResponse<Topic>>('/keywords', { word })
  return res.data
}

export async function updateTopic(id: number, active: boolean): Promise<Topic> {
  const res = await client.patch<unknown, ApiResponse<Topic>>(`/keywords/${id}`, { active })
  return res.data
}

export async function deleteTopic(id: number): Promise<void> {
  await client.delete(`/keywords/${id}`)
}
