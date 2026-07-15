import client from './client'
import type { RadarSignal, SourceConfig } from '../types'

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

export const updateSourceConfig = async (source: string, enabled: boolean): Promise<void> => {
  await client.put(`/source-configs/${source}`, { enabled })
}

export const updateRedditCommunities = async (communities: string[]): Promise<void> => {
  await client.put('/source-configs/reddit', { redditCommunities: communities })
}
