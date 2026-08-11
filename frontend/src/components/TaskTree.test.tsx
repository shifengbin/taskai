import {cleanup, fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {readFileSync} from 'node:fs'
import {afterEach, describe, expect, it, vi} from 'vitest'

import {TaskTree} from './TaskTree'
import type {TaskMenuItem, TaskRecord, TerminalRecord} from '../types'

const appStyles = readFileSync('src/App.css', 'utf8')
const motionStyles = readFileSync('src/style.css', 'utf8')

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

  it('将任务和终端子项的操作按钮限定在上下文操作容器内', () => {
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
        onCloseTerminal={vi.fn()}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    const taskRow = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!(taskRow instanceof HTMLElement)) {
      throw new Error('未找到任务行')
    }
    const leadingTaskActions = screen.getByTestId('task-row-leading-actions-task-1')
    const trailingTaskActions = screen.getByTestId('task-row-trailing-actions-task-1')
    expect(leadingTaskActions).toContainElement(within(leadingTaskActions).getByRole('button', {name: '收起终端'}))
    expect(trailingTaskActions).toContainElement(within(trailingTaskActions).getByRole('button', {name: '结束'}))
    expect(trailingTaskActions).toContainElement(within(trailingTaskActions).getByRole('button', {name: '任务操作'}))

    const terminalItem = screen.getByText('终端').closest('[role="button"]')
    if (!(terminalItem instanceof HTMLElement)) {
      throw new Error('未找到终端条目')
    }
    const terminalActions = screen.getByTestId('task-terminal-actions-terminal-1')
    expect(terminalActions).toContainElement(within(terminalActions).getByRole('button', {name: '关闭终端'}))
    expect(terminalActions).not.toContainElement(within(terminalItem).getByRole('status', {name: '终端状态：空闲'}))
  })

  it('打开任务操作菜单时标记所属任务的操作区域为活动', async () => {
    const user = userEvent.setup()
    render(
      <TaskTree
        tasks={[runningTask]}
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

    const taskRow = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!(taskRow instanceof HTMLElement)) {
      throw new Error('未找到任务行')
    }
    await user.click(screen.getByRole('button', {name: '任务操作'}))

    expect(taskRow).toHaveAttribute('data-task-actions-active', 'true')
  })

  it('在窗口底部打开任务操作下拉菜单时，将菜单翻转到按钮上方', async () => {
    const user = userEvent.setup()
    const originalInnerHeight = Object.getOwnPropertyDescriptor(window, 'innerHeight')
    Object.defineProperty(window, 'innerHeight', {configurable: true, value: 720})
    const getBoundingClientRect = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.getAttribute('aria-label') === '任务操作') {
        return {top: 672, bottom: 700, left: 250, right: 278, width: 28, height: 28} as DOMRect
      }
      if (this.getAttribute('role') === 'menu') {
        return {top: 0, bottom: 160, left: 0, right: 192, width: 192, height: 160} as DOMRect
      }
      return {top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0} as DOMRect
    })
    try {
      render(
        <TaskTree
          tasks={[runningTask]}
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

      await user.click(screen.getByRole('button', {name: '任务操作'}))

      expect(screen.getByRole('menu')).toHaveStyle({top: '508px'})
    } finally {
      getBoundingClientRect.mockRestore()
      if (originalInnerHeight) {
        Object.defineProperty(window, 'innerHeight', originalInnerHeight)
      }
    }
  })

  it('在视口右下角打开任务右键菜单时，将菜单移动到视口内', () => {
    const onOpenTaskFolder = vi.fn()
    const originalInnerHeight = Object.getOwnPropertyDescriptor(window, 'innerHeight')
    const originalInnerWidth = Object.getOwnPropertyDescriptor(window, 'innerWidth')
    Object.defineProperty(window, 'innerHeight', {configurable: true, value: 720})
    Object.defineProperty(window, 'innerWidth', {configurable: true, value: 1024})
    const getBoundingClientRect = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.getAttribute('role') === 'menu') {
        return {top: 0, bottom: 160, left: 0, right: 192, width: 192, height: 160} as DOMRect
      }
      return {top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0} as DOMRect
    })
    try {
      render(
        <TaskTree
          tasks={[runningTask]}
          terminals={[]}
          selectedTerminalId={undefined}
          onSelectTask={vi.fn()}
          onSelectTerminal={vi.fn()}
          onCreateTerminal={vi.fn()}
          onEditTask={vi.fn()}
          onOpenTaskFolder={onOpenTaskFolder}
          onStartTask={vi.fn()}
          onFinishTask={vi.fn()}
          activeStatus="running"
          onChangeStatus={vi.fn()}
        />,
      )

      fireEvent.contextMenu(screen.getByText('整理发布说明'), {clientX: 1000, clientY: 700})

      expect(screen.getByRole('menu')).toHaveStyle({top: '530px', left: '824px'})
      fireEvent.click(screen.getByRole('menuitem', {name: '打开任务文件夹'}))
      expect(onOpenTaskFolder).toHaveBeenCalledWith('task-1')
      expect(screen.queryByRole('menu')).not.toBeInTheDocument()

      fireEvent.contextMenu(screen.getByText('整理发布说明'), {clientX: 1000, clientY: 700})
      fireEvent.keyDown(document, {key: 'Escape'})
      expect(screen.queryByRole('menu')).not.toBeInTheDocument()

      fireEvent.contextMenu(screen.getByText('整理发布说明'), {clientX: 1000, clientY: 700})
      fireEvent.pointerDown(document.body)
      expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    } finally {
      getBoundingClientRect.mockRestore()
      if (originalInnerHeight) {
        Object.defineProperty(window, 'innerHeight', originalInnerHeight)
      }
      if (originalInnerWidth) {
        Object.defineProperty(window, 'innerWidth', originalInnerWidth)
      }
    }
  })

  it('菜单高于可用视口时，限制菜单高度并保留内部滚动', async () => {
    const user = userEvent.setup()
    const originalInnerHeight = Object.getOwnPropertyDescriptor(window, 'innerHeight')
    const originalInnerWidth = Object.getOwnPropertyDescriptor(window, 'innerWidth')
    Object.defineProperty(window, 'innerHeight', {configurable: true, value: 720})
    Object.defineProperty(window, 'innerWidth', {configurable: true, value: 1024})
    const getBoundingClientRect = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.getAttribute('aria-label') === '任务操作') {
        return {top: 400, bottom: 428, left: 250, right: 278, width: 28, height: 28} as DOMRect
      }
      if (this.getAttribute('role') === 'menu') {
        return {top: 0, bottom: 800, left: 0, right: 192, width: 192, height: 800} as DOMRect
      }
      return {top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0} as DOMRect
    })
    try {
      render(
        <TaskTree
          tasks={[runningTask]}
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

      await user.click(screen.getByRole('button', {name: '任务操作'}))

      expect(screen.getByRole('menu')).toHaveStyle({top: '8px', maxHeight: '388px', maxWidth: '1008px', overflowY: 'auto'})
    } finally {
      getBoundingClientRect.mockRestore()
      if (originalInnerHeight) {
        Object.defineProperty(window, 'innerHeight', originalInnerHeight)
      }
      if (originalInnerWidth) {
        Object.defineProperty(window, 'innerWidth', originalInnerWidth)
      }
    }
  })

  it('任务列表滚动后重新计算打开菜单的位置', async () => {
    const user = userEvent.setup()
    const originalInnerHeight = Object.getOwnPropertyDescriptor(window, 'innerHeight')
    const originalInnerWidth = Object.getOwnPropertyDescriptor(window, 'innerWidth')
    Object.defineProperty(window, 'innerHeight', {configurable: true, value: 720})
    Object.defineProperty(window, 'innerWidth', {configurable: true, value: 1024})
    let buttonBounds = {top: 200, bottom: 228, left: 250, right: 278, width: 28, height: 28}
    const getBoundingClientRect = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.getAttribute('aria-label') === '任务操作') {
        return buttonBounds as DOMRect
      }
      if (this.getAttribute('role') === 'menu') {
        return {top: 0, bottom: 160, left: 0, right: 192, width: 192, height: 160} as DOMRect
      }
      return {top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0} as DOMRect
    })
    try {
      render(
        <TaskTree
          tasks={[runningTask]}
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

      await user.click(screen.getByRole('button', {name: '任务操作'}))
      expect(screen.getByRole('menu')).toHaveStyle({top: '232px'})

      buttonBounds = {top: 672, bottom: 700, left: 250, right: 278, width: 28, height: 28}
      fireEvent.scroll(window)

      expect(screen.getByRole('menu')).toHaveStyle({top: '508px'})
    } finally {
      getBoundingClientRect.mockRestore()
      if (originalInnerHeight) {
        Object.defineProperty(window, 'innerHeight', originalInnerHeight)
      }
      if (originalInnerWidth) {
        Object.defineProperty(window, 'innerWidth', originalInnerWidth)
      }
    }
  })

  it('窗口尺寸变化后重新计算打开菜单的位置', async () => {
    const user = userEvent.setup()
    const originalInnerHeight = Object.getOwnPropertyDescriptor(window, 'innerHeight')
    const originalInnerWidth = Object.getOwnPropertyDescriptor(window, 'innerWidth')
    Object.defineProperty(window, 'innerHeight', {configurable: true, value: 720})
    Object.defineProperty(window, 'innerWidth', {configurable: true, value: 1024})
    const getBoundingClientRect = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.getAttribute('aria-label') === '任务操作') {
        return {top: 672, bottom: 700, left: 250, right: 278, width: 28, height: 28} as DOMRect
      }
      if (this.getAttribute('role') === 'menu') {
        return {top: 0, bottom: 160, left: 0, right: 192, width: 192, height: 160} as DOMRect
      }
      return {top: 0, bottom: 0, left: 0, right: 0, width: 0, height: 0} as DOMRect
    })
    try {
      render(
        <TaskTree
          tasks={[runningTask]}
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

      await user.click(screen.getByRole('button', {name: '任务操作'}))
      expect(screen.getByRole('menu')).toHaveStyle({top: '508px'})

      Object.defineProperty(window, 'innerHeight', {configurable: true, value: 900})
      fireEvent.resize(window)

      expect(screen.getByRole('menu')).toHaveStyle({top: '704px'})
    } finally {
      getBoundingClientRect.mockRestore()
      if (originalInnerHeight) {
        Object.defineProperty(window, 'innerHeight', originalInnerHeight)
      }
      if (originalInnerWidth) {
        Object.defineProperty(window, 'innerWidth', originalInnerWidth)
      }
    }
  })

  it('仅在可悬停设备中以悬停、焦点或任务菜单状态显示上下文操作', () => {
    expect(appStyles).toContain('@media (hover: hover)')
    expect(appStyles).toContain('.taskai-contextual-actions')
    expect(appStyles).toContain('.taskai-contextual-container:hover .taskai-contextual-actions')
    expect(appStyles).toContain('.taskai-contextual-container:focus-within .taskai-contextual-actions')
    expect(appStyles).toContain('[data-task-actions-active="true"] .taskai-contextual-actions')
  })

  it('在执行中列表末尾以默认收起的同级区域展示搁置任务', async () => {
    const user = userEvent.setup()
    const shelvedTask = {...runningTask, id: 'task-2', title: '暂缓发布', shelved: true}
    render(
      <TaskTree
        tasks={[runningTask, shelvedTask]}
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

    expect(screen.getByText('整理发布说明')).toBeInTheDocument()
    expect(screen.queryByText('暂缓发布')).not.toBeInTheDocument()
    const toggle = screen.getByRole('button', {name: '展开已搁置任务'})
    expect(toggle).toHaveTextContent('已搁置 (1)')

    await user.click(toggle)
    expect(screen.getByText('暂缓发布')).toBeInTheDocument()

    await user.click(screen.getByRole('button', {name: '收起已搁置任务'}))
    expect(screen.queryByText('暂缓发布')).not.toBeInTheDocument()
  })

  it('为执行中任务的操作菜单提供搁置操作', async () => {
    const user = userEvent.setup()
    const onSetTaskShelved = vi.fn()
    const toggleShelvedItem: TaskMenuItem = {id: 'system.toggle-shelved', kind: 'toggle-shelved', name: '收纳任务', unshelveName: '恢复任务', showTerminal: false}
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        onSetTaskShelved={onSetTaskShelved}
        menuItems={[toggleShelvedItem]}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', {name: '任务操作'}))
    const shelveMenuItem = screen.getByRole('menuitem', {name: '收纳任务'})
    expect(shelveMenuItem).toBeInTheDocument()
    expect(shelveMenuItem.querySelector('svg')).toBeInTheDocument()
    await user.click(shelveMenuItem)
    expect(onSetTaskShelved).toHaveBeenCalledWith('task-1', true)

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    expect(screen.getByRole('menuitem', {name: '收纳任务'}).querySelector('svg')).toBeInTheDocument()
  })

  it('为已搁置任务的操作菜单提供取消搁置图标', async () => {
    const user = userEvent.setup()
    const shelvedTask = {...runningTask, shelved: true}
    const onSetTaskShelved = vi.fn()
    const toggleShelvedItem: TaskMenuItem = {id: 'system.toggle-shelved', kind: 'toggle-shelved', name: '收纳任务', unshelveName: '恢复任务', showTerminal: false}
    render(
      <TaskTree
        tasks={[shelvedTask]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        onSetTaskShelved={onSetTaskShelved}
        menuItems={[toggleShelvedItem]}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', {name: '展开已搁置任务'}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    const unshelveMenuItem = screen.getByRole('menuitem', {name: '恢复任务'})
    expect(unshelveMenuItem.querySelector('svg')).toBeInTheDocument()
    await user.click(unshelveMenuItem)
    expect(onSetTaskShelved).toHaveBeenCalledWith('task-1', false)

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    expect(screen.getByRole('menuitem', {name: '恢复任务'}).querySelector('svg')).toBeInTheDocument()
  })

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

  it('空闲终端以状态点表示状态，不显示状态文字', () => {
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
    expect(within(terminalItem).getByRole('status', {name: '终端状态：空闲'})).toBeInTheDocument()
    expect(within(terminalItem).queryByText('空闲', {exact: true})).not.toBeInTheDocument()
  })

  it('为当前选中的任务提供 Nebula 选择态标记而不改变卡片结构', () => {
    render(
      <TaskTree
        {...({
          tasks: [runningTask],
          terminals: [],
          selectedTerminalId: undefined,
          selectedTaskID: 'task-1',
          onSelectTask: vi.fn(),
          onSelectTerminal: vi.fn(),
          onCreateTerminal: vi.fn(),
          onEditTask: vi.fn(),
          onOpenTaskFolder: vi.fn(),
          onStartTask: vi.fn(),
          onFinishTask: vi.fn(),
          activeStatus: 'running',
          onChangeStatus: vi.fn(),
        } as any)}
      />,
    )

    const row = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!(row instanceof HTMLElement)) {
      throw new Error('未找到任务行')
    }
    expect(row).toHaveAttribute('data-task-selected', 'true')
    expect(row).toHaveClass('taskai-task-row--selected')
    expect(within(row).getByText('整理发布说明', {exact: true})).toBeInTheDocument()
    expect(within(row).getByText('完成发布说明', {exact: true})).toBeInTheDocument()
    expect(within(row).getByRole('button', {name: '结束'})).toBeInTheDocument()
    expect(within(row).getByRole('button', {name: '任务操作'})).toBeInTheDocument()
  })

  it('任务卡使用 Nebula 细边框、10px 圆角和内嵌色条', () => {
    render(
      <TaskTree
        {...({
          tasks: [runningTask],
          terminals: [],
          selectedTerminalId: undefined,
          onSelectTask: vi.fn(),
          onSelectTerminal: vi.fn(),
          onCreateTerminal: vi.fn(),
          onEditTask: vi.fn(),
          onOpenTaskFolder: vi.fn(),
          onStartTask: vi.fn(),
          onFinishTask: vi.fn(),
          activeStatus: 'running',
          onChangeStatus: vi.fn(),
        } as any)}
      />,
    )

    const row = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!(row instanceof HTMLElement)) {
      throw new Error('未找到任务行')
    }
    expect(row).toHaveStyle({'--task-color': '#2563eb'})
    expect(appStyles).toContain('.taskai-task-row {')
    expect(appStyles).toContain('border: 1px solid transparent;')
    expect(appStyles).toContain('border-radius: 10px;')
    expect(appStyles).toContain('box-shadow: inset 3px 0 0 var(--task-color);')
    expect(appStyles).not.toContain('2.5px solid var(--snap-outline)')
    expect(appStyles).not.toContain('box-shadow: 3px 3px 0 var(--snap-outline)')
    expect(appStyles).toContain('.taskai-task-row:hover')
    expect(appStyles).toContain('border-color: var(--snap-outline);')
  })

  it('将顶层任务项固定为 60px 高的两行 Nebula 排版', () => {
    render(
      <TaskTree
        tasks={[runningTask]}
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

    const row = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!(row instanceof HTMLElement)) {
      throw new Error('未找到任务行')
    }
    const title = within(row).getByText('整理发布说明', {exact: true})
    const description = within(row).getByText('完成发布说明', {exact: true})

    expect(row).toHaveClass('h-[60px]')
    expect(title).toHaveClass('font-display', 'text-[13.5px]', 'font-extrabold')
    expect(description).toHaveClass('font-sans', 'text-[11.5px]', 'font-medium', 'mt-px')
    expect(title.parentElement).toBe(description.parentElement)
    expect(title).toBe(title.parentElement?.firstElementChild)
    expect(description).toBe(title.parentElement?.lastElementChild)
  })

  it('缺少描述时仍在任务项的第二行显示暂无描述', () => {
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

    const row = screen.getByText('整理发布说明').closest('[data-task-id]')
    if (!(row instanceof HTMLElement)) {
      throw new Error('未找到任务行')
    }
    const title = within(row).getByText('整理发布说明', {exact: true})
    const description = within(row).getByText('暂无描述', {exact: true})

    expect(description).toHaveClass('font-sans', 'text-[11.5px]', 'font-medium', 'mt-px')
    expect(title).toBe(title.parentElement?.firstElementChild)
    expect(description).toBe(title.parentElement?.lastElementChild)
  })

  it('选中任务以青色描边和玻璃层保持 Nebula 卡片层级', () => {
    render(
      <TaskTree
        {...({
          tasks: [runningTask],
          terminals: [],
          selectedTerminalId: undefined,
          selectedTaskID: 'task-1',
          onSelectTask: vi.fn(),
          onSelectTerminal: vi.fn(),
          onCreateTerminal: vi.fn(),
          onEditTask: vi.fn(),
          onOpenTaskFolder: vi.fn(),
          onStartTask: vi.fn(),
          onFinishTask: vi.fn(),
          activeStatus: 'running',
          onChangeStatus: vi.fn(),
        } as any)}
      />,
    )

    expect(screen.getByText('整理发布说明').closest('[data-task-id]')).toHaveAttribute('data-task-selected', 'true')
    expect(appStyles).toContain('.taskai-task-row[data-task-selected="true"]')
    expect(appStyles).toContain('border-color: var(--snap-cobalt);')
    expect(appStyles).not.toContain('box-shadow: 3px 3px 0 var(--snap-cobalt)')
  })

  it('终端子项使用 8px Nebula 次级玻璃行，并让启动反馈独立于卡片表面', () => {
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[terminal]}
        selectedTerminalId="terminal-1"
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
    expect(terminalItem).toHaveClass('taskai-terminal-row')
    expect(terminalItem).toHaveAttribute('aria-pressed', 'true')
    expect(appStyles).toContain('.taskai-terminal-row {')
    expect(appStyles).toContain('border-radius: 8px;')
    expect(appStyles).toContain('background-color: var(--snap-surface-2);')
    expect(appStyles).toContain('.taskai-terminal-row:hover')
    expect(appStyles).toContain('.taskai-terminal-row[aria-pressed="true"]')
    expect(appStyles).toContain('border-color: var(--snap-cobalt);')
    expect(appStyles).not.toContain('box-shadow: 2px 2px 0 var(--snap-outline)')
    expect(motionStyles).toContain('.taskai-task-row[data-task-start-feedback="flash"]::after')
    expect(motionStyles).toContain('.taskai-task-row[data-task-start-feedback="static"]::after { opacity: 1; }')
    expect(motionStyles).not.toMatch(/@keyframes taskai-task-start-feedback\s*\{[^}]*box-shadow/)
  })

  it('任务收起时显示聚合状态点，展开后仅展示终端状态点', () => {
    const props = {
      tasks: [{...runningTask, realtimeStatus: 'unread' as const}],
      terminals: [],
      selectedTerminalId: undefined,
      onSelectTask: vi.fn(),
      onSelectTerminal: vi.fn(),
      onCreateTerminal: vi.fn(),
      onEditTask: vi.fn(),
      onOpenTaskFolder: vi.fn(),
      onStartTask: vi.fn(),
      onFinishTask: vi.fn(),
      activeStatus: 'running' as const,
      onChangeStatus: vi.fn(),
    }
    const {rerender} = render(<TaskTree {...props} expandedTasks={{[runningTask.id]: false}}/>)

    expect(screen.getByRole('status', {name: '终端状态：未读'})).toHaveAttribute('data-status', 'unread')

    rerender(<TaskTree {...props} expandedTasks={{[runningTask.id]: true}}/>)

    expect(screen.queryByRole('status', {name: '终端状态：未读'})).not.toBeInTheDocument()
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

  it('通过状态标签筛选任务并使用同色标记与极浅底色', async () => {
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

    expect(screen.getByText('待处理任务').closest('[data-task-id]')).toHaveStyle({'--task-color': '#f97316'})
    expect(appStyles).toContain('background-color: color-mix(in srgb, var(--task-color) 4%, var(--snap-surface));')
    expect(screen.queryByText('整理发布说明')).not.toBeInTheDocument()
    expect(screen.queryByText('已完成任务')).not.toBeInTheDocument()

    await user.click(screen.getByRole('tab', {name: /执行中/}))
    expect(onChangeStatus).toHaveBeenCalledWith('running')
  })

  it.each(['pending', 'running', 'completed'] as const)('在%s状态下使用可收缩的无滚动条列表并保留任务选择', async (activeStatus) => {
    const user = userEvent.setup()
    const onSelectTask = vi.fn()
    const visibleTasks = Array.from({length: 12}, (_, index) => ({
      ...runningTask,
      id: `task-${index + 1}`,
      title: `${activeStatus}任务 ${index + 1}`,
      status: activeStatus,
    }))
    render(
      <TaskTree
        tasks={visibleTasks}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={onSelectTask}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        activeStatus={activeStatus}
        onChangeStatus={vi.fn()}
      />,
    )

    const taskList = screen.getByTestId('task-tree-list')
    expect(taskList).toHaveStyle({minHeight: '0'})
    expect(taskList).toHaveStyle({overflowX: 'hidden'})
    expect(taskList).toHaveStyle({overflowY: 'auto'})
    expect(taskList).toHaveStyle({scrollbarWidth: 'none'})

    taskList.scrollTop = 48
    fireEvent.scroll(taskList)
    await user.click(screen.getByText(`${activeStatus}任务 12`))
    expect(onSelectTask).toHaveBeenCalledWith(expect.objectContaining({id: 'task-12'}))
  })

  it('异常退出终端以状态点表示异常状态，并允许关闭', async () => {
    const user = userEvent.setup()
    const onCloseTerminal = vi.fn()
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
        onCloseTerminal={onCloseTerminal}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    const terminalItem = screen.getByText('终端').closest('[role="button"]')
    if (!(terminalItem instanceof HTMLElement)) {
      throw new Error('未找到终端条目')
    }
    expect(within(terminalItem).getByRole('status', {name: '终端状态：异常'})).toBeInTheDocument()
    expect(within(terminalItem).queryByText('异常', {exact: true})).not.toBeInTheDocument()

    await user.click(within(terminalItem).getByRole('button', {name: '关闭终端'}))
    expect(onCloseTerminal).toHaveBeenCalledWith({...terminal, state: 'exited'})
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

  it('悬浮任务条目时按原始换行显示描述', async () => {
    const user = userEvent.setup()
    const description = '第一行说明\n  缩进的第二行说明'
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
    const tooltip = await screen.findByRole('tooltip')
    expect(tooltip.textContent).toBe(description)
    expect(tooltip).toHaveClass('whitespace-pre-wrap')
  })

  it('悬浮缺少描述的任务时使用暂无描述作为提示回退', async () => {
    const user = userEvent.setup()
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

    await user.hover(screen.getByText('整理发布说明'))
    expect(await screen.findByRole('tooltip')).toHaveTextContent(/^暂无描述$/)
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
    const toggleShelvedItem: TaskMenuItem = {id: 'system.toggle-shelved', kind: 'toggle-shelved', name: '收纳任务', unshelveName: '恢复任务', showTerminal: false}
    render(
      <TaskTree
        tasks={[runningTask]}
        terminals={[]}
        selectedTerminalId={undefined}
        menuItems={[toggleShelvedItem, codexMenuItem, {id: 'system.edit-task', kind: 'edit-task', name: '编辑任务', showTerminal: false}]}
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
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual(['收纳任务', 'Codex', '编辑任务'])
    await user.click(screen.getByRole('menuitem', {name: 'Codex'}))
    expect(onRunMenuCommand).toHaveBeenCalledWith('task-1', codexMenuItem.id)

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    expect(screen.getAllByRole('menuitem').map((item) => item.textContent)).toEqual(['收纳任务', 'Codex', '编辑任务'])
  })

  it('显示失败命令链进度、禁用任务操作并允许从首命令重试', async () => {
    const user = userEvent.setup()
    const onRetryLifecycle = vi.fn()
    const lockedTask: TaskRecord = {
      ...runningTask,
      lifecycleExecution: {
        hook: 'postStart', chainId: 'chain-1', currentCommandId: 'deploy', currentCommandName: '部署项目',
        currentIndex: 2, commandCount: 3, state: 'failed', error: '命令退出码 1',
      },
    }
    render(
      <TaskTree
        tasks={[lockedTask]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        onRetryLifecycle={onRetryLifecycle}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    expect(screen.getByText('开始后 · 部署项目 2/3')).toBeInTheDocument()
    expect(screen.getByText('开始后 · 部署项目 2/3').closest('.taskai-lifecycle-chip')).toHaveClass('taskai-lifecycle-chip--error')
    expect(screen.getByRole('button', {name: '任务操作'})).toBeDisabled()
    expect(screen.getByRole('button', {name: '结束'})).toBeDisabled()
    await user.click(screen.getByRole('button', {name: '重试命令链'}))
    expect(onRetryLifecycle).toHaveBeenCalledWith('task-1')

    fireEvent.contextMenu(screen.getByText('整理发布说明'))
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('运行中的命令链仅显示当前步骤并锁定任务操作，不显示重试', () => {
    const lockedTask: TaskRecord = {
      ...runningTask,
      lifecycleExecution: {
        runId: 'run-1', revision: 1,
        hook: 'updateTask', chainId: 'chain-1', currentCommandId: 'prepare', currentCommandName: '准备环境',
        currentIndex: 1, commandCount: 3, state: 'running',
      },
    }
    render(
      <TaskTree
        tasks={[lockedTask]}
        terminals={[]}
        selectedTerminalId={undefined}
        onSelectTask={vi.fn()}
        onSelectTerminal={vi.fn()}
        onCreateTerminal={vi.fn()}
        onEditTask={vi.fn()}
        onOpenTaskFolder={vi.fn()}
        onStartTask={vi.fn()}
        onFinishTask={vi.fn()}
        onRetryLifecycle={vi.fn()}
        activeStatus="running"
        onChangeStatus={vi.fn()}
      />,
    )

    expect(screen.getByText('更新后 · 准备环境 1/3')).toBeInTheDocument()
    expect(screen.getByText('更新后 · 准备环境 1/3').closest('.taskai-lifecycle-chip')).toHaveClass('taskai-lifecycle-chip--warning')
    expect(screen.getByRole('button', {name: '任务操作'})).toBeDisabled()
    expect(screen.getByRole('button', {name: '结束'})).toBeDisabled()
    expect(screen.queryByRole('button', {name: '重试命令链'})).not.toBeInTheDocument()
  })

  it('仅为刚开始的目标任务显示快速反馈', () => {
    const otherTask: TaskRecord = {...runningTask, id: 'task-2', title: '处理发布反馈'}
    render(
      <TaskTree
        tasks={[runningTask, otherTask]}
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
        startedTaskFeedback={{taskID: 'task-1', sequence: 1}}
      />,
    )

    expect(screen.getByText('整理发布说明').closest('[data-task-id]')).toHaveAttribute('data-task-start-feedback', 'flash')
    expect(screen.getByText('处理发布反馈').closest('[data-task-id]')).not.toHaveAttribute('data-task-start-feedback')
  })

  it('开始反馈时将列表下方被裁切的目标任务滚动到可视区域', () => {
    const startedTask: TaskRecord = {...runningTask, id: 'task-2', title: '刚开始的发布任务'}
    const renderTaskTree = (startedTaskFeedback?: {taskID: string, sequence: number}) => (
      <TaskTree
        tasks={[runningTask, startedTask]}
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
        startedTaskFeedback={startedTaskFeedback}
      />
    )
    const {rerender} = render(renderTaskTree())
    const taskList = screen.getByTestId('task-tree-list')
    const taskItem = screen.getByText('刚开始的发布任务').closest('[data-task-id]')
    if (!taskItem) {
      throw new Error('未找到刚开始的任务条目')
    }
    Object.defineProperty(taskList, 'scrollTop', {configurable: true, writable: true, value: 80})
    vi.spyOn(taskList, 'getBoundingClientRect').mockReturnValue({top: 100, bottom: 200} as DOMRect)
    vi.spyOn(taskItem, 'getBoundingClientRect').mockReturnValue({top: 210, bottom: 258} as DOMRect)

    rerender(renderTaskTree({taskID: 'task-2', sequence: 1}))

    expect(taskList.scrollTop).toBe(138)
    expect(taskItem).toHaveAttribute('data-task-start-feedback', 'flash')
  })

  it('开始反馈时将列表上方被裁切的目标任务滚动到可视区域', () => {
    const startedTask: TaskRecord = {...runningTask, id: 'task-2', title: '刚开始的发布任务'}
    const renderTaskTree = (startedTaskFeedback?: {taskID: string, sequence: number}) => (
      <TaskTree
        tasks={[runningTask, startedTask]}
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
        startedTaskFeedback={startedTaskFeedback}
      />
    )
    const {rerender} = render(renderTaskTree())
    const taskList = screen.getByTestId('task-tree-list')
    const taskItem = screen.getByText('刚开始的发布任务').closest('[data-task-id]')
    if (!taskItem) {
      throw new Error('未找到刚开始的任务条目')
    }
    Object.defineProperty(taskList, 'scrollTop', {configurable: true, writable: true, value: 80})
    vi.spyOn(taskList, 'getBoundingClientRect').mockReturnValue({top: 100, bottom: 200} as DOMRect)
    vi.spyOn(taskItem, 'getBoundingClientRect').mockReturnValue({top: 60, bottom: 108} as DOMRect)

    rerender(renderTaskTree({taskID: 'task-2', sequence: 1}))

    expect(taskList.scrollTop).toBe(40)
  })

  it('开始反馈目标已完整可见或未渲染时保持列表滚动位置', () => {
    const startedTask: TaskRecord = {...runningTask, id: 'task-2', title: '刚开始的发布任务'}
    const renderTaskTree = (startedTaskFeedback?: {taskID: string, sequence: number}) => (
      <TaskTree
        tasks={[runningTask, startedTask]}
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
        startedTaskFeedback={startedTaskFeedback}
      />
    )
    const {rerender} = render(renderTaskTree())
    const taskList = screen.getByTestId('task-tree-list')
    const taskItem = screen.getByText('刚开始的发布任务').closest('[data-task-id]')
    if (!taskItem) {
      throw new Error('未找到刚开始的任务条目')
    }
    Object.defineProperty(taskList, 'scrollTop', {configurable: true, writable: true, value: 80})
    vi.spyOn(taskList, 'getBoundingClientRect').mockReturnValue({top: 100, bottom: 200} as DOMRect)
    vi.spyOn(taskItem, 'getBoundingClientRect').mockReturnValue({top: 120, bottom: 168} as DOMRect)

    rerender(renderTaskTree({taskID: 'task-2', sequence: 1}))
    expect(taskList.scrollTop).toBe(80)

    rerender(renderTaskTree({taskID: 'missing-task', sequence: 2}))
    expect(taskList.scrollTop).toBe(80)
  })

  it('减少动态效果时将列表外开始任务定位并替换为静态高亮', () => {
    const originalMatchMedia = window.matchMedia
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn(() => ({matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn()})),
    })
    try {
      const renderTaskTree = (startedTaskFeedback?: {taskID: string, sequence: number}) => (
        <TaskTree
          tasks={[runningTask]}
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
          startedTaskFeedback={startedTaskFeedback}
        />
      )
      const {rerender} = render(renderTaskTree())
      const taskList = screen.getByTestId('task-tree-list')
      const taskItem = screen.getByText('整理发布说明').closest('[data-task-id]')
      if (!taskItem) {
        throw new Error('未找到刚开始的任务条目')
      }
      Object.defineProperty(taskList, 'scrollTop', {configurable: true, writable: true, value: 80})
      vi.spyOn(taskList, 'getBoundingClientRect').mockReturnValue({top: 100, bottom: 200} as DOMRect)
      vi.spyOn(taskItem, 'getBoundingClientRect').mockReturnValue({top: 210, bottom: 258} as DOMRect)

      rerender(renderTaskTree({taskID: 'task-1', sequence: 1}))

      expect(taskList.scrollTop).toBe(138)
      expect(taskItem).toHaveAttribute('data-task-start-feedback', 'static')
    } finally {
      Object.defineProperty(window, 'matchMedia', {configurable: true, value: originalMatchMedia})
    }
  })

  it('在未执行选择模式中仅允许勾选无生命周期执行记录的任务', async () => {
    const user = userEvent.setup()
    const selectableTask: TaskRecord = {...runningTask, id: 'pending-1', title: '可删除任务', status: 'pending'}
    const lockedTask: TaskRecord = {
      ...runningTask,
      id: 'pending-2',
      title: '待重试开始任务',
      status: 'pending',
      lifecycleExecution: {hook: 'beforeStart', chainId: 'prepare', currentIndex: 1, commandCount: 1, state: 'failed'},
    }
    const onToggleTaskDeletion = vi.fn()
    render(
      <TaskTree
        tasks={[selectableTask, lockedTask]}
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
        onChangeStatus={vi.fn()}
        taskDeletionSelectionMode
        selectedTaskIDs={['pending-1']}
        onToggleTaskDeletion={onToggleTaskDeletion}
      />,
    )

    const selectableCheckbox = screen.getByRole('checkbox', {name: '选择任务 可删除任务'})
    expect(selectableCheckbox).toBeChecked()
    await user.click(selectableCheckbox)
    expect(onToggleTaskDeletion).toHaveBeenCalledWith('pending-1')

    const lockedCheckbox = screen.getByRole('checkbox', {name: '选择任务 待重试开始任务'})
    expect(lockedCheckbox).toBeDisabled()
    expect(onToggleTaskDeletion).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('button', {name: '执行'})).not.toBeInTheDocument()
    expect(screen.queryByRole('button', {name: '重试命令链'})).not.toBeInTheDocument()
  })
})
