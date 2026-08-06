import {useEffect, useMemo, useRef, type CSSProperties} from 'react'
import {Terminal as TerminalIcon, X} from 'lucide-react'
import '@xterm/xterm/css/xterm.css'

import {TerminalSessionRegistry, terminalVisualTheme} from '../terminal-session'
import {terminalDisplayName, terminalRealtimeStatus, type TerminalRecord} from '../types'
import {api} from '../api'
import {ClipboardGetText, OnFileDrop, OnFileDropOff} from '../../wailsjs/runtime/runtime'
import {TerminalStatusDot} from './TerminalStatusDot'
import {IconButton} from './ui'

interface TerminalViewProps {
  terminal: TerminalRecord
  sessionRegistry: TerminalSessionRegistry
  /** 当前色彩模式，用于注入 xterm 主题；默认亮色。 */
  mode?: 'light' | 'dark'
  onResize(columns: number, rows: number): void
  onClose(): void
  onError?(error: unknown): void
}

export function TerminalView({terminal, sessionRegistry, mode = 'light', onResize, onClose, onError}: TerminalViewProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const onResizeRef = useRef(onResize)
  const onErrorRef = useRef(onError)
  const terminalTheme = useMemo(() => terminalVisualTheme(mode), [mode])

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

  useEffect(() => {
    onErrorRef.current = onError
  }, [onError])

  useEffect(() => {
    let active = true
    try {
      OnFileDrop((_x, _y, paths) => {
        if (!active || paths.length === 0) {
          return
        }
        void api.writeTerminalFilePaths(terminal.taskId, terminal.id, paths).catch((error) => onErrorRef.current?.(error))
      }, true)
    } catch {
      // The browser development server has no Wails drag-and-drop runtime.
    }
    return () => {
      active = false
      try {
        OnFileDropOff()
      } catch {
        // The browser development server has no Wails drag-and-drop runtime.
      }
    }
  }, [terminal.id, terminal.taskId])

  return (
    <div className="taskai-terminal grid h-full min-w-0" style={{gridTemplateRows: '44px minmax(0, 1fr)'}}>
      <div className="taskai-terminal__header taskai-contextual-container flex items-center gap-2 border-b-2 border-snap-outline bg-snap-surface px-2" data-testid="terminal-view-header">
        <TerminalIcon className="h-4 w-4 shrink-0 text-snap-cobalt"/>
        <div data-testid="terminal-view-title-container" style={{flex: 1, minWidth: 0}}>
          <span
            data-testid="terminal-view-title"
            style={{whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'clip'}}
            className="block font-display text-sm font-bold text-snap-ink"
          >
            {terminalDisplayName(terminal)}
          </span>
        </div>
        <TerminalStatusDot status={terminalRealtimeStatus(terminal)}/>
        {terminal.state === 'active' && (
          <div className="taskai-contextual-actions flex shrink-0 items-center" data-testid="terminal-view-actions">
            <IconButton aria-label="关闭终端" title="关闭终端" className="h-7 w-7" onClick={onClose}>
              <X className="h-4 w-4"/>
            </IconButton>
          </div>
        )}
      </div>
      <div
        ref={containerRef}
        className="taskai-terminal__content relative min-h-0 overflow-hidden p-1"
        data-testid="terminal-content"
        onContextMenu={(event) => {
          event.preventDefault()
          void ClipboardGetText().then((clipboard) => {
            if (clipboard) {
              sessionRegistry.writeInput(terminal.taskId, terminal.id, clipboard)
            }
          }).catch(() => {})
        }}
        style={{backgroundColor: terminalTheme.background, '--wails-drop-target': 'drop'} as CSSProperties}
      >
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0"
          style={{
            backgroundImage: 'radial-gradient(var(--snap-dots) 1px, transparent 1px)',
            backgroundSize: '4px 4px',
          }}
        />
      </div>
    </div>
  )
}
