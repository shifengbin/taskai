import {act, cleanup, fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

const bindings = vi.hoisted(() => ({
  ClearSelectedTerminal: vi.fn(),
  CreateTask: vi.fn(),
	CreateTaskWithExtraInfo: vi.fn(),
  UpdateTask: vi.fn(),
	UpdateTaskWithExtraInfo: vi.fn(),
	ListTasks: vi.fn(),
	ListExtraInfoCatalogues: vi.fn(),
	ListExtraInfoTemplates: vi.fn(),
	SaveExtraInfoCatalogue: vi.fn(),
	SaveExtraInfoTemplate: vi.fn(),
	DeleteExtraInfoCatalogue: vi.fn(),
	DeleteExtraInfoTemplate: vi.fn(),
  ReorderTasks: vi.fn(),
	ReportTerminalTitleActivity: vi.fn(),
  StartTask: vi.fn(),
  FinishTask: vi.fn(),
  GetSettings: vi.fn(),
  SaveSettings: vi.fn(),
	SelectTerminal: vi.fn(),
  DetectShells: vi.fn(),
  CreateTerminal: vi.fn(),
  CreateCommandTerminal: vi.fn(),
  ExecuteTaskMenuCommand: vi.fn(),
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
vi.mock('./components/TerminalView', async () => {
  const {terminalDisplayName} = await vi.importActual<typeof import('./types')>('./types')
  return {
    TerminalView: ({terminal}: {terminal: {title?: string, output?: string}}) => <>
      <div>终端视图</div>
      <div data-testid="terminal-view-title-container">
        <div data-testid="terminal-view-title" aria-label="右侧终端标题">{terminalDisplayName(terminal)}</div>
      </div>
      <div aria-label="右侧终端原始输出">{terminal.output}</div>
    </>,
  }
})

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
		bindings.ClearSelectedTerminal.mockResolvedValue(undefined)
		bindings.SelectTerminal.mockResolvedValue(undefined)
		bindings.ReportTerminalTitleActivity.mockResolvedValue(true)
		bindings.ListExtraInfoCatalogues.mockResolvedValue([])
		bindings.ListExtraInfoTemplates.mockResolvedValue([])
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

  it('额外信息模板接口返回空值时仍可渲染工作台并响应原生关闭请求', async () => {
    let closeRequested: (() => void) | undefined
    runtime.EventsOn.mockImplementation((eventName, listener) => {
      if (eventName === 'application:close-requested') {
        closeRequested = listener
      }
    })
    bindings.ListExtraInfoTemplates.mockResolvedValue(null)

    render(<App/>)

    await screen.findByRole('navigation', {name: '任务和终端'})
    expect(screen.getByRole('button', {name: '额外信息管理'})).toBeInTheDocument()
    if (!closeRequested) {
      throw new Error('未注册原生关闭请求监听器')
    }
    act(() => closeRequested!())
    expect(screen.getByRole('dialog', {name: '仍有执行中的任务'})).toBeInTheDocument()
  })

  it('保存颜色模式并在当前会话中保留选择', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '设置'}))
    await user.click(screen.getByLabelText('颜色模式'))
    await user.click(screen.getByRole('option', {name: '暗色'}))
    await user.click(screen.getByRole('tab', {name: '终端 Shell'}))
    await user.click(screen.getByRole('tab', {name: '工作区与外观'}))
    expect(screen.getByLabelText('颜色模式')).toHaveTextContent('暗色')
    await user.click(screen.getByRole('button', {name: '保存'}))

    expect(bindings.SaveSettings).toHaveBeenCalledWith({
      workspaceRoot: '/tmp/workspaces',
      taskTreeWidth: 360,
      colorScheme: 'dark',
      shellPath: '/bin/sh',
      taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'pending',
		statusManagementMode: 'title-change',
		statusManagementHTTPPort: 0,
		httpServiceEnabled: false,
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
    await user.click(screen.getByRole('tab', {name: '终端 Shell'}))
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
		statusManagementMode: 'title-change',
		statusManagementHTTPPort: 0,
		httpServiceEnabled: false,
    })
  })

  it('允许手动设置 Shell 路径', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockImplementation(async (next) => next)
    render(<App/>)

    await waitFor(() => expect(bindings.DetectShells).toHaveBeenCalledOnce())
    await user.click(screen.getByRole('button', {name: '设置'}))
    await user.click(screen.getByRole('tab', {name: '终端 Shell'}))
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
		statusManagementMode: 'title-change',
		statusManagementHTTPPort: 0,
		httpServiceEnabled: false,
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

	it('在设置按钮旁管理分类，并以选择方式为多字段信息保存分类', async () => {
		const user = userEvent.setup()
		bindings.SaveExtraInfoCatalogue.mockResolvedValue('git')
		bindings.SaveExtraInfoTemplate.mockResolvedValue({
			id: 'template-1', catalogue: 'git', displayName: 'API 仓库', fields: [
				{key: 'repository', displayName: '仓库', value: 'git@example.com:team/api.git'},
				{key: 'remote', displayName: '远程名称', value: 'origin'},
			],
			parameters: [{key: 'branch', displayName: '分支', required: true}],
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		expect(screen.getByRole('dialog', {name: '额外信息管理'})).toBeInTheDocument()
		await user.type(screen.getByRole('textbox', {name: '分类名称'}), 'git')
		await user.click(screen.getByRole('button', {name: '新增分类'}))
		await waitFor(() => expect(bindings.SaveExtraInfoCatalogue).toHaveBeenCalledWith('git'))
		await waitFor(() => expect(screen.getByRole('button', {name: '新增信息'})).toBeEnabled())
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		expect(screen.getByRole('combobox')).toHaveTextContent('git')
		await user.type(screen.getByRole('textbox', {name: '信息展示名称'}), 'API 仓库')
		await user.type(screen.getByRole('textbox', {name: '固定键 1'}), 'repository')
		await user.type(screen.getByRole('textbox', {name: '固定字段显示名称 1'}), '仓库')
		await user.type(screen.getByRole('textbox', {name: '固定值 1'}), 'git@example.com:team/api.git')
		await user.click(screen.getByRole('button', {name: '新增固定字段'}))
		await user.type(screen.getByRole('textbox', {name: '固定键 2'}), 'remote')
		await user.type(screen.getByRole('textbox', {name: '固定字段显示名称 2'}), '远程名称')
		await user.type(screen.getByRole('textbox', {name: '固定值 2'}), 'origin')
		await user.click(screen.getByRole('button', {name: '新增参数'}))
		await user.type(screen.getByRole('textbox', {name: '参数键 1'}), 'branch')
		await user.type(screen.getByRole('textbox', {name: '参数显示名称 1'}), '分支')
		await user.click(screen.getByRole('checkbox', {name: '参数 1 必填'}))
		await user.click(screen.getByRole('button', {name: '保存信息'}))

		await waitFor(() => expect(bindings.SaveExtraInfoTemplate).toHaveBeenCalledWith({
			id: '', catalogue: 'git', displayName: 'API 仓库', fields: [
				{key: 'repository', displayName: '仓库', value: 'git@example.com:team/api.git'},
				{key: 'remote', displayName: '远程名称', value: 'origin'},
			],
			parameters: [{key: 'branch', displayName: '分支', required: true}],
		}))
	})

	it('创建任务时切换分类选择多个信息并保留已选项', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoCatalogues.mockResolvedValue(['git', 'issue'])
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'template-1', catalogue: 'git', displayName: 'API 仓库', fields: [{key: 'repository', displayName: '仓库', value: 'git@example.com:team/api.git'}],
			parameters: [{key: 'branch', displayName: '分支', required: true}],
		}, {
			id: 'template-2', catalogue: 'issue', displayName: '缺陷单', fields: [{key: 'project', displayName: '项目', value: 'TASK'}], parameters: [],
		}])
		bindings.CreateTaskWithExtraInfo.mockResolvedValue({
			id: 'task-2', title: '关联 API', description: '', status: 'pending', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z',
			extraInfo: [{
				id: 'template-1', catalogue: 'git', displayName: 'API 仓库', fields: [{key: 'repository', displayName: '仓库', value: 'git@example.com:team/api.git'}],
				parameters: [{key: 'branch', displayName: '分支', required: true, value: 'main'}],
			}, {
				id: 'template-2', catalogue: 'issue', displayName: '缺陷单', fields: [{key: 'project', displayName: '项目', value: 'TASK'}], parameters: [],
			}],
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		await user.type(screen.getByRole('textbox', {name: '标题'}), '关联 API')
		expect(screen.getByText('已选择 0 项')).toBeInTheDocument()
		await user.click(screen.getByRole('checkbox', {name: 'API 仓库'}))
		await user.type(screen.getByRole('textbox', {name: '分支'}), 'main')
		await user.click(screen.getByLabelText('选择分类'))
		await user.click(screen.getByRole('option', {name: 'issue'}))
		await user.click(screen.getByRole('checkbox', {name: '缺陷单'}))
		expect(screen.getByText('git · API 仓库')).toBeInTheDocument()
		expect(screen.getByText('issue · 缺陷单')).toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '创建'}))

		await waitFor(() => expect(bindings.CreateTaskWithExtraInfo).toHaveBeenCalledWith('关联 API', '', '#4f46e5', [{
			id: 'template-1', catalogue: 'git', displayName: 'API 仓库', fields: [{key: 'repository', displayName: '仓库', value: 'git@example.com:team/api.git'}],
			parameters: [{key: 'branch', displayName: '分支', required: true, value: 'main'}],
		}, {
			id: 'template-2', catalogue: 'issue', displayName: '缺陷单', fields: [{key: 'project', displayName: '项目', value: 'TASK'}], parameters: [],
		}]))
	})

	it('编辑任务时保留已删除模板快照并允许修改参数', async () => {
		const user = userEvent.setup()
		bindings.ListTasks.mockResolvedValue([{
			id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
			extraInfo: [{
				id: 'removed-template', catalogue: 'git', displayName: '旧 API 仓库', fields: [{key: 'repository', displayName: '仓库', value: 'git@example.com:team/old-api.git'}],
				parameters: [{key: 'branch', displayName: '分支', required: true, value: 'main'}],
			}],
		}])
		bindings.UpdateTaskWithExtraInfo.mockResolvedValue({
			id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
			extraInfo: [{
				id: 'removed-template', catalogue: 'git', displayName: '旧 API 仓库', fields: [{key: 'repository', displayName: '仓库', value: 'git@example.com:team/old-api.git'}],
				parameters: [{key: 'branch', displayName: '分支', required: true, value: 'release/1.0'}],
			}],
		})
		render(<App/>)

		await user.click(await screen.findByRole('tab', {name: /执行中/}))
		await user.click(screen.getByRole('button', {name: '任务操作'}))
		await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
		expect(screen.getByText('旧 API 仓库（已删除）')).toBeInTheDocument()
		const branch = screen.getByRole('textbox', {name: '分支'})
		await user.clear(branch)
		await user.type(branch, 'release/1.0')
		await user.click(screen.getByRole('button', {name: '保存'}))

		await waitFor(() => expect(bindings.UpdateTaskWithExtraInfo).toHaveBeenCalledWith('task-1', '清理临时文件', '', '#4f46e5', expect.arrayContaining([
			expect.objectContaining({id: 'removed-template', parameters: [expect.objectContaining({key: 'branch', value: 'release/1.0'})]}),
		])))
	})

  it('将新建任务按钮放在任务与终端标题栏中', async () => {
    render(<App/>)

    const taskTreeHeader = (await screen.findByText('任务与终端')).parentElement
    if (!taskTreeHeader) {
      throw new Error('未找到任务树标题栏')
    }

    expect(within(taskTreeHeader).getByRole('button', {name: '新建任务'})).toBeInTheDocument()
  })

  it('将一键展开和收起按钮放在新建任务按钮旁边', async () => {
    const user = userEvent.setup()
    render(<App/>)

    const taskTreeHeader = (await screen.findByText('任务与终端')).parentElement
    if (!taskTreeHeader) {
      throw new Error('未找到任务树标题栏')
    }

    expect(within(taskTreeHeader).getByRole('button', {name: '收起全部任务'})).toBeInTheDocument()
    expect(within(screen.getByRole('tablist', {name: '任务状态筛选'})).queryByRole('button', {name: /全部任务/})).not.toBeInTheDocument()

    await user.click(within(taskTreeHeader).getByRole('button', {name: '收起全部任务'}))
    expect(within(taskTreeHeader).getByRole('button', {name: '展开全部任务'})).toBeInTheDocument()
  })

  it('暗色模式默认使用低对比但可见的面板分割条', async () => {
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'dark', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
    })
    render(<App/>)

    const divider = await screen.findByRole('separator', {name: '调整任务树宽度'})
    expect(getComputedStyle(divider).backgroundColor).toBe('rgb(30, 41, 59)')
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

  it('主动关闭终端成功后从树和右侧终端视图移除', async () => {
    const user = userEvent.setup()
    bindings.CreateTerminal.mockResolvedValue({id: 'terminal-1', taskId: 'task-1', state: 'active', title: '关闭目标'})
    bindings.CloseTerminal.mockResolvedValue(undefined)
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '新增终端'}))
    await screen.findByText('终端视图')

    await user.click(screen.getByRole('button', {name: '关闭终端'}))

    await waitFor(() => expect(bindings.CloseTerminal).toHaveBeenCalledWith('task-1', 'terminal-1'))
    await waitFor(() => expect(screen.queryAllByText('关闭目标')).toHaveLength(0))
    expect(screen.queryByText('终端视图')).not.toBeInTheDocument()
  })

  it('终端输出 OSC 标题后实时更新树节点和右侧终端栏', async () => {
    const user = userEvent.setup()
    let terminalEventListener: ((event: {taskId: string; terminalId: string; type: 'output'; data: string}) => void) | undefined
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
    expect(screen.getAllByText('终端')).toHaveLength(2)

    if (!terminalEventListener) {
      throw new Error('未注册终端事件监听器')
    }
    terminalEventListener({taskId: 'task-1', terminalId: 'terminal-1', type: 'output', data: '\x1b]2;正在构建\x07'})

    await waitFor(() => expect(screen.getAllByText('正在构建')).toHaveLength(2))
    expect(bindings.ReportTerminalTitleActivity).toHaveBeenCalledWith('task-1', 'terminal-1')
    terminalEventListener({taskId: 'task-1', terminalId: 'terminal-1', type: 'output', data: '\x1b]2;正在构建\x07'})
    await waitFor(() => expect(bindings.ReportTerminalTitleActivity).toHaveBeenCalledTimes(1))
    expect(screen.queryByText('终端 1')).not.toBeInTheDocument()
  })

  it('接收实时状态事件并在选择终端时通知后端清除未读状态', async () => {
    const user = userEvent.setup()
    let realtimeStatusListener: ((event: {version: number, taskId: string, taskStatus: 'unread' | 'working', terminalId: string, terminalStatus: 'unread' | 'working'}) => void) | undefined
    runtime.EventsOn.mockImplementation((eventName, listener) => {
      if (eventName === 'task-realtime-status:event') {
        realtimeStatusListener = listener
      }
    })
    bindings.CreateTerminal.mockResolvedValue({id: 'terminal-1', taskId: 'task-1', state: 'active'})
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '新增终端'}))
    await screen.findByText('终端视图')
    if (!realtimeStatusListener) {
      throw new Error('未注册实时状态监听器')
    }

    realtimeStatusListener({version: 2, taskId: 'task-1', taskStatus: 'unread', terminalId: 'terminal-1', terminalStatus: 'unread'})

    await screen.findByRole('status', {name: '终端状态：未读'})
    realtimeStatusListener({version: 1, taskId: 'task-1', taskStatus: 'working', terminalId: 'terminal-1', terminalStatus: 'working'})
    expect(screen.getByRole('status', {name: '终端状态：未读'})).toBeInTheDocument()
    await user.click(screen.getByText('清理临时文件'))
    await waitFor(() => expect(bindings.ClearSelectedTerminal).toHaveBeenCalled())
    const terminalItem = screen.getByText('终端').closest('[role="button"]')
    if (!terminalItem) {
      throw new Error('未找到终端条目')
    }
    await user.click(terminalItem)
    await waitFor(() => expect(bindings.SelectTerminal).toHaveBeenLastCalledWith('task-1', 'terminal-1'))
  })

  it('在设置中配置 HTTP 状态管理并查看接口说明', async () => {
    const user = userEvent.setup()
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '设置'}))
    await user.click(screen.getByRole('tab', {name: '实时状态'}))
    await user.click(screen.getByLabelText('状态管理方式'))
    await user.click(screen.getByRole('option', {name: '通过 HTTP 接口'}))
    expect(screen.getByRole('switch', {name: '通过 HTTP 状态管理自动启用本机 HTTP 服务'})).toBeChecked()
    expect(screen.getByRole('switch', {name: '通过 HTTP 状态管理自动启用本机 HTTP 服务'})).toBeDisabled()
    const port = screen.getByRole('spinbutton', {name: 'HTTP 端口'})
    await user.clear(port)
    await user.type(port, '38561')
    await user.click(screen.getByRole('button', {name: '查看 HTTP 接口使用说明'}))

    expect(await screen.findByRole('dialog', {name: 'HTTP 状态接口使用说明'})).toHaveTextContent('TASKAI_STATUS_API')
    expect(screen.getByText('服务与设置')).toBeInTheDocument()
    expect(screen.getByText('查询接口')).toBeInTheDocument()
    expect(screen.getByText('状态更新')).toBeInTheDocument()
    expect(screen.getByText('GET /api/v1/tasks?status=pending|running|completed')).toBeInTheDocument()
    expect(screen.getByText('GET /api/v1/tasks/:taskId')).toBeInTheDocument()
    expect(screen.getByText('PUT /api/v1/tasks/:taskId/status')).toBeInTheDocument()
    expect(screen.getByText('PUT /api/v1/tasks/:taskId/terminals/:terminalId/status')).toBeInTheDocument()
    expect(screen.getByText(/任务列表查询参数：可省略.*pending.*running.*completed/)).toBeInTheDocument()
    expect(screen.getByText(/状态更新请求体的 status：必填.*idle.*working.*unread.*error/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', {name: '关闭'}))
    await waitFor(() => expect(screen.queryByRole('dialog', {name: 'HTTP 状态接口使用说明'})).not.toBeInTheDocument())
    await user.click(screen.getByRole('button', {name: '保存'}))

    await waitFor(() => expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining({
      statusManagementMode: 'http', statusManagementHTTPPort: 38561,
    })))
  })

  it('在标题变化模式可独立启用本机 HTTP 服务', async () => {
    const user = userEvent.setup()
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '设置'}))
    await user.click(screen.getByRole('tab', {name: '实时状态'}))
    const httpService = screen.getByRole('switch', {name: '启用本机 HTTP 服务'})
    expect(httpService).not.toBeChecked()
    await user.click(httpService)
    const port = screen.getByRole('spinbutton', {name: 'HTTP 端口'})
    await user.clear(port)
    await user.type(port, '38562')
    await user.click(screen.getByRole('button', {name: '查看 HTTP 接口使用说明'}))
    expect(await screen.findByRole('dialog', {name: 'HTTP 状态接口使用说明'})).toBeInTheDocument()
    await user.click(screen.getByRole('button', {name: '关闭'}))
    await waitFor(() => expect(screen.queryByRole('dialog', {name: 'HTTP 状态接口使用说明'})).not.toBeInTheDocument())
    await user.click(screen.getByRole('button', {name: '保存'}))

    await waitFor(() => expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining({
      statusManagementMode: 'title-change', statusManagementHTTPPort: 38562, httpServiceEnabled: true,
    })))
  })

  it('保存无效 HTTP 状态配置时显示后端错误', async () => {
    const user = userEvent.setup()
    bindings.SaveSettings.mockRejectedValue(new Error('HTTP 状态管理需要配置端口'))
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '设置'}))
    await user.click(screen.getByRole('tab', {name: '实时状态'}))
    await user.click(screen.getByLabelText('状态管理方式'))
    await user.click(screen.getByRole('option', {name: '通过 HTTP 接口'}))
    await user.click(screen.getByRole('button', {name: '保存'}))

    expect(await screen.findByText('HTTP 状态管理需要配置端口')).toBeInTheDocument()
  })

  it('终端输出空白 OSC 标题后在树节点和右侧终端栏回退为终端', async () => {
    const user = userEvent.setup()
    let terminalEventListener: ((event: {taskId: string; terminalId: string; type: 'output'; data: string}) => void) | undefined
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
    terminalEventListener({taskId: 'task-1', terminalId: 'terminal-1', type: 'output', data: '\x1b]2;正在构建\x07'})
    await waitFor(() => expect(screen.getByLabelText('右侧终端标题').textContent).toBe('正在构建'))
    terminalEventListener({taskId: 'task-1', terminalId: 'terminal-1', type: 'output', data: '\x1b]2;   \x07'})

    await waitFor(() => expect(screen.getByLabelText('右侧终端标题').textContent).toBe('终端'))
  })

  it('右键新增终端时合并创建返回前到达的标题与原始输出', async () => {
    const user = userEvent.setup()
    const output = '\x1b]2;启动标题\x07'
    let resolveTerminal: (terminal: {id: string, taskId: string, state: 'active'}) => void
    let terminalEventListener: ((event: {taskId: string, terminalId: string, type: 'output', data: string}) => void) | undefined
    runtime.EventsOn.mockImplementation((eventName, listener) => {
      if (eventName === 'task-terminal:event') {
        terminalEventListener = listener
      }
    })
    bindings.CreateTerminal.mockImplementation(() => new Promise((resolve) => {
      resolveTerminal = resolve
    }))
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '新增终端'}))
    await waitFor(() => expect(bindings.CreateTerminal).toHaveBeenCalledOnce())
    if (!terminalEventListener) {
      throw new Error('未注册终端事件监听器')
    }
    terminalEventListener({taskId: 'task-1', terminalId: 'terminal-1', type: 'output', data: output})

    resolveTerminal!({id: 'terminal-1', taskId: 'task-1', state: 'active'})

    await waitFor(() => expect(screen.getAllByText('启动标题')).toHaveLength(2))
    expect(screen.getByLabelText('右侧终端原始输出').textContent).toBe(output)
  })

  it('任务结束后忽略右键新增终端的晚到创建结果', async () => {
    const user = userEvent.setup()
    let resolveTerminal: (terminal: {id: string, taskId: string, state: 'active'}) => void
    bindings.CreateTerminal.mockImplementation(() => new Promise((resolve) => {
      resolveTerminal = resolve
    }))
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '新增终端'}))
    await waitFor(() => expect(bindings.CreateTerminal).toHaveBeenCalledOnce())
    await user.click(screen.getByRole('button', {name: '结束'}))
    await user.click(screen.getByRole('button', {name: '结束并删除'}))
    await waitFor(() => expect(bindings.FinishTask).toHaveBeenCalledWith('task-1'))

    await act(async () => {
      resolveTerminal!({id: 'terminal-1', taskId: 'task-1', state: 'active'})
      await Promise.resolve()
    })

    expect(screen.queryByText('终端视图')).not.toBeInTheDocument()
  })

  it('任务结束后忽略显示终端命令的晚到创建结果', async () => {
    const user = userEvent.setup()
    let resolveCommand: (result: {terminal: {id: string, taskId: string, state: 'active'}}) => void
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh',
      taskMenuItems: [{id: 'custom-codex', kind: 'command', name: 'Codex', command: 'codex', showTerminal: true}],
    })
    bindings.ExecuteTaskMenuCommand.mockImplementation(() => new Promise((resolve) => {
      resolveCommand = resolve
    }))
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: 'Codex'}))
    await waitFor(() => expect(bindings.ExecuteTaskMenuCommand).toHaveBeenCalledOnce())
    await user.click(screen.getByRole('button', {name: '结束'}))
    await user.click(screen.getByRole('button', {name: '结束并删除'}))
    await waitFor(() => expect(bindings.FinishTask).toHaveBeenCalledWith('task-1'))

    await act(async () => {
      resolveCommand!({terminal: {id: 'terminal-codex', taskId: 'task-1', state: 'active'}})
      await Promise.resolve()
    })

    expect(screen.queryByText('终端视图')).not.toBeInTheDocument()
  })

  it('显示终端的自定义命令时合并创建返回前到达的标题与原始输出', async () => {
    const user = userEvent.setup()
    const output = '\x1b]2;命令标题\x07'
    let resolveCommand: (result: {terminal: {id: string, taskId: string, state: 'active'}}) => void
    let terminalEventListener: ((event: {taskId: string, terminalId: string, type: 'output', data: string}) => void) | undefined
    runtime.EventsOn.mockImplementation((eventName, listener) => {
      if (eventName === 'task-terminal:event') {
        terminalEventListener = listener
      }
    })
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh',
      taskMenuItems: [{id: 'custom-codex', kind: 'command', name: 'Codex', command: 'codex', showTerminal: true}],
    })
    bindings.ExecuteTaskMenuCommand.mockImplementation(() => new Promise((resolve) => {
      resolveCommand = resolve
    }))
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: 'Codex'}))
    await waitFor(() => expect(bindings.ExecuteTaskMenuCommand).toHaveBeenCalledOnce())
    if (!terminalEventListener) {
      throw new Error('未注册终端事件监听器')
    }
    terminalEventListener({taskId: 'task-1', terminalId: 'terminal-codex', type: 'output', data: output})

    resolveCommand!({terminal: {id: 'terminal-codex', taskId: 'task-1', state: 'active'}})

    await waitFor(() => expect(screen.getAllByText('命令标题')).toHaveLength(2))
    expect(screen.getByLabelText('右侧终端原始输出').textContent).toBe(output)
  })

  it('自定义菜单通过保存的菜单项 ID 统一执行命令', async () => {
    const user = userEvent.setup()
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh',
      taskMenuItems: [
        {id: 'custom-codex', kind: 'command', name: 'Codex', command: 'codex', arguments: ['--full-auto'], showTerminal: true},
        {id: 'custom-vscode', kind: 'command', name: '打开 VS Code', command: 'code', arguments: ['.'], showTerminal: false},
      ],
    })
    bindings.ExecuteTaskMenuCommand
      .mockResolvedValueOnce({terminal: {id: 'terminal-codex', taskId: 'task-1', state: 'active'}})
      .mockResolvedValueOnce({})
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: 'Codex'}))
    expect(bindings.ExecuteTaskMenuCommand).toHaveBeenCalledWith('task-1', 'custom-codex', 100, 32)
    await screen.findByText('终端视图')

    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '打开 VS Code'}))
    expect(bindings.ExecuteTaskMenuCommand).toHaveBeenCalledWith('task-1', 'custom-vscode', 100, 32)
  })

  it('收到后置脚本错误事件时显示错误提示', async () => {
	const user = userEvent.setup()
	let scriptErrorListener: ((message: string) => void) | undefined
	runtime.EventsOn.mockImplementation((eventName, listener) => {
	  if (eventName === 'task-script:error') {
		scriptErrorListener = listener
	  }
	})
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
	if (!scriptErrorListener) {
	  throw new Error('未注册后置脚本错误监听器')
	}
	scriptErrorListener('执行后置脚本: 清理失败')

	expect(await screen.findByText('执行后置脚本: 清理失败')).toBeInTheDocument()
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
    await user.click(screen.getByRole('tab', {name: '任务操作'}))
    await user.click(screen.getByRole('button', {name: '新增菜单项'}))

    const createDialog = screen.getByRole('dialog', {name: '新增菜单项'})
    await user.clear(within(createDialog).getByRole('textbox', {name: '菜单名称'}))
    await user.type(within(createDialog).getByRole('textbox', {name: '菜单名称'}), 'Codex')
    await user.type(within(createDialog).getByRole('textbox', {name: '启动命令'}), 'codex')
    await user.type(within(createDialog).getByRole('textbox', {name: '启动参数（每行一个）'}), '--full-auto\n--dangerously-bypass-approvals-and-sandbox')
	await user.click(within(createDialog).getByRole('tab', {name: '前后置脚本'}))
	expect(within(createDialog).getByRole('textbox', {name: '前置脚本（命令或路径）'})).toBeInTheDocument()
	await user.type(within(createDialog).getByRole('textbox', {name: '前置脚本（命令或路径）'}), 'before-script')
	await user.type(within(createDialog).getByRole('textbox', {name: '前置脚本参数（每行一个）'}), '--before\n\n--with-space')
	await user.type(within(createDialog).getByRole('textbox', {name: '后置脚本（命令或路径）'}), 'after-script')
	await user.type(within(createDialog).getByRole('textbox', {name: '后置脚本参数（每行一个）'}), '--after')
	await user.click(within(createDialog).getByRole('button', {name: '前后置脚本使用说明'}))
    expect(await screen.findByText('taskId')).toBeInTheDocument()
    expect(screen.getByText('directory')).toBeInTheDocument()
    expect(screen.getByText('command')).toBeInTheDocument()
    expect(screen.getByText('arguments')).toBeInTheDocument()
    expect(screen.getByText(/标准输入/)).toBeInTheDocument()
	expect(screen.getByText('脚本填写路径或 Shell PATH 中的可执行脚本；参数每行传递为一个独立参数，空白行会忽略。')).toBeInTheDocument()
    expect(screen.getByText(/占位符/)).toBeInTheDocument()
    await user.keyboard('{Escape}')
    await user.click(within(createDialog).getByRole('tab', {name: '基本配置'}))
    expect(within(createDialog).getByRole('textbox', {name: '菜单名称'})).toHaveValue('Codex')
	await user.click(within(createDialog).getByRole('tab', {name: '前后置脚本'}))
	expect(within(createDialog).getByRole('textbox', {name: '前置脚本（命令或路径）'})).toHaveValue('before-script')
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
	  beforeScript: {script: 'before-script', arguments: ['--before', '--with-space']},
	  afterScript: {script: 'after-script', arguments: ['--after']},
    })
  })
})
