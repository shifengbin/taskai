import {type CSSProperties} from 'react'

import {realtimeStatusLabel, type RealtimeStatus} from '../types'

interface TerminalStatusDotProps {
  status: RealtimeStatus
}

// Nebula 状态点配色：空闲=蓝灰、工作中=青绿、未读=紫色、异常=红。
const statusColor: Record<RealtimeStatus, string> = {
  idle: 'var(--snap-muted)',
  working: 'var(--snap-cobalt)',
  unread: 'var(--snap-violet)',
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
