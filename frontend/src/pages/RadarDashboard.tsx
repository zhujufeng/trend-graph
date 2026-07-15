import { useState, type FormEvent } from 'react'
import { ExternalLink, LogOut, RefreshCw, Sparkles } from 'lucide-react'

import type { RadarSignal, SourceConfig } from '../types'

interface RadarDashboardProps {
  signals: RadarSignal[]
  sources: SourceConfig[]
  loading: boolean
  error: string
  onRefresh: () => void
  onLogout: () => void
  onSourceChange: (source: string, enabled: boolean) => void
  onRedditCommunitiesChange: (communities: string[]) => void
}

export function RadarDashboard({
  signals,
  sources,
  loading,
  error,
  onRefresh,
  onLogout,
  onSourceChange,
  onRedditCommunitiesChange,
}: RadarDashboardProps) {
  const qualified = signals.filter((signal) => signal.qualification === 'qualified')
  const pending = signals.filter((signal) => signal.qualification === 'pending')
  const tools = qualified.filter((signal) => signal.source === 'github' || signal.source === 'skillsmp')
  const content = qualified.filter((signal) => Boolean(signal.analysis?.contentOpportunity))
  const redditCommunities = sources.find((source) => source.source === 'reddit')?.settings.communities

  return (
    <main className="min-h-full bg-base-bg text-text-primary">
      <header className="border-b border-border bg-surface/80">
        <div className="mx-auto flex max-w-7xl items-center gap-3 px-5 py-4">
          <Sparkles className="h-6 w-6 text-accent" aria-hidden="true" />
          <div>
            <h1 className="text-xl font-semibold">AI 信号雷达</h1>
            <p className="text-xs text-text-muted">只看能学习、能实践、能转化为内容的 AI 动态</p>
          </div>
          <button className="ml-auto radar-button" onClick={onRefresh} disabled={loading} type="button">
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} aria-hidden="true" />
            刷新
          </button>
          <button className="radar-button" onClick={onLogout} type="button">
            <LogOut className="h-4 w-4" aria-hidden="true" />
            退出
          </button>
        </div>
      </header>

      <div className="mx-auto grid max-w-7xl gap-6 px-5 py-6 lg:grid-cols-[1fr_280px]">
        <div className="space-y-8">
          {error && <p className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">{error}</p>}
          {pending.length > 0 && <SignalSection title="最新采集（待分析）" signals={pending.slice(0, 12)} empty="" />}
          <SignalSection title="今日必读" signals={qualified.slice(0, 6)} empty="暂时没有新信号" />
          <SignalSection title="可用工具与 Skill" signals={tools} empty="等待 GitHub 与 SkillsMP 的可用项目" />
          <SignalSection title="内容素材" signals={content} empty="信号分析后会在这里出现可用选题" />
        </div>

        <aside className="h-fit rounded-xl border border-border bg-surface p-4">
          <h2 className="font-medium">信息源</h2>
          <p className="mt-1 text-xs text-text-muted">关闭后采集任务会跳过该来源</p>
          <div className="mt-4 space-y-3">
            {sources.map((source) => (
              <label key={source.source} className="flex cursor-pointer items-center justify-between gap-3 text-sm">
                <span>
                  <span className="block">{sourceLabel(source.source)}</span>
                  <span className="block text-xs text-text-muted">
                    {source.lastFailure ? `异常：${source.lastFailure}` : source.lastSuccessAt ? '最近采集正常' : '等待首次采集'}
                  </span>
                </span>
                <input
                  type="checkbox"
                  checked={source.enabled}
                  onChange={(event) => onSourceChange(source.source, event.target.checked)}
                  aria-label={`${sourceLabel(source.source)}采集开关`}
                />
              </label>
            ))}
          </div>
          {redditCommunities && (
            <RedditAllowlist
              key={redditCommunities.join(',')}
              communities={redditCommunities}
              onSave={onRedditCommunitiesChange}
            />
          )}
          <div className="mt-4 border-t border-border pt-4 text-xs text-text-muted">
            X：保留为后续关键词搜索采集，目前未启用。
          </div>
        </aside>
      </div>
    </main>
  )
}

function RedditAllowlist({ communities, onSave }: { communities: string[]; onSave: (communities: string[]) => void }) {
  const [value, setValue] = useState(communities.join(', '))
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    onSave(value.split(/[,\n]+/).map((item) => item.trim()).filter(Boolean))
  }
  return (
    <form className="mt-5 border-t border-border pt-4" onSubmit={submit}>
      <label className="text-xs font-medium" htmlFor="reddit-communities">Reddit 社区白名单</label>
      <textarea
        id="reddit-communities"
        className="mt-2 min-h-24 w-full rounded-lg border border-border bg-base-bg p-2 text-xs outline-none focus:border-accent"
        value={value}
        onChange={(event) => setValue(event.target.value)}
      />
      <button className="mt-2 w-full rounded-lg border border-border px-3 py-2 text-xs hover:border-accent hover:text-accent" type="submit">
        保存白名单
      </button>
    </form>
  )
}

function SignalSection({ title, signals, empty }: { title: string; signals: RadarSignal[]; empty: string }) {
  return (
    <section>
      <div className="mb-3 flex items-end justify-between">
        <h2 className="text-lg font-semibold">{title}</h2>
        <span className="text-xs text-text-muted">{signals.length} 条</span>
      </div>
      {signals.length === 0 ? (
        <p className="rounded-xl border border-dashed border-border p-6 text-sm text-text-muted">{empty}</p>
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {signals.map((signal) => (
            <article key={`${title}-${signal.id}`} className="rounded-xl border border-border bg-surface p-4 glow-border">
              <div className="flex items-center gap-2 text-xs text-text-muted">
                <span className="rounded bg-accent/10 px-2 py-0.5 text-accent">{sourceLabel(signal.source)}</span>
                <span>{signal.qualification === 'qualified' ? '已筛选' : '待分析'}</span>
                {signal.evidence && <span>· {evidenceLabel(signal.evidence.evidenceClass)}</span>}
              </div>
              <h3 className="mt-3 font-medium leading-6">{signal.title}</h3>
              <p className="mt-2 line-clamp-3 text-sm text-text-secondary">
                {signal.analysis?.whatChanged ?? signal.evidence?.excerpt ?? '等待详情采集'}
              </p>
              {signal.analysis?.action && <p className="mt-3 text-sm text-emerald-300">落地：{signal.analysis.action}</p>}
              {signal.analysis?.contentOpportunity && (
                <p className="mt-2 text-sm text-purple-300">选题：{signal.analysis.contentOpportunity}</p>
              )}
              <a className="mt-4 inline-flex items-center gap-1 text-sm text-accent hover:text-accent-hover" href={signal.originalUrl} target="_blank" rel="noreferrer">
                查看原始来源 <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
              </a>
            </article>
          ))}
        </div>
      )}
    </section>
  )
}

function sourceLabel(source: string) {
  return ({ waytoagi: 'WaytoAGI', skillsmp: 'SkillsMP', github: 'GitHub', reddit: 'Reddit' } as Record<string, string>)[source] ?? source
}

function evidenceLabel(evidenceClass: string) {
  return ({
    original_documentation: '官方/原始文档',
    documented_third_party_practice: '第三方实践',
    community_discussion: '社区讨论',
    user_verified: '本人验证',
    catalog_discovery: '目录线索',
  } as Record<string, string>)[evidenceClass] ?? evidenceClass
}
