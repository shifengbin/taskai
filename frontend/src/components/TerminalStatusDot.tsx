import {type CSSProperties} from 'react'

import {realtimeStatusLabel, type RealtimeStatus} from '../types'

interface TerminalStatusDotProps {
  status: RealtimeStatus
}

// 快门波普状态点配色（覆盖 design-preview 的绿/紫）：
// 空闲=墨灰、工作中=钴蓝、未读=琥珀、异常=红。颜色取自令牌，亮暗自适应。
const statusColor: Record<RealtimeStatus, string> = {
  idle: 'var(--snap-muted)',
  working: 'var(--snap-cobalt)',
  unread: 'var(--snap-amber)',
  error: 'var(--snap-error)',
}

export function TerminalStatusDot({status}: TerminalStatusDotProps) {
  const hasPulse = status === 'working' || status === 'unread'
  const color = statusColor[status]

  return (
    <span
      role="status"
      aria-label={`终端状态：${realtimeStatusLabel[status]}`}
      data-status={status}
      data-pulse={hasPulse ? 'active' : undefined}
      className="relative inline-block h-2 w-2 shrink-0 rounded-full"
      style={{backgroundColor: color, ...({'--snap-dot': color} as CSSProperties)}}
    >
      {hasPulse && (
        <span
          aria-hidden
          className="snap-status-pulse-ring pointer-events-none absolute -inset-1 rounded-full border"
          style={{borderColor: color}}
        />
      )}
    </span>
  )
}
