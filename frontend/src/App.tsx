import { useCallback, useEffect, useState } from 'react'

import { APIError } from './api/client'
import {
  listRadarSignals,
  listSourceConfigs,
  login,
  logout,
  updateRedditCommunities,
  updateSourceConfig,
} from './api/radar'
import { LoginPage } from './pages/LoginPage'
import { RadarDashboard } from './pages/RadarDashboard'
import type { RadarSignal, SourceConfig } from './types'

export function App() {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null)
  const [signals, setSignals] = useState<RadarSignal[]>([])
  const [sources, setSources] = useState<SourceConfig[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const loadDashboard = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [nextSignals, nextSources] = await Promise.all([listRadarSignals(), listSourceConfigs()])
      setSignals(nextSignals)
      setSources(nextSources)
      setAuthenticated(true)
    } catch (cause) {
      if (cause instanceof APIError && cause.status === 401) {
        setAuthenticated(false)
      } else {
        setError(cause instanceof Error ? cause.message : String(cause))
        setAuthenticated((current) => current ?? false)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadDashboard()
  }, [loadDashboard])

  const handleLogin = async (password: string) => {
    setLoading(true)
    setError('')
    try {
      await login(password)
      await loadDashboard()
    } catch (cause) {
      setAuthenticated(false)
      setError(cause instanceof Error ? cause.message : String(cause))
    } finally {
      setLoading(false)
    }
  }

  const handleLogout = async () => {
    try {
      await logout()
    } finally {
      setAuthenticated(false)
      setSignals([])
    }
  }

  const handleSourceChange = async (source: string, enabled: boolean) => {
    try {
      await updateSourceConfig(source, enabled)
      setSources((current) => current.map((item) => (item.source === source ? { ...item, enabled } : item)))
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause))
    }
  }

  const handleRedditCommunitiesChange = async (communities: string[]) => {
    try {
      await updateRedditCommunities(communities)
      await loadDashboard()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause))
    }
  }

  if (authenticated === null) {
    return <main className="grid min-h-full place-items-center bg-base-bg text-text-secondary">正在连接私人雷达…</main>
  }
  if (!authenticated) {
    return <LoginPage loading={loading} error={error} onLogin={handleLogin} />
  }
  return (
    <RadarDashboard
      signals={signals}
      sources={sources}
      loading={loading}
      error={error}
      onRefresh={loadDashboard}
      onLogout={handleLogout}
      onSourceChange={handleSourceChange}
      onRedditCommunitiesChange={handleRedditCommunitiesChange}
    />
  )
}
