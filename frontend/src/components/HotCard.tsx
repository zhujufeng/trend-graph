// HotCard.tsx
//
// 热点卡片组件，列表里的每一项就是一张卡。
//
// 这是一种"容器组件"：组合多个展示组件（SourceBadge / RelevanceBar）+
// 调用一组 props，它本身也不调 API，纯展示用。
//
// Tailwind 类名解释：
//   bg-surface: 用我们 index.css 里定义的主题色
//   rounded-xl: 圆角
//   glow-border: 自定义带光晕的边框（hover 增强）
//   transition: 动画过渡

import { ExternalLink, Flame, ShieldCheck, ShieldAlert, Clock } from 'lucide-react'
import type { HotItem } from '../types'
import { SourceBadge } from './SourceBadge'
import { RelevanceBar } from './RelevanceBar'

interface HotCardProps {
  item: HotItem
  onAnalyze?: (item: HotItem) => void
  analyzing?: boolean
}

// 时间格式化工具：把 unix 秒转成相对时间（如"3小时前"）
function timeAgo(unixSec: number): string {
  const diff = Date.now() / 1000 - unixSec
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`
  return `${Math.floor(diff / 86400)}天前`
}

// 解析 entities 字段（后端存的是 JSON 字符串）
function parseEntities(item: HotItem): string[] {
  if (!item.entities) return []
  try {
    const arr = JSON.parse(item.entities) as unknown
    if (Array.isArray(arr)) return arr as string[]
  } catch {
    return []
  }
  return []
}

export function HotCard({ item, onAnalyze, analyzing }: HotCardProps) {
  const entities = parseEntities(item)
  const hasAnalysis = item.relevance !== undefined && item.summary !== ''
  const url = item.url && item.url !== '' ? item.url : '#'
  // 是否已在分析中
  const isLoading = analyzing === true

  return (
    <article className="bg-surface rounded-xl glow-border p-5 transition-all duration-200 flex flex-col">
      {/* 头部：来源 + 热度 + 时间 */}
      <header className="flex items-center gap-3 mb-3">
        <SourceBadge source={item.source} />
        {item.hot > 0 && (
          <span className="flex items-center gap-1 text-xs text-text-muted">
            <Flame className="w-3 h-3 text-orange-500" />
            <span className="tabular-nums">{item.hot}</span>
          </span>
        )}
        <span className="flex items-center gap-1 text-xs text-text-muted ml-auto">
          <Clock className="w-3 h-3" />
          <span>{timeAgo(item.publishedAt)}</span>
        </span>
      </header>

      {/* 标题 + 外链图标 */}
      <h3 className="text-base font-medium text-text-primary mb-2 leading-relaxed">
        {url !== '#' ? (
          <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-accent transition flex items-start gap-1"
          >
            <span>{item.title}</span>
            <ExternalLink className="w-3.5 h-3.5 mt-1 text-text-muted shrink-0" />
          </a>
        ) : (
          <span>{item.title}</span>
        )}
      </h3>

      {/* AI 摘要 */}
      {hasAnalysis && (
        <p className="text-sm text-text-secondary mb-3 leading-relaxed">
          {item.summary}
        </p>
      )}

      {/* 实体（如果 AI 提取了） */}
      {entities.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-3">
          {entities.map((e, i) => (
            <span
              key={i}
              className="text-xs px-1.5 py-0.5 rounded bg-accent/10 text-accent border border-accent/20"
            >
              {e}
            </span>
          ))}
        </div>
      )}

      {/* AI 分析相关数据 */}
      {hasAnalysis && (
        <>
          <div className="flex items-center gap-3 mb-2">
            {/* 真假标签 */}
            {item.isAuthentic !== undefined && (
              <span
                className={`flex items-center gap-1 text-xs ${
                  item.isAuthentic ? 'text-emerald-400' : 'text-orange-400'
                }`}
                title={item.isAuthentic ? 'AI 判断可信' : 'AI 疑似夸大/谣言'}
              >
                {item.isAuthentic ? (
                  <ShieldCheck className="w-3.5 h-3.5" />
                ) : (
                  <ShieldAlert className="w-3.5 h-3.5" />
                )}
                {item.isAuthentic ? '可信' : '存疑'}
              </span>
            )}
            {/* 相关性条 */}
            <div className="flex-1 min-w-0">
              <RelevanceBar value={item.relevance ?? 0} />
            </div>
          </div>
        </>
      )}

      {/* 操作区 */}
      {onAnalyze && (
        <div className="mt-auto pt-3">
          <button
            onClick={() => onAnalyze(item)}
            disabled={isLoading || hasAnalysis}
            className="text-xs px-3 py-1.5 rounded-md border border-accent/30 text-accent hover:bg-accent/10 transition disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {isLoading ? 'AI 分析中…' : hasAnalysis ? '已分析 ✓' : 'AI 分析'}
          </button>
        </div>
      )}
    </article>
  )
}