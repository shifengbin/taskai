import {cleanup, fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

const bindings = vi.hoisted(() => ({
  CreateTask: vi.fn(),
  UpdateTask: vi.fn(),
  ListTasks: vi.fn(),
  ReorderTasks: vi.fn(),
  StartTask: vi.fn(),
  FinishTask: vi.fn(),
  GetSettings: vi.fn(),
  SaveSettings: vi.fn(),
  DetectShells: vi.fn(),
  CreateTerminal: vi.fn(),
  CreateCommandTerminal: vi.fn(),
  OpenTaskFolder: vi.fn(),
  RunTaskCommand: vi.fn(),
  WriteTerminal: vi.fn(),
  ResizeTerminal: vi.fn(),
  CloseTerminal: vi.fn(),
  HasRunningTasks: vi.fn(),
  PrepareQuit: vi.fn(),
}))
const runtime = vi.hoisted(() => ({EventsOn: vi.fn(), EventsOff: vi.fn(), Quit: vi.fn()}))

vi.mock('../wailsjs/go/main/App', () => bindings)
vi.mock('../wailsjs/runtime/runtime', () => runtime)
vi.mock('./components/TerminalView', () => ({TerminalView: () => <div>终端视图</div>}))

import App from './App'

const fixedTaskMenuItems = [
  {id: 'system.edit-task', kind: 'edit-task', name: '编辑任务', showTerminal: false},
  {id: 'system.create-terminal', kind: 'create-terminal', name: '新增终端', showTerminal: false},
  {id: 'system.open-folder', kind: 'open-folder', name: '打开任务文件夹', showTerminal: false},
]

function dispatchPointerEvent(target: Element, type: string, pointerId: number, clientX: number, clientY: number) {
  const event = new MouseEvent(type, {bubbles: true, cancelable: true, button: 0, clientX, clientY})
  Object.defineProperty(event, 'pointerId', {configurable: true, value: pointerId})
  fireEvent(target, event)
}

describe('App confirmation flows', () => {
  afterEach(cleanup)

  beforeEach(() => {
    Object.values(bindings).forEach((mock) => mock.mockReset())
    Object.values(runtime).forEach((mock) => mock.mockReset())
    bindings.SaveSettings.mockImplementation(async (next) => next)
    bindings.ListTasks.mockResolvedValue([{
      id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
    }])
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
    })
    bindings.DetectShells.mockResolvedValue(['/bin/sh', '/bin/zsh'])
    bindings.FinishTask.mockResolvedValue({
      id: 'task-1', title: '清理临时文件', description: '', status: 'completed', createdAt: '2026-07-22T00:00:00Z',
    })
  })

  it('结束任务在确认前不会调用后端，并在确认后执行', async () => {
    const user = userEvent.setup()
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await screen.findByText('清理临时文件')
    await user.click(screen.getByRole('button', {name: '结束'}))
    expect(screen.getByText('结束任务？')).toBeInTheDocument()
    expect(bindings.FinishTask).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', {name: '取消'}))
    expect(bindings.FinishTask).not.toHaveBeenCalled()
    await waitFor(() => expect(screen.queryByText('结束任务？')).not.toBeInTheDocument())

    await user.click(screen.getByRole('button', {name: '结束'}))
    await user.click(screen.getByRole('button', {name: '结束并删除'}))
    expect(bindings.FinishTask).toHaveBeenCalledWith('task-1')
  })

  it('退出前确认会保留任务状态，只请求关闭终端', async () => {
    const user = userEvent.setup()
    bindings.HasRunningTasks.mockResolvedValue(true)
    bindings.PrepareQuit.mockResolvedValue(undefined)
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await screen.findByText('清理临时文件')
    await user.click(screen.getByRole('button', {name: '退出应用'}))
    expect(screen.getByText('仍有执行中的任务')).toBeInTheDocument()
    expect(bindings.PrepareQuit).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', {name: '关闭终端并退出'}))
    expect(bindings.PrepareQuit).toHaveBeenCalledOnce()
    expect(bindings.FinishTask).not.toHaveBeenCalled()
  })

  it('恢复上次选中的任务标签，并在切换后持久化选择', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'running',
    })
    render(<App/>)

    const runningTab = await screen.findByRole('tab', {name: /执行中/})
    expect(runningTab).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByText('清理临时文件')).toBeInTheDocument()

    await user.click(screen.getByRole('tab', {name: /已完成/}))
    await waitFor(() => expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining({activeTaskStatus: 'completed'})))
  })

  it('指针拖动任务条目后持久化同状态内的新顺序', async () => {
    bindings.ListTasks.mockResolvedValue([
      {id: 'task-1', title: '先处理的任务', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z'},
      {id: 'task-2', title: '后处理的任务', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z'},
    ])
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'running',
    })
    bindings.ReorderTasks.mockResolvedValue([
      {id: 'task-2', title: '后处理的任务', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z'},
      {id: 'task-1', title: '先处理的任务', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z'},
    ])
    render(<App/>)

    const source = (await screen.findByText('先处理的任务')).closest('[data-task-id]')
    const target = screen.getByText('后处理的任务').closest('[data-task-id]')
    if (!source || !target) {
      throw new Error('未找到可拖动的任务条目')
    }
    Object.defineProperty(target, 'getBoundingClientRect', {
      configurable: true,
      value: () => ({top: 100, height: 48}),
    })
    const descriptor = Object.getOwnPropertyDescriptor(document, 'elementFromPoint')
    Object.defineProperty(document, 'elementFromPoint', {configurable: true, value: () => target})
    try {
      dispatchPointerEvent(source, 'pointerdown', 1, 20, 0)
      dispatchPointerEvent(source, 'pointermove', 1, 20, 140)
      dispatchPointerEvent(source, 'pointerup', 1, 20, 140)

      await waitFor(() => expect(bindings.ReorderTasks).toHaveBeenCalledWith('running', ['task-2', 'task-1']))
    } finally {
      if (descriptor) {
        Object.defineProperty(document, 'elementFromPoint', descriptor)
      } else {
        Reflect.deleteProperty(document, 'elementFromPoint')
      }
    }
  })

  it('初始数据加载完成前仅显示稳定的加载页面', async () => {
    let resolveTasks: (tasks: Array<Record<string, unknown>>) => void
    bindings.ListTasks.mockImplementation(() => new Promise((resolve) => {
      resolveTasks = resolve
    }))

    render(<App/>)

    expect(screen.getByRole('status', {name: '正在加载任务工作台'})).toBeInTheDocument()
    expect(screen.queryByRole('navigation', {name: '任务和终端'})).not.toBeInTheDocument()

    resolveTasks!([{
      id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
    }])

    await screen.findByRole('navigation', {name: '任务和终端'})
    expect(screen.queryByRole('status', {name: '正在加载任务工作台'})).not.toBeInTheDocument()
  })

  it('保存颜色模式并在当前会话中保留选择', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '设置'}))
    await user.click(screen.getByLabelText('颜色模式'))
    await user.click(screen.getByRole('option', {name: '暗色'}))
    await user.click(screen.getByRole('button', {name: '保存'}))

    expect(bindings.SaveSettings).toHaveBeenCalledWith({
      workspaceRoot: '/tmp/workspaces',
      taskTreeWidth: 360,
      colorScheme: 'dark',
      shellPath: '/bin/sh',
      taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'pending',
    })

    await waitFor(() => expect(screen.queryByRole('dialog', {name: '设置'})).not.toBeInTheDocument())
    await user.click(await screen.findByRole('button', {name: '设置'}))
    expect(screen.getByLabelText('颜色模式')).toHaveTextContent('暗色')
  })

  it('选择探测到的 Shell 并保存', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    render(<App/>)

    await waitFor(() => expect(bindings.DetectShells).toHaveBeenCalledOnce())
    await user.click(await screen.findByRole('button', {name: '设置'}))
    await user.click(screen.getByLabelText('探测到的 Shell'))
    await user.click(screen.getByRole('option', {name: '/bin/zsh'}))
    await user.click(screen.getByRole('button', {name: '保存'}))

    expect(bindings.SaveSettings).toHaveBeenCalledWith({
      workspaceRoot: '/tmp/workspaces',
      taskTreeWidth: 360,
      colorScheme: 'light',
      shellPath: '/bin/zsh',
      taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'pending',
    })
  })

  it('允许手动设置 Shell 路径', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    render(<App/>)

    await waitFor(() => expect(bindings.DetectShells).toHaveBeenCalledOnce())
    await user.click(screen.getByRole('button', {name: '设置'}))
    const shellPath = await screen.findByRole('textbox', {name: 'Shell 路径'})
    await user.clear(shellPath)
    await user.type(shellPath, '/custom/shell')
    await user.click(screen.getByRole('button', {name: '保存'}))

    expect(bindings.SaveSettings).toHaveBeenCalledWith({
      workspaceRoot: '/tmp/workspaces',
      taskTreeWidth: 360,
      colorScheme: 'light',
      shellPath: '/custom/shell',
      taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'pending',
    })
  })

  it('通过颜色选择器创建未执行任务', async () => {
    const user = userEvent.setup()
    bindings.CreateTask.mockResolvedValue({
      id: 'task-2', title: '彩色任务', description: '', status: 'pending', color: '#22c55e', createdAt: '2026-07-22T00:00:00Z',
    })
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '新建任务'}))
    await user.type(await screen.findByRole('textbox', {name: '标题'}), '彩色任务')
    fireEvent.change(screen.getByLabelText('任务颜色'), {target: {value: '#22c55e'}})
    await user.click(screen.getByRole('button', {name: '创建'}))

    expect(bindings.CreateTask).toHaveBeenCalledWith('彩色任务', '', '#22c55e')
  })

  it('通过任务操作编辑标题、描述和颜色', async () => {
    const user = userEvent.setup()
    bindings.UpdateTask.mockResolvedValue({
      id: 'task-1', title: '更新后的临时文件', description: '已补充说明', status: 'running', color: '#22c55e', createdAt: '2026-07-22T00:00:00Z',
    })
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
    expect(screen.getByText('编辑任务')).toBeInTheDocument()
    await user.clear(screen.getByRole('textbox', {name: '标题'}))
    await user.type(screen.getByRole('textbox', {name: '标题'}), '更新后的临时文件')
    await user.type(screen.getByRole('textbox', {name: '任务描述'}), '已补充说明')
    fireEvent.change(screen.getByLabelText('任务颜色'), {target: {value: '#22c55e'}})
    await user.click(screen.getByRole('button', {name: '保存'}))

    expect(bindings.UpdateTask).toHaveBeenCalledWith('task-1', '更新后的临时文件', '已补充说明', '#22c55e')
    await screen.findByText('更新后的临时文件')
  })

  it('终端退出后不再显示右侧终端视图', async () => {
    const user = userEvent.setup()
    let terminalEventListener: ((event: {taskId: string; terminalId: string; type: 'exited'}) => void) | undefined
    runtime.EventsOn.mockImplementation((eventName, listener) => {
      if (eventName === 'task-terminal:event') {
        terminalEventListener = listener
      }
    })
    bindings.CreateTerminal.mockResolvedValue({id: 'terminal-1', taskId: 'task-1', state: 'active'})
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '新增终端'}))
    await screen.findByText('终端视图')

    if (!terminalEventListener) {
      throw new Error('未注册终端事件监听器')
    }
    terminalEventListener({taskId: 'task-1', terminalId: 'terminal-1', type: 'exited'})

    await waitFor(() => expect(screen.queryByText('终端视图')).not.toBeInTheDocument())
  })

  it('自定义菜单可创建独立终端或后台启动命令', async () => {
    const user = userEvent.setup()
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh',
      taskMenuItems: [
        {id: 'custom-codex', kind: 'command', name: 'Codex', command: 'codex', arguments: ['--full-auto'], showTerminal: true},
        {id: 'custom-vscode', kind: 'command', name: '打开 VS Code', command: 'code', arguments: ['.'], showTerminal: false},
      ],
    })
    bindings.CreateCommandTerminal.mockResolvedValue({id: 'terminal-codex', taskId: 'task-1', state: 'active'})
    bindings.RunTaskCommand.mockResolvedValue(undefined)
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: 'Codex'}))
    expect(bindings.CreateCommandTerminal).toHaveBeenCalledWith('task-1', 'codex', ['--full-auto'], 100, 32)
    await screen.findByText('终端视图')

    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '打开 VS Code'}))
    expect(bindings.RunTaskCommand).toHaveBeenCalledWith('task-1', 'code', ['.'])
  })

  it('设置中通过独立弹窗配置菜单项并调整系统项顺序', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh',
      taskMenuItems: [
        {id: 'system.edit-task', kind: 'edit-task', name: '编辑任务', showTerminal: false},
        {id: 'system.create-terminal', kind: 'create-terminal', name: '新增终端', showTerminal: false},
        {id: 'system.open-folder', kind: 'open-folder', name: '打开任务文件夹', showTerminal: false},
      ],
    })
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '设置'}))
    expect(screen.getByText('工作区与外观')).toBeInTheDocument()
    expect(screen.getByText('终端 Shell')).toBeInTheDocument()
    await user.click(screen.getByRole('button', {name: '新增菜单项'}))

    const createDialog = screen.getByRole('dialog', {name: '新增菜单项'})
    await user.clear(within(createDialog).getByRole('textbox', {name: '菜单名称'}))
    await user.type(within(createDialog).getByRole('textbox', {name: '菜单名称'}), 'Codex')
    await user.type(within(createDialog).getByRole('textbox', {name: '启动命令'}), 'codex')
    await user.type(within(createDialog).getByRole('textbox', {name: '启动参数（每行一个）'}), '--full-auto\n--dangerously-bypass-approvals-and-sandbox')
    await user.click(within(createDialog).getByRole('button', {name: '添加菜单项'}))
    await waitFor(() => expect(screen.queryByRole('dialog', {name: '新增菜单项'})).not.toBeInTheDocument())
    expect(bindings.SaveSettings).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', {name: '编辑菜单项 Codex'}))
    const editDialog = screen.getByRole('dialog', {name: '编辑菜单项'})
    await user.clear(within(editDialog).getByRole('textbox', {name: '菜单名称'}))
    await user.type(within(editDialog).getByRole('textbox', {name: '菜单名称'}), 'Codex CLI')
    expect(within(editDialog).getByRole('switch', {name: '显示终端'})).toBeChecked()
    await user.click(within(editDialog).getByRole('button', {name: '保存菜单项'}))

    await user.click(screen.getByRole('button', {name: '上移 打开任务文件夹'}))
    expect(screen.queryByRole('button', {name: '编辑菜单项 编辑任务'})).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', {name: '保存'}))

    const saved = bindings.SaveSettings.mock.calls[0][0]
    expect(saved.taskMenuItems.map((item: {id: string}) => item.id)).toEqual([
      'system.edit-task', 'system.open-folder', 'system.create-terminal', saved.taskMenuItems[3].id,
    ])
    expect(saved.taskMenuItems[3]).toMatchObject({
      kind: 'command', name: 'Codex CLI', command: 'codex', arguments: ['--full-auto', '--dangerously-bypass-approvals-and-sandbox'], showTerminal: true,
    })
  })
})
