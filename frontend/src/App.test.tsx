import {cleanup, fireEvent, render, screen, waitFor} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

const bindings = vi.hoisted(() => ({
  CreateTask: vi.fn(),
  ListTasks: vi.fn(),
  StartTask: vi.fn(),
  FinishTask: vi.fn(),
  GetSettings: vi.fn(),
  SaveSettings: vi.fn(),
  DetectShells: vi.fn(),
  CreateTerminal: vi.fn(),
  OpenTaskFolder: vi.fn(),
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

describe('App confirmation flows', () => {
  afterEach(cleanup)

  beforeEach(() => {
    Object.values(bindings).forEach((mock) => mock.mockReset())
    Object.values(runtime).forEach((mock) => mock.mockReset())
    bindings.ListTasks.mockResolvedValue([{
      id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
    }])
    bindings.GetSettings.mockResolvedValue({workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh'})
    bindings.DetectShells.mockResolvedValue(['/bin/sh', '/bin/zsh'])
    bindings.FinishTask.mockResolvedValue({
      id: 'task-1', title: '清理临时文件', description: '', status: 'completed', createdAt: '2026-07-22T00:00:00Z',
    })
  })

  it('结束任务在确认前不会调用后端，并在确认后执行', async () => {
    const user = userEvent.setup()
    render(<App/>)

    await user.click(screen.getByRole('tab', {name: /执行中/}))
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

    await user.click(screen.getByRole('tab', {name: /执行中/}))
    await screen.findByText('清理临时文件')
    await user.click(screen.getByRole('button', {name: '退出应用'}))
    expect(screen.getByText('仍有执行中的任务')).toBeInTheDocument()
    expect(bindings.PrepareQuit).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', {name: '关闭终端并退出'}))
    expect(bindings.PrepareQuit).toHaveBeenCalledOnce()
    expect(bindings.FinishTask).not.toHaveBeenCalled()
  })

  it('保存颜色模式并在当前会话中保留选择', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    render(<App/>)

    await user.click(screen.getByRole('button', {name: '设置'}))
    await user.click(screen.getByLabelText('颜色模式'))
    await user.click(screen.getByRole('option', {name: '暗色'}))
    await user.click(screen.getByRole('button', {name: '保存'}))

    expect(bindings.SaveSettings).toHaveBeenCalledWith({
      workspaceRoot: '/tmp/workspaces',
      taskTreeWidth: 360,
      colorScheme: 'dark',
      shellPath: '/bin/sh',
    })

    await waitFor(() => expect(screen.queryByRole('dialog', {name: '设置'})).not.toBeInTheDocument())
    await user.click(screen.getByRole('button', {name: '设置'}))
    expect(screen.getByLabelText('颜色模式')).toHaveTextContent('暗色')
  })

  it('选择探测到的 Shell 并保存', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    render(<App/>)

    await waitFor(() => expect(bindings.DetectShells).toHaveBeenCalledOnce())
    await user.click(screen.getByRole('button', {name: '设置'}))
    await user.click(screen.getByLabelText('探测到的 Shell'))
    await user.click(screen.getByRole('option', {name: '/bin/zsh'}))
    await user.click(screen.getByRole('button', {name: '保存'}))

    expect(bindings.SaveSettings).toHaveBeenCalledWith({
      workspaceRoot: '/tmp/workspaces',
      taskTreeWidth: 360,
      colorScheme: 'light',
      shellPath: '/bin/zsh',
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
    })
  })

  it('通过颜色选择器创建未执行任务', async () => {
    const user = userEvent.setup()
    bindings.CreateTask.mockResolvedValue({
      id: 'task-2', title: '彩色任务', description: '', status: 'pending', color: '#22c55e', createdAt: '2026-07-22T00:00:00Z',
    })
    render(<App/>)

    await user.click(screen.getByRole('button', {name: '新建任务'}))
    await user.type(await screen.findByRole('textbox', {name: '标题'}), '彩色任务')
    fireEvent.change(screen.getByLabelText('任务颜色'), {target: {value: '#22c55e'}})
    await user.click(screen.getByRole('button', {name: '创建'}))

    expect(bindings.CreateTask).toHaveBeenCalledWith('彩色任务', '', '#22c55e')
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

    await user.click(screen.getByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '新增终端'}))
    await screen.findByText('终端视图')

    if (!terminalEventListener) {
      throw new Error('未注册终端事件监听器')
    }
    terminalEventListener({taskId: 'task-1', terminalId: 'terminal-1', type: 'exited'})

    await waitFor(() => expect(screen.queryByText('终端视图')).not.toBeInTheDocument())
  })
})
