// RelevanceBar.tsx
//
// 相关性进度条组件，把 0~1 的浮点数变成可视化条。
// 颜色随相关性变化：低=红，中=黄，高=绿。
//
// 这是前端典型的"展示组件"：只关心 props，不调 API，不带状态。
// 把它做得纯粹，方便复用和测试。

import type { ComponentPropsWithoutRef } from 'react'

interface RelevanceBarProps extends ComponentPropsWithoutRef<'div'> {
  value: number // 0~1
  showLabel?: boolean
}

// 根据 relevance 数值选颜色
function colorClass(v: number): string {
  if (v >= 0.7) return 'bg-emerald-500'
  if (v >= 0.4) return 'bg-yellow-500'
  if (v >= 0.1) return 'bg-orange-500'
  return 'bg-red-500/70'
}

export function RelevanceBar({ value, showLabel = true, className = '', ...rest }: RelevanceBarProps) {
  // 边界 clamp：value 可能越界
  const v = Math.max(0, Math.min(1, value))
  // 0 时显示一点最低高度
  const widthPct = Math.max(v * 100, v > 0 ? 4 : 0)

  return (
    <div className={`flex items-center gap-2 ${className}`} {...rest}>
      <div className="flex-1 h-1.5 bg-surface rounded overflow-hidden">
        <div
          className={`h-full ${colorClass(v)} transition-all duration-500 rounded`}
          style={{ width: `${widthPct}%` }}
        />
      </div>
      {showLabel && (
        <span className="text-xs text-text-muted font-mono tabular-nums w-12 text-right shrink-0">
          {v.toFixed(2)}
        </span>
      )}
    </div>
  )
}