import {cleanup, fireEvent, render, screen, waitFor, within} from '@testing-library/react'
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

function mockPointerTarget(target: Element) {
  const descriptor = Object.getOwnPropertyDescriptor(document, 'elementFromPoint')
  Object.defineProperty(document, 'elementFromPoint', {configurable: true, value: () => target})
  return () => {
    if (descriptor) {
      Object.defineProperty(document, 'elementFromPoint', descriptor)
      return
    }
    Reflect.deleteProperty(document, 'elementFromPoint')
  }
}

function dispatchPointerEvent(target: Element, type: string, pointerId: number, clientX: number, clientY: number) {
  const event = new MouseEvent(type, {bubbles: true, cancelable: true, button: 0, clientX, clientY})
  Object.defineProperty(event, 'pointerId', {configurable: true, value: pointerId})
  fireEvent(target, event)
}

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

    expect(screen.getByText('终端')).toBeInTheDocument()
    expect(screen.queryByText('终端 1')).not.toBeInTheDocument()
    await user.click(screen.getByText('终端'))
    expect(onSelectTerminal).toHaveBeenCalledWith(terminal)
  })

  it('优先显示终端真实标题，缺失标题时使用统一回退名称', () => {
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[{...terminal, id: 'terminal-titled', title: '正在构建'}, terminal]}
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

    expect(screen.getByText('正在构建')).toBeInTheDocument()
    expect(screen.getByText('终端')).toBeInTheDocument()
    expect(screen.queryByText('终端 1')).not.toBeInTheDocument()
    expect(screen.queryByText('终端 2')).not.toBeInTheDocument()
  })

  it('活跃终端以状态点表示运行状态，不显示状态文字', () => {
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[terminal]}
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

    const terminalItem = screen.getByText('终端').closest('[role="button"]')
    if (!(terminalItem instanceof HTMLElement)) {
      throw new Error('未找到终端条目')
    }
    expect(within(terminalItem).getByRole('status', {name: '终端状态：运行中'})).toBeInTheDocument()
    expect(within(terminalItem).queryByText('运行中', {exact: true})).not.toBeInTheDocument()
  })

  it('终端标题在可收缩容器中单行裁剪而不显示省略号', () => {
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[{...terminal, title: '这是用于验证任务树终端标题布局的超长真实终端名称'}]}
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

    const title = screen.getByTestId('task-tree-terminal-title-terminal-1')
    expect(title).toHaveStyle({whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'clip'})
    expect(title.parentElement).toHaveStyle({flex: '1', minWidth: '0'})
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

  it('仅执行中任务显示展开或收起终端按钮', () => {
    const taskProps = {
      terminals: [],
      selectedTerminalId: undefined,
      onSelectTask: vi.fn(),
      onSelectTerminal: vi.fn(),
      onCreateTerminal: vi.fn(),
      onEditTask: vi.fn(),
      onOpenTaskFolder: vi.fn(),
      onStartTask: vi.fn(),
      onFinishTask: vi.fn(),
      onChangeStatus: vi.fn(),
    }
    const {rerender} = render(
      <TaskTree
        {...taskProps}
        tasks={[{...runningTask, status: 'pending'}]}
        activeStatus="pending"
      />,
    )

    expect(screen.queryByRole('button', {name: '收起终端'})).not.toBeInTheDocument()

    rerender(
      <TaskTree
        {...taskProps}
        tasks={[{...runningTask, status: 'completed'}]}
        activeStatus="completed"
      />,
    )
    expect(screen.queryByRole('button', {name: '收起终端'})).not.toBeInTheDocument()

    rerender(
      <TaskTree
        {...taskProps}
        tasks={[runningTask]}
        activeStatus="running"
      />,
    )
    expect(screen.getByRole('button', {name: '收起终端'})).toBeInTheDocument()
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

  it('已退出终端以状态点表示退出状态，不显示状态文字', () => {
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

    const terminalItem = screen.getByText('终端').closest('[role="button"]')
    if (!(terminalItem instanceof HTMLElement)) {
      throw new Error('未找到终端条目')
    }
    expect(within(terminalItem).getByRole('status', {name: '终端状态：已退出'})).toBeInTheDocument()
    expect(within(terminalItem).queryByText('已退出', {exact: true})).not.toBeInTheDocument()
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

  it('双击任务条目的非按钮区域可展开或收起终端', async () => {
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[terminal]}
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

    fireEvent.doubleClick(screen.getByText('整理发布说明'))
    await waitFor(() => expect(screen.queryByText('终端')).not.toBeInTheDocument())

    fireEvent.doubleClick(screen.getByText('整理发布说明'))
    expect(screen.getByText('终端')).toBeInTheDocument()
  })

  it('指针拖动任务时显示明确的插入位置，并按位置请求重排', () => {
    const onReorderTasks = vi.fn()
    const secondTask: TaskRecord = {...runningTask, id: 'task-2', title: '补充发布说明'}
    render(
      <TaskTree
        tasks={[runningTask, secondTask]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        onReorderTasks={onReorderTasks}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    const source = screen.getByText('整理发布说明').closest('[data-task-id]')
    const target = screen.getByText('补充发布说明').closest('[data-task-id]')
    if (!source || !target) {
      throw new Error('未找到可拖动的任务条目')
    }
    Object.defineProperty(target, 'getBoundingClientRect', {
      configurable: true,
      value: () => ({top: 100, height: 48}),
    })
    const restorePointerTarget = mockPointerTarget(target)
    try {
      dispatchPointerEvent(source, 'pointerdown', 1, 20, 0)
      dispatchPointerEvent(source, 'pointermove', 1, 20, 105)

      expect(screen.getByRole('status', {name: '将任务插入“补充发布说明”之前'})).toBeInTheDocument()

      dispatchPointerEvent(source, 'pointerup', 1, 20, 105)
      expect(onReorderTasks).toHaveBeenCalledWith('task-1', 'task-2', 'before')

      dispatchPointerEvent(source, 'pointerdown', 2, 20, 0)
      dispatchPointerEvent(source, 'pointermove', 2, 20, 140)

      expect(screen.getByRole('status', {name: '将任务插入“补充发布说明”之后'})).toBeInTheDocument()
    } finally {
      restorePointerTarget()
    }
  })

  it('指针移动到目标任务的终端子项时按目标任务之后排序', () => {
    const onReorderTasks = vi.fn()
    const secondTask: TaskRecord = {...runningTask, id: 'task-2', title: '补充发布说明'}
    const secondTerminal: TerminalRecord = {...terminal, id: 'terminal-2', taskId: 'task-2'}
    render(
      <TaskTree
        tasks={[runningTask, secondTask]}
        terminals={[secondTerminal]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        onReorderTasks={onReorderTasks}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    const source = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!source) {
      throw new Error('未找到可拖动的任务条目')
    }
    const restorePointerTarget = mockPointerTarget(screen.getByText('终端'))
    try {
      dispatchPointerEvent(source, 'pointerdown', 1, 20, 0)
      dispatchPointerEvent(source, 'pointermove', 1, 20, 100)
      dispatchPointerEvent(source, 'pointerup', 1, 20, 100)

      expect(onReorderTasks).toHaveBeenCalledWith('task-1', 'task-2', 'after')
    } finally {
      restorePointerTarget()
    }
  })

  it('指针排序时显示被移动任务的预览，并在释放后隐藏', () => {
    const secondTask: TaskRecord = {...runningTask, id: 'task-2', title: '补充发布说明'}
    render(
      <TaskTree
        tasks={[runningTask, secondTask]}
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

    const source = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!source) {
      throw new Error('未找到可拖动的任务条目')
    }
    const restorePointerTarget = mockPointerTarget(screen.getByText('补充发布说明'))
    try {
      dispatchPointerEvent(source, 'pointerdown', 1, 20, 0)
      dispatchPointerEvent(source, 'pointermove', 1, 20, 100)

      expect(screen.getByRole('status', {name: '正在调整任务“整理发布说明”'})).toHaveTextContent('整理发布说明')

      dispatchPointerEvent(source, 'pointerup', 1, 20, 100)
      expect(screen.queryByRole('status', {name: '正在调整任务“整理发布说明”'})).not.toBeInTheDocument()

      dispatchPointerEvent(source, 'pointerdown', 2, 20, 0)
      dispatchPointerEvent(source, 'pointermove', 2, 20, 100)
      dispatchPointerEvent(source, 'pointercancel', 2, 20, 100)
      expect(screen.queryByRole('status', {name: '正在调整任务“整理发布说明”'})).not.toBeInTheDocument()
    } finally {
      restorePointerTarget()
    }
  })

  it('指针排序时保持终端子节点可见且不改变展开状态', () => {
    const secondTask: TaskRecord = {...runningTask, id: 'task-2', title: '补充发布说明'}
    const secondTerminal: TerminalRecord = {...terminal, id: 'terminal-2', taskId: 'task-2'}
    render(
      <TaskTree
        tasks={[runningTask, secondTask]}
        terminals={[terminal, secondTerminal]}
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

    const taskItem = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!taskItem) {
      throw new Error('未找到可拖动的任务条目')
    }
    expect(screen.getAllByText('终端')).toHaveLength(2)

    const taskTree = screen.getByRole('navigation', {name: '任务和终端'})
    const restorePointerTarget = mockPointerTarget(screen.getByText('补充发布说明'))
    try {
      dispatchPointerEvent(taskItem, 'pointerdown', 1, 20, 0)
      dispatchPointerEvent(taskItem, 'pointermove', 1, 20, 100)
      expect(taskTree).not.toHaveAttribute('data-task-dragging')
      expect(screen.getAllByText('终端')).toHaveLength(2)
      expect(screen.getAllByText('终端')[0]).toBeVisible()

      dispatchPointerEvent(taskItem, 'pointerup', 1, 20, 100)
      expect(taskTree).not.toHaveAttribute('data-task-dragging')
      expect(screen.getAllByText('终端')[0]).toBeVisible()
    } finally {
      restorePointerTarget()
    }
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
    expect(onRunMenuCommand).toHaveBeenCalledWith('task-1', codexMenuItem.id)

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual(['Codex', '编辑任务'])
  })
})
