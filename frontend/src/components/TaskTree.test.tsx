import {cleanup, fireEvent, render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {afterEach, describe, expect, it, vi} from 'vitest'

import {TaskTree} from './TaskTree'
import type {TaskMenuItem, TaskRecord, TerminalRecord} from '../types'

const runningTask: TaskRecord = {
  id: 'task-1',
  title: '整理发布说明',
  description: '完成发布说明',
  status: 'running',
  color: '#2563eb',
  createdAt: '2026-07-22T00:00:00Z',
}

const terminal: TerminalRecord = {id: 'terminal-1', taskId: 'task-1', state: 'active'}

describe('TaskTree', () => {
  afterEach(cleanup)

  it('任务操作下拉菜单和右键菜单均可创建终端、打开任务文件夹或编辑任务，并可选择终端子节点', async () => {
    const user = userEvent.setup()
    const onCreateTerminal = vi.fn()
    const onEditTask = vi.fn()
    const onOpenTaskFolder = vi.fn()
    const onSelectTerminal = vi.fn()
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[terminal]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={onSelectTerminal}
        onCreateTerminal={onCreateTerminal}
        onEditTask={onEditTask}
        onOpenTaskFolder={onOpenTaskFolder}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', {name: '任务操作'}))
    expect(onCreateTerminal).not.toHaveBeenCalled()
    await user.click(screen.getByRole('menuitem', {name: '新增终端'}))
    expect(onCreateTerminal).toHaveBeenCalledWith('task-1')

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    await user.click(screen.getByRole('menuitem', {name: '打开任务文件夹'}))
    expect(onOpenTaskFolder).toHaveBeenCalledWith('task-1')

    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '打开任务文件夹'}))
    expect(onOpenTaskFolder).toHaveBeenCalledTimes(2)

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
    expect(onEditTask).toHaveBeenCalledWith('task-1')

    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
    expect(onEditTask).toHaveBeenCalledTimes(2)

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
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        activeStatus="completed"
        onChangeStatus={vi.fn()}
      />,
    )

    expect(screen.queryByRole('button', {name: '执行'})).not.toBeInTheDocument()
  })

  it('状态由标签页表达，不在任务条目中重复显示', () => {
    render(
      <TaskTree
        tasks={[{...runningTask, description: ''}]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    expect(screen.getByText('整理发布说明').closest('[data-task-id]')).not.toHaveTextContent('执行中')
  })

  it('通过状态标签筛选任务并使用任务颜色标记节点', async () => {
    const user = userEvent.setup()
    const onChangeStatus = vi.fn()
    const pendingTask: TaskRecord = {...runningTask, id: 'task-pending', title: '待处理任务', status: 'pending', color: '#f97316'}
    const completedTask: TaskRecord = {...runningTask, id: 'task-completed', title: '已完成任务', status: 'completed', color: '#22c55e'}
    render(
      <TaskTree
        tasks={[pendingTask, runningTask, completedTask]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        activeStatus="pending"
        onChangeStatus={onChangeStatus}
      />,
    )

    expect(screen.getByText('待处理任务').closest('[data-task-id]')).toHaveStyle({borderLeftColor: 'rgb(249, 115, 22)'})
    expect(screen.queryByText('整理发布说明')).not.toBeInTheDocument()
    expect(screen.queryByText('已完成任务')).not.toBeInTheDocument()

    await user.click(screen.getByRole('tab', {name: /执行中/}))
    expect(onChangeStatus).toHaveBeenCalledWith('running')
  })

  it('不显示已退出终端', () => {
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[{...terminal, state: 'exited'}]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    expect(screen.queryByText('终端 1')).not.toBeInTheDocument()
    expect(screen.queryByText('已退出')).not.toBeInTheDocument()
  })

  it('悬浮任务条目时显示完整描述', async () => {
    const user = userEvent.setup()
    const description = '这是完整的任务描述，用于确认悬浮提示不会因条目宽度而被截断。'
    render(
      <TaskTree
        tasks={[{...runningTask, description}]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    await user.hover(screen.getByText('整理发布说明'))
    expect(await screen.findByRole('tooltip')).toHaveTextContent(description)
  })

  it('按配置顺序从右键和下拉菜单运行自定义任务命令', async () => {
    const user = userEvent.setup()
    const onRunMenuCommand = vi.fn()
    const codexMenuItem: TaskMenuItem = {id: 'custom-codex', kind: 'command', name: 'Codex', command: 'codex', arguments: ['--full-auto'], showTerminal: true}
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[]}
        selectedTerminalId={undefined}
        menuItems={[codexMenuItem, {id: 'system.edit-task', kind: 'edit-task', name: '编辑任务', showTerminal: false} as TaskMenuItem]}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onRunMenuCommand={onRunMenuCommand}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', {name: '任务操作'}))
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual(['Codex', '编辑任务'])
    await user.click(screen.getByRole('menuitem', {name: 'Codex'}))
    expect(onRunMenuCommand).toHaveBeenCalledWith('task-1', codexMenuItem)

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual(['Codex', '编辑任务'])
  })
})
