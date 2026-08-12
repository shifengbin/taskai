import {cleanup, fireEvent, render, screen} from '@testing-library/react'
import {afterEach, describe, expect, it, vi} from 'vitest'

import type {TerminalRecord} from '../types'
import {TerminalName} from './TerminalName'
import {TooltipProvider} from './ui'

const terminal: TerminalRecord = {
  id: 'terminal-1',
  taskId: 'task-1',
  state: 'active',
  title: 'npm run dev',
  command: 'zsh',
}

function renderTerminalName(record: TerminalRecord, onAliasChange = vi.fn()) {
  return render(
    <TooltipProvider delayDuration={0}>
      <TerminalName terminal={record} onAliasChange={onAliasChange}/>
    </TooltipProvider>,
  )
}

afterEach(cleanup)

describe('TerminalName', () => {
  it('双击后自动聚焦，并在 Enter 时保存规范化别名', () => {
    const onAliasChange = vi.fn()
    renderTerminalName(terminal, onAliasChange)

    fireEvent.doubleClick(screen.getByText('npm run dev'))

    const input = screen.getByRole('textbox', {name: '终端别名'})
    expect(input).toHaveFocus()
    fireEvent.change(input, {target: {value: ' 前端调试 '}})
    fireEvent.keyDown(input, {key: 'Enter'})

    expect(onAliasChange).toHaveBeenCalledWith('前端调试')
  })

  it('失焦时清空别名，按 Escape 时保留原别名', () => {
    const onAliasChange = vi.fn()
    const namedTerminal = {...terminal, alias: '前端调试'}
    const {rerender} = renderTerminalName(namedTerminal, onAliasChange)

    fireEvent.doubleClick(screen.getByText('前端调试'))
    fireEvent.change(screen.getByRole('textbox', {name: '终端别名'}), {target: {value: '  '}})
    fireEvent.blur(screen.getByRole('textbox', {name: '终端别名'}))

    expect(onAliasChange).toHaveBeenCalledWith(undefined)
    rerender(
      <TooltipProvider delayDuration={0}>
        <TerminalName terminal={namedTerminal} onAliasChange={onAliasChange}/>
      </TooltipProvider>,
    )
    fireEvent.doubleClick(screen.getByText('前端调试'))
    fireEvent.change(screen.getByRole('textbox', {name: '终端别名'}), {target: {value: '接口调试'}})
    fireEvent.keyDown(screen.getByRole('textbox', {name: '终端别名'}), {key: 'Escape'})

    expect(onAliasChange).toHaveBeenCalledTimes(1)
    expect(screen.getByText('前端调试')).toBeInTheDocument()
  })

  it('仅为别名显示带标签的两行实际会话信息', async () => {
    const {rerender} = renderTerminalName({...terminal, alias: '前端调试'})

    fireEvent.pointerMove(screen.getByText('前端调试'))

    expect(await screen.findByTestId('terminal-alias-details')).toHaveTextContent('标题: npm run dev')
    expect(screen.getByTestId('terminal-alias-details')).toHaveTextContent('命令: zsh')
    rerender(
      <TooltipProvider delayDuration={0}>
        <TerminalName terminal={terminal} onAliasChange={vi.fn()}/>
      </TooltipProvider>,
    )

    expect(screen.queryByTestId('terminal-alias-details')).not.toBeInTheDocument()
  })

  it('按调用场景定位提示且不接收鼠标事件', async () => {
    render(
      <TooltipProvider delayDuration={0}>
        <TerminalName
          terminal={{...terminal, alias: '前端调试'}}
          onAliasChange={vi.fn()}
          tooltipPlacement={{side: 'right', align: 'start', sideOffset: 8, avoidCollisions: false}}
        />
      </TooltipProvider>,
    )

    fireEvent.pointerMove(screen.getByText('前端调试'))

    const tooltip = await screen.findByTestId('terminal-alias-details')
    expect(tooltip).toHaveAttribute('data-side', 'right')
    expect(tooltip).toHaveAttribute('data-align', 'start')
    expect(tooltip).toHaveClass('pointer-events-none')
    expect(tooltip.parentElement).toHaveStyle({transform: 'translate(8px, 0px)'})
  })
})
