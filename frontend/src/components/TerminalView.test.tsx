import {cleanup, fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

const terminalInstances = vi.hoisted(() => [] as Array<{
  cols: number
  rows: number
  element?: HTMLElement
  getSelection: ReturnType<typeof vi.fn>
  focus: ReturnType<typeof vi.fn>
  loadAddon: ReturnType<typeof vi.fn>
  onData: ReturnType<typeof vi.fn>
  onSelectionChange: ReturnType<typeof vi.fn>
  open: ReturnType<typeof vi.fn>
  options: {scrollback?: number, theme?: {background?: string}}
  dispose: ReturnType<typeof vi.fn>
  refresh: ReturnType<typeof vi.fn>
  triggerSelectionChange(): void
}> )
const fitAddonInstances = vi.hoisted(() => [] as Array<{fit: ReturnType<typeof vi.fn>}>)
const animationFrameCallbacks = vi.hoisted(() => new Map<number, FrameRequestCallback>())
const animationFrameID = vi.hoisted(() => ({next: 0}))
const runtime = vi.hoisted(() => ({ClipboardGetText: vi.fn(), ClipboardSetText: vi.fn(), OnFileDrop: vi.fn(), OnFileDropOff: vi.fn()}))
const api = vi.hoisted(() => ({writeTerminalFilePaths: vi.fn()}))

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 100
    rows = 30
    element: HTMLElement | undefined
    getSelection = vi.fn(() => '')
    focus = vi.fn()
    loadAddon = vi.fn()
    onData = vi.fn(() => ({dispose: vi.fn()}))
    onSelectionChange = vi.fn((listener: () => void) => {
      this.selectionChangeListener = listener
      return {dispose: vi.fn()}
    })
    open = vi.fn((container: HTMLElement) => {
      this.element = document.createElement('div')
      container.append(this.element)
    })
    options: {scrollback?: number, theme?: {background?: string}}
    dispose = vi.fn()
    refresh = vi.fn()
    selectionChangeListener: (() => void) | undefined

    triggerSelectionChange() {
      this.selectionChangeListener?.()
    }

    constructor(options: {scrollback?: number, theme?: {background?: string}}) {
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
vi.mock('../api', () => ({api}))

import {TerminalView} from './TerminalView'
import {TerminalSessionRegistry} from '../terminal-session'

const terminal = {id: 'terminal-1', taskId: 'task-1', state: 'active' as const}

function runAnimationFrame() {
  const callbacks = Array.from(animationFrameCallbacks.values())
  animationFrameCallbacks.clear()
  callbacks.forEach((callback) => callback(performance.now()))
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
    animationFrameCallbacks.clear()
    animationFrameID.next = 0
    resetResizeObservers()
    vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
      const id = ++animationFrameID.next
      animationFrameCallbacks.set(id, callback)
      return id
    }))
    vi.stubGlobal('cancelAnimationFrame', vi.fn((id: number) => animationFrameCallbacks.delete(id)))
    runtime.ClipboardGetText.mockReset()
    runtime.ClipboardGetText.mockResolvedValue('')
    runtime.ClipboardSetText.mockReset()
    runtime.ClipboardSetText.mockResolvedValue(true)
    runtime.OnFileDrop.mockReset()
    runtime.OnFileDropOff.mockReset()
    api.writeTerminalFilePaths.mockReset()
    api.writeTerminalFilePaths.mockResolvedValue(undefined)
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('右侧终端标题在可收缩容器中单行裁剪而不显示省略号', () => {
    render(
      <TerminalView
        terminal={{...terminal, title: '这是用于验证右侧终端标题布局的超长真实终端名称'}}
        sessionRegistry={new TerminalSessionRegistry(vi.fn())}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    expect(screen.getByTestId('terminal-view-title')).toHaveStyle({whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'clip'})
    expect(screen.getByTestId('terminal-view-title-container')).toHaveStyle({flex: '1', minWidth: '0'})
  })

  it('将关闭按钮限定在终端标题栏的上下文操作容器内', () => {
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(vi.fn())} onResize={vi.fn()} onClose={vi.fn()} />)

    const header = screen.getByTestId('terminal-view-header')
    const actions = screen.getByTestId('terminal-view-actions')
    expect(header).toHaveClass('taskai-contextual-container')
    expect(actions).toHaveClass('taskai-contextual-actions')
    expect(actions).toContainElement(within(actions).getByRole('button', {name: '关闭终端'}))
    expect(actions).not.toContainElement(within(header).getByRole('status', {name: '终端状态：空闲'}))
  })

  it('暗色模式注入快门波普深表面终端底色', () => {
    const darkRegistry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView mode="dark" terminal={terminal} sessionRegistry={darkRegistry} onResize={vi.fn()} onClose={vi.fn()} />,
    )
    expect(terminalInstances[0].options.theme?.background).toBe('#16242B')
  })

  it('挂载活动终端后限制滚屏并自动聚焦 xterm 输入区', () => {
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(vi.fn())} onResize={vi.fn()} onClose={vi.fn()} />)
    runAnimationFrame()

    expect(terminalInstances[0].options.scrollback).toBe(1000)
    expect(terminalInstances[0].focus).toHaveBeenCalledOnce()
  })

  it('视图卸载时保留终端会话，以便再次切换时复用', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const first = render(<TerminalView terminal={terminal} sessionRegistry={registry} onResize={vi.fn()} onClose={vi.fn()} />)
    first.unmount()
    expect(terminalInstances[0].dispose).not.toHaveBeenCalled()

    render(<TerminalView terminal={terminal} sessionRegistry={registry} onResize={vi.fn()} onClose={vi.fn()} />)
    expect(terminalInstances).toHaveLength(1)
  })

  it('在终端容器调整尺寸后重新适配并重绘全部可见行', () => {
    const onResize = vi.fn()
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(vi.fn())} onResize={onResize} onClose={vi.fn()} />)
    terminalInstances[0].refresh.mockClear()
    onResize.mockClear()

    notifyResizeObservers()

    expect(onResize).toHaveBeenCalledWith(100, 30)
    expect(terminalInstances[0].refresh).toHaveBeenCalledWith(0, 29)
  })

  it('选中终端文本后自动写入系统剪贴板', () => {
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(vi.fn())} onResize={vi.fn()} onClose={vi.fn()} />)
    terminalInstances[0].getSelection.mockReturnValue('selected terminal output')
    terminalInstances[0].triggerSelectionChange()

    expect(runtime.ClipboardSetText).toHaveBeenCalledWith('selected terminal output')
  })

  it('右键终端时将系统剪贴板内容写入终端', async () => {
    const onWrite = vi.fn()
    runtime.ClipboardGetText.mockResolvedValue('paste from system clipboard')
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(onWrite)} onResize={vi.fn()} onClose={vi.fn()} />)
    const event = new MouseEvent('contextmenu', {bubbles: true, cancelable: true})
    fireEvent(screen.getByTestId('terminal-content'), event)

    expect(event.defaultPrevented).toBe(true)
    await waitFor(() => expect(onWrite).toHaveBeenCalledWith('task-1', 'terminal-1', 'paste from system clipboard'))
  })

  it('仅将投放到终端内容区的原生文件路径交给后端写入', async () => {
    let onFileDrop: ((x: number, y: number, paths: string[]) => void) | undefined
    runtime.OnFileDrop.mockImplementation((listener: (x: number, y: number, paths: string[]) => void) => {
      onFileDrop = listener
    })
    const onWrite = vi.fn()
    const view = render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(onWrite)} onResize={vi.fn()} onClose={vi.fn()} />)

    expect(runtime.OnFileDrop).toHaveBeenCalledWith(expect.any(Function), true)
    expect(screen.getByTestId('terminal-content')).toHaveStyle({'--wails-drop-target': 'drop'})

    onFileDrop?.(20, 30, ['/tmp/My Project/file.txt'])

    await waitFor(() => expect(api.writeTerminalFilePaths).toHaveBeenCalledWith('task-1', 'terminal-1', ['/tmp/My Project/file.txt']))
    expect(onWrite).not.toHaveBeenCalled()

    view.unmount()
    expect(runtime.OnFileDropOff).toHaveBeenCalledOnce()
  })

  it('忽略空投放，在后端拒绝时不输入原始路径并报告错误', async () => {
    let onFileDrop: ((x: number, y: number, paths: string[]) => void) | undefined
    runtime.OnFileDrop.mockImplementation((listener: (x: number, y: number, paths: string[]) => void) => {
      onFileDrop = listener
    })
    const failure = new Error('终端已关闭')
    api.writeTerminalFilePaths.mockRejectedValueOnce(failure)
    const onWrite = vi.fn()
    const onError = vi.fn()
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(onWrite)} onResize={vi.fn()} onClose={vi.fn()} onError={onError} />)

    onFileDrop?.(20, 30, [])
    expect(api.writeTerminalFilePaths).not.toHaveBeenCalled()

    onFileDrop?.(20, 30, ['/tmp/rejected file.txt'])
    await waitFor(() => expect(api.writeTerminalFilePaths).toHaveBeenCalledOnce())
    expect(onWrite).not.toHaveBeenCalled()
    expect(onError).toHaveBeenCalledWith(failure)
  })

  it('缺少 Wails 拖放运行时时保持终端可用', () => {
    runtime.OnFileDrop.mockImplementation(() => {
      throw new Error('Wails runtime is unavailable')
    })

    expect(() => render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(vi.fn())} onResize={vi.fn()} onClose={vi.fn()} />)).not.toThrow()
    expect(api.writeTerminalFilePaths).not.toHaveBeenCalled()
  })
})
