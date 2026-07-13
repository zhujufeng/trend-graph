// HotListPage.tsx
//
// 热点列表主页面，是项目第一屏，也是用户最常看的页面。
//
// 这是一个"容器组件"：
//   - 管理状态（来源、关键词、列表数据、loading、错误）
//   - 调用 API
//   - 把数据分发给展示组件 HotCard
//
// 这是 React 函数组件的核心模式，掌握这套写法基本上 80% 的页面都会写了。

// useState / useEffect 是最基础的两个 React Hook
// - useState: 给组件加"状态变量"
// - useEffect: 处理副作用（请求 API、订阅事件、定时等）
import { useEffect, useState, useCallback } from 'react'
// 图标库
import { RefreshCw, Plus, Loader2, Sparkles, AlertCircle } from 'lucide-react'

// 我们自己写的 API 客户端和类型
import { listHots, listSources, triggerCrawl, expandKeyword, getHot, analyzeHot } from '../api'
import type { HotItem, ListParams } from '../types'
import { HotCard } from '../components/HotCard'

// 默认值
const DEFAULT_SINCE = '24h'
const DEFAULT_LIMIT = 20

export function HotListPage() {
  // ====== 状态 ======
  // useState 返回 [当前值, 修改值的函数]
  // 修改时 React 会自动重新渲染组件

  // 所有可选信息源
  const [sources, setSources] = useState<string[]>([])
  // 当前选中的来源(""表示全部)
  const [activeSource, setActiveSource] = useState<string>('')
  // 监控关键词
  const [keyword, setKeyword] = useState<string>('')
  // 时间范围下拉
  const [since, setSince] = useState<string>(DEFAULT_SINCE)
  // 热点列表
  const [items, setItems] = useState<HotItem[]>([])
  // 列表总数（分页用）
  const [total, setTotal] = useState<number>(0)
  // 当前页码（前端转换 offset）
  const [page, setPage] = useState<number>(1)

  // 加载态、错误态、AI 分析中态
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')
  const [crawling, setCrawling] = useState(false)
  const [analyzingId, setAnalyzingId] = useState<number | null>(null)
  const [expandedKeywords, setExpandedKeywords] = useState<string[] | null>(null)
  const [expanding, setExpanding] = useState(false)

  // ====== 副作用 ======
  // useEffect 在组件渲染后执行
  // 第二个参数是依赖数组，依赖变了会重新执行
  // [] 表示只在挂载时执行一次

  // 初次加载：拉信息源
  useEffect(() => {
    listSources()
      .then(setSources)
      .catch((e) => console.error('listSources failed', e))
  }, [])

  // 筛选条件变了就重新拉热点
  // 把 fetchList 抽成 useCallback 避免每次重渲染都重建函数
  const fetchList = useCallback(async () => {
    setLoading(true)
    setError('')
    const params: ListParams = {
      source: activeSource || undefined,
      since,
      limit: DEFAULT_LIMIT,
      offset: (page - 1) * DEFAULT_LIMIT,
    }
    try {
      const { items: list, meta } = await listHots(params)
      setItems(list)
      setTotal(meta.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [activeSource, since, page])

  useEffect(() => {
    fetchList()
  }, [fetchList])

  // ====== 事件处理 ======

  // 点击"立即抓取"按钮
  const handleCrawl = async () => {
    setCrawling(true)
    setError('')
    try {
      // analyze=true 让后端抓完同时做 AI 分析
      await triggerCrawl(activeSource || 'hn', keyword, 10, true)
      // 抓完立刻刷新列表
      await fetchList()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setCrawling(false)
    }
  }

  // 触发 AI 查询扩展
  const handleExpand = async () => {
    if (!keyword.trim()) {
      setError('请输入关键词再扩展')
      return
    }
    setExpanding(true)
    setError('')
    try {
      const words = await expandKeyword(keyword)
      setExpandedKeywords(words)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setExpanding(false)
    }
  }

  // 单条热点的 AI 分析
  const handleAnalyze = async (item: HotItem) => {
    setAnalyzingId(item.id)
    try {
      // 后端会直接 update DB，并返回分析结果
      await analyzeHot(item.id, keyword)
      // 单条更新：再用 getHot 拉拿到带 AI 字段的版本
      const updated = await getHot(item.id)
      setItems((prev) => prev.map((it) => (it.id === item.id ? updated : it)))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setAnalyzingId(null)
    }
  }

  // ====== 渲染 ======
  // JSX：HTML 和 JS 混写的语法，最终编译成 React.createElement 调用
  // 条件渲染用 && 或 三元
  // 列表渲染用 array.map((item) => <element key={item.id} />)

  return (
    <div className="min-h-full flex flex-col">
      {/* ====== 顶部 Header ====== */}
      <header className="border-b border-border bg-surface/50 backdrop-blur sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center gap-2 mb-1">
            <Sparkles className="w-6 h-6 text-accent" />
            <h1 className="text-xl font-bold">
              <span className="text-gradient">trend-graph</span>
            </h1>
            <span className="text-xs text-text-muted ml-2 hidden sm:inline">
              AI 热点监控 + 关联图谱
            </span>
          </div>

          {/* 控制条 */}
          <div className="flex flex-wrap items-center gap-3 mt-4">
            {/* 关键词输入框 */}
            <input
              type="text"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleExpand()}
              placeholder="监控关键词，如 AI"
              className="flex-1 min-w-40 px-3 py-1.5 bg-base-bg border border-border rounded-md text-sm focus:border-accent focus:outline-none transition"
            />

            {/* 来源筛选 */}
            <select
              value={activeSource}
              onChange={(e) => {
                setActiveSource(e.target.value)
                setPage(1)
              }}
              className="px-3 py-1.5 bg-base-bg border border-border rounded-md text-sm focus:border-accent focus:outline-none"
            >
              <option value="">全部来源</option>
              {sources.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>

            {/* 时间范围 */}
            <select
              value={since}
              onChange={(e) => {
                setSince(e.target.value)
                setPage(1)
              }}
              className="px-3 py-1.5 bg-base-bg border border-border rounded-md text-sm focus:border-accent focus:outline-none"
            >
              <option value="1h">最近 1 小时</option>
              <option value="24h">最近 24 小时</option>
              <option value="3d">最近 3 天</option>
              <option value="7d">最近 7 天</option>
              <option value="30d">最近 30 天</option>
            </select>

            {/* AI 扩展按钮 */}
            <button
              onClick={handleExpand}
              disabled={expanding}
              className="flex items-center gap-1 px-3 py-1.5 text-sm border border-purple-500/30 text-purple-400 rounded-md hover:bg-purple-500/10 transition disabled:opacity-40"
            >
              {expanding ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Plus className="w-3.5 h-3.5" />}
              AI 扩展
            </button>

            {/* 立即抓取 */}
            <button
              onClick={handleCrawl}
              disabled={crawling}
              className="flex items-center gap-1 px-3 py-1.5 text-sm bg-accent text-base-bg rounded-md hover:bg-accent-hover transition disabled:opacity-40 font-medium"
            >
              {crawling ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RefreshCw className="w-3.5 h-3.5" />}
              立即抓取 + AI
            </button>
          </div>

          {/* AI 扩展结果展示 */}
          {expandedKeywords && (
            <div className="flex flex-wrap items-center gap-1.5 mt-3">
              <span className="text-xs text-text-muted">AI 扩展词：</span>
              {expandedKeywords.map((w, i) => (
                <span
                  key={i}
                  className="text-xs px-2 py-0.5 rounded bg-purple-500/10 text-purple-400 border border-purple-500/20"
                >
                  {w}
                </span>
              ))}
              <button
                onClick={() => setExpandedKeywords(null)}
                className="text-xs text-text-muted hover:text-text-primary ml-1"
              >
                ✕
              </button>
            </div>
          )}
        </div>
      </header>

      {/* ====== 错误提示 ====== */}
      {error && (
        <div className="max-w-7xl mx-auto w-full px-4 mt-4">
          <div className="flex items-start gap-2 p-3 bg-red-500/10 border border-red-500/30 rounded-md text-sm text-red-400">
            <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
            <span>{error}</span>
            <button onClick={() => setError('')} className="ml-auto text-red-400/60 hover:text-red-400">
              ✕
            </button>
          </div>
        </div>
      )}

      {/* ====== 列表 ====== */}
      <main className="max-w-7xl mx-auto w-full px-4 py-6 flex-1">
        <div className="flex items-center justify-between mb-4">
          <p className="text-sm text-text-muted">
            共 <span className="text-text-primary font-mono">{total}</span> 条
            {loading && <Loader2 className="w-3 h-3 inline ml-2 animate-spin" />}
          </p>
        </div>

        {/* 卡片网格：响应式 */}
        {items.length === 0 && !loading ? (
          <div className="text-center py-20 text-text-muted">
            <p className="mb-2">还没有热点数据</p>
            <button
              onClick={handleCrawl}
              disabled={crawling}
              className="text-accent hover:text-accent-hover transition"
            >
              点这里立即抓取一批试试
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {items.map((item) => (
              <HotCard
                key={item.id}
                item={item}
                onAnalyze={handleAnalyze}
                analyzing={analyzingId === item.id}
              />
            ))}
          </div>
        )}

        {/* 分页 */}
        {total > DEFAULT_LIMIT && (
          <div className="flex items-center justify-center gap-2 mt-8">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1 || loading}
              className="px-3 py-1.5 text-sm border border-border rounded-md hover:border-accent transition disabled:opacity-40"
            >
              上一页
            </button>
            <span className="text-sm text-text-secondary px-3 font-mono">
              {page} / {Math.ceil(total / DEFAULT_LIMIT)}
            </span>
            <button
              onClick={() => setPage((p) => p + 1)}
              disabled={page >= Math.ceil(total / DEFAULT_LIMIT) || loading}
              className="px-3 py-1.5 text-sm border border-border rounded-md hover:border-accent transition disabled:opacity-40"
            >
              下一页
            </button>
          </div>
        )}
      </main>

      {/* ====== Footer ====== */}
      <footer className="border-t border-border py-4 mt-auto">
        <div className="max-w-7xl mx-auto px-4 text-center text-xs text-text-muted">
          trend-graph · Go + TypeScript + PostgreSQL + DeepSeek ·{' '}
          <a
            href="https://github.com/zhujufeng/trend-graph"
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-accent transition"
          >
            GitHub
          </a>
        </div>
      </footer>
    </div>
  )
}