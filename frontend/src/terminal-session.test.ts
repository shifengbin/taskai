import {beforeEach, describe, expect, it, vi} from 'vitest'

const terminalInstances = vi.hoisted(() => [] as Array<{
  cols: number
  rows: number
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
  options: {scrollback?: number, theme?: unknown}
  refresh: ReturnType<typeof vi.fn>
  write: ReturnType<typeof vi.fn>
}>)
const fitAddonInstances = vi.hoisted(() => [] as Array<{fit: ReturnType<typeof vi.fn>}>)
const runtime = vi.hoisted(() => ({ClipboardSetText: vi.fn()}))

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 100
    rows = 30
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
    onSelectionChange = vi.fn(() => {
      this.onSelectionDisposable = {dispose: vi.fn()}
      return this.onSelectionDisposable
    })
    open = vi.fn((container: HTMLElement) => {
      this.element = document.createElement('div')
      container.append(this.element)
    })
    options: {scrollback?: number, theme?: unknown}
    refresh = vi.fn()
    write = vi.fn()

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

import {TerminalSessionRegistry, terminalVisualTheme, resolveTerminalCreateDimensions, defaultTerminalCreateDimensions} from './terminal-session'

const terminal = {id: 'terminal-1', taskId: 'task-1', state: 'active' as const}

describe('TerminalSessionRegistry', () => {
  beforeEach(() => {
    terminalInstances.length = 0
    fitAddonInstances.length = 0
    runtime.ClipboardSetText.mockReset()
  })

  it('直接写入按终端键持有的会话，并将滚屏限制为 1000 行', () => {
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'first output chunk'})

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].options.scrollback).toBe(1000)
    expect(terminalInstances[0].write).toHaveBeenCalledWith('first output chunk')
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

  it('lastDimensions 在首次适配前为空，适配后返回最近一次列/行', () => {
    const registry = new TerminalSessionRegistry(vi.fn())

    expect(registry.lastDimensions()).toBeUndefined()

    registry.attach(terminal, document.createElement('div'), terminalVisualTheme('light'), vi.fn())

    expect(registry.lastDimensions()).toEqual({columns: 100, rows: 30})
  })

  it('终端创建尺寸解析：无缓存回退默认，有缓存复用共享面板几何', () => {
    const registry = new TerminalSessionRegistry(vi.fn())

    // 首个终端尚无任何已适配会话 → 回退默认尺寸（不再以脱离容器的猜测驱动首批输出）
    expect(resolveTerminalCreateDimensions(registry.lastDimensions())).toEqual(defaultTerminalCreateDimensions)
    expect(resolveTerminalCreateDimensions({columns: 118, rows: 34})).toEqual({columns: 118, rows: 34})

    registry.attach(terminal, document.createElement('div'), terminalVisualTheme('light'), vi.fn())

    // 已有会话后，复用最近一次 fit 得到的共享面板几何
    expect(resolveTerminalCreateDimensions(registry.lastDimensions())).toEqual({columns: 100, rows: 30})
  })

  it('相同列/行的连续适配不重复同步 PTY，仅在尺寸变化时同步', () => {
    const onResize = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.attach(terminal, document.createElement('div'), terminalVisualTheme('light'), onResize)
    expect(onResize).toHaveBeenCalledTimes(1)
    expect(onResize).toHaveBeenCalledWith(100, 30)

    // 同尺寸再次适配：不应再次同步 PTY
    registry.fitAndRefresh(terminal.taskId, terminal.id, onResize)
    expect(onResize).toHaveBeenCalledTimes(1)

    // 尺寸真正变化后：同步新值
    terminalInstances[0].cols = 124
    terminalInstances[0].rows = 36
    registry.fitAndRefresh(terminal.taskId, terminal.id, onResize)
    expect(onResize).toHaveBeenCalledTimes(2)
    expect(onResize).toHaveBeenLastCalledWith(124, 36)
  })

  it('resize 同步期间到达的输出先缓冲，PTY 同步完成后再按序写入', async () => {
    let resolveResize!: () => void
    const pendingResize = () => new Promise<void>((resolve) => {
      resolveResize = resolve
    })
    const registry = new TerminalSessionRegistry(vi.fn())

    // 建立会话并以同步 onResize 完成首次适配，让 lastSent 落到 100×30
    registry.attach(terminal, document.createElement('div'), terminalVisualTheme('light'), vi.fn())
    await new Promise((resolve) => setTimeout(resolve, 0))

    // 触发一次尺寸变化，但让 PTY 同步挂起：此刻网格已变、PTY 尚未跟上
    terminalInstances[0].cols = 124
    terminalInstances[0].rows = 36
    registry.fitAndRefresh(terminal.taskId, terminal.id, pendingResize)

    // 同步未完成期间到达的输出必须被缓冲，不得立即写入（否则按新网格渲染旧宽度输出→错位）
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'during-resize'})
    expect(terminalInstances[0].write).not.toHaveBeenCalledWith('during-resize')

    // PTY 同步完成后，缓冲内容按到达顺序写入
    resolveResize()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(terminalInstances[0].write).toHaveBeenCalledWith('during-resize')
  })
})
