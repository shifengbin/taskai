import {fireEvent, render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {describe, expect, it, vi} from 'vitest'

import {TaskTree} from './TaskTree'
import type {TaskRecord, TerminalRecord} from '../types'

const runningTask: TaskRecord = {
  id: 'task-1',
  title: '整理发布说明',
  description: '完成发布说明',
  status: 'running',
  createdAt: '2026-07-22T00:00:00Z',
}

const terminal: TerminalRecord = {id: 'terminal-1', taskId: 'task-1', state: 'active'}

describe('TaskTree', () => {
  it('仅通过右键菜单创建终端，并可选择终端子节点', async () => {
    const user = userEvent.setup()
    const onCreateTerminal = vi.fn()
    const onSelectTerminal = vi.fn()
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[terminal]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={onSelectTerminal}
        onCreateTerminal={onCreateTerminal}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
      />,
    )

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    expect(onCreateTerminal).not.toHaveBeenCalled()
    await user.click(screen.getByRole('menuitem', {name: '新增终端'}))
    expect(onCreateTerminal).toHaveBeenCalledWith('task-1')

    await user.click(screen.getByText('终端 1'))
    expect(onSelectTerminal).toHaveBeenCalledWith(terminal)
  })

  it('已完成任务不显示重新执行操作', () => {
    render(
      <TaskTree
        tasks={[{...runningTask, status: 'completed'}]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
      />,
    )

    expect(screen.queryByRole('button', {name: '执行'})).not.toBeInTheDocument()
  })
})
