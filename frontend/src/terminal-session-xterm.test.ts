import {afterAll, afterEach, beforeAll, describe, expect, it, vi} from 'vitest'
import type {TerminalSessionRegistry as TerminalSessionRegistryType} from './terminal-session'

vi.mock('../wailsjs/runtime/runtime', () => ({ClipboardSetText: vi.fn()}))

let TerminalSessionRegistry: typeof import('./terminal-session').TerminalSessionRegistry

beforeAll(async () => {
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => ({
    createLinearGradient: () => ({addColorStop: () => {}}),
  } as unknown as CanvasRenderingContext2D))
  ;({TerminalSessionRegistry} = await import('./terminal-session'))
})

afterAll(() => {
  vi.restoreAllMocks()
})

const terminalKey = JSON.stringify(['task-1', 'terminal-1'])

interface TerminalCoreAccess {
  coreService: {
    isCursorHidden: boolean
  }
}

function sessionTerminal(registry: TerminalSessionRegistryType): {terminal: import('@xterm/xterm').Terminal} {
  const sessions = (registry as unknown as {sessions: Map<string, {terminal: import('@xterm/xterm').Terminal}>}).sessions
  return sessions.get(terminalKey)!
}

function cursorHidden(terminal: import('@xterm/xterm').Terminal): boolean {
  const core = terminal as unknown as {_core: TerminalCoreAccess}
  return core._core.coreService.isCursorHidden
}

describe('TerminalSessionRegistry 与真实 xterm 光标状态', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('重绘后跟随普通输出时不遗留隐藏光标', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const redraw = `${'x'.repeat(100)}\x1b[?25h`

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: redraw})
    vi.advanceTimersByTime(32)
    const terminal = sessionTerminal(registry).terminal
    vi.advanceTimersByTime(18)
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'plain output'})
    vi.advanceTimersByTime(32)
    vi.runOnlyPendingTimers()

    expect(cursorHidden(terminal)).toBe(false)
  })

  it('程序主动隐藏光标时不会被应用层强制显示', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: `\x1b[?25l${'x'.repeat(100)}`})
    vi.advanceTimersByTime(32)
    vi.runOnlyPendingTimers()
    const terminal = sessionTerminal(registry).terminal

    expect(cursorHidden(terminal)).toBe(true)
  })

  it('重绘后字符输入和左右方向键不会让光标保持隐藏', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const redraw = `${'x'.repeat(100)}\x1b[?25h`

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: redraw})
    vi.advanceTimersByTime(32)
    const terminal = sessionTerminal(registry).terminal
    terminal.input('\x1b[D')
    terminal.input('a')
    vi.runOnlyPendingTimers()

    expect(cursorHidden(terminal)).toBe(false)
  })
})
