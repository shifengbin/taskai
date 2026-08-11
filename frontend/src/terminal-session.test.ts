import {beforeEach, describe, expect, it, vi} from 'vitest'

type TerminalCellOverrides = Partial<{
  chars: string
  width: number
  foregroundColor: number
  foregroundColorMode: number
  backgroundColor: number
  backgroundColorMode: number
  bold: number
  dim: number
  italic: number
  underline: number
  blink: number
  inverse: number
  invisible: number
  strikethrough: number
  overline: number
}>

const terminalInstances = vi.hoisted(() => [] as Array<{
  buffer: {
    active: {
      cursorX: number
      cursorY: number
      getLine: ReturnType<typeof vi.fn>
      type: 'normal' | 'alternate'
      viewportY: number
    }
  }
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
	onScroll: ReturnType<typeof vi.fn>
	onScrollDisposable: {dispose: ReturnType<typeof vi.fn>}
  onWriteParsed: ReturnType<typeof vi.fn>
  onWriteParsedDisposable: {dispose: ReturnType<typeof vi.fn>}
  open: ReturnType<typeof vi.fn>
  paste: ReturnType<typeof vi.fn>
  options: {fontFamily?: string, fontSize?: number, scrollback?: number, theme?: unknown}
  refresh: ReturnType<typeof vi.fn>
  triggerSelectionChange(): void
	triggerData(data: string): void
	triggerScroll(viewportY: number): void
  triggerWriteParsed(): void
  write: ReturnType<typeof vi.fn>
  setCell(row: number, column: number, cell: TerminalCellOverrides): void
}>)
const fitAddonInstances = vi.hoisted(() => [] as Array<{fit: ReturnType<typeof vi.fn>}>)
const runtime = vi.hoisted(() => ({ClipboardSetText: vi.fn()}))

function terminalCell(overrides: TerminalCellOverrides = {}) {
  const cell = {
    chars: '',
    width: 1,
    foregroundColor: 0,
    foregroundColorMode: 0,
    backgroundColor: 0,
    backgroundColorMode: 0,
    bold: 0,
    dim: 0,
    italic: 0,
    underline: 0,
    blink: 0,
    inverse: 0,
    invisible: 0,
    strikethrough: 0,
    overline: 0,
    ...overrides,
  }
  return {
    getChars: () => cell.chars,
    getWidth: () => cell.width,
    getFgColor: () => cell.foregroundColor,
    getFgColorMode: () => cell.foregroundColorMode,
    getBgColor: () => cell.backgroundColor,
    getBgColorMode: () => cell.backgroundColorMode,
    isBold: () => cell.bold,
    isDim: () => cell.dim,
    isItalic: () => cell.italic,
    isUnderline: () => cell.underline,
    isBlink: () => cell.blink,
    isInverse: () => cell.inverse,
    isInvisible: () => cell.invisible,
    isStrikethrough: () => cell.strikethrough,
    isOverline: () => cell.overline,
  }
}

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
    dataListener: ((data: string) => void) | undefined
    onData = vi.fn((listener: (data: string) => void) => {
			this.dataListener = listener
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
		onScrollDisposable = {dispose: vi.fn()}
		scrollListener: ((viewportY: number) => void) | undefined
		onScroll = vi.fn((listener: (viewportY: number) => void) => {
			this.scrollListener = listener
			this.onScrollDisposable = {dispose: vi.fn()}
			return this.onScrollDisposable
		})
    onWriteParsedDisposable = {dispose: vi.fn()}
    writeParsedListener: (() => void) | undefined
    onWriteParsed = vi.fn((listener: () => void) => {
      this.writeParsedListener = listener
      this.onWriteParsedDisposable = {dispose: vi.fn()}
      return this.onWriteParsedDisposable
    })
    open = vi.fn((container: HTMLElement) => {
      this.element = document.createElement('div')
      container.append(this.element)
    })
    paste = vi.fn()
    options: {fontFamily?: string, fontSize?: number, scrollback?: number, theme?: unknown}
    refresh = vi.fn()
    write = vi.fn()
    lines = Array.from({length: 30}, () => Array.from({length: 100}, () => terminalCell()))
    buffer = {
      active: {
        cursorX: 0,
        cursorY: 0,
        getLine: vi.fn((row: number) => ({
          getCell: (column: number) => this.lines[row]?.[column],
        })),
        type: 'normal' as const,
        viewportY: 0,
      },
    }

    triggerSelectionChange() {
      this.selectionChangeListener?.()
    }

	triggerData(data: string) {
		this.dataListener?.(data)
	}

		triggerScroll(viewportY: number) {
			this.buffer.active.viewportY = viewportY
			this.scrollListener?.(viewportY)
		}

    triggerWriteParsed() {
      this.writeParsedListener?.()
    }

    setCell(row: number, column: number, overrides: TerminalCellOverrides) {
      this.lines[row][column] = terminalCell(overrides)
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

describe('terminalVisualTheme', () => {
	it('使用保存的独立主题，不随工作台亮暗模式切换', () => {
		const savedTheme = {...terminalVisualTheme(), background: '#102030', foreground: '#E0F0FF'}

		expect(terminalVisualTheme(undefined as never)).toMatchObject({
			background: '#070A16',
			foreground: '#E8ECFF',
		})
		expect(terminalVisualTheme(savedTheme as never)).toMatchObject(savedTheme)
	})

})

describe('TerminalSessionRegistry', () => {
  beforeEach(() => {
    terminalInstances.length = 0
    fitAddonInstances.length = 0
    runtime.ClipboardSetText.mockReset()
    runtime.ClipboardSetText.mockResolvedValue(true)
  })

  it('在创建会话时使用当前已保存的字体，并保持已有会话字体不变', () => {
    let savedFontFamily = 'Fira Code'
    const registry = new TerminalSessionRegistry(vi.fn(), () => savedFontFamily)

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'first'})
    savedFontFamily = 'Cascadia Mono'
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'again'})
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-2', data: 'second'})

    expect(terminalInstances[0].options.fontFamily).toBe('"Fira Code", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace')
    expect(terminalInstances[1].options.fontFamily).toBe('"Cascadia Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace')
  })

  it('创建会话时使用当前已保存的独立主题', () => {
    const savedTheme = {...terminalVisualTheme(), background: '#102030'}
    const registry = new (TerminalSessionRegistry as unknown as new (...arguments_: unknown[]) => TerminalSessionRegistry)(
      vi.fn(),
      () => '',
      () => 13,
      () => savedTheme,
    )

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'first'})

    expect(terminalInstances[0].options.theme).toMatchObject(savedTheme)
  })

  it('创建和批量更新会话时使用已保存字号，保留视图挂载前的输出', () => {
    let savedFontSize = 16
    const registry = new TerminalSessionRegistry(vi.fn(), () => '', () => savedFontSize)
    const container = document.createElement('div')
    const onResize = vi.fn()

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'before attach'})
    expect(terminalInstances[0].options.fontSize).toBe(16)

    savedFontSize = 20
    registry.setFontSize(savedFontSize)
    registry.attach(terminal, container, terminalVisualTheme(), onResize)

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].options.fontSize).toBe(20)
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(1)
    expect(terminalInstances[0].dispose).not.toHaveBeenCalled()
    expect(terminalInstances[0].onDataDisposable.dispose).not.toHaveBeenCalled()
    expect(onResize).toHaveBeenCalledWith(100, 30)
  })

  it('保存后将主题、字体和字号更新到已有会话', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    const updatedTheme = {...terminalVisualTheme(), background: '#102030', foreground: '#E0F0FF'}

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'ready'})
    registry.attach(terminal, container, terminalVisualTheme(), vi.fn())
    ;(registry as unknown as {setAppearance(fontFamily: string, fontSize: number, theme: typeof updatedTheme): void}).setAppearance('Fira Code', 16, updatedTheme)

    expect(terminalInstances[0].options.theme).toMatchObject(updatedTheme)
    expect(terminalInstances[0].options.fontFamily).toBe('"Fira Code", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace')
    expect(terminalInstances[0].options.fontSize).toBe(16)
    expect(terminalInstances[0].refresh).toHaveBeenCalledWith(0, 29)
    expect(terminalInstances[0].write).toHaveBeenCalledWith('ready')
  })

  it('直接写入按终端键持有的会话，并将滚屏限制为 1000 行', () => {
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'first output chunk'})

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].options.scrollback).toBe(1000)
    expect(terminalInstances[0].write).toHaveBeenCalledWith('first output chunk')
  })

  it('仅在解析后的终端画面内容或颜色变化时上报活动', () => {
    const onVisualActivity = vi.fn()
    const registry = new (TerminalSessionRegistry as unknown as new (
      onWrite: (taskID: string, terminalID: string, data: string) => void,
      terminalFontFamily?: () => string,
      terminalFontSize?: () => number,
      terminalTheme?: () => unknown,
      onVisualActivity?: (taskID: string, terminalID: string) => void,
    ) => TerminalSessionRegistry)(vi.fn(), undefined, undefined, undefined, onVisualActivity)

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'pending parse'})
    const session = terminalInstances[0]

    session.setCell(0, 0, {chars: 'A'})
    session.triggerWriteParsed()
    expect(onVisualActivity).toHaveBeenCalledWith('task-1', 'terminal-1')

    onVisualActivity.mockClear()
    session.triggerWriteParsed()
    expect(onVisualActivity).not.toHaveBeenCalled()

    session.setCell(0, 0, {chars: 'A', foregroundColorMode: 1, foregroundColor: 2})
    session.triggerWriteParsed()
    expect(onVisualActivity).toHaveBeenCalledWith('task-1', 'terminal-1')

    onVisualActivity.mockClear()
    session.setCell(0, 0, {chars: ' '})
    session.triggerWriteParsed()
    expect(onVisualActivity).toHaveBeenCalledWith('task-1', 'terminal-1')
  })

  it('将空白单元格变化视为活动，但忽略仅有光标或后续样式的变化', () => {
    const onVisualActivity = vi.fn()
    const registry = new (TerminalSessionRegistry as unknown as new (
      onWrite: (taskID: string, terminalID: string, data: string) => void,
      terminalFontFamily?: () => string,
      terminalFontSize?: () => number,
      terminalTheme?: () => unknown,
      onVisualActivity?: (taskID: string, terminalID: string) => void,
    ) => TerminalSessionRegistry)(vi.fn(), undefined, undefined, undefined, onVisualActivity)

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'pending parse'})
    const session = terminalInstances[0]

    session.buffer.active.cursorX = 12
    session.buffer.active.cursorY = 8
    session.triggerWriteParsed()
    session.triggerWriteParsed()
    expect(onVisualActivity).not.toHaveBeenCalled()

    session.setCell(0, 0, {chars: ' ', backgroundColorMode: 1, backgroundColor: 4})
    session.triggerWriteParsed()
    expect(onVisualActivity).toHaveBeenCalledWith('task-1', 'terminal-1')
  })

	it('用户滚动可见历史后重建基线，不把后续无画面变化的输出视为活动', () => {
		const onVisualActivity = vi.fn()
		const registry = new TerminalSessionRegistry(vi.fn(), undefined, undefined, undefined, onVisualActivity)

		registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'initial'})
		terminalInstances[0].setCell(1, 0, {chars: '滚'})
		terminalInstances[0].triggerScroll(1)
		terminalInstances[0].triggerWriteParsed()

		expect(onVisualActivity).not.toHaveBeenCalled()
	})

  it('在重新适配或更新外观时重置画面基线并在释放时清理解析监听', () => {
    const onVisualActivity = vi.fn()
    const registry = new (TerminalSessionRegistry as unknown as new (
      onWrite: (taskID: string, terminalID: string, data: string) => void,
      terminalFontFamily?: () => string,
      terminalFontSize?: () => number,
      terminalTheme?: () => unknown,
      onVisualActivity?: (taskID: string, terminalID: string) => void,
    ) => TerminalSessionRegistry)(vi.fn(), undefined, undefined, undefined, onVisualActivity)

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'pending parse'})
    const session = terminalInstances[0]

    session.setCell(0, 0, {chars: 'A'})
    registry.fitAndRefresh('task-1', 'terminal-1', vi.fn())
    session.triggerWriteParsed()
    expect(onVisualActivity).not.toHaveBeenCalled()

    session.setCell(0, 0, {chars: 'B'})
    registry.setFontSize(16)
    session.triggerWriteParsed()
    expect(onVisualActivity).not.toHaveBeenCalled()

    session.setCell(0, 0, {chars: 'C'})
    registry.setAppearance('Fira Code', 16, terminalVisualTheme())
    session.triggerWriteParsed()
    expect(onVisualActivity).not.toHaveBeenCalled()

    registry.dispose('task-1', 'terminal-1')
    expect(session.onWriteParsedDisposable.dispose).toHaveBeenCalledOnce()
  })

  it('将多行文本作为模拟粘贴写入指定活动会话，不追加执行字符', () => {
    const onWrite = vi.fn()
    const registry = new TerminalSessionRegistry(onWrite)

    expect(registry.pasteInput('task-1', 'terminal-1', 'ignored before session')).toBe(false)
    expect(onWrite).not.toHaveBeenCalled()

	registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'ready'})

    expect(registry.pasteInput('task-1', 'terminal-1', 'git status\ngit push')).toBe(true)
    expect(terminalInstances[0].paste).toHaveBeenCalledWith('git status\ngit push')
    expect(onWrite).not.toHaveBeenCalled()

    registry.dispose('task-1', 'terminal-1')
    expect(registry.pasteInput('task-1', 'terminal-1', 'ignored')).toBe(false)
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
    registry.attach(terminal, firstContainer, terminalVisualTheme(), onResize)
    registry.attach(terminal, secondContainer, terminalVisualTheme(), onResize)

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(1)
    expect(secondContainer.contains(terminalInstances[0].element!)).toBe(true)
    expect(fitAddonInstances[0].fit).toHaveBeenCalledTimes(2)
    expect(onResize).toHaveBeenCalledWith(100, 30)
  })

  it('挂载禁用策略的终端时不自动复制选区', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')

    registry.attach({...terminal, disableTaskAIMouseClipboard: true}, container, terminalVisualTheme(), vi.fn())
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

    registry.attach(terminal, firstAContainer, terminalVisualTheme(), vi.fn())
    registry.attach(terminalB, bContainer, terminalVisualTheme(), vi.fn())
    registry.attach(terminal, secondAContainer, terminalVisualTheme(), vi.fn())

    expect(terminalInstances).toHaveLength(2)
    expect(secondAContainer.contains(terminalInstances[0].element!)).toBe(true)
    expect(terminalInstances.every((instance) => instance.write.mock.calls.length === 0)).toBe(true)
  })

  it('终端退出后释放会话并忽略迟到的输出', () => {
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'before exit'})
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'exited', exitReason: 'normal'})
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'late output'})

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].dispose).toHaveBeenCalledOnce()
    expect(terminalInstances[0].onDataDisposable.dispose).toHaveBeenCalledOnce()
    expect(terminalInstances[0].onSelectionDisposable.dispose).toHaveBeenCalledOnce()
    expect(terminalInstances[0].write).toHaveBeenCalledOnce()
  })

  it('未分类退出时保留输出快照，避免终端标签与会话状态不一致', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    const onResize = vi.fn()
    const snapshot = {...terminal, state: 'exited' as const}

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'diagnostic output'})
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'exited'})

    expect(registry.attach(snapshot, container, terminalVisualTheme(), onResize)).toBe(true)
    expect(terminalInstances[0].write).toHaveBeenCalledWith('diagnostic output')
    expect(terminalInstances[0].dispose).not.toHaveBeenCalled()
    expect(onResize).not.toHaveBeenCalled()
  })

  it('异常退出冻结输出快照并拒绝后续终端交互', () => {
		const onWrite = vi.fn()
    const registry = new TerminalSessionRegistry(onWrite)
    const container = document.createElement('div')
    const onResize = vi.fn()
    const snapshot = {...terminal, state: 'exited' as const}

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'failure output'})
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'exited', exitReason: 'unexpected', exitCode: 1})

    expect(registry.attach(snapshot, container, terminalVisualTheme(), onResize)).toBe(true)
    expect(terminalInstances[0].dispose).not.toHaveBeenCalled()
    expect(terminalInstances[0].write).toHaveBeenNthCalledWith(1, 'failure output')
    expect(terminalInstances[0].write).toHaveBeenNthCalledWith(2, '\r\n终端已退出\x1b[?25l')
    expect(onResize).not.toHaveBeenCalled()
    expect(registry.writeInput(snapshot.taskId, snapshot.id, 'retry')).toBe(false)
    expect(registry.pasteInput(snapshot.taskId, snapshot.id, 'retry')).toBe(false)
    expect(registry.setCustomKeyEventHandler(snapshot.taskId, snapshot.id, vi.fn())).toBe(false)
		terminalInstances[0].triggerData('retry')
		expect(onWrite).not.toHaveBeenCalled()

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'late output'})
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(2)

    registry.dispose(snapshot.taskId, snapshot.id)
    expect(terminalInstances[0].dispose).toHaveBeenCalledOnce()
  })

	it('异常退出写入内部快照提示时不报告画面活动', () => {
		const onVisualActivity = vi.fn()
		const registry = new TerminalSessionRegistry(vi.fn(), undefined, undefined, undefined, onVisualActivity)

		registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'failure output'})
		terminalInstances[0].setCell(0, 0, {chars: '终'})
		registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'exited', exitReason: 'unexpected', exitCode: 1})
		terminalInstances[0].triggerWriteParsed()

		expect(onVisualActivity).not.toHaveBeenCalled()
	})

  it('无输出的异常终端也保留无光标的退出提示快照', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    const onResize = vi.fn()
    const snapshot = {...terminal, state: 'exited' as const}

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'exited', exitReason: 'unexpected', exitCode: 1})

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].write).toHaveBeenCalledWith('\r\n终端已退出\x1b[?25l')
    expect(registry.attach(snapshot, container, terminalVisualTheme(), onResize)).toBe(true)
    expect(onResize).not.toHaveBeenCalled()
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
