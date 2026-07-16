import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { RadarDashboard } from './RadarDashboard'
import type { RadarSignal, SourceConfig } from '../types'

describe('RadarDashboard', () => {
  it('renders the personal AI radar outcomes with evidence links', () => {
    const signals: RadarSignal[] = [
      {
        id: 7,
        source: 'github',
        title: 'MCP Inspector',
        originalUrl: 'https://github.com/owner/repo',
        score: 42,
        qualification: 'qualified',
        lifecycleState: 'new',
        createdAt: '2026-07-15T08:00:00Z',
        evidence: {
          id: 1,
          signalId: 7,
          sourceUrl: 'https://github.com/owner/repo/blob/main/SKILL.md',
          evidenceClass: 'original_documentation',
          excerpt: 'Install and run the inspector.',
          contentHash: 'hash',
          capturedAt: '2026-07-15T08:00:00Z',
          createdAt: '2026-07-15T08:00:00Z',
        },
        analysis: {
          whatChanged: '新增 MCP 检查流程',
          action: '用测试服务器复现',
          contentOpportunity: '做一期 MCP 排错清单',
          installation: '按 SKILL.md 配置 MCP 服务器',
          compatibleClients: ['Codex', 'Claude Code'],
        },
      },
    ]
    const sources: SourceConfig[] = [
      { source: 'github', enabled: true, settings: {}, updatedAt: '2026-07-15T08:00:00Z' },
      {
        source: 'reddit',
        enabled: true,
        settings: { communities: ['r/claudeai', 'r/cursor'] },
        updatedAt: '2026-07-15T08:00:00Z',
      },
    ]

    const html = renderToStaticMarkup(
      <RadarDashboard
        signals={signals}
        sources={sources}
        loading={false}
        error=""
        onRefresh={vi.fn()}
        onLogout={vi.fn()}
        onSourceChange={vi.fn()}
        onRedditCommunitiesChange={vi.fn()}
        onLifecycleChange={vi.fn()}
        contentPackage={null}
        onGenerateContent={vi.fn()}
        onSaveContent={vi.fn()}
        onApproveContent={vi.fn()}
      />,
    )

    expect(html).toContain('AI 实践雷达')
    expect(html).toContain('待实践')
    expect(html).toContain('MCP Inspector')
    expect(html).toContain('https://github.com/owner/repo')
    expect(html).toContain('用测试服务器复现')
    expect(html).toContain('按 SKILL.md 配置 MCP 服务器')
    expect(html).toContain('Codex、Claude Code')
    expect(html).toContain('Reddit 社区白名单')
    expect(html).toContain('r/claudeai, r/cursor')
    expect(html).toContain('保存白名单')
    expect(html).toContain('加入实践')
    expect(html).not.toContain('生成三平台内容包')
  })

  it('keeps rejected signals out of outcome sections', () => {
    const rejected: RadarSignal = {
      id: 8,
      source: 'bluesky',
      title: '仅目录收录、尚无 GitHub 证据',
      originalUrl: 'https://bsky.app/profile/example/post/1',
      score: 10,
      qualification: 'rejected',
      qualificationReason: 'github_verification_required',
      lifecycleState: 'new',
      createdAt: '2026-07-15T08:00:00Z',
    }

    const html = renderToStaticMarkup(
      <RadarDashboard
        signals={[rejected]}
        sources={[]}
        loading={false}
        error=""
        onRefresh={vi.fn()}
        onLogout={vi.fn()}
        onSourceChange={vi.fn()}
        onRedditCommunitiesChange={vi.fn()}
        onLifecycleChange={vi.fn()}
        contentPackage={null}
        onGenerateContent={vi.fn()}
        onSaveContent={vi.fn()}
        onApproveContent={vi.fn()}
      />,
    )

    expect(html).not.toContain(rejected.title)
    expect(html).toContain('当前没有值得实践的新信号')
  })

  it('shows newly collected signals while they wait for analysis', () => {
    const pending: RadarSignal = {
      id: 9,
      source: 'dev',
      title: '刚采集到的 AI 工作流',
      originalUrl: 'https://dev.to/example/ai-workflow',
      score: 0,
      qualification: 'pending',
      lifecycleState: 'new',
      createdAt: '2026-07-15T08:00:00Z',
    }

    const html = renderToStaticMarkup(
      <RadarDashboard
        signals={[pending]}
        sources={[]}
        loading={false}
        error=""
        onRefresh={vi.fn()}
        onLogout={vi.fn()}
        onSourceChange={vi.fn()}
        onRedditCommunitiesChange={vi.fn()}
        onLifecycleChange={vi.fn()}
        contentPackage={null}
        onGenerateContent={vi.fn()}
        onSaveContent={vi.fn()}
        onApproveContent={vi.fn()}
      />,
    )

    expect(html).toContain('采集队列')
    expect(html).toContain(pending.title)
    expect(html).toContain(pending.originalUrl)
    expect(html).toContain('待分析')
  })

  it('does not render raw pending evidence and only offers content generation after practice', () => {
    const rawEvidence = 'RAW-README-SHOULD-NOT-RENDER'
    const pending: RadarSignal = {
      id: 9,
      source: 'github',
      title: '待分析项目',
      originalUrl: 'https://github.com/example/pending',
      score: 1,
      qualification: 'pending',
      lifecycleState: 'new',
      createdAt: '2026-07-15T08:00:00Z',
      evidence: { id: 9, signalId: 9, sourceUrl: 'https://github.com/example/pending', evidenceClass: 'original_documentation', excerpt: rawEvidence, contentHash: 'hash', capturedAt: '2026-07-15T08:00:00Z', createdAt: '2026-07-15T08:00:00Z' },
    }
    const practiced: RadarSignal = {
      ...pending,
      id: 10,
      title: '已实践项目',
      qualification: 'qualified',
      lifecycleState: 'practiced',
      analysis: { whatChanged: '新增流程', action: '本地运行' },
    }
    const html = renderToStaticMarkup(
      <RadarDashboard
        signals={[pending, practiced]}
        sources={[]}
        loading={false}
        error=""
        onRefresh={vi.fn()}
        onLogout={vi.fn()}
        onSourceChange={vi.fn()}
        onRedditCommunitiesChange={vi.fn()}
        onLifecycleChange={vi.fn()}
        contentPackage={null}
        onGenerateContent={vi.fn()}
        onSaveContent={vi.fn()}
        onApproveContent={vi.fn()}
      />,
    )
    expect(html).not.toContain(rawEvidence)
    expect(html).toContain('已实践，可创作')
    expect((html.match(/生成三平台内容包/g) ?? []).length).toBe(1)
  })

  it('renders editable three-platform drafts with evidence links', () => {
    const html = renderToStaticMarkup(
      <RadarDashboard
        signals={[]}
        sources={[]}
        loading={false}
        error=""
        onRefresh={vi.fn()}
        onLogout={vi.fn()}
        onSourceChange={vi.fn()}
        onRedditCommunitiesChange={vi.fn()}
        onLifecycleChange={vi.fn()}
        onGenerateContent={vi.fn()}
        onSaveContent={vi.fn()}
        onApproveContent={vi.fn()}
        contentPackage={{
          id: 11,
          signalId: 7,
          evidenceSnapshotId: 8,
          status: 'draft',
          strategy: { angle: '三步复现', audience: 'AI 用户', evidenceNote: '官方文档' },
          xiaohongshu: { title: '小红书标题', body: '小红书正文', tags: ['AI'], sourceLinks: ['https://example.com/docs'] },
          wechat: { title: '公众号标题', body: '公众号正文', tags: ['AI'], sourceLinks: ['https://example.com/docs'] },
          x: { chinese: 'X 中文稿', english: 'X English draft', sourceLinks: ['https://example.com/docs'] },
          visualPlan: [{ purpose: '封面', aspectRatio: '3:4', prompt: '中文信息图提示词' }],
          createdAt: '2026-07-15T08:00:00Z',
          updatedAt: '2026-07-15T08:00:00Z',
        }}
      />,
    )

    expect(html).toContain('三平台内容工作台')
    expect(html).toContain('小红书正文')
    expect(html).toContain('X English draft')
    expect(html).toContain('中文信息图提示词')
    expect(html).toContain('https://example.com/docs')
    expect(html).toContain('确认可发布')
  })
})
