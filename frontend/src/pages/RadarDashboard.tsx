import { useState, type FormEvent } from 'react'
import { CheckCircle2, ExternalLink, ListChecks, LogOut, RefreshCw, Sparkles, X } from 'lucide-react'

import type { ContentPackage, RadarSignal, SourceConfig } from '../types'

interface RadarDashboardProps {
  signals: RadarSignal[]
  sources: SourceConfig[]
  loading: boolean
  error: string
  onRefresh: () => void
  onLogout: () => void
  onSourceChange: (source: string, enabled: boolean) => void
  onRedditCommunitiesChange: (communities: string[]) => void
  onLifecycleChange: (signalId: number, lifecycleState: string) => void
  contentPackage: ContentPackage | null
  onGenerateContent: (signalId: number) => void
  onSaveContent: (content: ContentPackage) => void
  onApproveContent: (content: ContentPackage) => void
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
  onLifecycleChange,
  contentPackage,
  onGenerateContent,
  onSaveContent,
  onApproveContent,
}: RadarDashboardProps) {
  const pending = signals.filter((signal) => signal.qualification === 'pending')
  const qualified = signals.filter((signal) => signal.qualification === 'qualified')
  const practiceQueue = qualified
    .filter((signal) => signal.lifecycleState === 'new' || signal.lifecycleState === 'queued')
    .sort((left, right) => Number(right.lifecycleState === 'queued') - Number(left.lifecycleState === 'queued') || right.score - left.score)
  const practiced = qualified.filter((signal) => signal.lifecycleState === 'practiced')

  return (
    <main className="min-h-full bg-base-bg text-text-primary">
      <header className="border-b border-border bg-surface/70 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center gap-3 px-5 py-4">
          <Sparkles className="h-6 w-6 text-amber-300" aria-hidden="true" />
          <div>
            <h1 className="text-xl font-semibold">AI 实践雷达</h1>
            <p className="text-xs text-text-muted">发现真实案例，完成实践，再生成内容</p>
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

      <div className="mx-auto max-w-6xl space-y-8 px-5 py-7">
        <section className="grid gap-3 sm:grid-cols-4" aria-label="工作区概览">
          <Metric label="待实践" value={practiceQueue.filter((signal) => signal.lifecycleState === 'new').length} />
          <Metric label="实践中" value={practiceQueue.filter((signal) => signal.lifecycleState === 'queued').length} />
          <Metric label="可创作" value={practiced.length} />
          <Metric label="待分析" value={pending.length} />
        </section>

        {error && <p className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">{error}</p>}
        {contentPackage && (
          <ContentPackageEditor
            key={`${contentPackage.id}-${contentPackage.updatedAt}`}
            content={contentPackage}
            loading={loading}
            onSave={onSaveContent}
            onApprove={onApproveContent}
          />
        )}

        <WorkflowSection
          title="待实践"
          description="AI 已完成初筛。选一个加入实践，不再刷一整页资讯。"
          empty="当前没有值得实践的新信号"
          signals={practiceQueue}
          loading={loading}
          onLifecycleChange={onLifecycleChange}
          onGenerateContent={onGenerateContent}
        />

        <WorkflowSection
          title="已实践，可创作"
          description="只有你确认实践过的内容，才进入小红书、公众号和 X 的创作流程。"
          empty="完成一次实践后，内容入口会出现在这里"
          signals={practiced}
          loading={loading}
          onLifecycleChange={onLifecycleChange}
          onGenerateContent={onGenerateContent}
        />

        <PendingQueue signals={pending.slice(0, 12)} />
        <SourceSettings
          sources={sources}
          onSourceChange={onSourceChange}
          onRedditCommunitiesChange={onRedditCommunitiesChange}
        />
      </div>
    </main>
  )
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-xl border border-border bg-surface px-4 py-3">
      <div className="text-2xl font-semibold text-white">{value}</div>
      <div className="mt-1 text-xs text-text-muted">{label}</div>
    </div>
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

function WorkflowSection({
  title, description, signals, empty, loading, onLifecycleChange, onGenerateContent,
}: {
  title: string
  description: string
  signals: RadarSignal[]
  empty: string
  loading: boolean
  onLifecycleChange: (signalId: number, lifecycleState: string) => void
  onGenerateContent: (signalId: number) => void
}) {
  return (
    <section>
      <div className="mb-4 flex items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold">{title}</h2>
          <p className="mt-1 text-sm text-text-muted">{description}</p>
        </div>
        <span className="shrink-0 text-xs text-text-muted">{signals.length} 条</span>
      </div>
      {signals.length === 0 ? (
        <p className="rounded-xl border border-dashed border-border bg-surface/40 p-8 text-center text-sm text-text-muted">{empty}</p>
      ) : (
        <div className="space-y-4">
          {signals.map((signal) => (
            <SignalCard
              key={signal.id}
              signal={signal}
              loading={loading}
              onLifecycleChange={onLifecycleChange}
              onGenerateContent={onGenerateContent}
            />
          ))}
        </div>
      )}
    </section>
  )
}

function SignalCard({
  signal, loading, onLifecycleChange, onGenerateContent,
}: {
  signal: RadarSignal
  loading: boolean
  onLifecycleChange: (signalId: number, lifecycleState: string) => void
  onGenerateContent: (signalId: number) => void
}) {
  const analysis = signal.analysis
  const queued = signal.lifecycleState === 'queued'
  const practiced = signal.lifecycleState === 'practiced'
  return (
    <article className={`rounded-2xl border bg-surface p-5 sm:p-6 ${queued ? 'border-amber-300/40' : practiced ? 'border-emerald-400/35' : 'border-border'}`}>
      <div className="flex flex-wrap items-center gap-2 text-xs text-text-muted">
        <span className="rounded-full bg-white/5 px-2.5 py-1 text-text-secondary">{sourceLabel(signal.source)}</span>
        <span>{evidenceLabel(signal.evidence?.evidenceClass ?? '')}</span>
        <span>· {signalDateLabel(signal)}</span>
        {analysis?.alertEligible && <span className="rounded-full bg-amber-400/10 px-2.5 py-1 text-amber-300">重要变化</span>}
        {(queued || practiced) && (
          <span className={`ml-auto rounded-full px-2.5 py-1 ${practiced ? 'bg-emerald-400/10 text-emerald-300' : 'bg-amber-400/10 text-amber-300'}`}>
            {practiced ? '已实践' : '实践中'}
          </span>
        )}
      </div>

      <h3 className="mt-4 text-xl font-semibold leading-8 text-white">{signal.title}</h3>
      {analysis?.whatChanged && <p className="mt-2 text-sm leading-6 text-text-secondary">{analysis.whatChanged}</p>}

      <div className="mt-5 grid gap-3 md:grid-cols-2">
        {analysis?.painPoint && (
          <div className="rounded-xl bg-base-bg/70 p-4">
            <div className="text-xs font-medium text-rose-300">解决什么问题</div>
            <p className="mt-2 text-sm leading-6 text-text-secondary">{analysis.painPoint}</p>
          </div>
        )}
        {analysis?.practicalUse && (
          <div className="rounded-xl bg-base-bg/70 p-4">
            <div className="text-xs font-medium text-cyan-300">能怎么用</div>
            <p className="mt-2 text-sm leading-6 text-text-secondary">{analysis.practicalUse}</p>
          </div>
        )}
      </div>

      {analysis?.action && (
        <div className="mt-4 rounded-xl border border-amber-300/20 bg-amber-300/5 p-4">
          <div className="flex items-center gap-2 text-sm font-medium text-amber-200">
            <ListChecks className="h-4 w-4" aria-hidden="true" /> 实践计划
          </div>
          <p className="mt-2 text-sm leading-6 text-text-primary">{analysis.action}</p>
          {(analysis.prerequisites || analysis.installation || analysis.compatibleClients?.length) && (
            <details className="mt-3 text-sm text-text-muted">
              <summary className="cursor-pointer select-none hover:text-text-secondary">查看准备与安装方式</summary>
              {analysis.prerequisites && <p className="mt-2">准备：{analysis.prerequisites}</p>}
              {analysis.installation && <p className="mt-1 break-words text-cyan-200">安装：{analysis.installation}</p>}
              {analysis.compatibleClients && analysis.compatibleClients.length > 0 && <p className="mt-1">兼容：{analysis.compatibleClients.join('、')}</p>}
            </details>
          )}
        </div>
      )}

      {analysis?.contentOpportunity && (
        <p className="mt-4 text-sm leading-6 text-purple-300"><span className="font-medium">可做选题：</span>{analysis.contentOpportunity}</p>
      )}

      <div className="mt-5 flex flex-wrap items-center gap-2 border-t border-border pt-4">
        <a className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm text-text-secondary hover:border-accent hover:text-accent" href={signal.originalUrl} target="_blank" rel="noreferrer">
          打开原始来源 <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
        </a>
        {!queued && !practiced && (
          <button className="rounded-lg bg-amber-300 px-3 py-2 text-sm font-medium text-slate-950 disabled:opacity-50" type="button" disabled={loading} onClick={() => onLifecycleChange(signal.id, 'queued')}>
            加入实践
          </button>
        )}
        {queued && (
          <button className="inline-flex items-center gap-1.5 rounded-lg bg-emerald-500 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" type="button" disabled={loading} onClick={() => onLifecycleChange(signal.id, 'practiced')}>
            <CheckCircle2 className="h-4 w-4" aria-hidden="true" /> 标记已实践
          </button>
        )}
        {practiced && (
          <button className="rounded-lg bg-purple-500 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" type="button" disabled={loading} onClick={() => onGenerateContent(signal.id)}>
            生成三平台内容包
          </button>
        )}
        {!practiced && (
          <button className="ml-auto inline-flex items-center gap-1 px-2 py-2 text-xs text-text-muted hover:text-text-secondary" type="button" disabled={loading} onClick={() => onLifecycleChange(signal.id, 'dismissed')}>
            <X className="h-3.5 w-3.5" aria-hidden="true" /> 暂不关注
          </button>
        )}
      </div>
    </article>
  )
}

function PendingQueue({ signals }: { signals: RadarSignal[] }) {
  if (signals.length === 0) return null
  return (
    <details className="rounded-xl border border-border bg-surface/60">
      <summary className="cursor-pointer select-none px-5 py-4 text-sm font-medium">
        采集队列 <span className="ml-2 text-xs font-normal text-text-muted">{signals.length} 条等待 AI 筛选</span>
      </summary>
      <div className="border-t border-border px-5 py-2">
        {signals.map((signal) => (
          <div className="flex items-center gap-3 border-b border-border/60 py-3 last:border-0" key={signal.id}>
            <span className="w-24 shrink-0 text-xs text-text-muted">{sourceLabel(signal.source)}</span>
            <a className="min-w-0 flex-1 truncate text-sm text-text-secondary hover:text-accent" href={signal.originalUrl} target="_blank" rel="noreferrer">{signal.title}</a>
            <span className="shrink-0 text-xs text-text-muted">待分析</span>
          </div>
        ))}
      </div>
    </details>
  )
}

function SourceSettings({
  sources, onSourceChange, onRedditCommunitiesChange,
}: {
  sources: SourceConfig[]
  onSourceChange: (source: string, enabled: boolean) => void
  onRedditCommunitiesChange: (communities: string[]) => void
}) {
  const redditCommunities = sources.find((source) => source.source === 'reddit')?.settings.communities
  return (
    <details className="rounded-xl border border-border bg-surface/40">
      <summary className="cursor-pointer select-none px-5 py-4 text-sm font-medium">来源设置与采集状态</summary>
      <div className="grid gap-5 border-t border-border p-5 md:grid-cols-2">
        <div className="space-y-3">
          {sources.map((source) => (
            <label key={source.source} className="flex cursor-pointer items-center justify-between gap-3 text-sm">
              <span>
                <span className="block">{sourceLabel(source.source)}</span>
                <span className="block text-xs text-text-muted">{sourceStatus(source)}</span>
              </span>
              <input type="checkbox" checked={source.enabled} onChange={(event) => onSourceChange(source.source, event.target.checked)} aria-label={`${sourceLabel(source.source)}采集开关`} />
            </label>
          ))}
        </div>
        <div>
          {redditCommunities && <RedditAllowlist key={redditCommunities.join(',')} communities={redditCommunities} onSave={onRedditCommunitiesChange} />}
          <p className="mt-4 text-xs text-text-muted">X 关键词采集暂未启用，不再用低质量目录站填充页面。</p>
        </div>
      </div>
    </details>
  )
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

function ContentPackageEditor({
  content,
  loading,
  onSave,
  onApprove,
}: {
  content: ContentPackage
  loading: boolean
  onSave: (content: ContentPackage) => void
  onApprove: (content: ContentPackage) => void
}) {
  const [draft, setDraft] = useState(content)
  const approved = draft.status === 'approved'
  const updatePlatform = (platform: 'xiaohongshu' | 'wechat', field: 'title' | 'body', value: string) => {
    setDraft((current) => ({ ...current, [platform]: { ...current[platform], [field]: value } }))
  }
  const updateX = (field: 'chinese' | 'english', value: string) => {
    setDraft((current) => ({ ...current, x: { ...current.x, [field]: value } }))
  }

  return (
    <section className="rounded-xl border border-purple-400/30 bg-surface p-5">
      <div className="flex flex-wrap items-center gap-3">
        <div>
          <h2 className="text-lg font-semibold">三平台内容工作台</h2>
          <p className="text-xs text-text-muted">证据说明：{draft.strategy.evidenceNote}</p>
        </div>
        <span className="ml-auto rounded bg-purple-400/10 px-2 py-1 text-xs text-purple-300">
          {approved ? '已确认可发布' : '草稿待确认'}
        </span>
      </div>

      <label className="mt-4 block text-xs text-text-muted">
        内容角度
        <input
          className="mt-1 w-full rounded-lg border border-border bg-base-bg p-2 text-sm text-text-primary"
          value={draft.strategy.angle}
          disabled={approved}
          onChange={(event) => setDraft((current) => ({ ...current, strategy: { ...current.strategy, angle: event.target.value } }))}
        />
      </label>

      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <DraftEditor label="小红书" title={draft.xiaohongshu.title} body={draft.xiaohongshu.body} disabled={approved} onTitle={(value) => updatePlatform('xiaohongshu', 'title', value)} onBody={(value) => updatePlatform('xiaohongshu', 'body', value)} />
        <DraftEditor label="公众号" title={draft.wechat.title} body={draft.wechat.body} disabled={approved} onTitle={(value) => updatePlatform('wechat', 'title', value)} onBody={(value) => updatePlatform('wechat', 'body', value)} />
        <DraftEditor label="X 中文" body={draft.x.chinese} disabled={approved} onBody={(value) => updateX('chinese', value)} />
        <DraftEditor label="X English" body={draft.x.english} disabled={approved} onBody={(value) => updateX('english', value)} />
      </div>

      <div className="mt-4 space-y-3">
        <h3 className="text-sm font-medium">视觉计划与图片提示词</h3>
        {draft.visualPlan.map((asset, index) => (
          <label className="block text-xs text-text-muted" key={`${asset.purpose}-${index}`}>
            {asset.purpose} · {asset.aspectRatio}
            <textarea
              className="mt-1 min-h-20 w-full rounded-lg border border-border bg-base-bg p-2 text-sm text-text-primary"
              value={asset.prompt}
              disabled={approved}
              onChange={(event) => setDraft((current) => ({
                ...current,
                visualPlan: current.visualPlan.map((item, itemIndex) => itemIndex === index ? { ...item, prompt: event.target.value } : item),
              }))}
            />
          </label>
        ))}
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        {!approved && (
          <>
            <button className="radar-button" type="button" disabled={loading} onClick={() => onSave(draft)}>保存修改</button>
            <button className="rounded-lg bg-purple-500 px-3 py-2 text-sm text-white disabled:opacity-50" type="button" disabled={loading} onClick={() => onApprove(draft)}>确认可发布</button>
          </>
        )}
        {draft.x.sourceLinks.map((link) => (
          <a className="inline-flex items-center gap-1 px-2 py-2 text-xs text-accent" href={link} target="_blank" rel="noreferrer" key={link}>
            证据来源 <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
        ))}
      </div>
    </section>
  )
}

function DraftEditor({
  label,
  title,
  body,
  disabled,
  onTitle,
  onBody,
}: {
  label: string
  title?: string
  body: string
  disabled: boolean
  onTitle?: (value: string) => void
  onBody: (value: string) => void
}) {
  return (
    <div className="rounded-lg border border-border p-3">
      <h3 className="text-sm font-medium">{label}</h3>
      {title !== undefined && onTitle && (
        <input className="mt-2 w-full rounded border border-border bg-base-bg p-2 text-sm" value={title} disabled={disabled} onChange={(event) => onTitle(event.target.value)} aria-label={`${label}标题`} />
      )}
      <textarea className="mt-2 min-h-40 w-full rounded border border-border bg-base-bg p-2 text-sm" value={body} disabled={disabled} onChange={(event) => onBody(event.target.value)} aria-label={`${label}正文`} />
    </div>
  )
}

function sourceLabel(source: string) {
  return ({ dev: 'DEV Community', github: 'GitHub', reddit: 'Reddit', bluesky: 'Bluesky' } as Record<string, string>)[source] ?? source
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
