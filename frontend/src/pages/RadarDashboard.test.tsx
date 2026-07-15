import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { RadarDashboard } from './RadarDashboard'
import type { RadarSignal, SourceConfig } from '../types'

describe('RadarDashboard', () => {
  it('renders the personal AI radar outcomes with evidence links', () => {
    const signals: RadarSignal[] = [
      {
        id: 7,
        source: 'skillsmp',
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
        },
      },
    ]
    const sources: SourceConfig[] = [
      { source: 'skillsmp', enabled: true, settings: {}, updatedAt: '2026-07-15T08:00:00Z' },
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
      />,
    )

    expect(html).toContain('AI 信号雷达')
    expect(html).toContain('今日必读')
    expect(html).toContain('可用工具与 Skill')
    expect(html).toContain('内容素材')
    expect(html).toContain('MCP Inspector')
    expect(html).toContain('https://github.com/owner/repo')
    expect(html).toContain('用测试服务器复现')
    expect(html).toContain('Reddit 社区白名单')
    expect(html).toContain('r/claudeai, r/cursor')
    expect(html).toContain('保存白名单')
  })

  it('keeps rejected signals out of outcome sections', () => {
    const rejected: RadarSignal = {
      id: 8,
      source: 'skillsmp',
      title: '仅目录收录、尚无 GitHub 证据',
      originalUrl: 'https://skillsmp.com/example',
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
      />,
    )

    expect(html).not.toContain(rejected.title)
    expect(html).toContain('暂时没有新信号')
  })

  it('shows newly collected signals while they wait for analysis', () => {
    const pending: RadarSignal = {
      id: 9,
      source: 'waytoagi',
      title: '刚采集到的 AI 工作流',
      originalUrl: 'https://www.waytoagi.com/zh/article/example',
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
      />,
    )

    expect(html).toContain('最新采集')
    expect(html).toContain(pending.title)
    expect(html).toContain(pending.originalUrl)
    expect(html).toContain('待分析')
  })
})
