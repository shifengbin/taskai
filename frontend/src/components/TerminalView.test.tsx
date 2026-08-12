import {act, cleanup, fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

const terminalInstances = vi.hoisted(() => [] as Array<{
  buffer: {
    active: {
      getLine: ReturnType<typeof vi.fn>
      type: 'normal'
      viewportY: number
    }
  }
  cols: number
  rows: number
  attachCustomKeyEventHandler: ReturnType<typeof vi.fn>
  element?: HTMLElement
  getSelection: ReturnType<typeof vi.fn>
  focus: ReturnType<typeof vi.fn>
  loadAddon: ReturnType<typeof vi.fn>
  onData: ReturnType<typeof vi.fn>
  onSelectionChange: ReturnType<typeof vi.fn>
	onScroll: ReturnType<typeof vi.fn>
  onWriteParsed: ReturnType<typeof vi.fn>
  open: ReturnType<typeof vi.fn>
  paste: ReturnType<typeof vi.fn>
  options: {fontSize?: number, scrollback?: number, theme?: {background?: string}}
  dispose: ReturnType<typeof vi.fn>
  refresh: ReturnType<typeof vi.fn>
	write: ReturnType<typeof vi.fn>
  triggerCustomKeyEvent(event: KeyboardEvent): boolean | undefined
  triggerSelectionChange(): void
}> )
const fitAddonInstances = vi.hoisted(() => [] as Array<{fit: ReturnType<typeof vi.fn>}>)
const animationFrameCallbacks = vi.hoisted(() => new Map<number, FrameRequestCallback>())
const animationFrameID = vi.hoisted(() => ({next: 0}))
const runtime = vi.hoisted(() => ({ClipboardGetText: vi.fn(), ClipboardSetText: vi.fn(), OnFileDrop: vi.fn(), OnFileDropOff: vi.fn()}))
const api = vi.hoisted(() => ({writeTerminalFilePaths: vi.fn()}))
const terminalBufferCell = {
  getChars: () => '',
  getWidth: () => 1,
  getFgColorMode: () => 0,
  getFgColor: () => 0,
  getBgColorMode: () => 0,
  getBgColor: () => 0,
  isBold: () => 0,
  isDim: () => 0,
  isItalic: () => 0,
  isUnderline: () => 0,
  isBlink: () => 0,
  isInverse: () => 0,
  isInvisible: () => 0,
  isStrikethrough: () => 0,
  isOverline: () => 0,
}

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 100
    rows = 30
    attachCustomKeyEventHandler = vi.fn((handler: (event: KeyboardEvent) => boolean) => {
      this.customKeyEventHandler = handler
    })
    element: HTMLElement | undefined
    getSelection = vi.fn(() => '')
    focus = vi.fn()
    loadAddon = vi.fn()
    onData = vi.fn(() => ({dispose: vi.fn()}))
    onSelectionChange = vi.fn((listener: () => void) => {
      this.selectionChangeListener = listener
      return {dispose: vi.fn()}
    })
		onScroll = vi.fn(() => ({dispose: vi.fn()}))
    onWriteParsed = vi.fn(() => ({dispose: vi.fn()}))
    open = vi.fn((container: HTMLElement) => {
      this.element = document.createElement('div')
      container.append(this.element)
    })
    paste = vi.fn()
    options: {fontSize?: number, scrollback?: number, theme?: {background?: string}}
    dispose = vi.fn()
    refresh = vi.fn()
		write = vi.fn()
    buffer = {
      active: {
        getLine: vi.fn(() => ({getCell: () => terminalBufferCell})),
        type: 'normal' as const,
        viewportY: 0,
      },
    }
    customKeyEventHandler: ((event: KeyboardEvent) => boolean) | undefined
    selectionChangeListener: (() => void) | undefined

    triggerCustomKeyEvent(event: KeyboardEvent) {
      return this.customKeyEventHandler?.(event)
    }

    triggerSelectionChange() {
      this.selectionChangeListener?.()
    }

    constructor(options: {fontSize?: number, scrollback?: number, theme?: {background?: string}}) {
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
import {TooltipProvider} from './ui'
import {TerminalSessionRegistry, terminalVisualTheme} from '../terminal-session'

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

  it('右侧终端标题栏仅保留始终可见的快捷输入入口', () => {
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(vi.fn())} onResize={vi.fn()} onClose={vi.fn()} />)

    expect(screen.getByTestId('terminal-view-header').parentElement).toHaveStyle({gridTemplateRows: '40px minmax(0, 1fr)'})
    const header = screen.getByTestId('terminal-view-header')
    const actions = screen.getByTestId('terminal-view-actions')
    expect(header).not.toHaveClass('taskai-contextual-container')
    expect(actions).toHaveClass('taskai-terminal__quick-input-actions')
    expect(actions).not.toHaveClass('taskai-contextual-actions')
    expect(actions).toContainElement(within(actions).getByRole('button', {name: /^快捷输入（/}))
    expect(within(header).queryByRole('button', {name: '关闭终端'})).not.toBeInTheDocument()
    expect(within(header).queryByRole('status')).not.toBeInTheDocument()
  })

  it('从终端热键打开搜索并以模拟粘贴插入，不追加 Enter', async () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        quickInputs={[
          {id: 'quick-1', name: '状态', content: 'git status'},
          {id: 'quick-2', name: '部署', content: 'git push origin main'},
        ]}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )
    runAnimationFrame()

    await act(async () => {
      expect(terminalInstances[0].triggerCustomKeyEvent(new KeyboardEvent('keydown', {key: 'P', ctrlKey: true, shiftKey: true}))).toBe(false)
    })
    const search = await screen.findByRole('textbox', {name: '搜索快捷输入'})
    expect(search).toHaveFocus()
    fireEvent.change(search, {target: {value: 'push'}})
    fireEvent.keyDown(search, {key: 'Enter'})

    expect(terminalInstances[0].paste).toHaveBeenCalledWith('git push origin main')
    expect(terminalInstances[0].paste).not.toHaveBeenCalledWith('git push origin main\n')
    expect(screen.queryByRole('textbox', {name: '搜索快捷输入'})).not.toBeInTheDocument()
    expect(terminalInstances[0].focus).toHaveBeenCalledTimes(2)
  })

  it('支持方向键选择、显示当前项、Escape 关闭并将焦点恢复到终端', async () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        quickInputs={[
          {id: 'quick-1', name: '状态', content: 'git status'},
          {id: 'quick-2', name: '部署', content: 'git push origin main'},
        ]}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )
    runAnimationFrame()

    fireEvent.click(screen.getByRole('button', {name: '快捷输入（Ctrl+Shift+P）'}))
    const search = await screen.findByRole('textbox', {name: '搜索快捷输入'})
    const status = screen.getByRole('option', {name: /状态/})
    const deploy = screen.getByRole('option', {name: /部署/})

    expect(status).toHaveAttribute('data-quick-input-selected', 'true')
    expect(deploy).toHaveAttribute('data-quick-input-selected', 'false')
    fireEvent.keyDown(search, {key: 'ArrowDown'})

    expect(deploy).toHaveAttribute('aria-selected', 'true')
    expect(deploy).toHaveAttribute('data-quick-input-selected', 'true')
    expect(status).toHaveAttribute('data-quick-input-selected', 'false')
    expect(search).toHaveFocus()
    fireEvent.keyDown(search, {key: 'Escape'})

    expect(screen.queryByRole('textbox', {name: '搜索快捷输入'})).not.toBeInTheDocument()
    expect(terminalInstances[0].focus).toHaveBeenCalledTimes(2)
  })

  it('会话关闭后提示无法插入快捷输入', async () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const onError = vi.fn()
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        quickInputs={[{id: 'quick-1', name: '状态', content: 'git status'}]}
        onResize={vi.fn()}
        onClose={vi.fn()}
        onError={onError}
      />,
    )
    registry.dispose(terminal.taskId, terminal.id)

    fireEvent.click(screen.getByRole('button', {name: '快捷输入（Ctrl+Shift+P）'}))
    fireEvent.click(await screen.findByRole('option', {name: /状态/}))

    expect(onError).toHaveBeenCalledWith(expect.objectContaining({message: '终端已关闭，无法插入快捷输入'}))
  })

  it('鼠标选中非空文本后打开备注输入并保存到当前终端', () => {
    const onAddNote = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        noteTemplate={{originalPrefix: '原文：', notePrefix: '备注：', listSuffix: ''}}
        onAddNote={onAddNote}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )
    terminalInstances[0].getSelection.mockReturnValue('编译失败')

    fireEvent.pointerUp(screen.getByTestId('terminal-content'), {clientX: 32, clientY: 40})
    act(() => runAnimationFrame())

    const addNote = screen.getByRole('button', {name: '添加备注'})
    expect(addNote).toHaveClass('opacity-70')
    expect(addNote).toHaveStyle({left: '0px', top: '0px'})
    fireEvent.click(addNote)

    expect(screen.getByRole('dialog')).toHaveTextContent('编译失败')
    const noteInput = screen.getByRole('textbox', {name: '备注内容'})
    fireEvent.pointerDown(noteInput)
    expect(screen.getByRole('dialog')).toHaveTextContent('编译失败')
    const save = screen.getByRole('button', {name: '保存备注'})
    expect(save).toBeDisabled()
    fireEvent.change(noteInput, {target: {value: '检查依赖版本'}})
    fireEvent.click(save)

    expect(onAddNote).toHaveBeenCalledWith({original: '编译失败', note: '检查依赖版本'})
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(terminalInstances[0].focus).toHaveBeenCalled()
  })

	it('点击终端其他区域时收起选区备注入口', () => {
		const registry = new TerminalSessionRegistry(vi.fn())
		render(<TerminalView terminal={terminal} sessionRegistry={registry} onResize={vi.fn()} onClose={vi.fn()}/>)
		terminalInstances[0].getSelection.mockReturnValue('编译失败')

		fireEvent.pointerUp(screen.getByTestId('terminal-content'), {clientX: 32, clientY: 40})
		act(() => runAnimationFrame())
		expect(screen.getByRole('button', {name: '添加备注'})).toBeInTheDocument()

		fireEvent.pointerDown(screen.getByTestId('terminal-view-header'))
		expect(screen.queryByRole('button', {name: '添加备注'})).not.toBeInTheDocument()
	})

  it('空选区或禁用鼠标剪贴板的终端不显示备注入口', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const {rerender} = render(<TerminalView terminal={terminal} sessionRegistry={registry} onResize={vi.fn()} onClose={vi.fn()}/>)

    fireEvent.pointerUp(screen.getByTestId('terminal-content'), {clientX: 20, clientY: 20})
    runAnimationFrame()
    expect(screen.queryByRole('button', {name: '添加备注'})).not.toBeInTheDocument()

    terminalInstances[0].getSelection.mockReturnValue('不应读取')
    rerender(<TerminalView terminal={{...terminal, disableTaskAIMouseClipboard: true}} sessionRegistry={registry} onResize={vi.fn()} onClose={vi.fn()}/>)
    fireEvent.pointerUp(screen.getByTestId('terminal-content'), {clientX: 20, clientY: 20})
    runAnimationFrame()

    expect(screen.queryByRole('button', {name: '添加备注'})).not.toBeInTheDocument()
  })

  it('默认勾选操作后清空时复制当前终端备注汇总并清空', async () => {
    const onClearNotes = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        notes={[{original: '一', note: '甲'}, {original: '二', note: '乙'}]}
        noteTemplate={{originalPrefix: '原文：', notePrefix: '备注：', listSuffix: '请处理'}}
        onClearNotes={onClearNotes}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByRole('button', {name: '终端备注（2 条）'}))

    expect(screen.getByRole('checkbox', {name: '操作后清空'})).toBeChecked()
    fireEvent.click(screen.getByRole('button', {name: '复制到剪贴板'}))

    await waitFor(() => expect(runtime.ClipboardSetText).toHaveBeenCalledWith('原文：一\n备注：甲\n原文：二\n备注：乙\n请处理\n'))
    expect(onClearNotes).toHaveBeenCalledOnce()
    expect(terminalInstances[0].paste).not.toHaveBeenCalled()
  })

  it('取消操作后清空时复制和发送均保留当前终端备注', async () => {
    const onClearNotes = vi.fn()
		const onClearNotesAfterActionChange = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        notes={[{original: '一', note: '甲'}, {original: '二', note: '乙'}]}
        noteTemplate={{originalPrefix: '原文：', notePrefix: '备注：', listSuffix: '请处理'}}
        clearNotesAfterAction={false}
        onClearNotes={onClearNotes}
			onClearNotesAfterActionChange={onClearNotesAfterActionChange}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByRole('button', {name: '终端备注（2 条）'}))

    const clearAfterAction = screen.getByRole('checkbox', {name: '操作后清空'})
		expect(clearAfterAction).not.toBeChecked()
		fireEvent.click(clearAfterAction)
		expect(onClearNotesAfterActionChange).toHaveBeenCalledWith(true)
    fireEvent.click(screen.getByRole('button', {name: '复制到剪贴板'}))
    await waitFor(() => expect(runtime.ClipboardSetText).toHaveBeenCalledWith('原文：一\n备注：甲\n原文：二\n备注：乙\n请处理\n'))
    expect(onClearNotes).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', {name: '终端备注（2 条）'}))
    fireEvent.click(screen.getByRole('button', {name: '发送到终端'}))

    expect(terminalInstances[0].paste).toHaveBeenCalledWith('原文：一\n备注：甲\n原文：二\n备注：乙\n请处理\n')
    expect(onClearNotes).not.toHaveBeenCalled()
  })

  it('从标题栏发送当前终端备注，并在发送后清空', () => {
    const onClearNotes = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        notes={[{original: '一', note: '甲'}, {original: '二', note: '乙'}]}
        noteTemplate={{originalPrefix: '原文：', notePrefix: '备注：', listSuffix: '请处理'}}
        onClearNotes={onClearNotes}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByRole('button', {name: '终端备注（2 条）'}))

    expect(screen.getByText('原文：一')).toBeInTheDocument()
    expect(screen.getByText('备注：乙')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', {name: '发送到终端'}))

    expect(terminalInstances[0].paste).toHaveBeenCalledWith('原文：一\n备注：甲\n原文：二\n备注：乙\n请处理\n')
    expect(onClearNotes).toHaveBeenCalledOnce()
  })

  it('目标会话关闭时仍清空备注并提示错误', () => {
    const onClearNotes = vi.fn()
    const onError = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        notes={[{original: '一', note: '甲'}]}
        onClearNotes={onClearNotes}
        onError={onError}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )
    registry.dispose(terminal.taskId, terminal.id)

    fireEvent.click(screen.getByRole('button', {name: '终端备注（1 条）'}))
    fireEvent.click(screen.getByRole('button', {name: '发送到终端'}))

    expect(onClearNotes).toHaveBeenCalledOnce()
    expect(onError).toHaveBeenCalledWith(expect.objectContaining({message: '终端已关闭，无法发送备注'}))
  })

  it('双击标题后将规范化别名交给父级', () => {
    const onAliasChange = vi.fn()
    const namedTerminal = {...terminal, title: 'zsh'}
    render(
      <TerminalView
        terminal={namedTerminal}
        sessionRegistry={new TerminalSessionRegistry(vi.fn())}
        onResize={vi.fn()}
        onClose={vi.fn()}
        onAliasChange={onAliasChange}
      />,
    )

    fireEvent.doubleClick(screen.getByTestId('terminal-view-title'))
    fireEvent.change(screen.getByRole('textbox', {name: '终端别名'}), {target: {value: ' 前端调试 '}})
    fireEvent.keyDown(screen.getByRole('textbox', {name: '终端别名'}), {key: 'Enter'})

    expect(onAliasChange).toHaveBeenCalledWith('前端调试')
  })

  it('右侧标题栏继续使用默认的终端提示定位', async () => {
    render(
      <TooltipProvider delayDuration={0}>
        <TerminalView
          terminal={{...terminal, title: 'npm run dev', command: 'zsh', alias: '前端调试'}}
          sessionRegistry={new TerminalSessionRegistry(vi.fn())}
          onResize={vi.fn()}
          onClose={vi.fn()}
        />
      </TooltipProvider>,
    )

    fireEvent.pointerMove(screen.getByTestId('terminal-view-title'))

    const tooltip = await screen.findByTestId('terminal-alias-details')
    expect(tooltip).toHaveAttribute('data-side', 'top')
    expect(tooltip).toHaveAttribute('data-align', 'center')
    expect(tooltip).not.toHaveClass('pointer-events-none')
  })

  it('将已配置的 Shift+Enter 作为一次有序输入写入终端', () => {
    const onWrite = vi.fn()
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={new TerminalSessionRegistry(onWrite)}
        terminalShortcuts={[{id: 'shortcut-1', shortcut: 'Shift+Enter', steps: [{kind: 'text', text: '\\'}, {kind: 'key', key: 'Enter', modifiers: []}]}]}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    const handled = terminalInstances[0].triggerCustomKeyEvent(new KeyboardEvent('keydown', {key: 'Enter', shiftKey: true, cancelable: true}))

    expect(handled).toBe(false)
    expect(onWrite).toHaveBeenCalledOnce()
    expect(onWrite).toHaveBeenCalledWith('task-1', 'terminal-1', '\\\r')
  })

  it('快捷键在生效程序范围外透传原始按键而不写入', () => {
    const onWrite = vi.fn()
    render(
      <TerminalView
        terminal={{...terminal, command: 'codex'}}
        sessionRegistry={new TerminalSessionRegistry(onWrite)}
        terminalShortcuts={[{id: 'shortcut-1', shortcut: 'Shift+Enter', steps: [{kind: 'text', text: '\\'}, {kind: 'key', key: 'Enter', modifiers: []}], includePrograms: ['powershell']}]}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    const handled = terminalInstances[0].triggerCustomKeyEvent(new KeyboardEvent('keydown', {key: 'Enter', shiftKey: true, cancelable: true}))

    expect(handled).toBe(true)
    expect(onWrite).not.toHaveBeenCalled()
  })

  it('快捷键在生效程序范围内仍写入已配置动作', () => {
    const onWrite = vi.fn()
    render(
      <TerminalView
        terminal={{...terminal, command: 'codex.exe'}}
        sessionRegistry={new TerminalSessionRegistry(onWrite)}
        terminalShortcuts={[{id: 'shortcut-1', shortcut: 'Shift+Enter', steps: [{kind: 'text', text: '\\'}, {kind: 'key', key: 'Enter', modifiers: []}], includePrograms: ['codex']}]}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    const handled = terminalInstances[0].triggerCustomKeyEvent(new KeyboardEvent('keydown', {key: 'Enter', shiftKey: true, cancelable: true}))

    expect(handled).toBe(false)
    expect(onWrite).toHaveBeenCalledOnce()
    expect(onWrite).toHaveBeenCalledWith('task-1', 'terminal-1', '\\\r')
  })

  it('快捷键目标终端关闭后停止动作并报告不可用', () => {
    const onWrite = vi.fn()
    const onError = vi.fn()
    const registry = new TerminalSessionRegistry(onWrite)
    render(
      <TerminalView
        terminal={terminal}
        sessionRegistry={registry}
        terminalShortcuts={[{id: 'shortcut-1', shortcut: 'Shift+Enter', steps: [{kind: 'text', text: '\\'}, {kind: 'key', key: 'Enter', modifiers: []}]}]}
        onResize={vi.fn()}
        onClose={vi.fn()}
        onError={onError}
      />,
    )
    registry.dispose(terminal.taskId, terminal.id)

    const handled = terminalInstances[0].triggerCustomKeyEvent(new KeyboardEvent('keydown', {key: 'Enter', shiftKey: true, cancelable: true}))

    expect(handled).toBe(false)
    expect(onWrite).not.toHaveBeenCalled()
    expect(onError).toHaveBeenCalledWith(expect.objectContaining({message: '终端已关闭，无法执行快捷键动作'}))
  })

  it('注入保存的独立终端配色', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    render(
      <TerminalView
        {...({terminalTheme: {background: '#102030', foreground: '#E0F0FF'}} as {})}
        terminal={terminal}
        sessionRegistry={registry}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )
    expect(terminalInstances[0].options.theme?.background).toBe('#102030')
    expect(screen.getByTestId('terminal-content')).toHaveStyle({backgroundColor: '#102030'})
  })

  it('挂载活动终端后限制滚屏并自动聚焦 xterm 输入区', () => {
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(vi.fn())} onResize={vi.fn()} onClose={vi.fn()} />)
    runAnimationFrame()

    expect(terminalInstances[0].options.scrollback).toBe(1000)
    expect(terminalInstances[0].focus).toHaveBeenCalledOnce()
  })

  it('异常退出快照只读且不注册后端终端交互', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const onResize = vi.fn()
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'failure output'})
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'exited', exitReason: 'unexpected', exitCode: 1})

    render(<TerminalView terminal={{...terminal, state: 'exited'}} sessionRegistry={registry} onResize={onResize} onClose={vi.fn()} />)
    runAnimationFrame()

    expect(onResize).not.toHaveBeenCalled()
    expect(runtime.OnFileDrop).not.toHaveBeenCalled()
    const event = new MouseEvent('contextmenu', {bubbles: true, cancelable: true})
    fireEvent(screen.getByTestId('terminal-content'), event)
    expect(event.defaultPrevented).toBe(false)
    expect(runtime.ClipboardGetText).not.toHaveBeenCalled()
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

  it('保存终端外观后立即重新适配当前终端并同步 PTY 尺寸', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const onResize = vi.fn()
    const view = render(<TerminalView terminal={terminal} sessionRegistry={registry} fontSize={13} onResize={onResize} onClose={vi.fn()} />)
    runAnimationFrame()
    onResize.mockClear()
    terminalInstances[0].refresh.mockClear()

    registry.setAppearance('', 16, terminalVisualTheme())
    view.rerender(<TerminalView terminal={terminal} sessionRegistry={registry} fontSize={16} onResize={onResize} onClose={vi.fn()} />)

    expect(terminalInstances[0].options.fontSize).toBe(16)
    expect(onResize).toHaveBeenCalledWith(100, 30)
    expect(terminalInstances[0].refresh).toHaveBeenCalledWith(0, 29)
  })

  it('选中终端文本后自动写入系统剪贴板', () => {
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(vi.fn())} onResize={vi.fn()} onClose={vi.fn()} />)
    terminalInstances[0].getSelection.mockReturnValue('selected terminal output')
    terminalInstances[0].triggerSelectionChange()

    expect(runtime.ClipboardSetText).toHaveBeenCalledWith('selected terminal output')
  })

  it('右键终端时将多行系统剪贴板内容作为模拟粘贴写入', async () => {
    const onWrite = vi.fn()
    runtime.ClipboardGetText.mockResolvedValue('git status\ngit push')
    render(<TerminalView terminal={terminal} sessionRegistry={new TerminalSessionRegistry(onWrite)} onResize={vi.fn()} onClose={vi.fn()} />)
    const event = new MouseEvent('contextmenu', {bubbles: true, cancelable: true})
    fireEvent(screen.getByTestId('terminal-content'), event)

    expect(event.defaultPrevented).toBe(true)
    await waitFor(() => expect(terminalInstances[0].paste).toHaveBeenCalledWith('git status\ngit push'))
    expect(onWrite).not.toHaveBeenCalled()
  })

  it('右键读取剪贴板期间终端关闭时不写入文本', async () => {
    const onWrite = vi.fn()
    let resolveClipboard: (clipboard: string) => void = () => {}
    runtime.ClipboardGetText.mockImplementation(() => new Promise<string>((resolve) => {
      resolveClipboard = resolve
    }))
    const registry = new TerminalSessionRegistry(onWrite)
    render(<TerminalView terminal={terminal} sessionRegistry={registry} onResize={vi.fn()} onClose={vi.fn()} />)

    const event = new MouseEvent('contextmenu', {bubbles: true, cancelable: true})
    fireEvent(screen.getByTestId('terminal-content'), event)
    registry.dispose('task-1', 'terminal-1')

    await act(async () => {
      resolveClipboard('git status\ngit push')
    })

    expect(event.defaultPrevented).toBe(true)
    expect(terminalInstances[0].paste).not.toHaveBeenCalled()
    expect(onWrite).not.toHaveBeenCalled()
  })

  it('禁用 TaskAI 鼠标剪贴板后不接管选区或右键', () => {
    const onWrite = vi.fn()
    runtime.ClipboardGetText.mockResolvedValue('stale system clipboard')
    render(
      <TerminalView
        terminal={{...terminal, disableTaskAIMouseClipboard: true}}
        sessionRegistry={new TerminalSessionRegistry(onWrite)}
        onResize={vi.fn()}
        onClose={vi.fn()}
      />,
    )
    terminalInstances[0].getSelection.mockReturnValue('selected terminal output')
    terminalInstances[0].triggerSelectionChange()

    expect(runtime.ClipboardSetText).not.toHaveBeenCalled()

    const event = new MouseEvent('contextmenu', {bubbles: true, cancelable: true})
    fireEvent(screen.getByTestId('terminal-content'), event)

    expect(event.defaultPrevented).toBe(false)
    expect(runtime.ClipboardGetText).not.toHaveBeenCalled()
    expect(onWrite).not.toHaveBeenCalled()
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
