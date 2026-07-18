import type { ComponentProps } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { RadarDashboard } from './RadarDashboard'
import type { RadarSignal, SourceConfig } from '../types'

type Props = ComponentProps<typeof RadarDashboard>

function baseProps(overrides: Partial<Props> = {}): Props {
  return {
    signals: [], sources: [], topics: [], loading: false, error: '', contentPackage: null,
    onRefresh: vi.fn(), onLogout: vi.fn(), onSourceChange: vi.fn(),
    onRedditCommunitiesChange: vi.fn(), onGitHubRepositoriesChange: vi.fn(), onRSSFeedsChange: vi.fn(),
    onCreateTopic: vi.fn(), onToggleTopic: vi.fn(), onDeleteTopic: vi.fn(),
    onLifecycleChange: vi.fn(), onGenerateContent: vi.fn(), onSaveContent: vi.fn(), onApproveContent: vi.fn(),
    ...overrides,
  }
}

describe('RadarDashboard', () => {
  it('renders a topic-driven personal inbox with source subscriptions', () => {
    const signal: RadarSignal = {
      id: 7, source: 'github', title: 'MCP Inspector', originalUrl: 'https://github.com/owner/repo/releases/tag/v2',
      score: 42, rankScore: 68, qualification: 'qualified', lifecycleState: 'inbox', createdAt: '2026-07-18T08:00:00Z',
      evidence: { id: 1, signalId: 7, sourceUrl: 'https://github.com/owner/repo/releases/tag/v2', evidenceClass: 'original_documentation', capturedAt: '2026-07-18T08:00:00Z' },
      analysis: { matchedTopics: ['AI', '开发工具'], valueScore: 4, whatChanged: '新增本地检查流程', practicalUse: '更快定位协议错误', action: '阅读发布说明并在测试项目中运行' },
    }
    const pending: RadarSignal = {
      id: 8, source: 'rss', title: '机器人产业周报', originalUrl: 'https://example.com/robot', score: 1, rankScore: 30,
      qualification: 'pending', lifecycleState: 'inbox', createdAt: '2026-07-18T08:00:00Z',
    }
    const sources: SourceConfig[] = [
      { source: 'github', enabled: true, settings: { repositories: ['owner/repo'] }, updatedAt: '2026-07-18T08:00:00Z' },
      { source: 'rss', enabled: true, settings: { feeds: ['https://example.com/feed.xml'] }, updatedAt: '2026-07-18T08:00:00Z' },
    ]
    const html = renderToStaticMarkup(<RadarDashboard {...baseProps({ signals: [signal, pending], sources, topics: [{ id: 1, word: 'AI', note: '', active: true, intervalMin: 180, createdAt: '2026-07-18T08:00:00Z' }] })} />)

    expect(html).toContain('个人信息雷达')
    expect(html).toContain('信息收件箱')
    expect(html).toContain('MCP Inspector')
    expect(html).toContain('价值 4/5')
    expect(html).toContain('更快定位协议错误')
    expect(html).toContain('保存')
    expect(html).toContain('已处理')
    expect(html).toContain('采集队列')
    expect(html).toContain('GitHub Release 订阅')
    expect(html).toContain('owner/repo')
    expect(html).toContain('RSS / Atom 订阅')
    expect(html).toContain('https://example.com/feed.xml')
  })

  it('keeps rejected and dismissed information out of the inbox', () => {
    const hidden: RadarSignal[] = [
      { id: 1, source: 'dev', title: '不相关信息', originalUrl: 'https://example.com/1', score: 1, rankScore: 10, qualification: 'rejected', lifecycleState: 'inbox', createdAt: '2026-07-18T08:00:00Z' },
      { id: 2, source: 'dev', title: '已忽略信息', originalUrl: 'https://example.com/2', score: 1, rankScore: 10, qualification: 'qualified', lifecycleState: 'dismissed', createdAt: '2026-07-18T08:00:00Z' },
    ]
    const html = renderToStaticMarkup(<RadarDashboard {...baseProps({ signals: hidden })} />)
    expect(html).not.toContain('不相关信息')
    expect(html).not.toContain('已忽略信息')
    expect(html).toContain('这个分组暂时是空的')
  })

  it('keeps content generation as an optional editable workspace', () => {
    const html = renderToStaticMarkup(<RadarDashboard {...baseProps({ contentPackage: {
      id: 11, signalId: 7, evidenceSnapshotId: 8, status: 'draft',
      strategy: { angle: '三步复现', audience: '开发者', evidenceNote: '官方文档' },
      xiaohongshu: { title: '标题', body: '小红书正文', tags: [], sourceLinks: [] },
      wechat: { title: '标题', body: '公众号正文', tags: [], sourceLinks: [] },
      x: { chinese: '中文稿', english: 'English draft', sourceLinks: [] },
      visualPlan: [], createdAt: '2026-07-18T08:00:00Z', updatedAt: '2026-07-18T08:00:00Z',
    } })} />)
    expect(html).toContain('内容草稿工作台（可选）')
    expect(html).toContain('小红书正文')
    expect(html).toContain('English draft')
    expect(html).toContain('确认内容')
  })
})
