import {FitAddon} from '@xterm/addon-fit'
import {Terminal, type ITheme} from '@xterm/xterm'

import {ClipboardSetText} from '../wailsjs/runtime/runtime'
import {type TerminalEvent, type TerminalRecord} from './types'
import {resolveTerminalFontFamily} from './terminal-font'
import {defaultTerminalFontSize, normalizeTerminalFontSize} from './terminal-font-size'

export const terminalScrollback = 1000

// 快门波普终端主题：直接注入 xterm ITheme。背景取自令牌（亮色=浅次表面 #E3EAE9，
// 暗色=深表面 #16242B），前景/光标用墨色/钴蓝；ANSI 调色板把成功→钴蓝、关键字→紫罗兰、
// 提示/错误→珊瑚、警告→琥珀，使原始 PTY 输出整体偏快门波普色系。
export type TerminalVisualTheme = ITheme

interface TerminalSession {
  taskID: string
  terminalID: string
  terminal: Terminal
  fitAddon: FitAddon
  selectionClipboard: {enabled: boolean}
  onData: {dispose(): void}
  onSelectionChange: {dispose(): void}
}

export class TerminalSessionRegistry {
  private readonly sessions = new Map<string, TerminalSession>()
  private readonly closedTerminalKeys = new Set<string>()

  constructor(
    private readonly onWrite: (taskID: string, terminalID: string, data: string) => void,
    private readonly terminalFontFamily: () => string = () => '',
    private readonly terminalFontSize: () => number = () => defaultTerminalFontSize,
  ) {}

  setFontSize(fontSize: number): void {
    const normalizedFontSize = normalizeTerminalFontSize(fontSize)
    for (const session of this.sessions.values()) {
      session.terminal.options.fontSize = normalizedFontSize
    }
  }

  handleTerminalEvent(event: TerminalEvent): void {
    if (event.type === 'output') {
      const session = this.getOrCreate(event.taskId, event.terminalId)
      if (session && event.data) {
        session.terminal.write(event.data)
      }
      return
    }
    if (event.type === 'exited') {
      this.dispose(event.taskId, event.terminalId)
    }
  }

  attach(
    terminal: TerminalRecord,
    container: HTMLElement,
    theme: TerminalVisualTheme,
    onResize: (columns: number, rows: number) => void,
  ): boolean {
    const session = this.getOrCreate(terminal.taskId, terminal.id)
    if (!session) {
      return false
    }
    session.selectionClipboard.enabled = terminal.disableTaskAIMouseClipboard !== true
    session.terminal.options.theme = theme
    if (session.terminal.element) {
      container.append(session.terminal.element)
    } else {
      session.terminal.open(container)
    }
    return this.fit(session, onResize)
  }

  fitAndRefresh(taskID: string, terminalID: string, onResize: (columns: number, rows: number) => void): boolean {
    const session = this.sessions.get(terminalSessionKey(taskID, terminalID))
    if (!session || !this.fit(session, onResize)) {
      return false
    }
    this.refreshVisibleRows(session)
    return true
  }

  focus(taskID: string, terminalID: string): void {
    this.sessions.get(terminalSessionKey(taskID, terminalID))?.terminal.focus()
  }

  writeInput(taskID: string, terminalID: string, data: string): void {
    if (!data || this.closedTerminalKeys.has(terminalSessionKey(taskID, terminalID))) {
      return
    }
    this.onWrite(taskID, terminalID, data)
  }

  pasteQuickInput(taskID: string, terminalID: string, content: string): boolean {
    const key = terminalSessionKey(taskID, terminalID)
    const session = this.sessions.get(key)
    if (!content || !session || this.closedTerminalKeys.has(key)) {
      return false
    }
    session.terminal.paste(content)
    return true
  }

  setCustomKeyEventHandler(taskID: string, terminalID: string, handler?: (event: KeyboardEvent) => boolean): boolean {
    const session = this.sessions.get(terminalSessionKey(taskID, terminalID))
    if (!session || this.closedTerminalKeys.has(terminalSessionKey(taskID, terminalID))) {
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
    this.sessions.delete(key)
    session.onData.dispose()
    session.onSelectionChange.dispose()
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
    if (this.closedTerminalKeys.has(key)) {
      return undefined
    }
    const existing = this.sessions.get(key)
    if (existing) {
      return existing
    }
    const terminal = new Terminal({
      cursorBlink: true,
      fontFamily: resolveTerminalFontFamily(this.terminalFontFamily()),
      fontSize: normalizeTerminalFontSize(this.terminalFontSize()),
      lineHeight: 1.35,
      scrollback: terminalScrollback,
    })
    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    const onData = terminal.onData((data) => this.onWrite(taskID, terminalID, data))
    const selectionClipboard = {enabled: true}
    const onSelectionChange = terminal.onSelectionChange(() => {
      if (!selectionClipboard.enabled) {
        return
      }
      const selection = terminal.getSelection()
      if (selection) {
        void ClipboardSetText(selection).catch(() => {})
      }
    })
    const session = {taskID, terminalID, terminal, fitAddon, selectionClipboard, onData, onSelectionChange}
    this.sessions.set(key, session)
    return session
  }

  private fit(session: TerminalSession, onResize: (columns: number, rows: number) => void): boolean {
    try {
      session.fitAddon.fit()
      if (session.terminal.cols > 0 && session.terminal.rows > 0) {
        onResize(session.terminal.cols, session.terminal.rows)
      }
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
}

export function terminalVisualTheme(mode: 'light' | 'dark'): TerminalVisualTheme {
  return mode === 'dark'
    ? {
        background: '#16242B',
        foreground: '#E6EEF1',
        cursor: '#5C8CFF',
        cursorAccent: '#16242B',
        selectionBackground: '#5C8CFF40',
        selectionForeground: '#FFFFFF',
        black: '#16242B',
        red: '#FF6A5A',
        green: '#5C8CFF',
        yellow: '#FFCD33',
        blue: '#5C8CFF',
        magenta: '#A98CFF',
        cyan: '#3FA9C0',
        white: '#8AA0A8',
        brightBlack: '#8AA0A8',
        brightRed: '#FF7A6E',
        brightGreen: '#5C8CFF',
        brightYellow: '#FFCD33',
        brightBlue: '#5C8CFF',
        brightMagenta: '#A98CFF',
        brightCyan: '#3FA9C0',
        brightWhite: '#E6EEF1',
      }
    : {
        background: '#E3EAE9',
        foreground: '#10212B',
        cursor: '#1E66F5',
        cursorAccent: '#E3EAE9',
        selectionBackground: '#1E66F540',
        selectionForeground: '#FFFFFF',
        black: '#10212B',
        red: '#E0341B',
        green: '#1E66F5',
        yellow: '#B07A00',
        blue: '#1E66F5',
        magenta: '#8B5CF6',
        cyan: '#0E7C9B',
        white: '#5A6E78',
        brightBlack: '#5A6E78',
        brightRed: '#FF5A4E',
        brightGreen: '#1E66F5',
        brightYellow: '#F5B700',
        brightBlue: '#1E66F5',
        brightMagenta: '#8B5CF6',
        brightCyan: '#0E7C9B',
        brightWhite: '#10212B',
      }
}

function terminalSessionKey(taskID: string, terminalID: string): string {
  return JSON.stringify([taskID, terminalID])
}
