import {cleanup, fireEvent, render, screen, waitFor} from '@testing-library/react'
import {ThemeProvider, createTheme} from '@mui/material'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

const terminalInstances = vi.hoisted(() => [] as Array<{
  cols: number
  rows: number
  getSelection: ReturnType<typeof vi.fn>
  focus: ReturnType<typeof vi.fn>
  loadAddon: ReturnType<typeof vi.fn>
  onSelectionChange: ReturnType<typeof vi.fn>
  open: ReturnType<typeof vi.fn>
  write: ReturnType<typeof vi.fn>
  refresh: ReturnType<typeof vi.fn>
  onData: ReturnType<typeof vi.fn>
  dispose: ReturnType<typeof vi.fn>
  reset: ReturnType<typeof vi.fn>
  options: {theme?: {background?: string}}
  triggerSelectionChange(): void
}>)

const fitAddonInstances = vi.hoisted(() => [] as Array<{fit: ReturnType<typeof vi.fn>}>)
const pendingWriteCallbacks = vi.hoisted(() => [] as Array<() => void>)
const animationFrameCallbacks = vi.hoisted(() => new Map<number, FrameRequestCallback>())
const animationFrameID = vi.hoisted(() => ({next: 0}))

const runtime = vi.hoisted(() => ({
  ClipboardGetText: vi.fn(),
  ClipboardSetText: vi.fn(),
}))

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 100
    rows = 30
    loadAddon = vi.fn()
    open = vi.fn()
    write = vi.fn((_data: string, callback?: () => void) => {
      if (callback) pendingWriteCallbacks.push(callback)
    })
    refresh = vi.fn()
    onData = vi.fn(() => ({dispose: vi.fn()}))
    getSelection = vi.fn(() => '')
    focus = vi.fn()
    options: {theme?: {background?: string}}
    selectionChangeListener: (() => void) | undefined
    onSelectionChange = vi.fn((listener: () => void) => {
      this.selectionChangeListener = listener
      return {dispose: vi.fn()}
    })
    dispose = vi.fn()
    reset = vi.fn()

    triggerSelectionChange() {
      this.selectionChangeListener?.()
    }

    constructor(options: {theme?: {background?: string}}) {
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
vi.mock('../../wailsjs/runtime/runtime', () => runtime)

import {TerminalView} from './TerminalView'

function runAnimationFrame() {
  const callbacks = Array.from(animationFrameCallbacks.values())
  animationFrameCallbacks.clear()
  callbacks.forEach((callback) => callback(performance.now()))
}

function flushPendingWriteCallbacks() {
  for (const callback of pendingWriteCallbacks.splice(0)) {
    callback()
  }
}

function notifyResizeObservers() {
  (window.ResizeObserver as unknown as {notify(): void}).notify()
}

function resetResizeObservers() {
  (window.ResizeObserver as unknown as {reset(): void}).reset()
}

describe('TerminalView', () => {
  beforeEach(() => {
    terminalInstances.length = 0
    fitAddonInstances.length = 0
    pendingWriteCallbacks.length = 0
    animationFrameCallbacks.clear()
    animationFrameID.next = 0
    resetResizeObservers()
    vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
      const id = ++animationFrameID.next
      animationFrameCallbacks.set(id, callback)
      return id
    }))
    vi.stubGlobal('cancelAnimationFrame', vi.fn((id: number) => {
      animationFrameCallbacks.delete(id)
    }))
    runtime.ClipboardGetText.mockReset()
    runtime.ClipboardGetText.mockResolvedValue('')
    runtime.ClipboardSetText.mockReset()
    runtime.ClipboardSetText.mockResolvedValue(true)
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('右侧终端标题在可收缩容器中单行裁剪而不显示省略号', () => {
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active', title: '这是用于验证右侧终端标题布局的超长真实终端名称'}}
        onWrite={vi.fn()}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    const title = screen.getByTestId('terminal-view-title')
    expect(title).toHaveStyle({whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'clip'})
    expect(screen.getByTestId('terminal-view-title-container')).toHaveStyle({flex: '1', minWidth: '0'})
  })

  it('暗色模式使用更深的庭院终端底色', () => {
    render(
      <ThemeProvider theme={createTheme({palette: {mode: 'dark'}})}>
        <TerminalView
          terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active'}}
          onWrite={vi.fn()}
          onResize={vi.fn()}
          onClose={vi.fn()}
        />
      </ThemeProvider>,
    )

    expect(terminalInstances[0].options.theme?.background).toBe('#101a14')
  })

  it('亮色模式仍使用深色庭院终端底色', () => {
    render(
      <ThemeProvider theme={createTheme({palette: {mode: 'light'}})}>
        <TerminalView
          terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active'}}
          onWrite={vi.fn()}
          onResize={vi.fn()}
          onClose={vi.fn()}
        />
      </ThemeProvider>,
    )

    expect(terminalInstances[0].options.theme?.background).toBe('#26352e')
  })

  it('挂载活动终端后自动聚焦 xterm 输入区', () => {
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active'}}
        onWrite={vi.fn()}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )
    runAnimationFrame()

    expect(terminalInstances[0].focus).toHaveBeenCalledOnce()
  })

  it('在回放终端历史输出前适配初始网格', () => {
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active', output: 'static terminal output'}}
        onWrite={vi.fn()}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    const instance = terminalInstances[0]
    const fit = fitAddonInstances[0].fit

    expect(fit).toHaveBeenCalledOnce()
    expect(fit.mock.invocationCallOrder[0]).toBeLessThan(instance.write.mock.invocationCallOrder[0])
  })

  it('在静态输出解析完毕后重绘全部可见终端行', () => {
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active', output: 'static terminal output'}}
        onWrite={vi.fn()}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    flushPendingWriteCallbacks()

    expect(terminalInstances[0].refresh).toHaveBeenCalledWith(0, 29)
  })

  it('在 A、B、A 终端切换时分别重绘每个终端的历史输出', () => {
    const onWrite = vi.fn()
    const onResize = vi.fn()
    const onClose = vi.fn()
    const {rerender} = render(
      <TerminalView
        terminal={{id: 'terminal-a', taskId: 'task-1', state: 'active', output: 'output from A'}}
        onWrite={onWrite}
        onResize={onResize}
        onClose={onClose}
      />,
    )
    flushPendingWriteCallbacks()

    rerender(
      <TerminalView
        terminal={{id: 'terminal-b', taskId: 'task-1', state: 'active', output: 'output from B'}}
        onWrite={onWrite}
        onResize={onResize}
        onClose={onClose}
      />,
    )
    flushPendingWriteCallbacks()

    rerender(
      <TerminalView
        terminal={{id: 'terminal-a', taskId: 'task-1', state: 'active', output: 'output from A'}}
        onWrite={onWrite}
        onResize={onResize}
        onClose={onClose}
      />,
    )
    flushPendingWriteCallbacks()

    expect(terminalInstances).toHaveLength(3)
    for (const instance of terminalInstances) {
      expect(instance.refresh).toHaveBeenCalledWith(0, 29)
    }
  })

  it('在终端容器调整尺寸后重绘全部可见行', () => {
    const onResize = vi.fn()
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active', output: 'static terminal output'}}
        onWrite={vi.fn()}
        onResize={onResize}
        onClose={vi.fn()}
      />,
    )
    flushPendingWriteCallbacks()
    terminalInstances[0].refresh.mockClear()

    notifyResizeObservers()

    expect(onResize).toHaveBeenCalledWith(100, 30)
    expect(terminalInstances[0].refresh).toHaveBeenCalledWith(0, 29)
  })

  it('实时追加输出时不触发全量终端重绘', () => {
    const onWrite = vi.fn()
    const onResize = vi.fn()
    const onClose = vi.fn()
    const {rerender} = render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active', output: 'initial output'}}
        onWrite={onWrite}
        onResize={onResize}
        onClose={onClose}
      />,
    )
    flushPendingWriteCallbacks()
    terminalInstances[0].refresh.mockClear()

    rerender(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active', output: 'initial output next'}}
        onWrite={onWrite}
        onResize={onResize}
        onClose={onClose}
      />,
    )

    expect(terminalInstances[0].write).toHaveBeenLastCalledWith(' next')
    expect(terminalInstances[0].refresh).not.toHaveBeenCalled()
  })

  it('终端卸载后不会因延迟的输出解析回调重绘已释放实例', () => {
    const {unmount} = render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active', output: 'static terminal output'}}
        onWrite={vi.fn()}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )
    const instance = terminalInstances[0]

    unmount()
    flushPendingWriteCallbacks()

    expect(instance.refresh).not.toHaveBeenCalled()
  })

  it('选中终端文本后自动写入系统剪贴板', () => {
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active'}}
        onWrite={vi.fn()}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    const instance = terminalInstances[0]
    instance.getSelection.mockReturnValue('selected terminal output')
    instance.triggerSelectionChange()

    expect(runtime.ClipboardSetText).toHaveBeenCalledWith('selected terminal output')
  })

  it('清空终端选区时不覆盖系统剪贴板', () => {
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active'}}
        onWrite={vi.fn()}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    terminalInstances[0].triggerSelectionChange()

    expect(runtime.ClipboardSetText).not.toHaveBeenCalled()
  })

  it('右键终端时将系统剪贴板内容写入终端', async () => {
    const onWrite = vi.fn()
    runtime.ClipboardGetText.mockResolvedValue('paste from system clipboard')
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active'}}
        onWrite={onWrite}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    const event = new MouseEvent('contextmenu', {bubbles: true, cancelable: true})
    fireEvent(screen.getByTestId('terminal-content'), event)

    expect(event.defaultPrevented).toBe(true)
    await waitFor(() => expect(onWrite).toHaveBeenCalledWith('paste from system clipboard'))
  })

  it('右键终端时不写入空剪贴板内容', async () => {
    const onWrite = vi.fn()
    render(
      <TerminalView
        terminal={{id: 'terminal-1', taskId: 'task-1', state: 'active'}}
        onWrite={onWrite}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    fireEvent.contextMenu(screen.getByTestId('terminal-content'))

    await waitFor(() => expect(runtime.ClipboardGetText).toHaveBeenCalledTimes(1))
    expect(onWrite).not.toHaveBeenCalled()
  })
})
