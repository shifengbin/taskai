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
// 按输出静默期合并 PTY 事件，避免逐事件渲染重绘中间态；连续输出按截止时限冲刷。
const terminalOutputQuietFlushMs = 32
const terminalOutputMaxFlushDelayMs = 64
const terminalOutputBufferLimit = 1024 * 1024
const terminalSynchronizedOutputMinBytes = 64
const terminalSynchronizedOutputEnableSequence = '\x1b[?2026h'
const terminalSynchronizedOutputDisableSequence = '\x1b[?2026l'
const cursorHideSequence = '\x1b[?25l'
const cursorShowSequence = '\x1b[?25h'

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
  pendingOutput: string[]
  pendingOutputLength: number
  pendingSinceMs?: number
  flushTimerHandle?: number
  synchronizedOutputOwned: boolean
  synchronizedOutputReleaseTimerHandle?: number
  lastNotifiedAtBottom?: boolean
  mouseGesture?: TerminalMouseGesture
  onAtBottomChange?: (atBottom: boolean) => void
  onData: {dispose(): void}
  onScroll?: {dispose(): void}
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
        this.enqueueOutput(session, event.data)
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
          // 退出前先冲刷缓冲（保持光标原状），退出通知在冲刷后写入
          this.flushOutput(session, {preserveCursor: true})
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
    onAtBottomChange?: (atBottom: boolean) => void,
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
		} else {
			this.detachMouseGesture(session)
		}
		this.attachScrollPosition(session, onAtBottomChange)
		this.attachedSessionKey = key
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

  scrollToBottom(taskID: string, terminalID: string): boolean {
    const session = this.sessions.get(terminalSessionKey(taskID, terminalID))
    if (!session) {
      return false
    }
    session.terminal.scrollToBottom()
    this.notifyScrollPosition(session)
    return true
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
    // 销毁会话：丢弃未冲刷缓冲并取消未决回调，不产生悬挂的定时器
    this.cancelScheduledFlush(session)
    this.cancelScheduledSynchronizedOutputRelease(session)
    session.pendingOutput.length = 0
    session.pendingOutputLength = 0
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
      pendingOutput: [],
      pendingOutputLength: 0,
      synchronizedOutputOwned: false,
      onData,
      onSelectionChange,
      onWriteParsed,
    }
    this.sessions.set(key, session)
    return session
  }

  private enqueueOutput(session: TerminalSession, data: string): void {
    const now = performance.now()
    if (session.pendingOutput.length === 0) {
      session.pendingSinceMs = now
    }
    session.pendingOutput.push(data)
    session.pendingOutputLength += data.length
    if (session.pendingOutputLength >= terminalOutputBufferLimit) {
      this.flushOutput(session)
      return
    }
    this.scheduleOutputFlush(session)
    if (session.synchronizedOutputOwned) {
      this.scheduleSynchronizedOutputRelease(session)
    }
  }

  // 每次到达都重置静默计时；连续输出超过 maxFlushDelay 时按剩余时间冲刷
  private scheduleOutputFlush(session: TerminalSession): void {
    if (session.flushTimerHandle !== undefined) {
      window.clearTimeout(session.flushTimerHandle)
    }
    const now = performance.now()
    const deadline = (session.pendingSinceMs ?? now) + terminalOutputMaxFlushDelayMs
    const delay = Math.max(0, Math.min(terminalOutputQuietFlushMs, deadline - now))
    session.flushTimerHandle = window.setTimeout(() => {
      session.flushTimerHandle = undefined
      this.flushOutput(session)
    }, delay)
  }

  private flushOutput(session: TerminalSession, options: {preserveCursor?: boolean} = {}): void {
    this.cancelScheduledFlush(session)
    if (session.pendingOutput.length === 0) {
      return
    }
    const merged = session.pendingOutput.join('')
    session.pendingOutput = []
    session.pendingOutputLength = 0
    session.pendingSinceMs = undefined
    if (!options.preserveCursor && this.shouldSynchronizeOutput(merged)) {
      this.beginSynchronizedOutput(session)
    }
    session.terminal.write(merged)
    if (session.synchronizedOutputOwned) {
      this.scheduleSynchronizedOutputRelease(session)
    }
  }

  private shouldSynchronizeOutput(data: string): boolean {
    return data.length >= terminalSynchronizedOutputMinBytes
      && (data.includes(cursorShowSequence) || data.includes(cursorHideSequence))
  }

  private beginSynchronizedOutput(session: TerminalSession): void {
    if (!session.synchronizedOutputOwned && session.terminal.modes.synchronizedOutputMode) {
      return
    }
    if (!session.synchronizedOutputOwned) {
      session.terminal.write(terminalSynchronizedOutputEnableSequence)
      session.synchronizedOutputOwned = true
      return
    }
    if (!session.terminal.modes.synchronizedOutputMode) {
      session.terminal.write(terminalSynchronizedOutputEnableSequence)
    }
  }

  private scheduleSynchronizedOutputRelease(session: TerminalSession): void {
    if (!session.synchronizedOutputOwned) {
      return
    }
    this.cancelScheduledSynchronizedOutputRelease(session)
    session.synchronizedOutputReleaseTimerHandle = window.setTimeout(() => {
      session.synchronizedOutputReleaseTimerHandle = undefined
      if (!session.synchronizedOutputOwned) {
        return
      }
      if (session.pendingOutput.length > 0) {
        this.flushOutput(session)
      }
      this.endSynchronizedOutput(session)
    }, terminalOutputQuietFlushMs)
  }

  private endSynchronizedOutput(session: TerminalSession): void {
    this.cancelScheduledSynchronizedOutputRelease(session)
    if (!session.synchronizedOutputOwned) {
      return
    }
    session.terminal.write(terminalSynchronizedOutputDisableSequence)
    session.synchronizedOutputOwned = false
  }

  private cancelScheduledSynchronizedOutputRelease(session: TerminalSession): void {
    if (session.synchronizedOutputReleaseTimerHandle !== undefined) {
      window.clearTimeout(session.synchronizedOutputReleaseTimerHandle)
      session.synchronizedOutputReleaseTimerHandle = undefined
    }
  }

  private cancelScheduledFlush(session: TerminalSession): void {
    if (session.flushTimerHandle !== undefined) {
      window.clearTimeout(session.flushTimerHandle)
      session.flushTimerHandle = undefined
    }
  }

  private fit(session: TerminalSession, onResize?: (columns: number, rows: number) => void): boolean {
    try {
      session.fitAddon.fit()
		if (onResize && session.terminal.cols > 0 && session.terminal.rows > 0) {
        onResize(session.terminal.cols, session.terminal.rows)
      }
      this.resetDisplay(session)
      this.notifyScrollPosition(session)
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

  private attachScrollPosition(session: TerminalSession, onAtBottomChange?: (atBottom: boolean) => void): void {
    this.detachScrollPosition(session)
    session.onAtBottomChange = onAtBottomChange
    if (onAtBottomChange) {
      session.onScroll = session.terminal.onScroll(() => this.notifyScrollPosition(session))
    }
  }

  private notifyScrollPosition(session: TerminalSession): void {
    const onAtBottomChange = session.onAtBottomChange
    if (!onAtBottomChange) {
      return
    }
    const buffer = session.terminal.buffer.active
    const atBottom = buffer.viewportY >= buffer.baseY
    if (session.lastNotifiedAtBottom === atBottom) {
      return
    }
    session.lastNotifiedAtBottom = atBottom
    onAtBottomChange(atBottom)
  }

  private detachScrollPosition(session: TerminalSession): void {
    session.onScroll?.dispose()
    session.onScroll = undefined
    session.onAtBottomChange = undefined
    session.lastNotifiedAtBottom = undefined
  }

  private detachMouseGesture(session: TerminalSession): void {
    session.mouseGesture?.dispose()
    session.mouseGesture = undefined
  }

  private detachByKey(key: string): void {
    const session = this.sessions.get(key)
    if (session) {
      this.detachMouseGesture(session)
      this.detachScrollPosition(session)
      this.endSynchronizedOutput(session)
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
