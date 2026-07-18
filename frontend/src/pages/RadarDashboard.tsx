import { useState, type FormEvent, type ReactNode } from 'react'
import { Bookmark, CheckCircle2, ExternalLink, Inbox, LogOut, RefreshCw, Sparkles, Trash2, X } from 'lucide-react'

import type { Topic } from '../api/keywords'
import type { ContentPackage, RadarSignal, SourceConfig } from '../types'

interface RadarDashboardProps {
  signals: RadarSignal[]
  sources: SourceConfig[]
  topics: Topic[]
  loading: boolean
  error: string
  onRefresh: () => void
  onLogout: () => void
  onSourceChange: (source: string, enabled: boolean) => void
  onRedditCommunitiesChange: (communities: string[]) => void
  onGitHubRepositoriesChange: (repositories: string[]) => void
  onRSSFeedsChange: (feeds: string[]) => void
  onCreateTopic: (word: string) => void
  onToggleTopic: (topic: Topic) => void
  onDeleteTopic: (id: number) => void
  onLifecycleChange: (signalId: number, lifecycleState: string) => void
  contentPackage: ContentPackage | null
  onGenerateContent: (signalId: number) => void
  onSaveContent: (content: ContentPackage) => void
  onApproveContent: (content: ContentPackage) => void
}

type InboxFilter = 'inbox' | 'saved' | 'done'

export function RadarDashboard(props: RadarDashboardProps) {
  const {
    signals, sources, topics, loading, error, onRefresh, onLogout, onSourceChange,
    onRedditCommunitiesChange, onGitHubRepositoriesChange, onRSSFeedsChange,
    onCreateTopic, onToggleTopic, onDeleteTopic, onLifecycleChange,
    contentPackage, onGenerateContent, onSaveContent, onApproveContent,
  } = props
  const [filter, setFilter] = useState<InboxFilter>('inbox')
  const pending = signals.filter((signal) => signal.qualification === 'pending')
  const qualified = signals.filter((signal) => signal.qualification === 'qualified' && signal.lifecycleState !== 'dismissed')
  const visible = qualified.filter((signal) => signal.lifecycleState === filter)

  return (
    <main className="min-h-full bg-base-bg text-text-primary">
      <header className="sticky top-0 z-10 border-b border-border/80 bg-base-bg/90 backdrop-blur-xl">
        <div className="mx-auto flex max-w-6xl items-center gap-3 px-5 py-4">
          <div className="grid h-10 w-10 place-items-center rounded-xl bg-accent/10 text-accent"><Sparkles className="h-5 w-5" /></div>
          <div>
            <h1 className="text-lg font-semibold text-white">个人信息雷达</h1>
            <p className="text-xs text-text-muted">把公开信息变成一个安静、可处理的收件箱</p>
          </div>
          <button className="ml-auto radar-button" onClick={onRefresh} disabled={loading} type="button" aria-label="刷新">
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} /> <span className="hidden sm:inline">刷新</span>
          </button>
          <button className="radar-button" onClick={onLogout} type="button" aria-label="退出"><LogOut className="h-4 w-4" /> <span className="hidden sm:inline">退出</span></button>
        </div>
      </header>

      <div className="mx-auto max-w-6xl space-y-7 px-5 py-7">
        {error && <p className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">{error}</p>}

        <section className="grid grid-cols-2 gap-3 sm:grid-cols-4" aria-label="信息概览">
          <Metric label="新收件" value={countState(qualified, 'inbox')} />
          <Metric label="已保存" value={countState(qualified, 'saved')} />
          <Metric label="已处理" value={countState(qualified, 'done')} />
          <Metric label="等待分析" value={pending.length} subtle />
        </section>

        <TopicPanel topics={topics} onCreate={onCreateTopic} onToggle={onToggleTopic} onDelete={onDeleteTopic} />

        {contentPackage && (
          <ContentPackageEditor key={`${contentPackage.id}-${contentPackage.updatedAt}`} content={contentPackage} loading={loading} onSave={onSaveContent} onApprove={onApproveContent} />
        )}

        <section aria-labelledby="inbox-title">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <h2 id="inbox-title" className="flex items-center gap-2 text-xl font-semibold"><Inbox className="h-5 w-5 text-accent" /> 信息收件箱</h2>
              <p className="mt-1 text-sm text-text-muted">已按实际价值、时效和重要性排序，相同信息只保留一条。</p>
            </div>
            <div className="flex rounded-xl border border-border bg-surface p-1" role="tablist" aria-label="收件箱状态">
              {([['inbox', '新收件'], ['saved', '已保存'], ['done', '已处理']] as const).map(([state, label]) => (
                <button className={`rounded-lg px-3 py-1.5 text-sm ${filter === state ? 'bg-white/10 text-white' : 'text-text-muted hover:text-text-secondary'}`} onClick={() => setFilter(state)} type="button" role="tab" aria-selected={filter === state} key={state}>
                  {label} {countState(qualified, state)}
                </button>
              ))}
            </div>
          </div>

          {visible.length === 0 ? (
            <p className="mt-4 rounded-2xl border border-dashed border-border bg-surface/40 p-10 text-center text-sm text-text-muted">这个分组暂时是空的。</p>
          ) : (
            <div className="mt-4 space-y-3">
              {visible.map((signal) => <SignalCard signal={signal} loading={loading} onLifecycleChange={onLifecycleChange} onGenerateContent={onGenerateContent} key={signal.id} />)}
            </div>
          )}
        </section>

        <PendingQueue signals={pending.slice(0, 12)} />
        <SourceSettings
          sources={sources}
          onSourceChange={onSourceChange}
          onRedditCommunitiesChange={onRedditCommunitiesChange}
          onGitHubRepositoriesChange={onGitHubRepositoriesChange}
          onRSSFeedsChange={onRSSFeedsChange}
        />
      </div>
    </main>
  )
}

function countState(signals: RadarSignal[], state: InboxFilter) {
  return signals.filter((signal) => signal.lifecycleState === state).length
}

function Metric({ label, value, subtle = false }: { label: string; value: number; subtle?: boolean }) {
  return (
    <div className="rounded-2xl border border-border bg-surface px-5 py-4">
      <div className={`text-2xl font-semibold ${subtle ? 'text-text-secondary' : 'text-white'}`}>{value}</div>
      <div className="mt-1 text-xs text-text-muted">{label}</div>
    </div>
  )
}

function TopicPanel({ topics, onCreate, onToggle, onDelete }: { topics: Topic[]; onCreate: (word: string) => void; onToggle: (topic: Topic) => void; onDelete: (id: number) => void }) {
  const [word, setWord] = useState('')
  const submit = (event: FormEvent) => {
    event.preventDefault()
    const value = word.trim()
    if (!value) return
    onCreate(value)
    setWord('')
  }
  return (
    <section className="rounded-2xl border border-border bg-surface p-5">
      <div className="flex flex-wrap items-center gap-3">
        <div>
          <h2 className="font-semibold text-white">关注主题</h2>
          <p className="mt-1 text-xs text-text-muted">采集与 AI 筛选都会使用这些主题，最多启用 10 个。</p>
        </div>
        <form className="flex w-full gap-2 sm:ml-auto sm:w-auto" onSubmit={submit}>
          <input className="min-w-0 flex-1 rounded-lg border border-border bg-base-bg px-3 py-2 text-sm outline-none focus:border-accent sm:w-40" value={word} onChange={(event) => setWord(event.target.value)} placeholder="例如：机器人" aria-label="新关注主题" />
          <button className="rounded-lg bg-accent px-3 py-2 text-sm font-medium text-slate-950" type="submit">添加</button>
        </form>
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        {topics.map((topic) => (
          <div className={`inline-flex items-center rounded-full border pl-3 text-sm ${topic.active ? 'border-accent/30 bg-accent/10 text-cyan-100' : 'border-border text-text-muted'}`} key={topic.id}>
            <button className="py-1.5" type="button" onClick={() => onToggle(topic)} title={topic.active ? '暂停主题' : '启用主题'}>{topic.word}</button>
            <button className="px-2 py-1.5 hover:text-red-300" type="button" onClick={() => onDelete(topic.id)} aria-label={`删除主题 ${topic.word}`}><Trash2 className="h-3.5 w-3.5" /></button>
          </div>
        ))}
        {topics.length === 0 && <span className="text-sm text-text-muted">没有启用主题时，只采集你明确订阅的 GitHub Release。</span>}
      </div>
    </section>
  )
}

function SignalCard({ signal, loading, onLifecycleChange, onGenerateContent }: { signal: RadarSignal; loading: boolean; onLifecycleChange: (signalId: number, state: string) => void; onGenerateContent: (signalId: number) => void }) {
  const analysis = signal.analysis
  const topics = analysis?.matchedTopics ?? []
  return (
    <article className="rounded-2xl border border-border bg-surface p-5 transition hover:border-slate-600 sm:p-6">
      <div className="flex flex-wrap items-center gap-2 text-xs text-text-muted">
        <span className="rounded-full bg-white/5 px-2.5 py-1 text-text-secondary">{sourceLabel(signal.source)}</span>
        {topics.map((topic) => <span className="rounded-full bg-accent/10 px-2.5 py-1 text-cyan-200" key={topic}>{topic}</span>)}
        <span>{signalDateLabel(signal)}</span>
        {analysis?.valueScore && <span className="ml-auto font-medium text-amber-200">价值 {analysis.valueScore}/5</span>}
      </div>
      <h3 className="mt-4 text-xl font-semibold leading-8 text-white">{signal.title}</h3>
      {analysis?.whatChanged && <p className="mt-2 text-sm leading-6 text-text-secondary">{analysis.whatChanged}</p>}

      <div className="mt-4 grid gap-3 md:grid-cols-2">
        {analysis?.practicalUse && <Insight label="为什么值得看" value={analysis.practicalUse} />}
        {analysis?.action && <Insight label="下一步" value={analysis.action} accent />}
      </div>
      {analysis?.uncertainty && <p className="mt-3 text-xs leading-5 text-text-muted">不确定性：{analysis.uncertainty}</p>}

      <div className="mt-5 flex flex-wrap items-center gap-2 border-t border-border pt-4">
        <a className="radar-button" href={signal.originalUrl} target="_blank" rel="noreferrer">查看原文 <ExternalLink className="h-3.5 w-3.5" /></a>
        {signal.lifecycleState !== 'saved' && signal.lifecycleState !== 'done' && <ActionButton onClick={() => onLifecycleChange(signal.id, 'saved')} disabled={loading}><Bookmark className="h-4 w-4" /> 保存</ActionButton>}
        {signal.lifecycleState !== 'done' && <ActionButton primary onClick={() => onLifecycleChange(signal.id, 'done')} disabled={loading}><CheckCircle2 className="h-4 w-4" /> 已处理</ActionButton>}
        {signal.lifecycleState === 'saved' && <button className="px-2 py-2 text-xs text-text-muted hover:text-text-secondary" onClick={() => onLifecycleChange(signal.id, 'inbox')} disabled={loading} type="button">移回收件箱</button>}
        {signal.lifecycleState === 'done' && <button className="rounded-lg bg-purple-500/90 px-3 py-2 text-sm text-white" onClick={() => onGenerateContent(signal.id)} disabled={loading} type="button">生成内容草稿（可选）</button>}
        {signal.lifecycleState !== 'done' && <button className="ml-auto inline-flex items-center gap-1 px-2 py-2 text-xs text-text-muted hover:text-red-300" onClick={() => onLifecycleChange(signal.id, 'dismissed')} disabled={loading} type="button"><X className="h-3.5 w-3.5" /> 忽略</button>}
      </div>
    </article>
  )
}

function Insight({ label, value, accent = false }: { label: string; value: string; accent?: boolean }) {
  return <div className={`rounded-xl p-4 ${accent ? 'border border-amber-300/15 bg-amber-300/5' : 'bg-base-bg/70'}`}><div className={`text-xs font-medium ${accent ? 'text-amber-200' : 'text-cyan-300'}`}>{label}</div><p className="mt-2 text-sm leading-6 text-text-secondary">{value}</p></div>
}

function ActionButton({ children, onClick, disabled, primary = false }: { children: ReactNode; onClick: () => void; disabled: boolean; primary?: boolean }) {
  return <button className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm disabled:opacity-50 ${primary ? 'bg-emerald-500 text-white' : 'border border-border text-text-secondary hover:border-accent'}`} onClick={onClick} disabled={disabled} type="button">{children}</button>
}

function PendingQueue({ signals }: { signals: RadarSignal[] }) {
  if (signals.length === 0) return null
  return (
    <details className="rounded-2xl border border-border bg-surface/60">
      <summary className="cursor-pointer select-none px-5 py-4 text-sm font-medium">采集队列 <span className="ml-2 text-xs font-normal text-text-muted">{signals.length} 条等待筛选</span></summary>
      <div className="border-t border-border px-5 py-2">{signals.map((signal) => <div className="flex gap-3 border-b border-border/60 py-3 last:border-0" key={signal.id}><span className="w-20 text-xs text-text-muted">{sourceLabel(signal.source)}</span><a className="min-w-0 flex-1 truncate text-sm text-text-secondary hover:text-accent" href={signal.originalUrl} target="_blank" rel="noreferrer">{signal.title}</a></div>)}</div>
    </details>
  )
}

function SourceSettings({ sources, onSourceChange, onRedditCommunitiesChange, onGitHubRepositoriesChange, onRSSFeedsChange }: { sources: SourceConfig[]; onSourceChange: (source: string, enabled: boolean) => void; onRedditCommunitiesChange: (items: string[]) => void; onGitHubRepositoriesChange: (items: string[]) => void; onRSSFeedsChange: (items: string[]) => void }) {
  const reddit = sources.find((item) => item.source === 'reddit')
  const github = sources.find((item) => item.source === 'github')
  const rss = sources.find((item) => item.source === 'rss')
  return (
    <details className="rounded-2xl border border-border bg-surface/40">
      <summary className="cursor-pointer select-none px-5 py-4 text-sm font-medium">来源设置与运行状态</summary>
      <div className="grid gap-6 border-t border-border p-5 lg:grid-cols-2">
        <div className="space-y-3">{sources.map((source) => <label className="flex cursor-pointer items-center justify-between gap-3 text-sm" key={source.source}><span><span className="block">{sourceLabel(source.source)}</span><span className="block text-xs text-text-muted">{sourceStatus(source)}</span></span><input type="checkbox" checked={source.enabled} onChange={(event) => onSourceChange(source.source, event.target.checked)} aria-label={`${sourceLabel(source.source)}采集开关`} /></label>)}</div>
        <div className="space-y-4">
          {github && <ListEditor label="GitHub Release 订阅" hint="每行一个 owner/repo，最多 20 个" values={github.settings.repositories ?? []} onSave={onGitHubRepositoriesChange} />}
          {rss && <ListEditor label="RSS / Atom 订阅" hint="每行一个 HTTP(S) Feed 地址，最多 20 个" values={rss.settings.feeds ?? []} onSave={onRSSFeedsChange} />}
          {reddit && <ListEditor label="Reddit 社区" hint="逗号或换行分隔，例如 r/LocalLLaMA" values={reddit.settings.communities ?? []} onSave={onRedditCommunitiesChange} />}
        </div>
      </div>
    </details>
  )
}

function ListEditor({ label, hint, values, onSave }: { label: string; hint: string; values: string[]; onSave: (values: string[]) => void }) {
  const [value, setValue] = useState(values.join('\n'))
  const submit = (event: FormEvent) => {
    event.preventDefault()
    onSave(value.split(/[,\n]+/).map((item) => item.trim()).filter(Boolean))
  }
  return <form className="rounded-xl border border-border p-4" onSubmit={submit}><label className="text-sm font-medium">{label}<span className="mt-1 block text-xs font-normal text-text-muted">{hint}</span><textarea className="mt-3 min-h-24 w-full rounded-lg border border-border bg-base-bg p-2 text-xs outline-none focus:border-accent" value={value} onChange={(event) => setValue(event.target.value)} /></label><button className="mt-2 rounded-lg border border-border px-3 py-2 text-xs hover:border-accent hover:text-accent" type="submit">保存设置</button></form>
}

function ContentPackageEditor({ content, loading, onSave, onApprove }: { content: ContentPackage; loading: boolean; onSave: (content: ContentPackage) => void; onApprove: (content: ContentPackage) => void }) {
  const [draft, setDraft] = useState(content)
  const approved = draft.status === 'approved'
  return (
    <section className="rounded-2xl border border-purple-400/30 bg-surface p-5">
      <div className="flex items-center gap-3"><div><h2 className="font-semibold">内容草稿工作台（可选）</h2><p className="mt-1 text-xs text-text-muted">基于已处理信息生成，可继续编辑后再确认。</p></div><span className="ml-auto text-xs text-purple-300">{approved ? '已确认' : '草稿'}</span></div>
      <label className="mt-4 block text-xs text-text-muted">内容角度<input className="mt-1 w-full rounded-lg border border-border bg-base-bg p-2 text-sm" value={draft.strategy.angle} disabled={approved} onChange={(event) => setDraft((current) => ({ ...current, strategy: { ...current.strategy, angle: event.target.value } }))} /></label>
      <div className="mt-4 grid gap-3 lg:grid-cols-2">
        <DraftEditor label="小红书" value={draft.xiaohongshu.body} disabled={approved} onChange={(body) => setDraft((current) => ({ ...current, xiaohongshu: { ...current.xiaohongshu, body } }))} />
        <DraftEditor label="公众号" value={draft.wechat.body} disabled={approved} onChange={(body) => setDraft((current) => ({ ...current, wechat: { ...current.wechat, body } }))} />
        <DraftEditor label="X 中文" value={draft.x.chinese} disabled={approved} onChange={(chinese) => setDraft((current) => ({ ...current, x: { ...current.x, chinese } }))} />
        <DraftEditor label="X English" value={draft.x.english} disabled={approved} onChange={(english) => setDraft((current) => ({ ...current, x: { ...current.x, english } }))} />
      </div>
      {!approved && <div className="mt-4 flex gap-2"><button className="radar-button" onClick={() => onSave(draft)} disabled={loading} type="button">保存草稿</button><button className="rounded-lg bg-purple-500 px-3 py-2 text-sm text-white" onClick={() => onApprove(draft)} disabled={loading} type="button">确认内容</button></div>}
    </section>
  )
}

function DraftEditor({ label, value, disabled, onChange }: { label: string; value: string; disabled: boolean; onChange: (value: string) => void }) {
  return <label className="text-xs text-text-muted">{label}<textarea className="mt-1 min-h-36 w-full rounded-lg border border-border bg-base-bg p-3 text-sm text-text-primary" value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)} /></label>
}

function sourceStatus(source: SourceConfig) {
  if (!source.lastRun) return source.lastSuccessAt ? '最近采集正常' : '等待首次采集'
  if (source.lastRun.status === 'success') return `最近采集 ${source.lastRun.itemCount} 条`
  return `异常：${source.lastRun.failureReason ?? source.lastFailure ?? '未知错误'}`
}

function signalDateLabel(signal: RadarSignal) {
  const value = signal.sourceUpdatedAt ?? signal.sourcePublishedAt ?? signal.createdAt
  return new Intl.DateTimeFormat('zh-CN', { month: 'numeric', day: 'numeric', timeZone: 'Asia/Shanghai' }).format(new Date(value))
}

function sourceLabel(source: string) {
  return ({ dev: 'DEV', github: 'GitHub', reddit: 'Reddit', bluesky: 'Bluesky', rss: 'RSS / Atom' } as Record<string, string>)[source] ?? source
}
