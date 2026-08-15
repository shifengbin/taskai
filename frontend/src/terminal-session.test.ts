import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

type RecordedMouseEvent = Pick<MouseEvent, 'altKey' | 'button' | 'clientX' | 'clientY' | 'detail' | 'shiftKey' | 'type'>

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
      baseY: number
      getLine: ReturnType<typeof vi.fn>
      type: 'normal' | 'alternate'
      viewportY: number
    }
  }
  cols: number
  rows: number
  modes: {mouseTrackingMode: 'none' | 'any'}
  mouseEvents: RecordedMouseEvent[]
  attachCustomKeyEventHandler: ReturnType<typeof vi.fn>
  element?: HTMLElement
  dispose: ReturnType<typeof vi.fn>
  focus: ReturnType<typeof vi.fn>
  getSelection: ReturnType<typeof vi.fn>
  getSelectionPosition: ReturnType<typeof vi.fn>
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
  options: {altClickMovesCursor?: boolean, fontFamily?: string, fontSize?: number, macOptionClickForcesSelection?: boolean, scrollback?: number, theme?: unknown}
  refresh: ReturnType<typeof vi.fn>
  select: ReturnType<typeof vi.fn>
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
    modes = {mouseTrackingMode: 'none' as const}
    mouseEvents: RecordedMouseEvent[] = []
    attachCustomKeyEventHandler = vi.fn()
    element: HTMLElement | undefined
    dispose = vi.fn()
    focus = vi.fn()
    getSelection = vi.fn(() => '')
    getSelectionPosition = vi.fn(() => undefined as {start: {x: number, y: number}, end: {x: number, y: number}} | undefined)
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
      for (const type of ['mousedown', 'mousemove', 'mouseup', 'wheel']) {
        this.element.addEventListener(type, (event) => {
          const mouseEvent = event as MouseEvent
          this.mouseEvents.push({
            altKey: mouseEvent.altKey,
            button: mouseEvent.button,
            clientX: mouseEvent.clientX,
            clientY: mouseEvent.clientY,
            detail: mouseEvent.detail,
            shiftKey: mouseEvent.shiftKey,
            type: mouseEvent.type,
          })
        })
      }
      container.append(this.element)
    })
    paste = vi.fn()
    options: {altClickMovesCursor?: boolean, fontFamily?: string, fontSize?: number, macOptionClickForcesSelection?: boolean, scrollback?: number, theme?: unknown}
    refresh = vi.fn()
    select = vi.fn()
    write = vi.fn()
    lines = Array.from({length: 60}, () => Array.from({length: 100}, () => terminalCell()))
    buffer = {
      active: {
        cursorX: 0,
        cursorY: 0,
        baseY: 0,
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

// 输出按静默期合批写入：推进假定时器触发冲刷回调后再断言 terminal.write。
// 常量与 terminal-session.ts 保持一致（静默 32ms / 恢复 48ms）。
function flushTerminalOutput() {
  vi.advanceTimersByTime(32)
}

function cursorRestoreSettled() {
  vi.advanceTimersByTime(48)
}

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

  afterEach(() => {
    vi.useRealTimers()
    document.body.replaceChildren()
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
    vi.useFakeTimers()
    let savedFontSize = 16
    const registry = new TerminalSessionRegistry(vi.fn(), () => '', () => savedFontSize)
    const container = document.createElement('div')
    const onResize = vi.fn()

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'before attach'})
    expect(terminalInstances[0].options.fontSize).toBe(16)

    savedFontSize = 20
    registry.setFontSize(savedFontSize)
    registry.attach(terminal, container, terminalVisualTheme(), onResize)
    flushTerminalOutput()

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].options.fontSize).toBe(20)
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(1)
    expect(terminalInstances[0].dispose).not.toHaveBeenCalled()
    expect(terminalInstances[0].onDataDisposable.dispose).not.toHaveBeenCalled()
    expect(onResize).toHaveBeenCalledWith(100, 30)
  })

  it('保存后将主题、字体和字号更新到已有会话', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)
    const updatedTheme = {...terminalVisualTheme(), background: '#102030', foreground: '#E0F0FF'}

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'ready'})
    registry.attach(terminal, container, terminalVisualTheme(), vi.fn())
    ;(registry as unknown as {setAppearance(fontFamily: string, fontSize: number, theme: typeof updatedTheme): void}).setAppearance('Fira Code', 16, updatedTheme)
    flushTerminalOutput()

    expect(terminalInstances[0].options.theme).toMatchObject(updatedTheme)
    expect(terminalInstances[0].options.fontFamily).toBe('"Fira Code", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace')
    expect(terminalInstances[0].options.fontSize).toBe(16)
    expect(terminalInstances[0].refresh).toHaveBeenCalledWith(0, 29)
    expect(terminalInstances[0].write).toHaveBeenCalledWith('ready')
  })

  it('直接写入按终端键持有的会话，并将滚屏限制为 1000 行', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'first output chunk'})
    flushTerminalOutput()

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].options.scrollback).toBe(1000)
    expect(terminalInstances[0].options).toMatchObject({altClickMovesCursor: false, macOptionClickForcesSelection: true})
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

  it('用户查看历史时，活动终端页变化仍上报活动', () => {
    const onVisualActivity = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn(), undefined, undefined, undefined, onVisualActivity)

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'initial'})
    const session = terminalInstances[0]
    session.buffer.active.baseY = 30
    session.buffer.active.viewportY = 30
    session.triggerWriteParsed()
    onVisualActivity.mockClear()

    session.triggerScroll(0)
    session.setCell(30, 0, {chars: '新'})
    session.triggerWriteParsed()

    expect(onVisualActivity).toHaveBeenCalledWith('task-1', 'terminal-1')
  })

  it('用户滚动或历史内容变化不改变活动终端页的状态基线', () => {
    const onVisualActivity = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn(), undefined, undefined, undefined, onVisualActivity)

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'initial'})
    const session = terminalInstances[0]
    session.buffer.active.baseY = 30
    session.buffer.active.viewportY = 30
    session.triggerWriteParsed()
    onVisualActivity.mockClear()

    session.triggerScroll(0)
    session.setCell(0, 0, {chars: '旧'})
    session.triggerWriteParsed()

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

  it('仅返回活动会话的当前选区，并忽略缺失或已关闭会话', () => {
    const registry = new TerminalSessionRegistry(vi.fn())

    expect(registry.selectionText('task-1', 'terminal-1')).toBe('')

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'ready'})
    terminalInstances[0].getSelection.mockReturnValue('选中的终端内容')

    expect(registry.selectionText('task-1', 'terminal-1')).toBe('选中的终端内容')
    expect(registry.selectionText('task-1', 'terminal-2')).toBe('')

    registry.dispose('task-1', 'terminal-1')

    expect(registry.selectionText('task-1', 'terminal-1')).toBe('')
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
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const firstContainer = document.createElement('div')
    const secondContainer = document.createElement('div')
    const onResize = vi.fn()

    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'output', data: 'existing output'})
    registry.attach(terminal, firstContainer, terminalVisualTheme(), onResize)
    registry.attach(terminal, secondContainer, terminalVisualTheme(), onResize)
    flushTerminalOutput()

    expect(terminalInstances).toHaveLength(1)
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(1)
    expect(secondContainer.contains(terminalInstances[0].element!)).toBe(true)
    expect(fitAddonInstances[0].fit).toHaveBeenCalledTimes(2)
    expect(onResize).toHaveBeenCalledWith(100, 30)
  })

  it('旧策略字段不再阻止自动复制选区', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)

    registry.attach({...terminal, disableTaskAIMouseClipboard: true} as never, container, terminalVisualTheme(), vi.fn())
    terminalInstances[0].getSelection.mockReturnValue('selected terminal output')
    terminalInstances[0].triggerSelectionChange()

    expect(runtime.ClipboardSetText).toHaveBeenCalledWith('selected terminal output')
  })

  it.each(['none', 'any'] as const)('鼠标追踪模式 %s 中延迟重放单击并容忍阈值内抖动', (mouseTrackingMode) => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)

    registry.attach(terminal, container, terminalVisualTheme(), vi.fn(), vi.fn())
    terminalInstances[0].modes.mouseTrackingMode = mouseTrackingMode
    const element = terminalInstances[0].element!
    element.dispatchEvent(new MouseEvent('mousedown', {bubbles: true, button: 0, buttons: 1, clientX: 10, clientY: 12, detail: 1}))
    element.dispatchEvent(new MouseEvent('mousemove', {bubbles: true, button: 0, buttons: 1, clientX: 12, clientY: 14, detail: 1}))
    element.dispatchEvent(new MouseEvent('mouseup', {bubbles: true, button: 0, clientX: 12, clientY: 14, detail: 1}))

    expect(terminalInstances[0].mouseEvents).toEqual([])
    vi.runAllTimers()
    expect(terminalInstances[0].mouseEvents).toEqual([
      expect.objectContaining({type: 'mousedown', button: 0, clientX: 10, clientY: 12, detail: 1, shiftKey: false, altKey: false}),
      expect.objectContaining({type: 'mouseup', button: 0, clientX: 12, clientY: 14, detail: 1, shiftKey: false, altKey: false}),
    ])
  })

  it.each([
    {clicks: 2, forceSelection: false, label: '普通模式双击单词', mouseTrackingMode: 'none' as const},
    {clicks: 3, forceSelection: false, label: '普通模式三击整行', mouseTrackingMode: 'none' as const},
    {clicks: 2, forceSelection: true, label: '鼠标追踪期间双击单词', mouseTrackingMode: 'any' as const},
    {clicks: 3, forceSelection: true, label: '鼠标追踪期间三击整行', mouseTrackingMode: 'any' as const},
  ])('$label仅重放一次本地选择', ({clicks, forceSelection, mouseTrackingMode}) => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)
    const onSelectionComplete = vi.fn()

    registry.attach(terminal, container, terminalVisualTheme(), vi.fn(), onSelectionComplete)
    terminalInstances[0].modes.mouseTrackingMode = mouseTrackingMode
    terminalInstances[0].getSelection.mockReturnValue('选择原文')
    const element = terminalInstances[0].element!
    for (let detail = 1; detail <= clicks; detail++) {
      element.dispatchEvent(new MouseEvent('mousedown', {bubbles: true, button: 0, buttons: 1, clientX: 30, clientY: 40, detail}))
      element.dispatchEvent(new MouseEvent('mouseup', {bubbles: true, button: 0, clientX: 30, clientY: 40, detail}))
    }

    vi.runAllTimers()
    expect(terminalInstances[0].mouseEvents).toEqual([
      expect.objectContaining({type: 'mousedown', detail: clicks, shiftKey: forceSelection}),
      expect.objectContaining({type: 'mouseup', detail: clicks, shiftKey: forceSelection}),
    ])
    expect(onSelectionComplete).toHaveBeenCalledOnce()
    expect(onSelectionComplete).toHaveBeenCalledWith({clientX: 30, clientY: 40, text: '选择原文'})
  })

  it('超过阈值的拖拽强制本地选择，同时透传滚轮和中键', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)
    const onSelectionComplete = vi.fn()

    registry.attach(terminal, container, terminalVisualTheme(), vi.fn(), onSelectionComplete)
    terminalInstances[0].modes.mouseTrackingMode = 'any'
    terminalInstances[0].getSelection.mockReturnValue('拖拽原文')
    const element = terminalInstances[0].element!
    element.dispatchEvent(new WheelEvent('wheel', {bubbles: true, deltaY: 40}))
    element.dispatchEvent(new MouseEvent('mousedown', {bubbles: true, button: 1, buttons: 4}))
    element.dispatchEvent(new MouseEvent('mouseup', {bubbles: true, button: 1}))
    element.dispatchEvent(new MouseEvent('mousedown', {bubbles: true, button: 0, buttons: 1, clientX: 10, clientY: 10, detail: 1}))
    element.dispatchEvent(new MouseEvent('mousemove', {bubbles: true, button: 0, buttons: 1, clientX: 30, clientY: 35, detail: 1}))
    element.dispatchEvent(new MouseEvent('mouseup', {bubbles: true, button: 0, clientX: 30, clientY: 35, detail: 1}))
    vi.runAllTimers()

    expect(terminalInstances[0].mouseEvents).toEqual([
      expect.objectContaining({type: 'wheel'}),
      expect.objectContaining({type: 'mousedown', button: 1}),
      expect.objectContaining({type: 'mouseup', button: 1}),
      expect.objectContaining({type: 'mousedown', button: 0, clientX: 10, clientY: 10, shiftKey: true}),
      expect.objectContaining({type: 'mousemove', button: 0, clientX: 30, clientY: 35, shiftKey: true}),
      expect.objectContaining({type: 'mouseup', button: 0, clientX: 30, clientY: 35, shiftKey: true}),
    ])
    expect(onSelectionComplete).toHaveBeenCalledWith({clientX: 30, clientY: 35, text: '拖拽原文'})
  })

  it('鼠标追踪期间选择后移动鼠标仍发送事件并重复恢复同一本地选区', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)

    registry.attach(terminal, container, terminalVisualTheme(), vi.fn(), vi.fn())
    const instance = terminalInstances[0]
    instance.modes.mouseTrackingMode = 'any'
    const selection = 'Claude 选区'
    const selectionPosition = {start: {x: 2, y: 4}, end: {x: 8, y: 5}}
    instance.getSelection.mockReturnValue(selection)
    instance.getSelectionPosition.mockReturnValue(selectionPosition)
    instance.select.mockImplementation(() => {
      instance.getSelection.mockReturnValue(selection)
      instance.getSelectionPosition.mockReturnValue(selectionPosition)
      instance.triggerSelectionChange()
    })
    const element = instance.element!
    element.dispatchEvent(new MouseEvent('mousedown', {bubbles: true, button: 0, buttons: 1, clientX: 10, clientY: 10, detail: 1}))
    element.dispatchEvent(new MouseEvent('mousemove', {bubbles: true, button: 0, buttons: 1, clientX: 30, clientY: 35, detail: 1}))
    element.dispatchEvent(new MouseEvent('mouseup', {bubbles: true, button: 0, clientX: 30, clientY: 35, detail: 1}))
    vi.runAllTimers()
    instance.mouseEvents.length = 0
    const clearSelection = () => {
      instance.getSelection.mockReturnValue('')
      instance.getSelectionPosition.mockReturnValue(undefined)
      instance.triggerSelectionChange()
    }
    element.addEventListener('mousemove', (event) => {
      if (!(event as MouseEvent).buttons) {
        clearSelection()
      }
    })

    element.dispatchEvent(new MouseEvent('mousemove', {bubbles: true, clientX: 42, clientY: 50}))
    window.setTimeout(clearSelection, 20)
    vi.advanceTimersByTime(20)

    expect(instance.mouseEvents).toEqual([
      expect.objectContaining({type: 'mousemove', clientX: 42, clientY: 50}),
    ])
    expect(instance.select).toHaveBeenCalledTimes(2)
    expect(instance.select).toHaveBeenNthCalledWith(1, 2, 4, 106)
    expect(instance.select).toHaveBeenNthCalledWith(2, 2, 4, 106)
    expect(instance.getSelection()).toBe(selection)

    element.dispatchEvent(new KeyboardEvent('keydown', {bubbles: true, key: 'a'}))
    clearSelection()
    expect(instance.select).toHaveBeenCalledTimes(2)
  })

  it('右键按下和松开不进入 xterm，仍保留 contextmenu 给视图粘贴', () => {
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)

    const onContextMenu = vi.fn()
    registry.attach(terminal, container, terminalVisualTheme(), vi.fn(), vi.fn(), onContextMenu)
    terminalInstances[0].modes.mouseTrackingMode = 'any'
    const element = terminalInstances[0].element!
    element.dispatchEvent(new MouseEvent('mousedown', {bubbles: true, button: 2, buttons: 2}))
    element.dispatchEvent(new MouseEvent('mouseup', {bubbles: true, button: 2}))
    element.dispatchEvent(new MouseEvent('contextmenu', {bubbles: true, button: 2}))

    expect(terminalInstances[0].mouseEvents).toEqual([])
    expect(onContextMenu).toHaveBeenCalledOnce()
  })

  it.each(['detach', 'switch', 'inactive', 'dispose'] as const)('%s 时取消尚未重放的左键单击', (action) => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)

    registry.attach(terminal, container, terminalVisualTheme(), vi.fn(), vi.fn())
    const firstElement = terminalInstances[0].element!
    firstElement.dispatchEvent(new MouseEvent('mousedown', {bubbles: true, button: 0, buttons: 1, detail: 1}))
    firstElement.dispatchEvent(new MouseEvent('mouseup', {bubbles: true, button: 0, detail: 1}))
    if (action === 'detach') {
      registry.detach(terminal.taskId, terminal.id)
    } else if (action === 'switch') {
      registry.attach({id: 'terminal-2', taskId: terminal.taskId, state: 'active'}, document.createElement('div'), terminalVisualTheme(), vi.fn(), vi.fn())
    } else if (action === 'inactive') {
      registry.attach({...terminal, state: 'exited'}, container, terminalVisualTheme(), vi.fn(), vi.fn())
    } else {
      registry.dispose(terminal.taskId, terminal.id)
    }

    vi.runAllTimers()
    expect(terminalInstances[0].mouseEvents).toEqual([])
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
    // 会话被释放，尚未冲刷的缓冲随之丢弃，不再写入已销毁的终端
    expect(terminalInstances[0].write).not.toHaveBeenCalled()
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

  it('异常退出时取消尚未重放的单击', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const container = document.createElement('div')
    document.body.append(container)

    registry.attach(terminal, container, terminalVisualTheme(), vi.fn(), vi.fn())
    const element = terminalInstances[0].element!
    element.dispatchEvent(new MouseEvent('mousedown', {bubbles: true, button: 0, buttons: 1, detail: 1}))
    element.dispatchEvent(new MouseEvent('mouseup', {bubbles: true, button: 0, detail: 1}))
    registry.handleTerminalEvent({...terminal, terminalId: terminal.id, type: 'exited', exitReason: 'unexpected', exitCode: 1})

    vi.runAllTimers()
    expect(terminalInstances[0].mouseEvents).toEqual([])
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

  it('同一静默期内的多次输出事件合并为一次写入且保持到达顺序', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'chunk-a'})
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'chunk-b'})
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'chunk-c'})
    expect(terminalInstances[0].write).not.toHaveBeenCalled()

    flushTerminalOutput()

    expect(terminalInstances[0].write).toHaveBeenCalledOnce()
    expect(terminalInstances[0].write).toHaveBeenCalledWith('chunk-achunk-bchunk-c')
  })

  it('跨静默期输出分批写入且顺序保持，退出时先冲刷缓冲再写退出通知', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'one'})
    flushTerminalOutput()
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'two'})
    flushTerminalOutput()
    expect(terminalInstances[0].write.mock.calls).toEqual([['one'], ['two']])

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'tail'})
    registry.handleTerminalEvent({type: 'exited', taskId: 'task-1', terminalId: 'terminal-1', exitReason: 'unexpected', exitCode: 1})
    expect(terminalInstances[0].write.mock.calls).toEqual([['one'], ['two'], ['tail'], ['\r\n终端已退出\x1b[?25l']])
  })

  it('ConPTY 绘制批次冲刷时追加合成隐藏序列，输出静默后补写恢复光标', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const batch = `${'x'.repeat(100)}\x1b[?25h`

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: batch})
    expect(terminalInstances[0].write).not.toHaveBeenCalled()
    flushTerminalOutput()
    expect(terminalInstances[0].write).toHaveBeenCalledOnce()
    expect(terminalInstances[0].write).toHaveBeenCalledWith(`${batch}\x1b[?25l`)

    cursorRestoreSettled()
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(2)
    expect(terminalInstances[0].write).toHaveBeenLastCalledWith('\x1b[?25h')
  })

  it('批间继续输出时跳过光标恢复，直至输出真正停止', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const batchA = `${'a'.repeat(100)}\x1b[?25h`
    const batchB = `${'b'.repeat(100)}\x1b[?25h`

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: batchA})
    flushTerminalOutput()
    // 18ms 后下一绘制批次到达（恢复定时器 48ms 尚未到期）
    vi.advanceTimersByTime(18)
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: batchB})
    // 恢复定时器在冲刷后 48ms 触发：此刻缓冲非空，跳过恢复
    vi.advanceTimersByTime(30)
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(1)
    // 第二批的静默冲刷落地
    flushTerminalOutput()
    expect(terminalInstances[0].write).toHaveBeenLastCalledWith(`${batchB}\x1b[?25l`)
    // 流停止后恢复光标
    cursorRestoreSettled()
    expect(terminalInstances[0].write).toHaveBeenLastCalledWith('\x1b[?25h')
  })

  it('小于阈值的击键回显冲刷原样写入且不隐藏光标', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'a'})
    flushTerminalOutput()
    expect(terminalInstances[0].write).toHaveBeenCalledOnce()
    expect(terminalInstances[0].write).toHaveBeenCalledWith('a')

    vi.runAllTimers()
    expect(terminalInstances[0].write).toHaveBeenCalledOnce()
  })

  it('不含光标序列的流式长文本保持光标可见', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())
    const logLine = `${'compiling module '.repeat(12)}\r\n`

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: logLine})
    flushTerminalOutput()
    vi.runAllTimers()

    expect(terminalInstances[0].write).toHaveBeenCalledOnce()
    expect(terminalInstances[0].write).toHaveBeenCalledWith(logLine)
  })

  it('连续输出超过截止时限时按截止冲刷已缓冲内容', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'c1'})
    vi.advanceTimersByTime(30)
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'c2'})
    vi.advanceTimersByTime(28)
    expect(terminalInstances[0].write).not.toHaveBeenCalled()
    vi.advanceTimersByTime(6)
    expect(terminalInstances[0].write).toHaveBeenCalledOnce()
    expect(terminalInstances[0].write).toHaveBeenCalledWith('c1c2')
  })

  it('静默期满定时器冲刷，缓冲超上限立即同步冲刷', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'fallback chunk'})
    expect(terminalInstances[0].write).not.toHaveBeenCalled()
    flushTerminalOutput()
    expect(terminalInstances[0].write).toHaveBeenCalledWith('fallback chunk')

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'x'.repeat(1024 * 1024)})
    expect(terminalInstances[0].write).toHaveBeenCalledTimes(2)
    expect(terminalInstances[0].write).toHaveBeenLastCalledWith('x'.repeat(1024 * 1024))
  })

  it('会话销毁后取消未决冲刷与恢复回调且不再写入', () => {
    vi.useFakeTimers()
    const registry = new TerminalSessionRegistry(vi.fn())

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'plain doomed'})
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: `${'x'.repeat(100)}\x1b[?25h`})
    registry.dispose('task-1', 'terminal-1')
    vi.runAllTimers()

    expect(terminalInstances[0].dispose).toHaveBeenCalledOnce()
    expect(terminalInstances[0].write).not.toHaveBeenCalled()
  })

  it('同一合并批次内瞬时改写又还原不计为画面活动', () => {
    vi.useFakeTimers()
    const onVisualActivity = vi.fn()
    const registry = new TerminalSessionRegistry(vi.fn(), undefined, undefined, undefined, onVisualActivity)

    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'redraw'})
    const session = terminalInstances[0]
    // 批次中的瞬时状态：先改写再还原，批末画面与基线一致
    session.setCell(0, 0, {chars: 'X'})
    session.setCell(0, 0, {chars: ''})
    flushTerminalOutput()
    session.triggerWriteParsed()
    expect(onVisualActivity).not.toHaveBeenCalled()

    // 对照：批末画面确实变化则正常上报
    registry.handleTerminalEvent({type: 'output', taskId: 'task-1', terminalId: 'terminal-1', data: 'visible change'})
    session.setCell(0, 0, {chars: 'Y'})
    flushTerminalOutput()
    session.triggerWriteParsed()
    expect(onVisualActivity).toHaveBeenCalledOnce()
  })
})
