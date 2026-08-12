import {useCallback, useEffect, useMemo, useRef, useState, type CSSProperties} from 'react'
import {ClipboardPaste, MessageSquarePlus, Send, Terminal as TerminalIcon} from 'lucide-react'
import '@xterm/xterm/css/xterm.css'

import {TerminalSessionRegistry, terminalVisualTheme} from '../terminal-session'
import {type QuickInput, type TerminalNoteTemplate, type TerminalRecord, type TerminalShortcut} from '../types'
import {findTerminalShortcut, terminalShortcutApplies, terminalShortcutInput} from '../terminal-shortcuts'
import {defaultTerminalNoteTemplate, formatTerminalNotes, type TerminalNote} from '../terminal-notes'
import {api} from '../api'
import {defaultTerminalFontSize} from '../terminal-font-size'
import {type TerminalTheme} from '../terminal-theme'
import {ClipboardGetText, OnFileDrop, OnFileDropOff} from '../../wailsjs/runtime/runtime'
import {Button, Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, IconButton, Input, Popover, PopoverContent, PopoverTrigger, Textarea} from './ui'
import {TerminalName} from './TerminalName'

interface TerminalViewProps {
  terminal: TerminalRecord
  sessionRegistry: TerminalSessionRegistry
  quickInputs?: QuickInput[]
  terminalShortcuts?: TerminalShortcut[]
	notes?: TerminalNote[]
	noteTemplate?: TerminalNoteTemplate
  fontSize?: number
  terminalTheme?: Partial<TerminalTheme>
  onResize(columns: number, rows: number): void
  onClose(): void
  onError?(error: unknown): void
	onAddNote?(note: TerminalNote): void
	onClearNotes?(): void
  onAliasChange?(alias: string | undefined): void
}

export function TerminalView({terminal, sessionRegistry, quickInputs = [], terminalShortcuts = [], notes = [], noteTemplate = defaultTerminalNoteTemplate, fontSize = defaultTerminalFontSize, terminalTheme, onResize, onClose, onError, onAddNote = () => {}, onClearNotes = () => {}, onAliasChange = () => {}}: TerminalViewProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const quickInputSearchRef = useRef<HTMLInputElement>(null)
  const noteInputRef = useRef<HTMLTextAreaElement>(null)
  const onResizeRef = useRef(onResize)
  const onErrorRef = useRef(onError)
  const [quickInputSelectorOpen, setQuickInputSelectorOpen] = useState(false)
  const [quickInputSearch, setQuickInputSearch] = useState('')
  const [selectedQuickInputIndex, setSelectedQuickInputIndex] = useState(0)
	const [selectionNote, setSelectionNote] = useState<{original: string, left: number, top: number}>()
	const [noteInputOpen, setNoteInputOpen] = useState(false)
	const [noteText, setNoteText] = useState('')
	const [notesPanelOpen, setNotesPanelOpen] = useState(false)
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

	const closeNoteInput = () => {
		setNoteInputOpen(false)
		setNoteText('')
		setSelectionNote(undefined)
		sessionRegistry.focus(terminal.taskId, terminal.id)
	}

	const saveNote = () => {
		if (!selectionNote || !noteText.trim()) {
			return
		}
		onAddNote({original: selectionNote.original, note: noteText})
		closeNoteInput()
	}

	const sendNotes = () => {
		if (notes.length === 0) {
			return
		}
		const wrote = sessionRegistry.pasteInput(terminal.taskId, terminal.id, formatTerminalNotes(notes, noteTemplate))
		onClearNotes()
		setNotesPanelOpen(false)
		sessionRegistry.focus(terminal.taskId, terminal.id)
		if (!wrote) {
			onErrorRef.current?.(new Error('终端已关闭，无法发送备注'))
		}
	}

  const insertQuickInput = (quickInput: QuickInput) => {
    if (!sessionRegistry.pasteInput(terminal.taskId, terminal.id, quickInput.content)) {
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
  }, [resolvedTerminalTheme, sessionRegistry, terminal.disableTaskAIMouseClipboard, terminal.id, terminal.state, terminal.taskId])

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
		if (noteInputOpen) {
			noteInputRef.current?.focus()
		}
	}, [noteInputOpen])

	useEffect(() => {
		if (terminal.state !== 'active') {
			setSelectionNote(undefined)
			setNoteInputOpen(false)
			setNotesPanelOpen(false)
		}
	}, [terminal.state])

  useEffect(() => {
		if (terminal.state !== 'active') {
			return
		}
    sessionRegistry.setCustomKeyEventHandler(terminal.taskId, terminal.id, (event) => {
      const isMac = typeof navigator !== 'undefined' && navigator.platform.toUpperCase().includes('MAC')
      const isQuickInputHotkey = event.type === 'keydown'
        && event.key.toLocaleLowerCase() === 'p'
        && event.shiftKey
        && (isMac ? event.metaKey : event.ctrlKey)
		if (isQuickInputHotkey) {
			event.preventDefault()
			event.stopPropagation()
			openQuickInputSelector()
			return false
		}

		const configuredShortcut = findTerminalShortcut(terminalShortcuts, event)
		if (!configuredShortcut) {
			return true
		}
		if (!terminalShortcutApplies(configuredShortcut, terminal.command)) {
			return true
		}
		event.preventDefault()
		event.stopPropagation()
		if (!sessionRegistry.writeInput(terminal.taskId, terminal.id, terminalShortcutInput(configuredShortcut.steps))) {
			onErrorRef.current?.(new Error('终端已关闭，无法执行快捷键动作'))
		}
		return false
    })
    return () => {
      sessionRegistry.setCustomKeyEventHandler(terminal.taskId, terminal.id)
    }
  }, [openQuickInputSelector, sessionRegistry, terminal.command, terminal.id, terminal.state, terminal.taskId, terminalShortcuts])

  useEffect(() => {
		if (terminal.state !== 'active') {
			return
		}
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
  }, [terminal.id, terminal.state, terminal.taskId])

  return (
    <div className="taskai-terminal grid h-full min-w-0" style={{gridTemplateRows: '40px minmax(0, 1fr)'}} onPointerDown={() => setSelectionNote(undefined)}>
      <div className="taskai-terminal__header flex items-center gap-2 border-b border-snap-outline bg-snap-surface px-2" data-testid="terminal-view-header">
        <TerminalIcon className="h-4 w-4 shrink-0 text-snap-cobalt"/>
        <div data-testid="terminal-view-title-container" style={{flex: 1, minWidth: 0}}>
          <TerminalName
            terminal={terminal}
            onAliasChange={onAliasChange}
            testId="terminal-view-title"
            className="block w-full truncate font-display text-sm font-bold text-snap-ink"
          />
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
				<Popover open={notesPanelOpen} onOpenChange={setNotesPanelOpen}>
					<PopoverTrigger asChild>
						<IconButton aria-label={`终端备注（${notes.length} 条）`} title={`终端备注（${notes.length} 条）`} className="relative h-7 w-7">
							<MessageSquarePlus className="h-4 w-4"/>
							{notes.length > 0 && <span aria-hidden className="absolute -right-1 -top-1 min-w-4 rounded-full bg-snap-cobalt px-1 text-[10px] font-bold leading-4 text-snap-surface">{notes.length}</span>}
						</IconButton>
					</PopoverTrigger>
					<PopoverContent align="end" className="w-96 p-3">
						<div className="grid gap-3">
							<div className="flex items-center justify-between gap-2">
								<span className="font-display text-sm font-bold text-snap-ink">终端备注</span>
								<span className="text-xs text-snap-muted">{notes.length} 条</span>
							</div>
							{notes.length === 0 ? <p className="py-3 text-sm text-snap-muted">还没有备注</p> : <div className="grid max-h-64 gap-2 overflow-y-auto pr-1">
								{notes.map((note, index) => <div key={`${note.original}-${note.note}-${index}`} className="grid gap-1 rounded-snap border border-snap-outline bg-snap-surface p-2 text-sm">
									<p className="whitespace-pre-wrap text-snap-ink">原文：{note.original}</p>
									<p className="whitespace-pre-wrap text-snap-muted">备注：{note.note}</p>
								</div>)}
							</div>}
							<Button variant="primary" size="sm" disabled={notes.length === 0} onClick={sendNotes}><Send className="mr-1 h-4 w-4"/>发送到终端</Button>
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
		onContextMenu={terminal.state === 'active' && taskAIMouseClipboardEnabled ? (event) => {
          event.preventDefault()
          void ClipboardGetText().then((clipboard) => {
            if (clipboard) {
              sessionRegistry.pasteInput(terminal.taskId, terminal.id, clipboard)
            }
          }).catch(() => {})
		} : undefined}
		onPointerUp={(event) => {
			if (terminal.state !== 'active' || !taskAIMouseClipboardEnabled) {
				return
			}
			const clientX = Number.isFinite(event.clientX) ? event.clientX : 0
			const clientY = Number.isFinite(event.clientY) ? event.clientY : 0
			requestAnimationFrame(() => {
				const original = sessionRegistry.selectionText(terminal.taskId, terminal.id)
				const bounds = containerRef.current?.getBoundingClientRect()
				if (!original || !bounds) {
					setSelectionNote(undefined)
					return
				}
				setSelectionNote({
					original,
					left: clampNoteButtonPosition(clientX - bounds.left + 8, bounds.width),
					top: clampNoteButtonPosition(clientY - bounds.top + 8, bounds.height),
				})
			})
		}}
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
			{selectionNote && terminal.state === 'active' && taskAIMouseClipboardEnabled && <IconButton
				aria-label="添加备注"
				title="添加备注"
				className="absolute z-10 h-8 w-8 border-snap-cobalt bg-snap-overlay opacity-70 hover:opacity-100"
				style={{left: selectionNote.left, top: selectionNote.top}}
				onPointerDown={(event) => event.stopPropagation()}
				onClick={() => setNoteInputOpen(true)}
			>
				<MessageSquarePlus className="h-4 w-4"/>
			</IconButton>}
      </div>
		<Dialog open={noteInputOpen} onOpenChange={(open) => { if (!open) closeNoteInput() }}>
			<DialogContent onPointerDown={(event) => event.stopPropagation()}>
				<DialogHeader>
					<DialogTitle>添加终端备注</DialogTitle>
				</DialogHeader>
				<div className="grid gap-2">
					<span className="text-xs font-bold uppercase tracking-wide text-snap-muted">原文</span>
					<pre className="max-h-40 overflow-auto whitespace-pre-wrap rounded-snap border border-snap-outline bg-snap-surface p-3 font-mono text-sm text-snap-ink">{selectionNote?.original}</pre>
					<Textarea ref={noteInputRef} aria-label="备注内容" rows={4} placeholder="输入备注…" value={noteText} onChange={(event) => setNoteText(event.target.value)}/>
				</div>
				<DialogFooter>
					<Button variant="secondary" onClick={closeNoteInput}>取消</Button>
					<Button variant="primary" disabled={!noteText.trim()} onClick={saveNote}>保存备注</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
    </div>
  )
}

function quickInputContentPreview(content: string): string {
  return content.replaceAll('\n', ' ↵ ').replaceAll('\t', ' ⇥ ')
}

function clampNoteButtonPosition(position: number, containerSize: number): number {
	return Math.max(0, Math.min(position, Math.max(0, containerSize - 40)))
}
