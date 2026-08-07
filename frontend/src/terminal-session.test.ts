import {beforeEach, describe, expect, it, vi} from 'vitest'

const terminalInstances = vi.hoisted(() => [] as Array<{
  cols: number
  rows: number
  attachCustomKeyEventHandler: ReturnType<typeof vi.fn>
  element?: HTMLElement
  dispose: ReturnType<typeof vi.fn>
  focus: ReturnType<typeof vi.fn>
  getSelection: ReturnType<typeof vi.fn>
  loadAddon: ReturnType<typeof vi.fn>
  onData: ReturnType<typeof vi.fn>
  onDataDisposable: {dispose: ReturnType<typeof vi.fn>}
  onSelectionChange: ReturnType<typeof vi.fn>
  onSelectionDisposable: {dispose: ReturnType<typeof vi.fn>}
  open: ReturnType<typeof vi.fn>
  paste: ReturnType<typeof vi.fn>
  options: {scrollback?: number, theme?: unknown}
  refresh: ReturnType<typeof vi.fn>
  triggerSelectionChange(): void
  write: ReturnType<typeof vi.fn>
}>)
const fitAddonInstances = vi.hoisted(() => [] as Array<{fit: ReturnType<typeof vi.fn>}>)
const runtime = vi.hoisted(() => ({ClipboardSetText: vi.fn()}))

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 100
    rows = 30
    attachCustomKeyEventHandler = vi.fn()
    element: HTMLElement | undefined
    dispose = vi.fn()
    focus = vi.fn()
    getSelection = vi.fn(() => '')
    loadAddon = vi.fn()
    onDataDisposable = {dispose: vi.fn()}
    onData = vi.fn(() => {
      this.onDataDisposable = {dispose: vi.fn()}
      return this.onDataDisposable
    })
    onSelectionDisposable = {dispose: vi.fn()}
    selectionChangeListener: (() => void) | undefined
    onSelectionChange = vi.fn((listener: () => void) => {
      this.selectionChangeListener = listener
      this.onSelectionDisposable = {dispose: vi.fn()}
      return this.onSelectionDisposable
    })
    open = vi.fn((container: HTMLElement) => {
      this.element = document.createElement('div')
      container.append(this.element)
    })
    paste = vi.fn()
    options: {scrollback?: number, theme?: unknown}
    refresh = vi.fn()
    write = vi.fn()

    triggerSelectionChange() {
      this.selectionChangeListener?.()
    }

    constructor(options: {scrollback?: number, theme?: unknown}) {
      this.options = options
      terminalInstances.push(this)
    }
  },
}))

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class {
    fit = vi.fn()

    constructor() {
      fitAddonInstances.push(this)
    }
  },
}))

vi.mock('../wailsjs/runtime/runtime', () => runtime)

import {TerminalSessionRegistry, terminalVisualTheme} from './terminal-session'

const terminal = {id: 'terminal-1', taskId: 'task-1', state: 'active' as const}

describe('TerminalSessionRegistry', () => {
  beforeEach(() => {
    terminalInstances.length = 0
    fitAddonInstances.length = 0
    runtime.ClipboardSetText.mockReset()
    runtime.ClipboardSetText.mockResolvedValue(true)
  })

  it('直接写入按终端键持有的会话，并将滚屏限制为 1000 行', () => {
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'first output chunk'})

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].options.scrollback).toBe(1000)
    expect(terminalInstances[0].write).toHaveBeenCalledWith('first output chunk')
  })

  it('将快捷输入作为模拟粘贴写入指定活动会话，不追加执行字符', () => {
    const onWrite = vi.fn()
    const registry = new TerminalSessionRegistry(onWrite)
	registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'ready'})

    expect(registry.pasteQuickInput('task-1', 'terminal-1', 'git status\ngit push')).toBe(true)
    expect(terminalInstances[0].paste).toHaveBeenCalledWith('git status\ngit push')
    expect(onWrite).not.toHaveBeenCalled()

    registry.dispose('task-1', 'terminal-1')
    expect(registry.pasteQuickInput('task-1', 'terminal-1', 'ignored')).toBe(false)
    expect(terminalInstances[0].paste).toHaveBeenCalledTimes(1)
  })

  it('仅为活动会话注册终端按键处理器', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const handler = vi.fn(() => false)

    expect(registry.setCustomKeyEventHandler('task-1', 'terminal-1', handler)).toBe(false)
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'ready'})

    expect(registry.setCustomKeyEventHandler('task-1', 'terminal-1', handler)).toBe(true)
    expect(terminalInstances[0].attachCustomKeyEventHandler).toHaveBeenCalledWith(handler)

    registry.dispose('task-1', 'terminal-1')
    expect(registry.setCustomKeyEventHandler('task-1', 'terminal-1', handler)).toBe(false)
  })

  it('重新挂载同一终端时复用实例且不回放已写入内容', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const firstContainer = document.createElement('div')
    const secondContainer = document.createElement('div')
    const onResize = vi.fn()

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'existing output'})
    registry.attach(terminal, firstContainer, terminalVisualTheme('dark'), onResize)
    registry.attach(terminal, secondContainer, terminalVisualTheme('dark'), onResize)

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(1)
    expect(secondContainer.contains(terminalInstances[0].element!)).toBe(true)
    expect(fitAddonInstances[0].fit).toHaveBeenCalledTimes(2)
    expect(onResize).toHaveBeenCalledWith(100, 30)
  })

  it('挂载禁用策略的终端时不自动复制选区', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')

    registry.attach({...terminal, disableTaskAIMouseClipboard: true}, container, terminalVisualTheme('light'), vi.fn())
    terminalInstances[0].getSelection.mockReturnValue('selected terminal output')
    terminalInstances[0].triggerSelectionChange()

    expect(runtime.ClipboardSetText).not.toHaveBeenCalled()
  })

  it('按 A、B、A 顺序切换时复用两个已有终端根节点', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const terminalB = {id: 'terminal-2', taskId: 'task-1', state: 'active' as const}
    const firstAContainer = document.createElement('div')
    const bContainer = document.createElement('div')
    const secondAContainer = document.createElement('div')

    registry.attach(terminal, firstAContainer, terminalVisualTheme('light'), vi.fn())
    registry.attach(terminalB, bContainer, terminalVisualTheme('light'), vi.fn())
    registry.attach(terminal, secondAContainer, terminalVisualTheme('light'), vi.fn())

    expect(terminalInstances).toHaveLength(2)
    expect(secondAContainer.contains(terminalInstances[0].element!)).toBe(true)
    expect(terminalInstances.every((instance) => instance.write.mock.calls.length === 0)).toBe(true)
  })

  it('终端退出后释放会话并忽略迟到的输出', () => {
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'before exit'})
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'exited'})
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'late output'})

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].dispose).toHaveBeenCalledOnce()
    expect(terminalInstances[0].onDataDisposable.dispose).toHaveBeenCalledOnce()
    expect(terminalInstances[0].onSelectionDisposable.dispose).toHaveBeenCalledOnce()
    expect(terminalInstances[0].write).toHaveBeenCalledOnce()
  })

  it('任务级和全局释放不会处置其他任务的活动会话', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const terminalB = {id: 'terminal-2', taskId: 'task-2', state: 'active' as const}

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'task one'})
    registry.handleTerminalEvent({...terminalB, terminalId: terminalB.id, type: 'output', data: 'task two'})
    registry.disposeTask('task-1')

    expect(terminalInstances[0].dispose).toHaveBeenCalledOnce()
    expect(terminalInstances[1].dispose).not.toHaveBeenCalled()

    registry.disposeAll()
    expect(terminalInstances[1].dispose).toHaveBeenCalledOnce()
  })
})
