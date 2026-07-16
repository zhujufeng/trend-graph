import client from './client'
import type { ContentPackage, RadarSignal, SourceConfig } from '../types'

interface ListResponse<T> {
  data: T[]
  count: number
}

export const login = async (password: string) => {
  await client.post('/auth/login', { password })
}

export const logout = async () => {
  await client.post('/auth/logout')
}

export const listRadarSignals = async (): Promise<RadarSignal[]> => {
  const response = await client.get<unknown, ListResponse<RadarSignal>>('/radar/signals', { params: { limit: 30 } })
  return response.data
}

export const listSourceConfigs = async (): Promise<SourceConfig[]> => {
  const response = await client.get<unknown, ListResponse<SourceConfig>>('/source-configs')
  return response.data
}

export const updateSignalLifecycle = async (id: number, state: RadarSignal['lifecycleState']): Promise<void> => {
  await client.patch(`/radar/signals/${id}/lifecycle`, { state })
}

export const updateSourceConfig = async (source: string, enabled: boolean): Promise<void> => {
  await client.put(`/source-configs/${source}`, { enabled })
}

export const updateRedditCommunities = async (communities: string[]): Promise<void> => {
  await client.put('/source-configs/reddit', { redditCommunities: communities })
}

export const createContentPackage = async (signalId: number): Promise<ContentPackage> => {
  const response = await client.post<unknown, { data: ContentPackage }>(`/radar/signals/${signalId}/content-packages`)
  return response.data
}

export const updateContentPackage = async (content: ContentPackage): Promise<ContentPackage> => {
  const response = await client.put<unknown, { data: ContentPackage }>(`/content-packages/${content.id}`, {
    strategy: content.strategy,
    xiaohongshu: content.xiaohongshu,
    wechat: content.wechat,
    x: content.x,
    visualPlan: content.visualPlan,
  })
  return response.data
}

export const approveContentPackage = async (id: number): Promise<void> => {
  await client.post(`/content-packages/${id}/approve`)
}
