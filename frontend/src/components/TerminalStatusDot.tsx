import {Box} from '@mui/material'

import {realtimeStatusLabel, type RealtimeStatus} from '../types'

interface TerminalStatusDotProps {
  status: RealtimeStatus
}

const statusColor: Record<RealtimeStatus, string> = {
  idle: 'grey.500',
  working: 'success.main',
  unread: '#8b5cf6',
  error: 'error.main',
}

export function TerminalStatusDot({status}: TerminalStatusDotProps) {
  const hasPulse = status === 'working' || status === 'unread'

  return (
    <Box
      role="status"
      aria-label={`终端状态：${realtimeStatusLabel[status]}`}
      data-status={status}
      data-pulse={hasPulse ? 'active' : undefined}
      sx={{
        width: 8,
        height: 8,
        flexShrink: 0,
        position: 'relative',
        borderRadius: '50%',
        bgcolor: statusColor[status],
        ...(hasPulse && {
          '&::before': {
            content: '""',
            position: 'absolute',
            inset: -4,
            pointerEvents: 'none',
            border: '1px solid',
            borderColor: status === 'working' ? 'success.main' : '#8b5cf6',
            borderRadius: 'inherit',
            animation: 'taskai-status-pulse 1.4s ease-out infinite',
          },
          '@keyframes taskai-status-pulse': {
            '0%': {opacity: 0.7, transform: 'scale(0.65)'},
            '80%, 100%': {opacity: 0, transform: 'scale(1.75)'},
          },
          '@media (prefers-reduced-motion: reduce)': {
            '&::before': {animation: 'none', opacity: 0},
          },
        }),
      }}
    />
  )
}
