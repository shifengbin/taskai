import {FitAddon} from '@xterm/addon-fit'
import {Terminal, type ITheme} from '@xterm/xterm'

import {ClipboardSetText} from '../wailsjs/runtime/runtime'
import {type TerminalEvent, type TerminalRecord} from './types'

export const terminalScrollback = 1000

// 终端创建时的回退尺寸：无任何已适配会话（首个终端）时使用。仅决定 PTY 创建瞬间的初始网格，
// 随后挂载 fit() 会同步到真实容器尺寸。
export const defaultTerminalCreateDimensions = {columns: 100, rows: 32}

// 解析新终端创建时应使用的初始尺寸：优先复用最近一次成功 fit 得到的共享面板几何，
// 避免以脱离容器的固定尺寸驱动首批输出折行；无缓存时回退到默认值。
export function resolveTerminalCreateDimensions(cached?: {columns: number; rows: number}): {columns: number; rows: number} {
  return cached ?? defaultTerminalCreateDimensions
}

// 快门波普终端主题：直接注入 xterm ITheme。背景取自令牌（亮色=浅次表面 #E3EAE9，
// 暗色=深表面 #16242B），前景/光标用墨色/钴蓝；ANSI 调色板把成功→钴蓝、关键字→紫罗兰、
// 提示/错误→珊瑚、警告→琥珀，使原始 PTY 输出整体偏快门波普色系。
export type TerminalVisualTheme = ITheme

interface TerminalSession {
  taskID: string
  terminalID: string
  terminal: Terminal
  fitAddon: FitAddon
  onData: {dispose(): void}
  onSelectionChange: {dispose(): void}
  lastSentColumns: number
  lastSentRows: number
  // PTY 尺寸同步期间的输出缓冲：网格已按新尺寸适配、ConPTY 尚未跟上时，到达的输出
  // 若立即写入会按新网格渲染旧宽度内容（Windows 显示缩放下表现为附近几行错位且自愈）。
  // 在 syncPty 期间暂存于此，同步完成后再按序写入。
  resizeInFlight: boolean
  outputQueue: string[]
}

export class TerminalSessionRegistry {
  private readonly sessions = new Map<string, TerminalSession>()
  private readonly closedTerminalKeys = new Set<string>()
  private lastFitDimensions: {columns: number; rows: number} | undefined

  constructor(private readonly onWrite: (taskID: string, terminalID: string, data: string) => void) {}

  // 最近一次成功 fit 得到的列/行。内容区面板对所有终端共享同一几何，故新终端可复用该尺寸
  // 作为 PTY 初始尺寸，避免按固定默认尺寸折行首批输出。
  lastDimensions(): {columns: number; rows: number} | undefined {
    return this.lastFitDimensions
  }

  handleTerminalEvent(event: TerminalEvent): void {
    if (event.type === 'output') {
      const session = this.getOrCreate(event.taskId, event.terminalId)
      if (session && event.data) {
        if (session.resizeInFlight) {
          session.outputQueue.push(event.data)
        } else {
          session.terminal.write(event.data)
        }
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
    onResize: (columns: number, rows: number) => void | Promise<void>,
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
    const dimensions = this.fit(session)
    if (dimensions) {
      void this.syncPty(session, dimensions, onResize)
    }
    return true
  }

  fitAndRefresh(taskID: string, terminalID: string, onResize: (columns: number, rows: number) => void | Promise<void>): boolean {
    const session = this.sessions.get(terminalSessionKey(taskID, terminalID))
    if (!session) {
      return false
    }
    const dimensions = this.fit(session)
    this.refreshVisibleRows(session)
    if (dimensions) {
      void this.syncPty(session, dimensions, onResize)
    }
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
    const session = {taskID, terminalID, terminal, fitAddon, onData, onSelectionChange, lastSentColumns: 0, lastSentRows: 0, resizeInFlight: false, outputQueue: []}
    this.sessions.set(key, session)
    return session
  }

  // 同步适配网格：立即 fitAddon.fit()（xterm 网格在此刻重排），返回需要同步给 PTY 的尺寸。
  // 仅在尺寸真正变化时返回非空：合并连续的相同尺寸事件，避免把抖动的中间列数灌给 ConPTY。
  // 注意：网格与 PTY 不在此处同步——见 syncPty。调用方应在拿到返回值后立即 refresh，
// 再以 syncPty 异步对齐 PTY，期间输出被缓冲（见 handleTerminalEvent）。
  private fit(session: TerminalSession): {columns: number; rows: number} | undefined {
    try {
      session.fitAddon.fit()
      const {cols, rows} = session.terminal
      if (cols > 0 && rows > 0) {
        this.lastFitDimensions = {columns: cols, rows: rows}
        if (cols !== session.lastSentColumns || rows !== session.lastSentRows) {
          session.lastSentColumns = cols
          session.lastSentRows = rows
          return {columns: cols, rows: rows}
        }
      }
    } catch {
      // 度量失败时不冒泡：保持上一次已知良好尺寸，等下一次 ResizeObserver 回调重试。
    }
    return undefined
  }

  // 异步对齐 PTY：网格已按新尺寸适配（fit 已返回），此刻到 ConPTY 真正生效之间存在
  // 异步窗口。期间到达的输出若按新网格立即写入会错位，故在 onResize 完成前缓冲，
  // 完成后按序刷入——使网格与 PTY 在输出渲染层面原子一致。
  private async syncPty(
    session: TerminalSession,
    dimensions: {columns: number; rows: number},
    onResize: (columns: number, rows: number) => void | Promise<void>,
  ): Promise<void> {
    session.resizeInFlight = true
    try {
      await onResize(dimensions.columns, dimensions.rows)
    } finally {
      session.resizeInFlight = false
      this.flushOutput(session)
    }
  }

  private flushOutput(session: TerminalSession): void {
    if (session.outputQueue.length === 0) {
      return
    }
    const queued = session.outputQueue.splice(0)
    session.terminal.write(queued.join(''))
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
