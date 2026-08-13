import {FitAddon} from '@xterm/addon-fit'
import {Terminal, type IBufferCell, type ITheme} from '@xterm/xterm'

import {ClipboardSetText} from '../wailsjs/runtime/runtime'
import {type TerminalEvent, type TerminalRecord} from './types'
import {resolveTerminalFontFamily} from './terminal-font'
import {defaultTerminalFontSize, normalizeTerminalFontSize} from './terminal-font-size'
import {normalizeTerminalTheme, type TerminalTheme} from './terminal-theme'
import {TerminalMouseGesture, type TerminalContextMenuHandler, type TerminalSelectionCompleteHandler} from './terminal-mouse-gesture'

export const terminalScrollback = 1000
const terminalExitSnapshotNotice = '\r\n终端已退出\x1b[?25l'

// Nebula 终端主题：直接注入 xterm ITheme。背景/前景/光标和 ANSI 调色板与
// 亮暗主题令牌保持一致，终端输出仍由现有 PTY 会话管理。
export type TerminalVisualTheme = ITheme

interface TerminalSession {
  taskID: string
  terminalID: string
  terminal: Terminal
  fitAddon: FitAddon
  display: TerminalDisplay
  suppressVisualActivity: boolean
  mouseGesture?: TerminalMouseGesture
  onData: {dispose(): void}
  onSelectionChange: {dispose(): void}
  onWriteParsed: {dispose(): void}
}

interface TerminalDisplay {
  bufferType: 'normal' | 'alternate'
  cells: string[]
  columns: number
  rows: number
}

export class TerminalSessionRegistry {
  private readonly sessions = new Map<string, TerminalSession>()
  private readonly closedTerminalKeys = new Set<string>()
  private attachedSessionKey?: string

  constructor(
    private readonly onWrite: (taskID: string, terminalID: string, data: string) => void,
    private readonly terminalFontFamily: () => string = () => '',
    private readonly terminalFontSize: () => number = () => defaultTerminalFontSize,
    private readonly terminalTheme: () => TerminalTheme = () => normalizeTerminalTheme(),
    private readonly onVisualActivity: (taskID: string, terminalID: string) => void = () => {},
  ) {}

  setFontSize(fontSize: number): void {
    const normalizedFontSize = normalizeTerminalFontSize(fontSize)
    for (const session of this.sessions.values()) {
      session.terminal.options.fontSize = normalizedFontSize
      this.resetDisplay(session)
    }
  }

  setAppearance(fontFamily: string, fontSize: number, theme: TerminalVisualTheme): void {
    const normalizedFontSize = normalizeTerminalFontSize(fontSize)
    const resolvedFontFamily = resolveTerminalFontFamily(fontFamily)
    for (const session of this.sessions.values()) {
      session.terminal.options.fontFamily = resolvedFontFamily
      session.terminal.options.fontSize = normalizedFontSize
      session.terminal.options.theme = theme
      this.refreshVisibleRows(session)
      this.resetDisplay(session)
    }
  }

  handleTerminalEvent(event: TerminalEvent): void {
    if (event.type === 'output') {
			if (this.closedTerminalKeys.has(terminalSessionKey(event.taskId, event.terminalId))) {
				return
			}
      const session = this.getOrCreate(event.taskId, event.terminalId)
      if (session && event.data) {
        session.terminal.write(event.data)
      }
      return
    }
    if (event.type === 'exited') {
      const key = terminalSessionKey(event.taskId, event.terminalId)
      this.detachByKey(key)
      if (!terminalExitDisposesSession(event.exitReason)) {
        if (this.closedTerminalKeys.has(key)) {
          return
        }

        const session = this.getOrCreate(event.taskId, event.terminalId)
        if (session) {
          session.suppressVisualActivity = true
          session.terminal.write(terminalExitSnapshotNotice)
        }
        this.closedTerminalKeys.add(key)
        return
      }
      this.dispose(event.taskId, event.terminalId)
    }
  }

  attach(
    terminal: TerminalRecord,
    container: HTMLElement,
    theme: TerminalVisualTheme,
    onResize: (columns: number, rows: number) => void,
    onSelectionComplete?: TerminalSelectionCompleteHandler,
    onContextMenu?: TerminalContextMenuHandler,
  ): boolean {
    const key = terminalSessionKey(terminal.taskId, terminal.id)
    if (this.attachedSessionKey && this.attachedSessionKey !== key) {
      this.detachByKey(this.attachedSessionKey)
    }
    const session = this.getOrCreate(terminal.taskId, terminal.id)
    if (!session) {
      return false
    }
    session.terminal.options.theme = theme
    if (session.terminal.element) {
      container.append(session.terminal.element)
    } else {
      session.terminal.open(container)
    }
		if (terminal.state === 'active' && !this.closedTerminalKeys.has(key) && session.terminal.element) {
			if (!session.mouseGesture) {
				session.mouseGesture = new TerminalMouseGesture(
					session.terminal.element,
					() => session.terminal.modes.mouseTrackingMode !== 'none',
					() => session.terminal.getSelection(),
					onSelectionComplete,
					onContextMenu,
					() => session.terminal.getSelectionPosition(),
					(selection) => {
						session.terminal.select(
							selection.start.x,
							selection.start.y,
							(selection.end.y - selection.start.y) * session.terminal.cols - selection.start.x + selection.end.x,
						)
						return session.terminal.getSelection() === selection.text
					},
				)
			} else {
				session.mouseGesture.setSelectionCompleteHandler(onSelectionComplete)
				session.mouseGesture.setContextMenuHandler(onContextMenu)
			}
			this.attachedSessionKey = key
		} else {
			this.detachByKey(key)
		}
		return this.fit(session, this.closedTerminalKeys.has(terminalSessionKey(terminal.taskId, terminal.id)) ? undefined : onResize)
  }

  detach(taskID: string, terminalID: string): void {
    this.detachByKey(terminalSessionKey(taskID, terminalID))
  }

  fitAndRefresh(taskID: string, terminalID: string, onResize?: (columns: number, rows: number) => void): boolean {
    const session = this.sessions.get(terminalSessionKey(taskID, terminalID))
		if (!session || !this.fit(session, this.closedTerminalKeys.has(terminalSessionKey(taskID, terminalID)) ? undefined : onResize)) {
      return false
    }
    this.refreshVisibleRows(session)
    return true
  }

  focus(taskID: string, terminalID: string): void {
    this.sessions.get(terminalSessionKey(taskID, terminalID))?.terminal.focus()
  }

  writeInput(taskID: string, terminalID: string, data: string): boolean {
    if (!data || this.closedTerminalKeys.has(terminalSessionKey(taskID, terminalID))) {
      return false
    }
    this.onWrite(taskID, terminalID, data)
    return true
  }

  pasteInput(taskID: string, terminalID: string, content: string): boolean {
    const key = terminalSessionKey(taskID, terminalID)
    const session = this.sessions.get(key)
    if (!content || !session || this.closedTerminalKeys.has(key)) {
      return false
    }
    session.terminal.paste(content)
    return true
  }

  selectionText(taskID: string, terminalID: string): string {
    const key = terminalSessionKey(taskID, terminalID)
    const session = this.sessions.get(key)
    if (!session || this.closedTerminalKeys.has(key)) {
      return ''
    }
    return session.terminal.getSelection()
  }

  setCustomKeyEventHandler(taskID: string, terminalID: string, handler?: (event: KeyboardEvent) => boolean): boolean {
		const key = terminalSessionKey(taskID, terminalID)
		const session = this.sessions.get(key)
		if (!session) {
			return false
		}
		if (this.closedTerminalKeys.has(key)) {
			session.terminal.attachCustomKeyEventHandler(() => false)
			return false
		}
    session.terminal.attachCustomKeyEventHandler(handler ?? (() => true))
    return true
  }

  dispose(taskID: string, terminalID: string): void {
    const key = terminalSessionKey(taskID, terminalID)
    this.closedTerminalKeys.add(key)
    const session = this.sessions.get(key)
    if (!session) {
      return
    }
    this.detachByKey(key)
    this.sessions.delete(key)
    session.onData.dispose()
    session.onSelectionChange.dispose()
    session.onWriteParsed.dispose()
    session.terminal.dispose()
  }

  disposeTask(taskID: string): void {
    for (const session of [...this.sessions.values()]) {
      if (session.taskID === taskID) {
        this.dispose(session.taskID, session.terminalID)
      }
    }
  }

  disposeAll(): void {
    for (const session of [...this.sessions.values()]) {
      this.dispose(session.taskID, session.terminalID)
    }
  }

  private getOrCreate(taskID: string, terminalID: string): TerminalSession | undefined {
    const key = terminalSessionKey(taskID, terminalID)
    const existing = this.sessions.get(key)
    if (existing) {
      return existing
    }
		if (this.closedTerminalKeys.has(key)) {
			return undefined
		}
    const terminal = new Terminal({
      altClickMovesCursor: false,
      cursorBlink: true,
      fontFamily: resolveTerminalFontFamily(this.terminalFontFamily()),
      fontSize: normalizeTerminalFontSize(this.terminalFontSize()),
      lineHeight: 1.35,
      macOptionClickForcesSelection: true,
      scrollback: terminalScrollback,
      theme: terminalVisualTheme(this.terminalTheme()),
    })
    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
		const onData = terminal.onData((data) => {
			if (!this.closedTerminalKeys.has(key)) {
				this.onWrite(taskID, terminalID, data)
			}
		})
    let session: TerminalSession
    const onSelectionChange = terminal.onSelectionChange(() => {
      const selection = terminal.getSelection()
      if (selection) {
        void ClipboardSetText(selection).catch(() => {})
      } else {
        session?.mouseGesture?.restorePreservedSelection()
      }
    })
    const onWriteParsed = terminal.onWriteParsed?.(() => {
      const display = captureTerminalDisplay(terminal)
      const suppressVisualActivity = session.suppressVisualActivity
      session.suppressVisualActivity = false
      if (!terminalDisplaysEqual(session.display, display)) {
        session.display = display
        if (!suppressVisualActivity) {
          this.onVisualActivity(taskID, terminalID)
        }
      }
    }) ?? {dispose() {}}
    session = {
      taskID,
      terminalID,
      terminal,
      fitAddon,
      display: captureTerminalDisplay(terminal),
      suppressVisualActivity: false,
      onData,
      onSelectionChange,
      onWriteParsed,
    }
    this.sessions.set(key, session)
    return session
  }

  private fit(session: TerminalSession, onResize?: (columns: number, rows: number) => void): boolean {
    try {
      session.fitAddon.fit()
		if (onResize && session.terminal.cols > 0 && session.terminal.rows > 0) {
        onResize(session.terminal.cols, session.terminal.rows)
      }
      this.resetDisplay(session)
      return true
    } catch {
      return false
    }
  }

  private refreshVisibleRows(session: TerminalSession): void {
    if (session.terminal.rows > 0) {
      session.terminal.refresh(0, session.terminal.rows - 1)
    }
  }

  private resetDisplay(session: TerminalSession): void {
    session.display = captureTerminalDisplay(session.terminal)
  }

  private detachByKey(key: string): void {
    const session = this.sessions.get(key)
    session?.mouseGesture?.dispose()
    if (session) {
      session.mouseGesture = undefined
    }
    if (this.attachedSessionKey === key) {
      this.attachedSessionKey = undefined
    }
  }

}

function captureTerminalDisplay(terminal: Terminal): TerminalDisplay {
  const buffer = terminal.buffer.active
  const cells: string[] = []
  for (let row = 0; row < terminal.rows; row++) {
    const line = buffer.getLine(buffer.baseY + row)
    for (let column = 0; column < terminal.cols; column++) {
      const cell = line?.getCell(column)
      cells.push(cell ? terminalCellSignature(cell) : '')
    }
  }
  return {
    bufferType: buffer.type,
    cells,
    columns: terminal.cols,
    rows: terminal.rows,
  }
}

function terminalCellSignature(cell: IBufferCell): string {
  return [
    cell.getChars(),
    cell.getWidth(),
    cell.getFgColorMode(),
    cell.getFgColor(),
    cell.getBgColorMode(),
    cell.getBgColor(),
    cell.isBold(),
    cell.isDim(),
    cell.isItalic(),
    cell.isUnderline(),
    cell.isBlink(),
    cell.isInverse(),
    cell.isInvisible(),
    cell.isStrikethrough(),
    cell.isOverline(),
  ].join('\u001F')
}

function terminalDisplaysEqual(left: TerminalDisplay, right: TerminalDisplay): boolean {
  if (left.bufferType !== right.bufferType || left.columns !== right.columns || left.rows !== right.rows || left.cells.length !== right.cells.length) {
    return false
  }
  return left.cells.every((cell, index) => cell === right.cells[index])
}

export function terminalVisualTheme(theme?: Partial<TerminalTheme>): TerminalVisualTheme {
  return normalizeTerminalTheme(theme)
}

function terminalExitDisposesSession(exitReason: TerminalEvent['exitReason']): boolean {
  return exitReason === 'normal' || exitReason === 'closed' || exitReason === 'task-ended' || exitReason === 'application-shutdown'
}

function terminalSessionKey(taskID: string, terminalID: string): string {
  return JSON.stringify([taskID, terminalID])
}
