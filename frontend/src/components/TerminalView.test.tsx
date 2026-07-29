import {cleanup, fireEvent, render, screen, waitFor} from '@testing-library/react'
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
  onData: ReturnType<typeof vi.fn>
  dispose: ReturnType<typeof vi.fn>
  reset: ReturnType<typeof vi.fn>
  triggerSelectionChange(): void
}>)

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
    write = vi.fn()
    onData = vi.fn(() => ({dispose: vi.fn()}))
    getSelection = vi.fn(() => '')
    focus = vi.fn()
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

    constructor() {
      terminalInstances.push(this)
    }
  },
}))

vi.mock('@xterm/addon-fit', () => ({FitAddon: class {fit = vi.fn()}}))
vi.mock('../../wailsjs/runtime/runtime', () => runtime)

import {TerminalView} from './TerminalView'

function runAnimationFrame() {
  const callbacks = Array.from(animationFrameCallbacks.values())
  animationFrameCallbacks.clear()
  callbacks.forEach((callback) => callback(performance.now()))
}

describe('TerminalView', () => {
  beforeEach(() => {
    terminalInstances.length = 0
    animationFrameCallbacks.clear()
    animationFrameID.next = 0
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
