import {Box} from '@mui/material'

import {terminalStatusLabel, type TerminalState} from '../types'

interface TerminalStatusDotProps {
  state: TerminalState
}

export function TerminalStatusDot({state}: TerminalStatusDotProps) {
  return (
    <Box
      role="status"
      aria-label={`终端状态：${terminalStatusLabel[state]}`}
      sx={{width: 8, height: 8, flexShrink: 0, borderRadius: '50%', bgcolor: state === 'active' ? 'success.main' : 'error.main'}}
    />
  )
}
