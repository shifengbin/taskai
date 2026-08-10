import {useCallback, useEffect, useMemo, useRef, useState, type CSSProperties} from 'react'
import {ClipboardPaste, Terminal as TerminalIcon} from 'lucide-react'
import '@xterm/xterm/css/xterm.css'

import {TerminalSessionRegistry, terminalVisualTheme} from '../terminal-session'
import {terminalDisplayName, type QuickInput, type TerminalRecord} from '../types'
import {api} from '../api'
import {defaultTerminalFontSize} from '../terminal-font-size'
import {type TerminalTheme} from '../terminal-theme'
import {ClipboardGetText, OnFileDrop, OnFileDropOff} from '../../wailsjs/runtime/runtime'
import {IconButton, Input, Popover, PopoverContent, PopoverTrigger} from './ui'

interface TerminalViewProps {
  terminal: TerminalRecord
  sessionRegistry: TerminalSessionRegistry
  quickInputs?: QuickInput[]
  fontSize?: number
  terminalTheme?: Partial<TerminalTheme>
  onResize(columns: number, rows: number): void
  onClose(): void
  onError?(error: unknown): void
}

export function TerminalView({terminal, sessionRegistry, quickInputs = [], fontSize = defaultTerminalFontSize, terminalTheme, onResize, onClose, onError}: TerminalViewProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const quickInputSearchRef = useRef<HTMLInputElement>(null)
  const onResizeRef = useRef(onResize)
  const onErrorRef = useRef(onError)
  const [quickInputSelectorOpen, setQuickInputSelectorOpen] = useState(false)
  const [quickInputSearch, setQuickInputSearch] = useState('')
  const [selectedQuickInputIndex, setSelectedQuickInputIndex] = useState(0)
  const resolvedTerminalTheme = useMemo(() => terminalVisualTheme(terminalTheme), [terminalTheme])
  const taskAIMouseClipboardEnabled = terminal.disableTaskAIMouseClipboard !== true
  const quickInputHotkey = typeof navigator !== 'undefined' && navigator.platform.toUpperCase().includes('MAC')
    ? 'Command+Shift+P'
    : 'Ctrl+Shift+P'
  const filteredQuickInputs = useMemo(() => {
    const search = quickInputSearch.trim().toLocaleLowerCase()
    if (!search) {
      return quickInputs
    }
    return quickInputs.filter((quickInput) => (
      quickInput.name.toLocaleLowerCase().includes(search)
      || quickInput.content.toLocaleLowerCase().includes(search)
    ))
  }, [quickInputSearch, quickInputs])

  const openQuickInputSelector = useCallback(() => {
    setQuickInputSearch('')
    setSelectedQuickInputIndex(0)
    setQuickInputSelectorOpen(true)
  }, [])

  const closeQuickInputSelector = () => {
    setQuickInputSelectorOpen(false)
    setQuickInputSearch('')
    sessionRegistry.focus(terminal.taskId, terminal.id)
  }

  const insertQuickInput = (quickInput: QuickInput) => {
    if (!sessionRegistry.pasteQuickInput(terminal.taskId, terminal.id, quickInput.content)) {
      onErrorRef.current?.(new Error('终端已关闭，无法插入快捷输入'))
      return
    }
    closeQuickInputSelector()
  }

  useEffect(() => {
    let active = true
    const fitAndRefresh = () => {
      if (active) {
        sessionRegistry.fitAndRefresh(terminal.taskId, terminal.id, onResizeRef.current)
      }
    }
    if (containerRef.current) {
      sessionRegistry.attach(terminal, containerRef.current, resolvedTerminalTheme, onResizeRef.current)
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
  }, [resolvedTerminalTheme, sessionRegistry, terminal.disableTaskAIMouseClipboard, terminal.id, terminal.taskId])

  useEffect(() => {
    onResizeRef.current = onResize
  }, [onResize])

  useEffect(() => {
    sessionRegistry.fitAndRefresh(terminal.taskId, terminal.id, onResizeRef.current)
  }, [fontSize, sessionRegistry, terminal.id, terminal.taskId])

  useEffect(() => {
    onErrorRef.current = onError
  }, [onError])

  useEffect(() => {
    if (quickInputSelectorOpen) {
      quickInputSearchRef.current?.focus()
    }
  }, [quickInputSelectorOpen])

  useEffect(() => {
    sessionRegistry.setCustomKeyEventHandler(terminal.taskId, terminal.id, (event) => {
      const isMac = typeof navigator !== 'undefined' && navigator.platform.toUpperCase().includes('MAC')
      const isQuickInputHotkey = event.type === 'keydown'
        && event.key.toLocaleLowerCase() === 'p'
        && event.shiftKey
        && (isMac ? event.metaKey : event.ctrlKey)
      if (!isQuickInputHotkey) {
        return true
      }
      event.preventDefault()
      event.stopPropagation()
      openQuickInputSelector()
      return false
    })
    return () => {
      sessionRegistry.setCustomKeyEventHandler(terminal.taskId, terminal.id)
    }
  }, [openQuickInputSelector, sessionRegistry, terminal.id, terminal.taskId])

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
    <div className="taskai-terminal grid h-full min-w-0" style={{gridTemplateRows: '40px minmax(0, 1fr)'}}>
      <div className="taskai-terminal__header flex items-center gap-2 border-b border-snap-outline bg-snap-surface px-2" data-testid="terminal-view-header">
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
        {terminal.state === 'active' && (
          <div className="taskai-terminal__quick-input-actions flex shrink-0 items-center" data-testid="terminal-view-actions">
            <Popover
              open={quickInputSelectorOpen}
              onOpenChange={(open) => {
                if (open) {
                  openQuickInputSelector()
                } else {
                  closeQuickInputSelector()
                }
              }}
            >
              <PopoverTrigger asChild>
                <IconButton aria-label={`快捷输入（${quickInputHotkey}）`} title={`快捷输入（${quickInputHotkey}）`} className="h-7 w-7">
                  <ClipboardPaste className="h-4 w-4"/>
                </IconButton>
              </PopoverTrigger>
              <PopoverContent
                align="end"
                className="w-80 p-3"
                onCloseAutoFocus={(event) => {
                  event.preventDefault()
                }}
                onEscapeKeyDown={(event) => {
                  event.preventDefault()
                  closeQuickInputSelector()
                }}
              >
                <Input
                  ref={quickInputSearchRef}
                  aria-label="搜索快捷输入"
                  placeholder="搜索名称或内容"
                  value={quickInputSearch}
                  onChange={(event) => {
                    setQuickInputSearch(event.target.value)
                    setSelectedQuickInputIndex(0)
                  }}
                  onKeyDown={(event) => {
                    if (event.key === 'ArrowDown') {
                      event.preventDefault()
                      setSelectedQuickInputIndex((index) => Math.min(index + 1, Math.max(filteredQuickInputs.length - 1, 0)))
                    } else if (event.key === 'ArrowUp') {
                      event.preventDefault()
                      setSelectedQuickInputIndex((index) => Math.max(index - 1, 0))
                    } else if (event.key === 'Enter') {
                      event.preventDefault()
                      const quickInput = filteredQuickInputs[selectedQuickInputIndex]
                      if (quickInput) {
                        insertQuickInput(quickInput)
                      }
                    }
                  }}
                />
                <div role="listbox" aria-label="快捷输入列表" className="mt-2 max-h-72 overflow-y-auto">
                  {filteredQuickInputs.length === 0 ? (
                    <p className="px-2 py-3 text-sm text-snap-muted">没有匹配的快捷输入</p>
                  ) : filteredQuickInputs.map((quickInput, index) => (
                    <button
                      key={quickInput.id}
                      type="button"
                      role="option"
                      aria-selected={index === selectedQuickInputIndex}
                      data-quick-input-selected={index === selectedQuickInputIndex}
                      className="block w-full border-b border-l-2 border-l-transparent border-snap-outline/30 px-2 py-2 text-left last:border-b-0 hover:bg-snap-cobalt/10 focus:bg-snap-cobalt/10 focus:outline-none data-[quick-input-selected=true]:border-l-snap-cobalt data-[quick-input-selected=true]:bg-snap-cobalt/10"
                      onMouseEnter={() => setSelectedQuickInputIndex(index)}
                      onClick={() => insertQuickInput(quickInput)}
                    >
                      <span className="block truncate text-sm font-bold text-snap-ink">{quickInput.name}</span>
                      <span className="mt-0.5 block truncate font-mono text-xs text-snap-muted">{quickInputContentPreview(quickInput.content)}</span>
                    </button>
                  ))}
                </div>
              </PopoverContent>
            </Popover>
          </div>
        )}
      </div>
      <div
        ref={containerRef}
        className="taskai-terminal__content relative min-h-0 overflow-hidden p-1"
        data-testid="terminal-content"
        onContextMenu={taskAIMouseClipboardEnabled ? (event) => {
          event.preventDefault()
          void ClipboardGetText().then((clipboard) => {
            if (clipboard) {
              sessionRegistry.writeInput(terminal.taskId, terminal.id, clipboard)
            }
          }).catch(() => {})
        } : undefined}
        style={{backgroundColor: resolvedTerminalTheme.background, '--wails-drop-target': 'drop'} as CSSProperties}
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

function quickInputContentPreview(content: string): string {
  return content.replaceAll('\n', ' ↵ ').replaceAll('\t', ' ⇥ ')
}
