import {useEffect, useMemo, useRef} from 'react'
import {Box, IconButton, Tooltip, Typography, useTheme} from '@mui/material'
import CloseOutlinedIcon from '@mui/icons-material/CloseOutlined'
import TerminalOutlinedIcon from '@mui/icons-material/TerminalOutlined'
import '@xterm/xterm/css/xterm.css'

import {TerminalSessionRegistry, terminalVisualTheme} from '../terminal-session'
import {terminalDisplayName, terminalRealtimeStatus, type TerminalRecord} from '../types'
import {ClipboardGetText} from '../../wailsjs/runtime/runtime'
import {TerminalStatusDot} from './TerminalStatusDot'

interface TerminalViewProps {
  terminal: TerminalRecord
  sessionRegistry: TerminalSessionRegistry
  onResize(columns: number, rows: number): void
  onClose(): void
}

export function TerminalView({terminal, sessionRegistry, onResize, onClose}: TerminalViewProps) {
  const appTheme = useTheme()
  const containerRef = useRef<HTMLDivElement>(null)
  const onResizeRef = useRef(onResize)
  const terminalTheme = useMemo(() => terminalVisualTheme(appTheme.palette.mode), [appTheme.palette.mode])

  useEffect(() => {
    let active = true
    const fitAndRefresh = () => {
      if (active) {
        sessionRegistry.fitAndRefresh(terminal.taskId, terminal.id, onResizeRef.current)
      }
    }
    if (containerRef.current) {
      sessionRegistry.attach(terminal, containerRef.current, terminalTheme, onResizeRef.current)
    }
    const observer = new ResizeObserver(fitAndRefresh)
    if (containerRef.current) {
      observer.observe(containerRef.current)
    }
    const animationFrame = requestAnimationFrame(() => {
      fitAndRefresh()
      if (active) {
        sessionRegistry.focus(terminal.taskId, terminal.id)
      }
    })

    return () => {
      active = false
      cancelAnimationFrame(animationFrame)
      observer.disconnect()
    }
  }, [sessionRegistry, terminal.id, terminal.taskId, terminalTheme])

  useEffect(() => {
    onResizeRef.current = onResize
  }, [onResize])

  return (
    <Box className="taskai-terminal" sx={{height: '100%', minWidth: 0, display: 'grid', gridTemplateRows: '44px minmax(0, 1fr)'}}>
      <Box className="taskai-terminal__header" sx={{display: 'flex', alignItems: 'center', gap: 1, px: 1.5, borderBottom: 1, borderColor: 'divider'}}>
        <TerminalOutlinedIcon fontSize="small" color="primary"/>
        <Box data-testid="terminal-view-title-container" sx={{flex: 1, minWidth: 0}}>
          <Typography
            data-testid="terminal-view-title"
            variant="subtitle2"
            sx={{whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'clip'}}
          >
            {terminalDisplayName(terminal)}
          </Typography>
        </Box>
        <TerminalStatusDot status={terminalRealtimeStatus(terminal)}/>
        {terminal.state === 'active' && (
          <Tooltip title="关闭终端">
            <IconButton aria-label="关闭终端" size="small" onClick={onClose}>
              <CloseOutlinedIcon fontSize="small"/>
            </IconButton>
          </Tooltip>
        )}
      </Box>
      <Box
        ref={containerRef}
        className="taskai-terminal__content"
        data-testid="terminal-content"
        onContextMenu={(event) => {
          event.preventDefault()
          void ClipboardGetText().then((clipboard) => {
            if (clipboard) {
              sessionRegistry.writeInput(terminal.taskId, terminal.id, clipboard)
            }
          }).catch(() => {})
        }}
        sx={{minHeight: 0, overflow: 'hidden', bgcolor: terminalTheme.background, p: 1}}
      />
    </Box>
  )
}
