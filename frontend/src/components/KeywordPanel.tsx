// KeywordPanel.tsx
//
// 监控关键词管理面板：增删改 + 暂停/激活 + 显示下次抓取时间。
//
// 这是阶段 7 前端关键改动：把后端 keywords CRUD 用一个独立组件展示，
// 用户配好关键词后，后端 scheduler 会自动按间隔跑抓取。
//
// 这是"独立子组件"，父级 HotListPage 用 toggle 控制显隐。

import { useEffect, useState, useCallback } from 'react'
import { Trash2, Pause, Play, Plus, Loader2, X, Clock, CheckCircle2 } from 'lucide-react'
import { listKeywords, createKeyword, updateKeyword, deleteKeyword } from '../api/keywords'

// 后端关键词结构（对应 store.Keyword 的 JSON）
interface Keyword {
  id: number
  word: string
  note: string
  active: boolean
  intervalMin: number
  lastFetchedAt?: string
  createdAt: string
}

interface KeywordPanelProps {
  onClose: () => void
  // 关键词变更后通知父组件（让用户在主列表里能切到该关键词筛选）
  onKeywordsChanged?: () => void
}

export function KeywordPanel({ onClose, onKeywordsChanged }: KeywordPanelProps) {
  const [keywords, setKeywords] = useState<Keyword[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // 新建表单
  const [newWord, setNewWord] = useState('')
  const [newNote, setNewNote] = useState('')
  const [newInterval, setNewInterval] = useState(30)
  const [creating, setCreating] = useState(false)

  const fetchKeywords = useCallback(async () => {
    setLoading(true)
    try {
      const ks = await listKeywords()
      setKeywords(ks)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchKeywords()
  }, [fetchKeywords])

  const handleCreate = async () => {
    if (!newWord.trim()) {
      setError('关键词不能为空')
      return
    }
    setCreating(true)
    setError('')
    try {
      await createKeyword(newWord.trim(), newNote, newInterval)
      setNewWord('')
      setNewNote('')
      setNewInterval(30)
      await fetchKeywords()
      onKeywordsChanged?.()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setCreating(false)
    }
  }

  const handleToggle = async (k: Keyword) => {
    try {
      await updateKeyword(k.id, !k.active, undefined)
      await fetchKeywords()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  const handleIntervalChange = async (k: Keyword, interval: number) => {
    try {
      await updateKeyword(k.id, undefined, interval)
      await fetchKeywords()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  const handleDelete = async (k: Keyword) => {
    if (!confirm(`确定删除关键词 "${k.word}"？`)) return
    try {
      await deleteKeyword(k.id)
      await fetchKeywords()
      onKeywordsChanged?.()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center p-4 bg-black/60 backdrop-blur-sm">
      <div className="bg-surface rounded-xl border border-border w-full max-w-3xl my-8 max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-5 border-b border-border">
          <div>
            <h2 className="text-lg font-medium">监控关键词管理</h2>
            <p className="text-xs text-text-muted mt-1">
              添加关键词后，后端会按设置间隔自动抓取 9 源并 AI 分析，命中高相关时给所有通知渠道推送
            </p>
          </div>
          <button onClick={onClose} className="text-text-muted hover:text-text-primary">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* 新建表单 */}
        <div className="p-5 border-b border-border">
          <div className="flex flex-wrap gap-3 items-end">
            <div className="flex-1 min-w-40">
              <label className="text-xs text-text-muted block mb-1">关键词 *</label>
              <input
                value={newWord}
                onChange={(e) => setNewWord(e.target.value)}
                placeholder="如：AI、Go、北京申办奥运"
                className="w-full px-3 py-1.5 bg-base-bg border border-border rounded-md text-sm focus:border-accent focus:outline-none"
              />
            </div>
            <div className="flex-1 min-w-40">
              <label className="text-xs text-text-muted block mb-1">备注</label>
              <input
                value={newNote}
                onChange={(e) => setNewNote(e.target.value)}
                placeholder="如：监控大模型更新"
                className="w-full px-3 py-1.5 bg-base-bg border border-border rounded-md text-sm focus:border-accent focus:outline-none"
              />
            </div>
            <div className="w-24">
              <label className="text-xs text-text-muted block mb-1">间隔（分钟）</label>
              <input
                type="number"
                min={1}
                value={newInterval}
                onChange={(e) => setNewInterval(Number(e.target.value) || 30)}
                className="w-full px-3 py-1.5 bg-base-bg border border-border rounded-md text-sm focus:border-accent focus:outline-none"
              />
            </div>
            <button
              onClick={handleCreate}
              disabled={creating}
              className="flex items-center gap-1 px-4 py-1.5 bg-accent text-base-bg text-sm rounded-md hover:bg-accent-hover transition disabled:opacity-40 font-medium"
            >
              {creating ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Plus className="w-3.5 h-3.5" />}
              添加
            </button>
          </div>
        </div>

        {/* 错误提示 */}
        {error && (
          <div className="px-5 py-3 bg-red-500/10 border-b border-red-500/30 text-sm text-red-400">
            {error}
            <button onClick={() => setError('')} className="ml-2 text-red-400/60">✕</button>
          </div>
        )}

        {/* 关键词列表 */}
        <div className="flex-1 overflow-y-auto p-5">
          {loading ? (
            <div className="flex justify-center py-12">
              <Loader2 className="w-5 h-5 animate-spin text-text-muted" />
            </div>
          ) : keywords.length === 0 ? (
            <div className="text-center py-12 text-text-muted text-sm">
              还没有监控关键词
            </div>
          ) : (
            <div className="space-y-2">
              {keywords.map((k) => (
                <div
                  key={k.id}
                  className={`flex items-center gap-3 p-3 rounded-md border ${
                    k.active ? 'border-accent/30 bg-accent/5' : 'border-border bg-base-bg/50'
                  }`}
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium truncate">{k.word}</span>
                      {k.active ? (
                        <span className="flex items-center gap-1 text-xs text-emerald-400">
                          <CheckCircle2 className="w-3 h-3" />激活
                        </span>
                      ) : (
                        <span className="text-xs text-text-muted">已暂停</span>
                      )}
                    </div>
                    {k.note && <p className="text-xs text-text-muted mt-0.5 truncate">{k.note}</p>}
                    {k.lastFetchedAt && (
                      <p className="text-xs text-text-muted mt-0.5 flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        最近一次 {new Date(k.lastFetchedAt).toLocaleString('zh-CN')}
                      </p>
                    )}
                  </div>

                  {/* 间隔 */}
                  <div className="flex items-center gap-1 text-xs text-text-muted">
                    <span>每</span>
                    <input
                      type="number"
                      min={1}
                      value={k.intervalMin}
                      onChange={(e) => handleIntervalChange(k, Number(e.target.value) || 30)}
                      className="w-14 px-1.5 py-0.5 bg-base-bg border border-border rounded text-xs focus:border-accent focus:outline-none"
                    />
                    <span>分</span>
                  </div>

                  {/* 暂停/激活 */}
                  <button
                    onClick={() => handleToggle(k)}
                    className={`p-1.5 rounded transition ${
                      k.active
                        ? 'text-orange-400 hover:bg-orange-500/10'
                        : 'text-emerald-400 hover:bg-emerald-500/10'
                    }`}
                    title={k.active ? '暂停' : '激活'}
                  >
                    {k.active ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                  </button>

                  {/* 删除 */}
                  <button
                    onClick={() => handleDelete(k)}
                    className="p-1.5 rounded text-red-400 hover:bg-red-500/10 transition"
                    title="删除"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}