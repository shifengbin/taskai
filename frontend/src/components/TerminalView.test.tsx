import {cleanup, render, screen} from '@testing-library/react'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

const terminalInstances = vi.hoisted(() => [] as Array<{
  cols: number
  rows: number
  loadAddon: ReturnType<typeof vi.fn>
  open: ReturnType<typeof vi.fn>
  write: ReturnType<typeof vi.fn>
  onData: ReturnType<typeof vi.fn>
  dispose: ReturnType<typeof vi.fn>
  reset: ReturnType<typeof vi.fn>
}>)

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 100
    rows = 30
    loadAddon = vi.fn()
    open = vi.fn()
    write = vi.fn()
    onData = vi.fn(() => ({dispose: vi.fn()}))
    dispose = vi.fn()
    reset = vi.fn()

    constructor() {
      terminalInstances.push(this)
    }
  },
}))

vi.mock('@xterm/addon-fit', () => ({FitAddon: class {fit = vi.fn()}}))

import {TerminalView} from './TerminalView'

describe('TerminalView', () => {
  beforeEach(() => {
    terminalInstances.length = 0
  })

  afterEach(() => {
    cleanup()
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
})
