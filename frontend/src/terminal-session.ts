import {FitAddon} from '@xterm/addon-fit'
import {Terminal} from '@xterm/xterm'

import {ClipboardSetText} from '../wailsjs/runtime/runtime'
import {type TerminalEvent, type TerminalRecord} from './types'

export const terminalScrollback = 1000

export interface TerminalVisualTheme {
  background: string
  foreground: string
  cursor: string
  selectionBackground: string
}

interface TerminalSession {
  taskID: string
  terminalID: string
  terminal: Terminal
  fitAddon: FitAddon
  onData: {dispose(): void}
  onSelectionChange: {dispose(): void}
}

export class TerminalSessionRegistry {
  private readonly sessions = new Map<string, TerminalSession>()
  private readonly closedTerminalKeys = new Set<string>()

  constructor(private readonly onWrite: (taskID: string, terminalID: string, data: string) => void) {}

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
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      fontSize: 13,
      lineHeight: 1.35,
      scrollback: terminalScrollback,
    })
    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    const onData = terminal.onData((data) => this.onWrite(taskID, terminalID, data))
    const onSelectionChange = terminal.onSelectionChange(() => {
      const selection = terminal.getSelection()
      if (selection) {
        void ClipboardSetText(selection).catch(() => {})
      }
    })
    const session = {taskID, terminalID, terminal, fitAddon, onData, onSelectionChange}
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
    ? {background: '#101a14', foreground: '#d8e8dc', cursor: '#9cc3ab', selectionBackground: '#2e4035'}
    : {background: '#26352e', foreground: '#e3eee5', cursor: '#9bd5ae', selectionBackground: '#3d5748'}
}

function terminalSessionKey(taskID: string, terminalID: string): string {
  return JSON.stringify([taskID, terminalID])
}
