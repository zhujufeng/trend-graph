// SourceBadge.tsx
//
// 显示信息源的"徽章"组件，比如 "hn" "weibo"。
// 不同源用不同颜色，让用户一眼看出热点来源。
//
// ...props 类似 React 的 children/className，做 props 透传。

// 从 React 引入类型，方便继承 HTML 元素的 props
import type { ComponentPropsWithoutRef } from 'react'

// 源 → 颜色样式映射
// 用 const + as const 让 TS 把键推断为字面量类型（避免被推断成 string）
const SOURCE_STYLES = {
  hn: 'bg-orange-500/15 text-orange-400 border-orange-500/30',
  weibo: 'bg-red-500/15 text-red-400 border-red-500/30',
  bilibili: 'bg-pink-500/15 text-pink-400 border-pink-500/30',
  github: 'bg-gray-500/15 text-gray-300 border-gray-500/30',
  reddit: 'bg-orange-600/15 text-orange-500 border-orange-600/30',
  dev: 'bg-zinc-500/15 text-zinc-300 border-zinc-500/30',
  bluesky: 'bg-sky-500/15 text-sky-400 border-sky-500/30',
  bing: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
  twitter: 'bg-sky-500/15 text-sky-400 border-sky-500/30',
  zhihu: 'bg-blue-600/15 text-blue-500 border-blue-600/30',
} as const

// 兜底样式（不在映射里的源）
const DEFAULT_STYLE = 'bg-accent/15 text-accent border-accent/30'

interface SourceBadgeProps extends ComponentPropsWithoutRef<'span'> {
  source: string
}

export function SourceBadge({ source, className = '', ...rest }: SourceBadgeProps) {
  // 用方括号语法查样式，找不到用默认
  // KEY 是为了让 TS 不抱怨键类型
  const styleClass = (SOURCE_STYLES as Record<string, string>)[source] ?? DEFAULT_STYLE
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-mono border ${styleClass} ${className}`}
      {...rest}
    >
      {source}
    </span>
  )
}
