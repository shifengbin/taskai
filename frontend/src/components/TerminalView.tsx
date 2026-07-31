import {useEffect, useMemo, useRef} from 'react'
import {Box, IconButton, Tooltip, Typography, useTheme} from '@mui/material'
import CloseOutlinedIcon from '@mui/icons-material/CloseOutlined'
import TerminalOutlinedIcon from '@mui/icons-material/TerminalOutlined'
import {FitAddon} from '@xterm/addon-fit'
import {Terminal} from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'

import {terminalDisplayName, terminalRealtimeStatus, type TerminalRecord} from '../types'
import {ClipboardGetText, ClipboardSetText} from '../../wailsjs/runtime/runtime'
import {TerminalStatusDot} from './TerminalStatusDot'

interface TerminalViewProps {
  terminal: TerminalRecord
  onWrite(data: string): void
  onResize(columns: number, rows: number): void
  onClose(): void
}

export function TerminalView({terminal, onWrite, onResize, onClose}: TerminalViewProps) {
  const appTheme = useTheme()
  const containerRef = useRef<HTMLDivElement>(null)
  const terminalRef = useRef<Terminal>()
  const outputRef = useRef('')
  const terminalTheme = useMemo(() => terminalVisualTheme(appTheme.palette.mode), [appTheme.palette.mode])

  useEffect(() => {
    const instance = new Terminal({
      cursorBlink: terminal.state === 'active',
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      fontSize: 13,
      lineHeight: 1.35,
      theme: terminalTheme,
    })
    const fitAddon = new FitAddon()
    instance.loadAddon(fitAddon)
    terminalRef.current = instance
    outputRef.current = terminal.output ?? ''
    if (containerRef.current) {
      instance.open(containerRef.current)
      instance.write(outputRef.current)
    }
    const fit = () => {
      try {
        fitAddon.fit()
        if (instance.cols > 0 && instance.rows > 0) {
          onResize(instance.cols, instance.rows)
        }
      } catch {
        // The terminal container may be temporarily hidden during a pane switch.
      }
    }
    const observer = new ResizeObserver(fit)
    if (containerRef.current) {
      observer.observe(containerRef.current)
    }
    const onData = instance.onData((data) => onWrite(data))
    const onSelectionChange = instance.onSelectionChange(() => {
      const selection = instance.getSelection()
      if (selection) {
        void ClipboardSetText(selection).catch(() => {})
      }
    })
    const animationFrame = requestAnimationFrame(() => {
      fit()
      instance.focus()
    })

    return () => {
      cancelAnimationFrame(animationFrame)
      onData.dispose()
      onSelectionChange.dispose()
      observer.disconnect()
      instance.dispose()
      terminalRef.current = undefined
    }
  }, [terminal.id])

  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.options.theme = terminalTheme
    }
  }, [terminalTheme])

  useEffect(() => {
    const instance = terminalRef.current
    const output = terminal.output ?? ''
    if (!instance) {
      return
    }
    if (output.startsWith(outputRef.current)) {
      instance.write(output.slice(outputRef.current.length))
    } else {
      instance.reset()
      instance.write(output)
    }
    outputRef.current = output
  }, [terminal.output])

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
              onWrite(clipboard)
            }
          }).catch(() => {})
        }}
        sx={{minHeight: 0, overflow: 'hidden', bgcolor: terminalTheme.background, p: 1}}
      />
    </Box>
  )
}

function terminalVisualTheme(mode: 'light' | 'dark') {
  return mode === 'dark'
    ? {background: '#101a14', foreground: '#d8e8dc', cursor: '#9cc3ab', selectionBackground: '#2e4035'}
    : {background: '#26352e', foreground: '#e3eee5', cursor: '#9bd5ae', selectionBackground: '#3d5748'}
}
