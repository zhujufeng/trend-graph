import { useState, type FormEvent } from 'react'
import { ExternalLink, LogOut, RefreshCw, Sparkles } from 'lucide-react'

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
  contentPackage,
  onGenerateContent,
  onSaveContent,
  onApproveContent,
}: RadarDashboardProps) {
  const qualified = signals.filter((signal) => signal.qualification === 'qualified')
  const todayKey = shanghaiDate(new Date())
  const today = qualified.filter((signal) => shanghaiDate(new Date(signal.sourceUpdatedAt ?? signal.sourcePublishedAt ?? signal.createdAt)) === todayKey)
  const pending = signals.filter((signal) => signal.qualification === 'pending')
  const tools = qualified.filter((signal) => signal.source === 'github')
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
          {contentPackage && (
            <ContentPackageEditor
              key={`${contentPackage.id}-${contentPackage.updatedAt}`}
              content={contentPackage}
              loading={loading}
              onSave={onSaveContent}
              onApprove={onApproveContent}
            />
          )}
          {pending.length > 0 && <SignalSection title="最新采集（待分析）" signals={pending.slice(0, 12)} empty="" />}
          <SignalSection title="今日必读" signals={today.slice(0, 6)} empty="暂时没有新信号" loading={loading} onGenerateContent={onGenerateContent} />
          <SignalSection title="可用工具与 Skill" signals={tools} empty="等待 GitHub 的可用项目" loading={loading} onGenerateContent={onGenerateContent} />
          <SignalSection title="内容素材" signals={content} empty="信号分析后会在这里出现可用选题" loading={loading} onGenerateContent={onGenerateContent} />
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
                    {source.lastRun
                      ? source.lastRun.status === 'success'
                        ? `最近采集 ${source.lastRun.itemCount} 条 · ${source.lastRun.durationMs}ms`
                        : `异常：${source.lastRun.failureReason ?? source.lastFailure ?? '未知错误'}`
                      : source.lastSuccessAt ? '最近采集正常' : '等待首次采集'}
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

function SignalSection({
  title,
  signals,
  empty,
  loading = false,
  onGenerateContent,
}: {
  title: string
  signals: RadarSignal[]
  empty: string
  loading?: boolean
  onGenerateContent?: (signalId: number) => void
}) {
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
              {signal.analysis?.installation && <p className="mt-2 text-sm text-cyan-300">安装：{signal.analysis.installation}</p>}
              {signal.analysis?.compatibleClients && signal.analysis.compatibleClients.length > 0 && (
                <p className="mt-2 text-xs text-text-muted">兼容：{signal.analysis.compatibleClients.join('、')}</p>
              )}
              {signal.analysis?.contentOpportunity && (
                <p className="mt-2 text-sm text-purple-300">选题：{signal.analysis.contentOpportunity}</p>
              )}
              <a className="mt-4 inline-flex items-center gap-1 text-sm text-accent hover:text-accent-hover" href={signal.originalUrl} target="_blank" rel="noreferrer">
                查看原始来源 <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
              </a>
              {signal.qualification === 'qualified' && onGenerateContent && (
                <button
                  className="mt-3 block rounded-lg border border-purple-400/40 px-3 py-2 text-sm text-purple-300 hover:border-purple-300 disabled:opacity-50"
                  type="button"
                  disabled={loading}
                  onClick={() => onGenerateContent(signal.id)}
                >
                  生成三平台内容包
                </button>
              )}
            </article>
          ))}
        </div>
      )}
    </section>
  )
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

function shanghaiDate(value: Date) {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai' }).format(value)
}
