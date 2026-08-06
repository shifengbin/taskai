import {act, cleanup, fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {readFileSync} from 'node:fs'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

const bindings = vi.hoisted(() => ({
  ClearSelectedTerminal: vi.fn(),
  CreateTask: vi.fn(),
	CreateTaskWithExtraInfo: vi.fn(),
	CreateTaskWithExtraInfoAndLifecycleChains: vi.fn(),
	CreateTaskWithExtraInfoAndTemplateFields: vi.fn(),
	CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains: vi.fn(),
	UpdateTask: vi.fn(),
	UpdateTaskWithExtraInfo: vi.fn(),
	UpdateTaskWithExtraInfoAndLifecycleChains: vi.fn(),
	UpdateTaskWithExtraInfoAndTemplateFields: vi.fn(),
	UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains: vi.fn(),
  ListTasks: vi.fn(),
	DeleteCompletedTasks: vi.fn(),
	ListExtraInfoCatalogues: vi.fn(),
	ListExtraInfoTemplates: vi.fn(),
	ListExtraInfos: vi.fn(),
	SaveExtraInfoCatalogue: vi.fn(),
	SaveExtraInfoTemplate: vi.fn(),
	SaveExtraInfo: vi.fn(),
	DeleteExtraInfoCatalogue: vi.fn(),
	DeleteExtraInfoTemplate: vi.fn(),
	DeleteExtraInfo: vi.fn(),
	ListLifecycleCommands: vi.fn(),
	SaveLifecycleCommand: vi.fn(),
	DeleteLifecycleCommand: vi.fn(),
	ListLifecycleCommandChains: vi.fn(),
	SaveLifecycleCommandChain: vi.fn(),
	CopyLifecycleCommandChain: vi.fn(),
	CopyLifecyclePreset: vi.fn(),
	GetLifecycleCommandInput: vi.fn(),
	DeleteLifecycleCommandChain: vi.fn(),
	DeleteLifecyclePreset: vi.fn(),
	ListLifecyclePresets: vi.fn(),
	SaveLifecyclePreset: vi.fn(),
	SaveDefaultLifecyclePreset: vi.fn(),
  ReorderTasks: vi.fn(),
	SetTaskShelved: vi.fn(),
	ReportTerminalTitleActivity: vi.fn(),
  StartTask: vi.fn(),
	RetryTaskLifecycleCommandChain: vi.fn(),
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
const runtime = vi.hoisted(() => ({ClipboardSetText: vi.fn(), EventsOn: vi.fn(), EventsOff: vi.fn(), Quit: vi.fn()}))
const terminalSessionRegistry = vi.hoisted(() => ({
  handleTerminalEvent: vi.fn(),
  disposeAll: vi.fn(),
  disposeTask: vi.fn(),
  dispose: vi.fn(),
}))

vi.mock('../wailsjs/go/main/App', () => bindings)
vi.mock('../wailsjs/runtime/runtime', () => runtime)
vi.mock('./terminal-session', () => ({
  TerminalSessionRegistry: class {
    handleTerminalEvent = terminalSessionRegistry.handleTerminalEvent
    disposeAll = terminalSessionRegistry.disposeAll
    disposeTask = terminalSessionRegistry.disposeTask
    dispose = terminalSessionRegistry.dispose

    constructor(_onWrite: unknown) {}
  },
}))
vi.mock('./components/TerminalView', async () => {
  const {terminalDisplayName} = await vi.importActual<typeof import('./types')>('./types')
  return {
    TerminalView: ({terminal}: {terminal: {title?: string}}) => <>
      <div>终端视图</div>
      <div data-testid="terminal-view-title-container">
        <div data-testid="terminal-view-title" aria-label="右侧终端标题">{terminalDisplayName(terminal)}</div>
      </div>
    </>,
  }
})

import App from './App'
import type {TaskRecord} from './types'

const appStyles = readFileSync('src/App.css', 'utf8')

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
    Object.values(terminalSessionRegistry).forEach((mock) => mock.mockReset())
    bindings.SaveSettings.mockImplementation(async (next) => next)
		bindings.ClearSelectedTerminal.mockResolvedValue(undefined)
		bindings.SelectTerminal.mockResolvedValue(undefined)
		bindings.ReportTerminalTitleActivity.mockResolvedValue(true)
		runtime.ClipboardSetText.mockResolvedValue(true)
		bindings.ListExtraInfoCatalogues.mockResolvedValue([])
		bindings.ListExtraInfoTemplates.mockResolvedValue([])
		bindings.ListExtraInfos.mockResolvedValue([])
		bindings.ListLifecycleCommands.mockResolvedValue([])
		bindings.ListLifecycleCommandChains.mockResolvedValue([])
		bindings.ListLifecyclePresets.mockResolvedValue([])
		bindings.SaveLifecyclePreset.mockImplementation(async (preset) => preset)
		bindings.CopyLifecyclePreset.mockResolvedValue(undefined)
		bindings.DeleteLifecyclePreset.mockResolvedValue(undefined)
		bindings.SaveDefaultLifecyclePreset.mockResolvedValue({defaultLifecyclePresetId: ''})
		bindings.DeleteCompletedTasks.mockResolvedValue([])
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
    expect(screen.getByText('确认后将结束“清理临时文件”并关闭其全部终端。结束后命令链会按该任务的配置执行。此操作无法撤销。')).toBeInTheDocument()
    expect(screen.queryByText(/删除其工作目录及所有内容/)).not.toBeInTheDocument()
    expect(bindings.FinishTask).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', {name: '取消'}))
    expect(bindings.FinishTask).not.toHaveBeenCalled()
    await waitFor(() => expect(screen.queryByText('结束任务？')).not.toBeInTheDocument())

    await user.click(screen.getByRole('button', {name: '结束'}))
    await user.click(screen.getByRole('button', {name: '结束任务'}))
    expect(bindings.FinishTask).toHaveBeenCalledWith('task-1')
    expect(terminalSessionRegistry.disposeTask).toHaveBeenCalledWith('task-1')
  })

	it('应用卸载时释放全部终端会话', async () => {
		const {unmount} = render(<App/>)

		await screen.findByRole('tab', {name: /执行中/})
		unmount()

		expect(terminalSessionRegistry.disposeAll).toHaveBeenCalledOnce()
	})

	it('没有活动终端时阻止文件拖放的默认导航', () => {
		render(<App/>)

		const event = new Event('drop', {bubbles: true, cancelable: true})
		Object.defineProperty(event, 'dataTransfer', {value: {types: ['Files']}})
		document.dispatchEvent(event)

		expect(event.defaultPrevented).toBe(true)
	})

	it('已完成任务仅选择可删除记录，并在确认后批量删除', async () => {
		const user = userEvent.setup()
		const lockedTask: TaskRecord = {
			id: 'completed-locked', title: '正在清理的任务', description: '', status: 'completed', createdAt: '2026-07-22T00:00:00Z',
			lifecycleExecution: {hook: 'postEnd', chainId: 'cleanup', currentIndex: 1, commandCount: 1, state: 'failed'},
		}
		bindings.ListTasks.mockResolvedValue([
			{id: 'completed-1', title: '已交付任务', description: '', status: 'completed', createdAt: '2026-07-22T00:00:00Z'},
			{id: 'completed-2', title: '已归档任务', description: '', status: 'completed', createdAt: '2026-07-21T00:00:00Z'},
			lockedTask,
		])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems, activeTaskStatus: 'completed',
		})
		bindings.DeleteCompletedTasks.mockResolvedValue([lockedTask])
		render(<App/>)

		await screen.findByText('已交付任务')
		await user.click(screen.getByText('已交付任务'))
		expect(screen.getByRole('heading', {name: '已交付任务'})).toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '选择已完成任务'}))
		expect(screen.getByRole('checkbox', {name: '选择任务 正在清理的任务'})).toBeDisabled()
		await user.click(screen.getByRole('button', {name: '全选已完成任务'}))
		expect(screen.getByRole('checkbox', {name: '选择任务 已交付任务'})).toBeChecked()
		expect(screen.getByRole('checkbox', {name: '选择任务 已归档任务'})).toBeChecked()

		await user.click(screen.getByRole('button', {name: '删除已选任务记录'}))
		expect(screen.getByText('删除 2 个任务记录？')).toBeInTheDocument()
		expect(screen.getByText('此操作只会移除任务记录，不会删除工作目录或运行生命周期命令。此操作无法撤销。')).toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '取消'}))
		expect(bindings.DeleteCompletedTasks).not.toHaveBeenCalled()
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())

		await user.click(screen.getByRole('button', {name: '删除已选任务记录'}))
		await user.click(screen.getByRole('button', {name: '删除记录'}))
		await waitFor(() => expect(bindings.DeleteCompletedTasks).toHaveBeenCalledWith(['completed-1', 'completed-2']))
		await waitFor(() => expect(screen.queryByText('已交付任务')).not.toBeInTheDocument())
		expect(screen.getByText('正在清理的任务')).toBeInTheDocument()
		expect(screen.queryByText('已选 2 项')).not.toBeInTheDocument()
		expect(screen.getByText('从左侧选择任务，或创建一个新任务开始。')).toBeInTheDocument()
	})

	it('批量删除失败时保留已选任务并展示错误', async () => {
		const user = userEvent.setup()
		bindings.ListTasks.mockResolvedValue([
			{id: 'completed-1', title: '保留选择任务', description: '', status: 'completed', createdAt: '2026-07-22T00:00:00Z'},
		])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems, activeTaskStatus: 'completed',
		})
		bindings.DeleteCompletedTasks.mockRejectedValue(new Error('任务记录无法删除'))
		render(<App/>)

		await screen.findByText('保留选择任务')
		await user.click(screen.getByRole('button', {name: '选择已完成任务'}))
		await user.click(screen.getByRole('checkbox', {name: '选择任务 保留选择任务'}))
		await user.click(screen.getByRole('button', {name: '删除已选任务记录'}))
		await user.click(screen.getByRole('button', {name: '删除记录'}))

		expect(await screen.findByText('任务记录无法删除')).toBeInTheDocument()
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
		expect(screen.getByRole('checkbox', {name: '选择任务 保留选择任务'})).toBeChecked()
		expect(screen.getByText('已选 1 项')).toBeInTheDocument()
	})

	it('离开已完成标签时退出选择模式并清空已选任务', async () => {
		const user = userEvent.setup()
		bindings.ListTasks.mockResolvedValue([
			{id: 'pending-1', title: '待执行任务', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z'},
			{id: 'completed-1', title: '已完成任务', description: '', status: 'completed', createdAt: '2026-07-21T00:00:00Z'},
		])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems, activeTaskStatus: 'completed',
		})
		render(<App/>)

		await screen.findByText('已完成任务')
		await user.click(screen.getByRole('button', {name: '选择已完成任务'}))
		await user.click(screen.getByRole('checkbox', {name: '选择任务 已完成任务'}))
		expect(screen.getByText('已选 1 项')).toBeInTheDocument()
		await user.click(screen.getByRole('tab', {name: /未执行/}))
		await screen.findByText('待执行任务')
		await user.click(screen.getByRole('tab', {name: /已完成/}))

		await screen.findByText('已完成任务')
		expect(screen.getByRole('button', {name: '选择已完成任务'})).toBeInTheDocument()
		expect(screen.queryByRole('checkbox', {name: '选择任务 已完成任务'})).not.toBeInTheDocument()
	})

  it('任务详情展示额外信息和系统变量，并复制当前命令链输入', async () => {
    const user = userEvent.setup()
    bindings.ListTasks.mockResolvedValue([{
      id: 'task-1', title: '发布 API', description: '准备部署', status: 'running', createdAt: '2026-07-22T00:00:00Z',
      extraInfo: [{
        id: 'info-1', catalogue: 'git', displayName: 'API 服务',
        fields: [{key: 'name', displayName: '项目名称', value: 'API 服务'}, {key: 'repository', displayName: '仓库地址', value: 'git@example.com:team/api.git'}],
        parameters: [{key: 'branch', displayName: '仓库分支', required: false, value: 'main'}, {key: 'deploy', displayName: '允许部署', inputType: 'checkbox', required: false, value: 'true'}],
      }],
      templateFields: {environment: 'production'},
    }])
    bindings.GetLifecycleCommandInput.mockResolvedValue('{"id":"task-1"}')
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByText('发布 API'))

    expect(screen.getByText('额外信息')).toBeInTheDocument()
    expect(screen.getAllByText('API 服务')).toHaveLength(2)
    expect(screen.getByText('仓库地址')).toBeInTheDocument()
    expect(screen.getByText('git@example.com:team/api.git')).toBeInTheDocument()
    expect(screen.getByText('允许部署')).toBeInTheDocument()
    expect(screen.getByText('是')).toBeInTheDocument()
    expect(screen.getByText('TASKAI_TASK_ID')).toBeInTheDocument()
    expect(screen.getByText('TASKAI_TERMINAL_ID')).toBeInTheDocument()
    expect(screen.getByText('TASKAI_STATUS_API')).toBeInTheDocument()
		expect(screen.getByText('本机 HTTP 服务正在监听时，注入到之后新建的终端')).toBeInTheDocument()
    expect(screen.queryByText('TASKAI_ENVIRONMENT')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', {name: '复制当前命令链输入 JSON'}))
    expect(bindings.GetLifecycleCommandInput).toHaveBeenCalledWith('task-1')
    expect(runtime.ClipboardSetText).toHaveBeenCalledWith('{"id":"task-1"}')
    expect(await screen.findByText('已复制当前命令链输入 JSON')).toBeInTheDocument()
    const successToast = screen.getByText('已复制当前命令链输入 JSON').closest('[role="alert"]') as HTMLElement
    expect(successToast).toHaveAttribute('data-severity', 'success')

    await user.click(within(successToast).getByRole('button', {name: '关闭'}))
    expect(screen.queryAllByRole('alert').some((alert) => alert.textContent?.trim() === '')).toBe(false)
  })

  it('复制当前命令链输入失败时不写入剪贴板', async () => {
    const user = userEvent.setup()
    bindings.GetLifecycleCommandInput.mockRejectedValue(new Error('任务不存在'))
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByText('清理临时文件'))
    await user.click(screen.getByRole('button', {name: '复制当前命令链输入 JSON'}))

    expect(await screen.findByText('任务不存在')).toBeInTheDocument()
    expect(screen.getByText('任务不存在').closest('[role="alert"]')).toHaveAttribute('data-severity', 'error')
    expect(runtime.ClipboardSetText).not.toHaveBeenCalled()
  })

  it('任务详情展示模板字段和环境变量', async () => {
    const user = userEvent.setup()
    bindings.ListTasks.mockResolvedValue([{
      id: 'task-1', title: '发布 API', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
      templateFields: {environment: 'production', deploy: true, note: '仅供参考'},
    }])
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
      activeTaskTemplateId: 'release',
      taskTemplates: [{
        id: 'release', name: '发布配置', fields: [
          {key: 'environment', displayName: '运行环境', inputType: 'string', required: true, defaultValue: 'development', injectEnvironment: true},
          {key: 'deploy', displayName: '允许部署', inputType: 'bool', required: false, defaultValue: false, injectEnvironment: true},
          {key: 'note', displayName: '内部备注', inputType: 'string', required: false, defaultValue: '', injectEnvironment: false},
        ],
      }],
    })
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByText('发布 API'))

    expect(screen.getByText('任务模板')).toBeInTheDocument()
    expect(screen.getByText('发布配置')).toBeInTheDocument()
    expect(screen.getByText('运行环境')).toBeInTheDocument()
    expect(screen.getByText('production')).toBeInTheDocument()
    expect(screen.getByText('TASKAI_ENVIRONMENT=production')).toBeInTheDocument()
    expect(screen.getByText('TASKAI_DEPLOY=true')).toBeInTheDocument()
    expect(screen.getByText('不生成环境变量')).toBeInTheDocument()
    expect(screen.getAllByText('仅自定义生命周期 Shell 命令').length).toBeGreaterThan(0)
  })

  it('任务详情在未启用模板时显示模板空状态', async () => {
    const user = userEvent.setup()
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByText('清理临时文件'))

    expect(screen.getByText('未启用任务模板')).toBeInTheDocument()
  })

  it('暗色任务详情使用高对比栏目标题并占满右侧区域', async () => {
    const user = userEvent.setup()
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'dark', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
    })
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByText('清理临时文件'))

    const sectionTitle = screen.getByText('任务模板')
    expect(sectionTitle).toHaveClass('taskai-detail-section__title')
    expect(sectionTitle.closest('.taskai-task-detail')).toHaveStyle({maxWidth: 'none', width: '100%'})
  })

  it('结束执行中的任务后保持当前任务标签', async () => {
    const user = userEvent.setup()
    let resolveFinishTask: ((task: TaskRecord) => void) | undefined
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'running',
    })
    bindings.FinishTask.mockImplementation(() => new Promise<TaskRecord>((resolve) => { resolveFinishTask = resolve }))
    render(<App/>)

    const runningTab = await screen.findByRole('tab', {name: /执行中/})
    await user.click(screen.getByRole('button', {name: '结束'}))
    await user.click(screen.getByRole('button', {name: '结束任务'}))
    await waitFor(() => expect(bindings.FinishTask).toHaveBeenCalledWith('task-1'))
    if (!resolveFinishTask) {
      throw new Error('结束任务绑定未等待返回')
    }
    const finishTaskResolver = resolveFinishTask

    await act(async () => {
      finishTaskResolver({
        id: 'task-1', title: '清理临时文件', description: '', status: 'completed', createdAt: '2026-07-22T00:00:00Z',
      })
      await Promise.resolve()
    })

    await waitFor(() => expect(screen.queryByRole('dialog', {name: '结束任务？'})).not.toBeInTheDocument())
    expect(runningTab).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', {name: /已完成/})).toHaveAttribute('aria-selected', 'false')
    expect(bindings.SaveSettings).not.toHaveBeenCalledWith(expect.objectContaining({activeTaskStatus: 'completed'}))
  })

  it('生命周期事件结束任务后保持当前任务标签', async () => {
    const user = userEvent.setup()
    let lifecycleEventListener: ((task: TaskRecord) => void) | undefined
    runtime.EventsOn.mockImplementation((eventName, listener) => {
      if (eventName === 'task-lifecycle:event') {
        lifecycleEventListener = listener
      }
    })
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'running',
    })
    render(<App/>)

    const runningTab = await screen.findByRole('tab', {name: /执行中/})
    await user.click(screen.getByText('清理临时文件'))
    if (!lifecycleEventListener) {
      throw new Error('未注册生命周期事件监听器')
    }

    act(() => lifecycleEventListener?.({
      id: 'task-1', title: '清理临时文件', description: '', status: 'completed', createdAt: '2026-07-22T00:00:00Z',
    }))

    expect(runningTab).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', {name: /已完成/})).toHaveAttribute('aria-selected', 'false')
    expect(bindings.SaveSettings).not.toHaveBeenCalledWith(expect.objectContaining({activeTaskStatus: 'completed'}))
  })

  it('开始任务后切换到执行中并突出目标任务', async () => {
    const user = userEvent.setup()
    bindings.ListTasks.mockResolvedValue([{
      id: 'task-1', title: '清理临时文件', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
    }])
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'pending',
    })
    bindings.StartTask.mockResolvedValue({
      id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
    })
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '执行'}))

    await waitFor(() => expect(screen.getByRole('tab', {name: /执行中/})).toHaveAttribute('aria-selected', 'true'))
    expect(within(screen.getByRole('navigation', {name: '任务和终端'})).getByText('清理临时文件').closest('[data-task-id]')).toHaveAttribute('data-task-start-feedback', 'flash')
    expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining({activeTaskStatus: 'running'}))
  })

  it('开始结果仍为未执行时不切换任务标签或显示反馈', async () => {
    const user = userEvent.setup()
    bindings.ListTasks.mockResolvedValue([{
      id: 'task-1', title: '清理临时文件', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
    }])
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'pending',
    })
    bindings.StartTask.mockResolvedValue({
      id: 'task-1', title: '清理临时文件', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
    })
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '执行'}))
    await waitFor(() => expect(bindings.StartTask).toHaveBeenCalledWith('task-1'))

    expect(screen.getByRole('tab', {name: /未执行/})).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', {name: /执行中/})).toHaveAttribute('aria-selected', 'false')
    expect(within(screen.getByRole('navigation', {name: '任务和终端'})).getByText('清理临时文件').closest('[data-task-id]')).not.toHaveAttribute('data-task-start-feedback')
    expect(bindings.SaveSettings).not.toHaveBeenCalledWith(expect.objectContaining({activeTaskStatus: 'running'}))
  })

  it('开始任务失败时不切换任务标签或显示反馈', async () => {
    const user = userEvent.setup()
    bindings.ListTasks.mockResolvedValue([{
      id: 'task-1', title: '清理临时文件', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
    }])
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
      activeTaskStatus: 'pending',
    })
    bindings.StartTask.mockRejectedValue(new Error('启动失败'))
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '执行'}))

    expect(await screen.findByText('启动失败')).toBeInTheDocument()
    expect(screen.getByRole('tab', {name: /未执行/})).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', {name: /执行中/})).toHaveAttribute('aria-selected', 'false')
    expect(within(screen.getByRole('navigation', {name: '任务和终端'})).getByText('清理临时文件').closest('[data-task-id]')).not.toHaveAttribute('data-task-start-feedback')
    expect(bindings.SaveSettings).not.toHaveBeenCalledWith(expect.objectContaining({activeTaskStatus: 'running'}))
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

  it('从任务操作菜单搁置任务后使用完整列表刷新任务树', async () => {
    const user = userEvent.setup()
    bindings.SetTaskShelved.mockResolvedValue([{
      id: 'task-1', title: '清理临时文件', description: '', status: 'running', shelved: true, createdAt: '2026-07-22T00:00:00Z',
    }])
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(await screen.findByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '搁置任务'}))

    expect(bindings.SetTaskShelved).toHaveBeenCalledWith('task-1', true)
    expect(await screen.findByRole('button', {name: '展开已搁置任务'})).toHaveTextContent('已搁置 (1)')
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

  it('为任务树保留可收缩的面板剩余高度', async () => {
    render(<App/>)

    const taskTree = await screen.findByRole('navigation', {name: '任务和终端'})
    const taskTreeRegion = taskTree.parentElement
    const taskPanel = taskTreeRegion?.parentElement
    if (!taskTreeRegion || !taskPanel) {
      throw new Error('未找到任务树的面板容器')
    }

    expect(taskPanel).toHaveStyle({display: 'grid', gridTemplateRows: '42px minmax(0, 1fr)', minHeight: '0'})
    expect(taskTreeRegion).toHaveStyle({minHeight: '0'})
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

	it('在设置中编辑任务模板字段并选择当前模板', async () => {
		const user = userEvent.setup()
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '任务模板'}))
		await user.click(screen.getByRole('button', {name: '新增模板'}))
		const editor = screen.getByRole('dialog', {name: '新增任务模板'})
		await user.type(within(editor).getByRole('textbox', {name: /模板名称/}), '发布任务')
		await user.type(within(editor).getByRole('textbox', {name: /字段 1 键/}), 'environment')
		await user.type(within(editor).getByRole('textbox', {name: /字段 1 显示名称/}), '环境')
		await user.type(within(editor).getByRole('textbox', {name: '字段 1 默认值'}), 'production')
		await user.click(within(editor).getByLabelText('必填'))
		await user.click(within(editor).getByLabelText('注入生命周期环境变量'))
		await user.click(within(editor).getByRole('button', {name: '新增字段'}))
		await user.type(within(editor).getByRole('textbox', {name: /字段 2 键/}), 'deploy')
		await user.type(within(editor).getByRole('textbox', {name: /字段 2 显示名称/}), '立即部署')
		await user.selectOptions(within(editor).getByRole('combobox', {name: '字段 2 类型'}), '布尔值')
		// 字段 2 类型已通过 selectOptions 选择
		await user.click(within(editor).getByLabelText('默认选中'))
		await user.click(within(editor).getByLabelText('必填（必须勾选）'))
		await user.click(within(editor).getAllByLabelText('注入生命周期环境变量')[1])
		await user.click(within(editor).getByRole('button', {name: '保存模板'}))

		await waitFor(() => expect(screen.queryByRole('dialog', {name: '新增任务模板'})).not.toBeInTheDocument())
		await user.selectOptions(screen.getByRole('combobox', {name: '当前任务模板'}), '发布任务')
		// 当前任务模板已通过 selectOptions 选择
		await user.click(screen.getByRole('button', {name: '保存'}))

		await waitFor(() => expect(bindings.SaveSettings).toHaveBeenCalledOnce())
		const saved = bindings.SaveSettings.mock.calls[0][0]
		expect(saved.taskTemplates).toEqual([expect.objectContaining({
			name: '发布任务',
			fields: [
				{key: 'environment', displayName: '环境', inputType: 'string', required: true, defaultValue: 'production', injectEnvironment: true},
				{key: 'deploy', displayName: '立即部署', inputType: 'bool', required: true, defaultValue: true, injectEnvironment: true},
			],
		})])
		expect(saved.activeTaskTemplateId).toBe(saved.taskTemplates[0].id)
	})

	it('保存非模板设置后重新打开设置仍显示已保存模板', async () => {
		const user = userEvent.setup()
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			taskTemplates: [{
				id: 'release', name: '发布模板', fields: [{key: 'environment', displayName: '环境', inputType: 'string', required: false, defaultValue: '', injectEnvironment: false}],
			}],
			activeTaskTemplateId: 'release',
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '终端 Shell'}))
		const shellPath = screen.getByRole('textbox', {name: 'Shell 路径'})
		await user.clear(shellPath)
		await user.type(shellPath, '/bin/zsh')
		await user.click(screen.getByRole('button', {name: '保存'}))
		await waitFor(() => expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining({
			taskTemplates: [expect.objectContaining({id: 'release'})],
			activeTaskTemplateId: 'release',
		})))

		await waitFor(() => expect(screen.queryByRole('dialog', {name: '设置'})).not.toBeInTheDocument())
		await user.click(screen.getByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '任务模板'}))
		expect(within(screen.getByRole('dialog', {name: '设置'})).getByText('1 个字段 · 当前使用')).toBeInTheDocument()
	})

	it('活动任务引用模板时禁用对应删除按钮，已完成任务不会阻止删除', async () => {
		const user = userEvent.setup()
		bindings.ListTasks.mockResolvedValue([
			{id: 'running-release', title: '正在发布', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z', taskTemplateId: 'release', templateFields: {environment: 'production'}},
			{id: 'completed-archive', title: '已归档', description: '', status: 'completed', createdAt: '2026-07-22T00:00:00Z', taskTemplateId: 'archive', templateFields: {retention: '30d'}},
		])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			taskTemplates: [
				{id: 'release', name: '发布模板', fields: [{key: 'environment', displayName: '环境', inputType: 'string', required: false, defaultValue: '', injectEnvironment: false}]},
				{id: 'archive', name: '归档模板', fields: [{key: 'retention', displayName: '保留期', inputType: 'string', required: false, defaultValue: '', injectEnvironment: false}]},
			],
			activeTaskTemplateId: 'release',
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '任务模板'}))
		const protectedDelete = screen.getByRole('button', {name: '删除任务模板 发布模板'})
		const completedDelete = screen.getByRole('button', {name: '删除任务模板 归档模板'})
		expect(protectedDelete).toBeDisabled()
		expect(completedDelete).toBeEnabled()
		await user.hover(protectedDelete.parentElement!)
		expect(protectedDelete).toHaveAttribute('title', '未执行或执行中的任务正在使用此模板，暂不能删除')

		await user.click(completedDelete)
		await user.click(screen.getByRole('button', {name: '保存'}))
		await waitFor(() => expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining({
			taskTemplates: [expect.objectContaining({id: 'release'})],
		})))
	})

	it('历史活动任务缺少模板 ID 时禁用所有模板删除按钮', async () => {
		const user = userEvent.setup()
		bindings.ListTasks.mockResolvedValue([{
			id: 'legacy-pending', title: '旧任务', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z', templateFields: {environment: 'production'},
		}])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			taskTemplates: [
				{id: 'release', name: '发布模板', fields: [{key: 'environment', displayName: '环境', inputType: 'string', required: false, defaultValue: '', injectEnvironment: false}]},
				{id: 'archive', name: '归档模板', fields: [{key: 'retention', displayName: '保留期', inputType: 'string', required: false, defaultValue: '', injectEnvironment: false}]},
			],
			activeTaskTemplateId: 'release',
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '任务模板'}))
		const protectedDelete = screen.getByRole('button', {name: '删除任务模板 发布模板'})
		expect(protectedDelete).toBeDisabled()
		expect(screen.getByRole('button', {name: '删除任务模板 归档模板'})).toBeDisabled()
		await user.hover(protectedDelete.parentElement!)
		expect(protectedDelete).toHaveAttribute('title', '未执行或执行中的旧任务包含无法归属的模板字段，完成这些任务后才能删除模板')
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

  it('长任务表单在弹窗内滚动并保持创建操作可达', async () => {
    const user = userEvent.setup()
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '新建任务'}))

    const dialog = screen.getByRole('dialog', {name: '新建任务'})
    const form = dialog.querySelector('form')
    expect(form).not.toBeNull()
    expect(form).toHaveStyle({display: 'flex', flexDirection: 'column', flex: '1 1 auto', minHeight: '0'})
    expect(within(dialog).getByTestId('task-dialog-content')).toHaveStyle({overflowY: 'auto', minHeight: '0'})
    expect(within(dialog).getByRole('button', {name: '创建'})).toBeVisible()
  })

  it('新建任务随机预选颜色并将其用于创建请求', async () => {
    const user = userEvent.setup()
    const random = vi.spyOn(Math, 'random').mockReturnValue(0)
    bindings.CreateTask.mockResolvedValue({
      id: 'task-2', title: '随机颜色任务', description: '', status: 'pending', color: '#ef4444', createdAt: '2026-07-22T00:00:00Z',
    })

    try {
      render(<App/>)

      await user.click(await screen.findByRole('button', {name: '新建任务'}))
      expect(screen.getByLabelText('任务颜色')).toHaveValue('#ef4444')
      await user.type(screen.getByRole('textbox', {name: '标题'}), '随机颜色任务')
      await user.click(screen.getByRole('button', {name: '创建'}))

      expect(bindings.CreateTask).toHaveBeenCalledWith('随机颜色任务', '', '#ef4444')
    } finally {
      random.mockRestore()
    }
  })

	it('当前模板字段在新建任务中显示默认值并随表单保存', async () => {
		const user = userEvent.setup()
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			taskTemplates: [{
				id: 'release', name: '发布任务', fields: [
					{key: 'environment', displayName: '环境', inputType: 'string', required: true, defaultValue: 'development', injectEnvironment: true},
					{key: 'deploy', displayName: '立即部署', inputType: 'bool', required: false, defaultValue: false, injectEnvironment: true},
				],
			}],
			activeTaskTemplateId: 'release',
		})
		bindings.CreateTaskWithExtraInfoAndTemplateFields.mockResolvedValue({
			id: 'task-template', title: '发布 API', description: '', color: '#4f46e5', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
			templateFields: {environment: 'production', deploy: false},
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		expect(screen.getByRole('textbox', {name: /环境/})).toHaveValue('development')
		expect(screen.getByLabelText('立即部署')).not.toBeChecked()
		const description = screen.getByRole('textbox', {name: '任务描述'})
		const templateFields = screen.getByTestId('task-template-fields')
		expect(description.compareDocumentPosition(templateFields) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
		expect(templateFields.compareDocumentPosition(screen.getByLabelText('任务颜色')) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
		await user.clear(screen.getByRole('textbox', {name: /环境/}))
		await user.type(screen.getByRole('textbox', {name: /环境/}), 'production')
		await user.type(screen.getByRole('textbox', {name: '标题'}), '发布 API')
		await user.click(screen.getByRole('button', {name: '创建'}))

		await waitFor(() => expect(bindings.CreateTaskWithExtraInfoAndTemplateFields).toHaveBeenCalledWith(
			'发布 API', '', expect.any(String), [], {environment: 'production', deploy: false},
		))
	})

	it('默认分支模板要求输入分支', async () => {
		const user = userEvent.setup()
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			taskTemplates: [{id: 'preset.task-template.default-branch', name: '默认分支', fields: [
				{key: 'branch', displayName: '默认分支', inputType: 'string', required: true, defaultValue: '', injectEnvironment: false},
			]}], activeTaskTemplateId: 'preset.task-template.default-branch',
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		expect(screen.getByRole('textbox', {name: /默认分支/})).toHaveValue('')
		const title = screen.getByRole('textbox', {name: '标题'})
		await user.type(title, '迭代任务')
		const form = title.closest('form')
		if (!form) {
			throw new Error('未找到任务表单')
		}
		fireEvent.submit(form)

		await screen.findByText('字段“默认分支”不能为空')
		expect(bindings.CreateTaskWithExtraInfoAndTemplateFields).not.toHaveBeenCalled()
	})

	it('编辑旧任务时恢复当前字段值并隐藏历史字段', async () => {
		const user = userEvent.setup()
		bindings.ListTasks.mockResolvedValue([{
			id: 'task-template-old', title: '旧发布任务', description: '', status: 'pending', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z',
			templateFields: {environment: 'production', legacy: 'preserved'},
		}])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			taskTemplates: [{id: 'release', name: '发布任务', fields: [
				{key: 'environment', displayName: '环境', inputType: 'string', required: true, defaultValue: 'staging', injectEnvironment: false},
			]}], activeTaskTemplateId: 'release',
		})
		bindings.UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains.mockResolvedValue({
			id: 'task-template-old', title: '旧发布任务', description: '', status: 'pending', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z',
			templateFields: {environment: 'production', legacy: 'preserved'},
		})
		render(<App/>)

		await screen.findByText('旧发布任务')
		await user.click(screen.getByRole('button', {name: '任务操作'}))
		await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
		expect(screen.getByRole('textbox', {name: /环境/})).toHaveValue('production')
		expect(screen.queryByText('legacy')).not.toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '保存'}))

		await waitFor(() => expect(bindings.UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains).toHaveBeenCalledWith(
			'task-template-old', '旧发布任务', '', '#4f46e5', [], {environment: 'production'}, {},
		))
	})

	it('模板必填字符串和布尔字段阻止提交，满足后才创建任务', async () => {
		const user = userEvent.setup()
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			taskTemplates: [{id: 'release', name: '发布任务', fields: [
				{key: 'environment', displayName: '环境', inputType: 'string', required: true, defaultValue: '', injectEnvironment: false},
				{key: 'deploy', displayName: '立即部署', inputType: 'bool', required: true, defaultValue: false, injectEnvironment: false},
			]}], activeTaskTemplateId: 'release',
		})
		bindings.CreateTaskWithExtraInfoAndTemplateFields.mockResolvedValue({
			id: 'task-required-template', title: '必填模板任务', description: '', status: 'pending', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z',
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		const title = screen.getByRole('textbox', {name: '标题'})
		await user.type(title, '必填模板任务')
		const form = title.closest('form')
		if (!form) {
			throw new Error('未找到任务表单')
		}
		fireEvent.submit(form)
		await screen.findByText('字段“环境”不能为空')
		expect(bindings.CreateTaskWithExtraInfoAndTemplateFields).not.toHaveBeenCalled()
		await user.type(screen.getByRole('textbox', {name: /环境/}), 'production')
		fireEvent.submit(form)
		await screen.findByText('字段“立即部署”必须勾选')
		expect(bindings.CreateTaskWithExtraInfoAndTemplateFields).not.toHaveBeenCalled()
		await user.click(screen.getByLabelText('立即部署'))
		fireEvent.submit(form)
		await waitFor(() => expect(bindings.CreateTaskWithExtraInfoAndTemplateFields).toHaveBeenCalledWith(
			'必填模板任务', '', expect.any(String), [], {environment: 'production', deploy: true},
		))
	})

	it('新建任务套用默认命令链预设，并通过链选择绑定保存', async () => {
		const user = userEvent.setup()
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleChains: [
				{id: 'start-chain', name: '启动准备', commands: [{commandId: 'system.lifecycle.create-workspace', arguments: []}], applicableHooks: ['postStart']},
				{id: 'end-chain', name: '结束清理', commands: [{commandId: 'system.lifecycle.delete-workspace', arguments: []}], applicableHooks: ['beforeEnd']},
			],
			lifecyclePresets: [{id: 'default', name: '默认流程', chains: {postStart: 'start-chain', beforeEnd: 'end-chain'}}],
			defaultLifecyclePresetId: 'default',
		})
		bindings.CreateTaskWithExtraInfoAndLifecycleChains.mockResolvedValue({
			id: 'task-new', title: '带链任务', description: '', color: '#4f46e5', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		expect(screen.getByLabelText('命令链预设')).toHaveTextContent('默认流程')
		expect(screen.getByLabelText('开始后')).toHaveTextContent('启动准备')
		const titleInput = document.querySelector<HTMLInputElement>('input[required]')
		const taskForm = titleInput?.closest('form')
		if (!titleInput || !taskForm) {
			throw new Error('未找到新建任务表单')
		}
		fireEvent.change(titleInput, {target: {value: '带链任务'}})
		fireEvent.submit(taskForm)

		await waitFor(() => expect(bindings.CreateTaskWithExtraInfoAndLifecycleChains).toHaveBeenCalledWith(
			'带链任务', '', expect.any(String), [], {postStart: 'start-chain', beforeEnd: 'end-chain'},
		))
	})

	it('新建任务选择不使用预设时显式保存空命令链映射', async () => {
		const user = userEvent.setup()
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleChains: [{id: 'prepare', name: '开始前准备', commands: [], applicableHooks: ['beforeStart']}],
			lifecyclePresets: [{id: 'default', name: '默认流程', chains: {beforeStart: 'prepare'}}], defaultLifecyclePresetId: 'default',
		})
		bindings.CreateTaskWithExtraInfoAndLifecycleChains.mockResolvedValue({
			id: 'task-empty-preset', title: '空预设任务', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		await user.click(screen.getByLabelText('命令链预设'))
		await user.click(await screen.findByRole('option', {name: '不使用预设'}))
		await user.type(screen.getByRole('textbox', {name: '标题'}), '空预设任务')
		await user.click(screen.getByRole('button', {name: '创建'}))

		await waitFor(() => expect(bindings.CreateTaskWithExtraInfoAndLifecycleChains).toHaveBeenCalledWith(
			'空预设任务', '', expect.any(String), [], {},
		))
	})

	it('按钩子范围筛选新建任务链，并在编辑任务时锁定链选择', async () => {
		const user = userEvent.setup()
		const beforeStartChain = {id: 'before-start-chain', name: '开始前准备', commands: [{commandId: 'prepare', arguments: []}], applicableHooks: ['beforeStart']}
		const postStartChain = {id: 'post-start-chain', name: '开始后通知', commands: [{commandId: 'notify', arguments: []}], applicableHooks: ['postStart']}
		bindings.ListTasks.mockResolvedValue([{
			id: 'task-scoped', title: '有范围的任务', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
			lifecycleChains: {beforeStart: beforeStartChain.id, postStart: postStartChain.id},
		}])
		const scopedSettings = {
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
		lifecycleChains: [beforeStartChain, postStartChain], lifecyclePresets: [{id: 'scoped-default', name: '范围默认预设', chains: {beforeStart: beforeStartChain.id, postStart: postStartChain.id}}], defaultLifecyclePresetId: 'scoped-default',
		}
		bindings.GetSettings.mockResolvedValue(scopedSettings)
		bindings.SaveSettings.mockImplementation(async (next) => ({...scopedSettings, ...next}))
		bindings.UpdateTask.mockResolvedValue({
			id: 'task-scoped', title: '更新后的范围任务', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
			lifecycleChains: {beforeStart: beforeStartChain.id, postStart: postStartChain.id},
		})
		bindings.UpdateTaskWithExtraInfoAndLifecycleChains.mockResolvedValue({
			id: 'task-scoped', title: '更新后的范围任务', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
			lifecycleChains: {beforeStart: beforeStartChain.id, postStart: postStartChain.id},
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		const beforeStartSelect = screen.getByLabelText('开始前')
		expect(within(beforeStartSelect).getByRole('option', {name: '开始前准备'})).toBeInTheDocument()
		expect(within(beforeStartSelect).queryByRole('option', {name: '开始后通知'})).not.toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '取消'}))

		await user.click(await screen.findByRole('tab', {name: /执行中/}))
		await user.click(screen.getByRole('button', {name: '任务操作'}))
		await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
		expect(screen.getByLabelText('开始前')).toHaveAttribute('aria-disabled', 'true')
		expect(screen.getByLabelText('开始后')).toHaveAttribute('aria-disabled', 'true')
		await user.clear(screen.getByRole('textbox', {name: '标题'}))
		await user.type(screen.getByRole('textbox', {name: '标题'}), '更新后的范围任务')
		await user.click(screen.getByRole('button', {name: '保存'}))

		await waitFor(() => expect(bindings.UpdateTask).toHaveBeenCalledWith('task-scoped', '更新后的范围任务', '', '#4f46e5'))
		expect(bindings.UpdateTaskWithExtraInfoAndLifecycleChains).not.toHaveBeenCalled()
	})

	it('编辑未执行任务时可修改命令链', async () => {
		const user = userEvent.setup()
		const beforeStartChain = {id: 'before-start-chain', name: '开始前准备', commands: [{commandId: 'prepare', arguments: []}], applicableHooks: ['beforeStart']}
		const postStartChain = {id: 'post-start-chain', name: '开始后通知', commands: [{commandId: 'notify', arguments: []}], applicableHooks: ['postStart']}
		bindings.ListTasks.mockResolvedValue([{
			id: 'task-pending-chain', title: '待调整链任务', description: '', status: 'pending', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z',
			lifecycleChains: {beforeStart: beforeStartChain.id},
		}])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleChains: [beforeStartChain, postStartChain], lifecyclePresets: [{id: 'prepare', name: '准备预设', chains: {beforeStart: beforeStartChain.id}}],
		})
		bindings.UpdateTaskWithExtraInfoAndLifecycleChains.mockResolvedValue({
			id: 'task-pending-chain', title: '待调整链任务', description: '', status: 'pending', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z',
			lifecycleChains: {beforeStart: beforeStartChain.id, postStart: postStartChain.id},
		})
		render(<App/>)

		await screen.findByText('待调整链任务')
		await user.click(screen.getByRole('button', {name: '任务操作'}))
		await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
		expect(screen.getByLabelText('命令链预设')).toHaveTextContent('准备预设')
		expect(screen.getByLabelText('开始后')).not.toHaveAttribute('aria-disabled', 'true')
		await user.click(screen.getByLabelText('开始后'))
		await user.click(await screen.findByRole('option', {name: '开始后通知'}))
		expect(screen.getByLabelText('命令链预设')).toHaveTextContent('自定义')
		await user.click(screen.getByRole('button', {name: '保存'}))

		await waitFor(() => expect(bindings.UpdateTaskWithExtraInfoAndLifecycleChains).toHaveBeenCalledWith(
			'task-pending-chain', '待调整链任务', '', '#4f46e5', [], {beforeStart: beforeStartChain.id, postStart: postStartChain.id},
		))
	})

	it('生命周期设置管理预设并可切换默认预设', async () => {
		const user = userEvent.setup()
		const beforeStartChain = {id: 'prepare-chain', name: '开始前准备', commands: [{commandId: 'prepare', arguments: []}], applicableHooks: ['beforeStart']}
		const lifecycleSettings = {
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleChains: [beforeStartChain], lifecyclePresets: [{id: 'default', name: '默认预设', chains: {beforeStart: beforeStartChain.id}}], defaultLifecyclePresetId: 'default',
		}
		bindings.GetSettings.mockResolvedValue(lifecycleSettings)
		bindings.SaveDefaultLifecyclePreset.mockResolvedValue({...lifecycleSettings, defaultLifecyclePresetId: ''})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		expect(screen.getByLabelText('默认预设')).toHaveTextContent('默认预设')
		await user.click(screen.getByRole('button', {name: '新增预设'}))
		const presetDialog = screen.getByRole('dialog', {name: '新增命令链预设'})
		await user.type(within(presetDialog).getByRole('textbox', {name: '预设名称'}), '发布预设')
		await user.click(within(presetDialog).getByLabelText('开始前'))
		await user.click(await screen.findByRole('option', {name: '开始前准备'}))
		await user.click(within(presetDialog).getByRole('button', {name: '保存预设'}))
		await waitFor(() => expect(bindings.SaveLifecyclePreset).toHaveBeenCalledWith(expect.objectContaining({name: '发布预设', chains: {beforeStart: 'prepare-chain'}})))
		await waitFor(() => expect(screen.queryByRole('dialog', {name: '新增命令链预设'})).not.toBeInTheDocument())

		await user.click(screen.getByLabelText('默认预设'))
		await user.click(await screen.findByRole('option', {name: '不设置默认预设'}))
		expect(bindings.SaveDefaultLifecyclePreset).toHaveBeenCalledWith('')
		const defaultPresetRow = screen.getByLabelText('删除预设 默认预设').parentElement?.parentElement as HTMLElement
		await user.click(within(defaultPresetRow).getByRole('button', {name: '复制'}))
		expect(bindings.CopyLifecyclePreset).toHaveBeenCalledWith('default')
		await user.click(screen.getByLabelText('删除预设 默认预设'))
		expect(bindings.DeleteLifecyclePreset).toHaveBeenCalledWith('default')
	})

	it('编辑已完成任务时锁定命令链', async () => {
		const user = userEvent.setup()
		const postEndChain = {id: 'post-end-chain', name: '结束后清理', commands: [{commandId: 'cleanup', arguments: []}], applicableHooks: ['postEnd']}
		bindings.ListTasks.mockResolvedValue([{
			id: 'task-completed-chain', title: '已完成链任务', description: '', status: 'completed', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z',
			lifecycleChains: {postEnd: postEndChain.id},
		}])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleChains: [postEndChain], lifecyclePresets: [], activeTaskStatus: 'completed',
		})
		bindings.UpdateTask.mockResolvedValue({
			id: 'task-completed-chain', title: '已完成链任务', description: '', status: 'completed', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z',
			lifecycleChains: {postEnd: postEndChain.id},
		})
		render(<App/>)

		await screen.findByText('已完成链任务')
		await user.click(screen.getByRole('button', {name: '任务操作'}))
		await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
		expect(screen.getByLabelText('结束后')).toHaveAttribute('aria-disabled', 'true')
		await user.click(screen.getByRole('button', {name: '保存'}))

		await waitFor(() => expect(bindings.UpdateTask).toHaveBeenCalledWith('task-completed-chain', '已完成链任务', '', '#4f46e5'))
		expect(bindings.UpdateTaskWithExtraInfoAndLifecycleChains).not.toHaveBeenCalled()
	})

	it('命令和链管理按适用范围保存并筛选命令', async () => {
		const user = userEvent.setup()
		const beforeStartCommand = {id: 'prepare', kind: 'custom', name: '开始前命令', command: 'prepare', arguments: [], applicableHooks: ['beforeStart']}
		const postStartCommand = {id: 'notify', kind: 'custom', name: '开始后命令', command: 'notify', arguments: [], applicableHooks: ['postStart']}
		bindings.ListLifecycleCommands.mockResolvedValue([beforeStartCommand, postStartCommand])
		bindings.ListLifecycleCommandChains.mockResolvedValue([])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleCommands: [beforeStartCommand, postStartCommand], lifecycleChains: [], lifecyclePresets: [],
		})
		bindings.SaveLifecycleCommand.mockResolvedValue({...beforeStartCommand, id: 'new-command', name: '新命令'})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		await screen.findByText('开始前命令')
		await user.click(screen.getByRole('button', {name: '新增链'}))
		const chainDialog = screen.getByRole('dialog', {name: '新增命令链'})
		expect(within(chainDialog).getByText('请先选择命令链适用范围。')).toBeInTheDocument()
		await user.click(within(chainDialog).getByLabelText('开始前'))
		expect(await within(chainDialog).findByLabelText('开始前命令')).toBeInTheDocument()
		expect(within(chainDialog).queryByLabelText('开始后命令')).not.toBeInTheDocument()
    await user.click(within(chainDialog).getByRole('button', {name: '取消'}))
    await waitFor(() => {
      expect(screen.queryByRole('dialog', {name: '新增命令链'})).not.toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', {name: '新增命令'}))
		const commandDialog = screen.getByRole('dialog', {name: '新增命令'})
		expect(within(commandDialog).getByRole('button', {name: '保存命令'})).toBeDisabled()
		expect(within(commandDialog).getByRole('textbox', {name: '固定参数（每行一个）'})).toBeInTheDocument()
		expect(within(commandDialog).getByLabelText('允许在命令链中追加参数')).not.toBeChecked()
		await user.type(within(commandDialog).getByRole('textbox', {name: '命令名称'}), '新命令')
		await user.type(within(commandDialog).getByRole('textbox', {name: '可执行命令'}), 'new-command')
		await user.click(within(commandDialog).getByLabelText('开始前'))
		await user.click(within(commandDialog).getByRole('button', {name: '保存命令'}))
		await waitFor(() => expect(bindings.SaveLifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({name: '新命令', command: 'new-command', chainArgumentMode: 'disabled', applicableHooks: ['beforeStart']})))
	})

	it('命令链管理提供扩展规则帮助', async () => {
		const user = userEvent.setup()
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		await user.click(screen.getByRole('button', {name: '查看命令链扩展规则'}))

		const helpDialog = screen.getByRole('dialog', {name: '命令链扩展规则'})
		expect(within(helpDialog).getByText(/固定参数会先于命令链追加参数传入/)).toBeInTheDocument()
		expect(within(helpDialog).getByText(/前一条自定义命令的原始标准输出/)).toBeInTheDocument()
		expect(within(helpDialog).getByText('baseURL')).toBeInTheDocument()
		expect(within(helpDialog).getByText(/只通过标准输入提供，不是终端环境变量/)).toBeInTheDocument()
		expect(within(helpDialog).getByText(/dir=<相对目录>/)).toBeInTheDocument()
		expect(within(helpDialog).getByText(/从链首重新执行/)).toBeInTheDocument()

		await user.click(within(helpDialog).getByRole('button', {name: '关闭'}))
		await waitFor(() => expect(screen.queryByRole('dialog', {name: '命令链扩展规则'})).not.toBeInTheDocument())
	})

	it('在命令链中配置 Git 参数并显示系统使用文档', async () => {
		const user = userEvent.setup()
		const gitCommand = {
			id: 'system.lifecycle.git-clone', kind: 'git-clone', name: 'Git 仓库克隆', arguments: [],
			chainArgumentMode: 'enabled',
			documentation: '参数可留空；留空时每个内置 Git 项目将克隆到任务工作目录下的 <项目名称>。填写时使用 dir=<相对目录>，将克隆到任务工作目录下的 <dir>/<项目名称>。',
			applicableHooks: ['beforeStart', 'beforeEnd', 'updateTask'],
		}
		bindings.ListLifecycleCommands.mockResolvedValue([gitCommand])
		bindings.ListLifecycleCommandChains.mockResolvedValue([])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleCommands: [gitCommand], lifecycleChains: [], lifecyclePresets: [],
		})
		bindings.SaveLifecycleCommandChain.mockImplementation(async (chain) => ({...chain, id: 'git-chain'}))
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		expect(await screen.findByText(/参数可留空/)).toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '新增链'}))
		const chainDialog = screen.getByRole('dialog', {name: '新增命令链'})
		await user.type(within(chainDialog).getByRole('textbox', {name: '命令链名称'}), '克隆仓库')
		await user.click(within(chainDialog).getByLabelText('开始前'))
		await user.click(within(chainDialog).getByLabelText('Git 仓库克隆'))
		const argumentsInput = within(chainDialog).getByRole('textbox', {name: 'Git 仓库克隆 追加参数（每行一个）'})
		await user.type(argumentsInput, 'dir=repositories')
		await user.click(within(chainDialog).getByRole('button', {name: '保存命令链'}))

		await waitFor(() => expect(bindings.SaveLifecycleCommandChain).toHaveBeenCalledWith({
			id: '', name: '克隆仓库', commands: [{commandId: 'system.lifecycle.git-clone', arguments: ['dir=repositories']}], applicableHooks: ['beforeStart'],
		}))
	})

	it('允许命令链将 Git 克隆目录参数留空', async () => {
		const user = userEvent.setup()
		const gitCommand = {
			id: 'system.lifecycle.git-clone', kind: 'git-clone', name: 'Git 仓库克隆', arguments: [],
			chainArgumentMode: 'enabled', documentation: '参数可留空；留空时克隆到任务工作目录。',
			applicableHooks: ['beforeStart', 'beforeEnd', 'updateTask'],
		}
		bindings.ListLifecycleCommands.mockResolvedValue([gitCommand])
		bindings.ListLifecycleCommandChains.mockResolvedValue([])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleCommands: [gitCommand], lifecycleChains: [], lifecyclePresets: [],
		})
		bindings.SaveLifecycleCommandChain.mockImplementation(async (chain) => ({...chain, id: 'git-chain'}))
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		await user.click(screen.getByRole('button', {name: '新增链'}))
		const chainDialog = screen.getByRole('dialog', {name: '新增命令链'})
		await user.type(within(chainDialog).getByRole('textbox', {name: '命令链名称'}), '默认目录克隆')
		await user.click(within(chainDialog).getByLabelText('开始前'))
		await user.click(within(chainDialog).getByLabelText('Git 仓库克隆'))
		await user.click(within(chainDialog).getByRole('button', {name: '保存命令链'}))

		await waitFor(() => expect(bindings.SaveLifecycleCommandChain).toHaveBeenCalledWith({
			id: '', name: '默认目录克隆', commands: [{commandId: 'system.lifecycle.git-clone', arguments: []}], applicableHooks: ['beforeStart'],
		}))
	})

	it('在命令链中配置指定 Git 仓库初始化参数并显示约束', async () => {
		const user = userEvent.setup()
		const cloneRepositoryCommand = {
			id: 'system.lifecycle.git-clone-repository', kind: 'git-clone-repository', name: '克隆指定 Git 仓库', arguments: [],
			chainArgumentMode: 'enabled',
			documentation: '参数：repository=<仓库地址>（必填）；dir=<相对目录>（可选）。仓库直接克隆到任务工作目录或指定子目录本身，不读取 Git 附加信息。目标必须为空目录，非空目录会失败。分支使用任务模板的 branch 字段；为空时由远程默认分支决定。',
			applicableHooks: ['beforeStart', 'postStart'],
		}
		bindings.ListLifecycleCommands.mockResolvedValue([cloneRepositoryCommand])
		bindings.ListLifecycleCommandChains.mockResolvedValue([])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleCommands: [cloneRepositoryCommand], lifecycleChains: [], lifecyclePresets: [],
		})
		bindings.SaveLifecycleCommandChain.mockImplementation(async (chain) => ({...chain, id: 'clone-template'}))
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		expect(await screen.findByText(/repository=<仓库地址>（必填）/)).toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '新增链'}))
		const chainDialog = screen.getByRole('dialog', {name: '新增命令链'})
		await user.type(within(chainDialog).getByRole('textbox', {name: '命令链名称'}), '初始化任务目录')
		await user.click(within(chainDialog).getByLabelText('开始前'))
		expect(within(chainDialog).getByLabelText('克隆指定 Git 仓库')).toBeInTheDocument()
		await user.click(within(chainDialog).getByLabelText('克隆指定 Git 仓库'))
		const argumentsInput = within(chainDialog).getByRole('textbox', {name: '克隆指定 Git 仓库 追加参数（每行一个）'})
		await user.type(argumentsInput, 'repository=https://example.com/template.git\ndir=template')
		expect(within(chainDialog).getByText(/目标必须为空目录，非空目录会失败/)).toBeInTheDocument()
		await user.click(within(chainDialog).getByRole('button', {name: '保存命令链'}))

		await waitFor(() => expect(bindings.SaveLifecycleCommandChain).toHaveBeenCalledWith({
			id: '', name: '初始化任务目录', commands: [{commandId: 'system.lifecycle.git-clone-repository', arguments: ['repository=https://example.com/template.git', 'dir=template']}], applicableHooks: ['beforeStart'],
		}))
	})

	it('在命令链中配置更新默认分支参数', async () => {
		const user = userEvent.setup()
		const updateDefaultBranchCommand = {
			id: 'system.lifecycle.update-default-branch', kind: 'update-default-branch', name: '更新默认分支', arguments: [],
			chainArgumentMode: 'enabled', documentation: '参数可留空；templateField=<字段键>（可选），省略时使用 branch。',
			applicableHooks: ['beforeStart', 'postStart', 'updateTask'],
		}
		bindings.ListLifecycleCommands.mockResolvedValue([updateDefaultBranchCommand])
		bindings.ListLifecycleCommandChains.mockResolvedValue([])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleCommands: [updateDefaultBranchCommand], lifecycleChains: [], lifecyclePresets: [],
		})
		bindings.SaveLifecycleCommandChain.mockImplementation(async (chain) => ({...chain, id: 'default-branch-chain'}))
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		expect(await screen.findByText(/templateField=<字段键>/)).toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '新增链'}))
		const chainDialog = screen.getByRole('dialog', {name: '新增命令链'})
		await user.type(within(chainDialog).getByRole('textbox', {name: '命令链名称'}), '设置发布分支')
		await user.click(within(chainDialog).getByLabelText('开始前'))
		await user.click(within(chainDialog).getByLabelText('更新默认分支'))
		await user.type(within(chainDialog).getByRole('textbox', {name: '更新默认分支 追加参数（每行一个）'}), 'templateField=release_branch')
		await user.click(within(chainDialog).getByRole('button', {name: '保存命令链'}))

		await waitFor(() => expect(bindings.SaveLifecycleCommandChain).toHaveBeenCalledWith({
			id: '', name: '设置发布分支', commands: [{commandId: 'system.lifecycle.update-default-branch', arguments: ['templateField=release_branch']}], applicableHooks: ['beforeStart'],
		}))
	})

	it('禁止链级追加参数时隐藏输入框但保留历史值', async () => {
		const user = userEvent.setup()
		const command = {
			id: 'deploy', kind: 'custom', name: '部署', command: 'deploy', arguments: ['--fixed'], chainArgumentMode: 'disabled', applicableHooks: ['beforeStart'],
		}
		const chain = {
			id: 'deploy-chain', name: '部署链', commands: [{commandId: 'deploy', arguments: ['--saved-extra']}], applicableHooks: ['beforeStart'],
		}
		bindings.ListLifecycleCommands.mockImplementation(async () => [command])
		bindings.ListLifecycleCommandChains.mockResolvedValue([chain])
		bindings.GetSettings.mockImplementation(async () => ({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleCommands: [command], lifecycleChains: [chain], lifecyclePresets: [],
		}))
		bindings.SaveLifecycleCommandChain.mockResolvedValue(chain)
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		await screen.findByText('部署链')
		await user.click(screen.getByRole('button', {name: '编辑命令链 部署链'}))
		const chainDialog = screen.getByRole('dialog', {name: '编辑命令链'})
		expect(within(chainDialog).queryByRole('textbox', {name: '部署 追加参数（每行一个）'})).not.toBeInTheDocument()
		await user.click(within(chainDialog).getByRole('button', {name: '保存命令链'}))
		await waitFor(() => expect(bindings.SaveLifecycleCommandChain).toHaveBeenCalledWith({
			id: 'deploy-chain', name: '部署链', commands: [{commandId: 'deploy', arguments: ['--saved-extra']}], applicableHooks: ['beforeStart'],
		}))
	})

	it('重新允许链级追加参数后显示既有参数', async () => {
		const user = userEvent.setup()
		const command = {
			id: 'deploy', kind: 'custom', name: '部署', command: 'deploy', arguments: ['--fixed'], chainArgumentMode: 'enabled', applicableHooks: ['beforeStart'],
		}
		const chain = {
			id: 'deploy-chain', name: '部署链', commands: [{commandId: 'deploy', arguments: ['--saved-extra']}], applicableHooks: ['beforeStart'],
		}
		bindings.ListLifecycleCommands.mockResolvedValue([command])
		bindings.ListLifecycleCommandChains.mockResolvedValue([chain])
		bindings.GetSettings.mockResolvedValue({
			workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'light', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
			lifecycleCommands: [command], lifecycleChains: [chain], lifecyclePresets: [],
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		await user.click(screen.getByRole('tab', {name: '生命周期编排'}))
		await screen.findByText('部署链')
		await user.click(screen.getByRole('button', {name: '编辑命令链 部署链'}))
		const chainDialog = screen.getByRole('dialog', {name: '编辑命令链'})
		expect(within(chainDialog).getByRole('textbox', {name: '部署 追加参数（每行一个）'})).toHaveValue('--saved-extra')
	})

	it('管理内置 Git 模板，并以默认值创建可复用信息', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [
				{key: 'name', displayName: '项目名称', defaultValue: ''},
				{key: 'repository', displayName: '仓库地址', defaultValue: 'git@example.com:team/api.git'},
			],
			parameters: [{key: 'branch', displayName: '仓库分支', required: true}],
		}])
		bindings.SaveExtraInfo.mockResolvedValue({
			id: 'api-info', templateId: 'git-template', catalogue: 'git', fields: [
				{key: 'name', displayName: '项目名称', value: 'API 服务'},
				{key: 'repository', displayName: '仓库地址', value: 'git@example.com:team/api.git'},
			],
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '分类模板'}))
		expect(screen.getByText('内置 Git')).toBeInTheDocument()
		expect(screen.getByRole('button', {name: '删除模板 git'})).toBeDisabled()
		expect(screen.getAllByRole('button', {name: '新增信息'})).toHaveLength(1)
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		expect(screen.getByRole('combobox', {name: '选择模板'})).toHaveTextContent('Git')
		await user.type(screen.getByRole('textbox', {name: '项目名称'}), 'API 服务')
		expect(screen.getByRole('textbox', {name: '仓库地址'})).toHaveValue('git@example.com:team/api.git')
		await user.click(screen.getByRole('button', {name: '保存信息'}))

		await waitFor(() => expect(bindings.SaveExtraInfo).toHaveBeenCalledWith(expect.objectContaining({
			id: '', templateId: 'git-template', catalogue: 'git', fields: expect.arrayContaining([
				expect.objectContaining({key: 'name', value: 'API 服务'}),
				expect.objectContaining({key: 'repository', value: 'git@example.com:team/api.git'}),
			]),
		})))
	})

	it('填写 Git 仓库地址时为未填写的项目名称自动提取仓库名', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [
				{key: 'name', displayName: '项目名称', defaultValue: '   '},
				{key: 'repository', displayName: '仓库地址', defaultValue: ''},
			],
			parameters: [],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		const repository = screen.getByRole('textbox', {name: '仓库地址'})
		await user.type(repository, '  git@gitlab.jiandan100.cn:webdev/interact-study.git  ')

		expect(screen.getByRole('textbox', {name: '项目名称'})).toHaveValue('interact-study')
		expect(repository).toHaveValue('  git@gitlab.jiandan100.cn:webdev/interact-study.git  ')
		await user.clear(repository)
		await user.type(repository, 'git@gitlab.jiandan100.cn:webdev/next-project.git')
		expect(screen.getByRole('textbox', {name: '项目名称'})).toHaveValue('interact-study')
	})

	it('填写 Git 仓库地址不会覆盖已有项目名称，且无效地址不回填', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [
				{key: 'name', displayName: '项目名称', defaultValue: ''},
				{key: 'repository', displayName: '仓库地址', defaultValue: ''},
			],
			parameters: [],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		const projectName = screen.getByRole('textbox', {name: '项目名称'})
		const repository = screen.getByRole('textbox', {name: '仓库地址'})
		await user.type(projectName, '互动学习')
		await user.type(repository, 'git@gitlab.jiandan100.cn:webdev/interact-study.git')
		expect(projectName).toHaveValue('互动学习')

		await user.clear(projectName)
		await user.clear(repository)
		await user.type(repository, 'git@gitlab.jiandan100.cn:webdev/interact-study')
		expect(projectName).toHaveValue('')

		await user.clear(repository)
		await user.type(repository, 'git@gitlab.jiandan100.cn:interact-study.git')
		expect(projectName).toHaveValue('')
	})

	it('非 Git 分类的仓库地址不自动修改名称', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'source-template', catalogue: 'source', displayName: '源代码', builtIn: false,
			fields: [
				{key: 'name', displayName: '名称', defaultValue: ''},
				{key: 'repository', displayName: '仓库地址', defaultValue: ''},
			],
			parameters: [],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		await user.type(screen.getByRole('textbox', {name: '仓库地址'}), 'git@gitlab.jiandan100.cn:webdev/interact-study.git')

		expect(screen.getByRole('textbox', {name: '名称'})).toHaveValue('')
	})

	it('收起分类模板时仍可新建模板，并按当前会话预选且切换新增信息模板', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([
			{id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: false, fields: [{key: 'name', displayName: '项目名称'}], parameters: []},
			{id: 'issue-template', catalogue: 'issue', displayName: '缺陷', builtIn: false, fields: [{key: 'name', displayName: '名称'}], parameters: []},
			{id: 'incident-template', catalogue: 'incident', displayName: '事件', builtIn: false, fields: [{key: 'name', displayName: '名称'}], parameters: []},
		])
		bindings.SaveExtraInfo.mockImplementation(async (draft) => ({...draft, id: 'git-info'}))
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		const templateSection = screen.getByRole('button', {name: '分类模板'})
		expect(templateSection).toHaveAttribute('aria-expanded', 'false')
		await user.click(screen.getByRole('button', {name: '新增模板'}))
		expect(screen.getByRole('dialog', {name: '新增模板'})).toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '取消'}))
		await waitFor(() => expect(screen.queryByRole('dialog', {name: '新增模板'})).not.toBeInTheDocument())

		await user.click(screen.getByRole('button', {name: '新增信息'}))
		const templateSelect = screen.getByRole('combobox', {name: '选择模板'})
		await user.click(templateSelect)
		await user.click(screen.getByRole('option', {name: 'Git（git）'}))
		await user.type(screen.getByRole('textbox', {name: '项目名称'}), 'API 服务')
		await user.click(screen.getByRole('button', {name: '保存信息'}))
		await waitFor(() => expect(screen.queryByRole('dialog', {name: '新增信息'})).not.toBeInTheDocument())

		await user.click(screen.getByRole('button', {name: '新增信息'}))
		const defaultTemplateSelect = screen.getByRole('combobox', {name: '选择模板'})
		expect(defaultTemplateSelect).toHaveTextContent('Git')
		await user.click(defaultTemplateSelect)
		await user.click(screen.getByRole('option', {name: '缺陷（issue）'}))
		expect(screen.getByRole('textbox', {name: '名称'})).toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '取消'}))
		await waitFor(() => expect(screen.queryByRole('dialog', {name: '新增信息'})).not.toBeInTheDocument())

		await user.click(templateSection)
		await user.click(screen.getByRole('button', {name: '删除模板 git'}))
		await waitFor(() => expect(screen.queryByRole('button', {name: '删除模板 git'})).not.toBeInTheDocument())
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		expect(screen.getByRole('combobox', {name: '选择模板'})).toBeInTheDocument()
	})

	it('仅有一个分类时新增信息预选该模板且仍显示选择控件', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}], parameters: [],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		expect(screen.getByRole('combobox', {name: '选择模板'})).toHaveTextContent('Git')
		expect(screen.getByRole('textbox', {name: '项目名称'})).toBeInTheDocument()
	})

	it('将新增模板和新增信息放在同一管理操作栏', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}], parameters: [],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		const actions = screen.getByTestId('extra-info-manager-actions')
		expect(within(actions).getByRole('button', {name: '新增模板'})).toBeInTheDocument()
		expect(within(actions).getByRole('button', {name: '新增信息'})).toBeEnabled()
	})

	it('模板和信息编辑弹窗使用中等宽度', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}], parameters: [],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增模板'}))
		expect(screen.getByRole('dialog', {name: '新增模板'})).toHaveClass('max-w-2xl')
		await user.click(screen.getByRole('button', {name: '取消'}))
		await waitFor(() => expect(screen.queryByRole('dialog', {name: '新增模板'})).not.toBeInTheDocument())

		await user.click(screen.getByRole('button', {name: '新增信息'}))
		expect(screen.getByRole('dialog', {name: '新增信息'})).toHaveClass('max-w-2xl')
	})

	it('模板动态参数可切换为复选框，且不再设置必填状态', async () => {
		const user = userEvent.setup()
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增模板'}))
		await user.click(screen.getByRole('button', {name: '新增参数'}))
		await user.click(screen.getByLabelText('参数 1 必填'))
		await user.click(screen.getByRole('combobox', {name: '参数类型 1'}))
		await user.click(screen.getByRole('option', {name: '复选框'}))

		expect(screen.queryByLabelText('参数 1 必填')).not.toBeInTheDocument()
	})

	it('模板和信息编辑器使用清晰字段布局与平面参数分隔行', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [
				{key: 'name', displayName: '项目名称', defaultValue: ''},
				{key: 'repository', displayName: '仓库地址', defaultValue: 'git@example.com:team/api.git'},
			],
			parameters: [{key: 'branch', displayName: '仓库分支', required: true}],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增模板'}))
		const templateBasicFields = screen.getByTestId('extra-info-template-basic-fields')
		expect(getComputedStyle(templateBasicFields).display).toBe('grid')
		expect(getComputedStyle(templateBasicFields).gridTemplateColumns).toBe('repeat(auto-fit, minmax(220px, 1fr))')
		expect(screen.getByRole('textbox', {name: '分类'})).toHaveClass('text-sm')
		expect(screen.getByRole('textbox', {name: '模板备注'})).toHaveClass('text-sm')
		const templateFixedField = screen.getByTestId('extra-info-template-fixed-field-0')
		expect(getComputedStyle(templateFixedField).borderTopStyle).toBe('solid')
		expect(getComputedStyle(templateFixedField).gridTemplateColumns).toBe('minmax(0, 1fr) auto')
		await user.click(screen.getByRole('button', {name: '新增参数'}))
		const templateParameter = screen.getByTestId('extra-info-template-parameter-0')
		expect(getComputedStyle(templateParameter).borderTopStyle).toBe('solid')
		expect(getComputedStyle(templateParameter).gridTemplateColumns).toBe('minmax(0, 1fr) auto')
		await user.click(screen.getByRole('button', {name: '取消'}))
		await waitFor(() => expect(screen.queryByRole('dialog', {name: '新增模板'})).not.toBeInTheDocument())

		await user.click(screen.getByRole('button', {name: '新增信息'}))
		expect(screen.getByRole('combobox', {name: '选择模板'})).toHaveTextContent('Git')
		const draftFields = screen.getByTestId('extra-info-draft-fields')
		expect(getComputedStyle(draftFields).display).toBe('grid')
		expect(getComputedStyle(draftFields).gridTemplateColumns).toBe('1fr')
		expect(screen.getByRole('textbox', {name: '项目名称'})).toHaveClass('text-sm')
		await user.click(screen.getByRole('button', {name: '新增动态参数'}))
		const draftParameter = screen.getByTestId('extra-info-draft-parameter-0')
		expect(getComputedStyle(draftParameter).borderTopStyle).toBe('solid')
	})

	it('信息级动态参数保存默认值并在任务中作为只读定义带入', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [
				{key: 'name', displayName: '项目名称', defaultValue: ''},
				{key: 'repository', displayName: '仓库地址', defaultValue: 'git@example.com:team/api.git'},
			],
			parameters: [{key: 'branch', displayName: '仓库分支', required: false}],
		}])
		bindings.SaveExtraInfo.mockResolvedValue({
			id: 'api-info', templateId: 'git-template', catalogue: 'git', fields: [
				{key: 'name', displayName: '项目名称', value: 'API 服务'},
				{key: 'repository', displayName: '仓库地址', value: 'git@example.com:team/api.git'},
			],
			parameters: [{key: 'environment', displayName: '环境', required: true, value: 'production'}],
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		await user.type(screen.getByRole('textbox', {name: '项目名称'}), 'API 服务')
		await user.click(screen.getByRole('button', {name: '新增动态参数'}))
		await user.type(screen.getByRole('textbox', {name: '参数键 1'}), 'environment')
		await user.type(screen.getByRole('textbox', {name: '参数显示名称 1'}), '环境')
		await user.type(screen.getByRole('textbox', {name: '默认值 1'}), 'production')
		await user.click(screen.getByLabelText('参数 1 必填'))
		await user.click(screen.getByRole('button', {name: '保存信息'}))

		await waitFor(() => expect(bindings.SaveExtraInfo).toHaveBeenCalledWith(expect.objectContaining({
			parameters: [expect.objectContaining({key: 'environment', displayName: '环境', required: true, value: 'production'})],
		})))
		await user.click(await screen.findByRole('button', {name: '关闭'}))
		await waitFor(() => expect(screen.queryByRole('dialog', {name: '额外信息管理'})).not.toBeInTheDocument())
		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		await user.click(screen.getByRole('checkbox', {name: 'API 服务'}))
		expect(screen.getByRole('textbox', {name: '环境'})).toHaveValue('production')
		expect(screen.queryByRole('textbox', {name: '参数键'})).not.toBeInTheDocument()
		expect(screen.queryByRole('textbox', {name: '显示名称'})).not.toBeInTheDocument()
	})

	it('新建信息只读预览模板动态参数', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([
			{
				id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
				fields: [{key: 'name', displayName: '项目名称'}],
				parameters: [
					{key: 'branch', displayName: '仓库分支', required: true, inputType: 'text'},
					{key: 'deploy', displayName: '自动部署', required: false, inputType: 'checkbox'},
				],
			},
			{id: 'issue-template', catalogue: 'issue', displayName: '缺陷', builtIn: false, fields: [{key: 'name', displayName: '名称'}], parameters: []},
		])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		await user.click(screen.getByRole('button', {name: '新增信息'}))
		await user.click(screen.getByRole('combobox', {name: '选择模板'}))
		await user.click(screen.getByRole('option', {name: 'Git（git）'}))

		const templateParameters = screen.getByTestId('extra-info-template-parameters')
		const branch = within(templateParameters).getByTestId('extra-info-template-parameter-preview-0')
		expect(branch).toHaveTextContent('参数键：branch')
		expect(branch).toHaveTextContent('显示名称：仓库分支')
		expect(branch).toHaveTextContent('默认值：空')
		expect(branch).toHaveTextContent('必填')
		expect(within(branch).queryByRole('textbox')).not.toBeInTheDocument()
		expect(within(branch).queryByRole('checkbox')).not.toBeInTheDocument()
		expect(within(branch).queryByRole('button')).not.toBeInTheDocument()

		const deploy = within(templateParameters).getByTestId('extra-info-template-parameter-preview-1')
		expect(deploy).toHaveTextContent('参数键：deploy')
		expect(deploy).toHaveTextContent('显示名称：自动部署')
		expect(deploy).toHaveTextContent('默认值：false')
		expect(deploy).not.toHaveTextContent('必填')
		expect(screen.getByRole('button', {name: '新增动态参数'})).toBeInTheDocument()
	})

	it('按模板折叠信息并按名称搜索', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}], parameters: [],
		}, {
			id: 'issue-template', catalogue: 'issue', displayName: '缺陷', builtIn: false,
			fields: [{key: 'name', displayName: '名称'}], parameters: [],
		}])
		bindings.ListExtraInfos.mockResolvedValue([{
			id: 'api-info', templateId: 'git-template', catalogue: 'git', fields: [{key: 'name', displayName: '项目名称', value: 'API 服务'}],
		}, {
			id: 'issue-info', templateId: 'issue-template', catalogue: 'issue', fields: [{key: 'name', displayName: '名称', value: '缺陷单'}],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		const gitGroup = screen.getByRole('button', {name: /Git.*git/})
		const issueGroup = screen.getByRole('button', {name: /缺陷.*issue/})
		expect(gitGroup).toHaveAttribute('aria-expanded', 'true')
		expect(issueGroup).toHaveAttribute('aria-expanded', 'false')
		await user.type(screen.getByRole('textbox', {name: '搜索信息'}), '缺陷')
		await waitFor(() => expect(issueGroup).toHaveAttribute('aria-expanded', 'true'))
		expect(gitGroup).toBeInTheDocument()
		expect(screen.getByText('缺陷单')).toBeInTheDocument()
		expect(screen.queryByText('API 服务')).not.toBeInTheDocument()
	})

	it('隐藏没有信息的分类，但保留搜索无匹配的分类', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}], parameters: [],
		}, {
			id: 'issue-template', catalogue: 'issue', displayName: '缺陷', builtIn: false,
			fields: [{key: 'name', displayName: '名称'}], parameters: [],
		}])
		bindings.ListExtraInfos.mockResolvedValue([{
			id: 'api-info', templateId: 'git-template', catalogue: 'git', fields: [{key: 'name', displayName: '项目名称', value: 'API 服务'}],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		expect(screen.getByRole('button', {name: /Git.*git/})).toBeInTheDocument()
		expect(screen.queryByRole('button', {name: /缺陷.*issue/})).not.toBeInTheDocument()
		await user.type(screen.getByRole('textbox', {name: '搜索信息'}), '不存在')
		expect(screen.getByRole('button', {name: /Git.*git/})).toBeInTheDocument()
		expect(screen.getByText('未找到匹配的信息。')).toBeInTheDocument()
	})

	it('分类模板可以整体折叠而不影响信息分组', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}], parameters: [],
		}])
		bindings.ListExtraInfos.mockResolvedValue([{
			id: 'api-info', templateId: 'git-template', catalogue: 'git', fields: [{key: 'name', displayName: '项目名称', value: 'API 服务'}],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		const templateSection = screen.getByRole('button', {name: /分类模板/})
		expect(templateSection).toHaveAttribute('aria-expanded', 'false')
		await user.click(templateSection)
		expect(templateSection).toHaveAttribute('aria-expanded', 'true')
		await user.click(templateSection)
		expect(templateSection).toHaveAttribute('aria-expanded', 'false')
		expect(screen.queryByText('git', {selector: 'p'})).not.toBeInTheDocument()
		expect(screen.getByRole('button', {name: /Git.*git/})).toBeInTheDocument()
	})

	it('分类模板列表使用可滚动的纵向布局，避免覆盖后续模板', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}], parameters: [],
		}, {
			id: 'legacy-template', catalogue: 'git-legacy', displayName: '旧仓库', builtIn: false,
			fields: [{key: 'name', displayName: '项目名称'}], parameters: [],
		}, {
			id: 'other-template', catalogue: 'other', displayName: '其他', builtIn: false,
			fields: [{key: 'name', displayName: '名称'}], parameters: [],
		}])
		bindings.ListExtraInfos.mockResolvedValue([{
			id: 'api-info', templateId: 'git-template', catalogue: 'git', fields: [{key: 'name', displayName: '项目名称', value: 'API 服务'}],
		}, {
			id: 'legacy-info', templateId: 'legacy-template', catalogue: 'git-legacy', fields: [{key: 'name', displayName: '项目名称', value: '遗留服务'}],
		}, {
			id: 'other-info', templateId: 'other-template', catalogue: 'other', fields: [{key: 'name', displayName: '名称', value: '归档条目'}],
		}])
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '额外信息管理'}))
		const content = screen.getByTestId('extra-info-manager-content')
		const scrollRegion = within(content).getByTestId('extra-info-manager-scroll')
		await user.click(within(content).getByRole('button', {name: '分类模板'}))
		expect(getComputedStyle(content).display).toBe('flex')
		expect(getComputedStyle(content).flexDirection).toBe('column')
		expect(scrollRegion).toHaveClass('max-h-[55vh]', 'overflow-y-auto')
		expect(within(content).getByRole('button', {name: '关闭'})).toBeInTheDocument()
		expect([...content.querySelectorAll('p')].map((item) => item.textContent)).toEqual(['git', 'git-legacy', 'other'])
		await user.click(within(content).getByRole('button', {name: '分类模板'}))
		expect(within(content).queryByText('git', {selector: 'p'})).not.toBeInTheDocument()
		await user.type(within(content).getByRole('textbox', {name: '搜索信息'}), '归档')
		await waitFor(() => expect(within(content).getByText('归档条目')).toBeInTheDocument())
		expect(within(content).queryByText('API 服务')).not.toBeInTheDocument()
	})

	it('创建任务时按名称搜索信息，且只填写动态参数', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}, {key: 'repository', displayName: '仓库地址'}],
			parameters: [{key: 'branch', displayName: '仓库分支', required: true}],
		}, {
			id: 'issue-template', catalogue: 'issue', builtIn: false,
			fields: [{key: 'name', displayName: '名称'}, {key: 'project', displayName: '项目'}], parameters: [],
		}])
		bindings.ListExtraInfos.mockResolvedValue([{
			id: 'api-info', templateId: 'git-template', catalogue: 'git', fields: [
				{key: 'name', displayName: '项目名称', value: 'API 服务'},
				{key: 'repository', displayName: '仓库地址', value: 'git@example.com:team/api.git'},
			],
		}, {
			id: 'issue-info', templateId: 'issue-template', catalogue: 'issue', fields: [
				{key: 'name', displayName: '名称', value: '缺陷单'}, {key: 'project', displayName: '项目', value: 'TASK-123'},
			],
		}])
		bindings.CreateTaskWithExtraInfo.mockResolvedValue({
			id: 'task-2', title: '关联 API', description: '', status: 'pending', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z', extraInfo: [],
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		await user.type(screen.getByRole('textbox', {name: '标题'}), '关联 API')
		const search = screen.getByRole('textbox', {name: '搜索信息'})
		await user.type(search, 'API')
		await user.click(screen.getByRole('checkbox', {name: 'API 服务'}))
		expect(screen.queryByRole('textbox', {name: '项目名称'})).not.toBeInTheDocument()
		expect(screen.queryByRole('textbox', {name: '仓库地址'})).not.toBeInTheDocument()
		const taskParameter = screen.getByTestId('task-extra-info-parameter-api-info-0')
		expect(getComputedStyle(taskParameter).borderTopStyle).toBe('solid')
		await user.type(screen.getByRole('textbox', {name: '仓库分支'}), 'main')
		expect(screen.queryByRole('button', {name: '新增动态参数'})).not.toBeInTheDocument()
		expect(screen.queryByRole('textbox', {name: '参数键'})).not.toBeInTheDocument()
		expect(screen.queryByRole('textbox', {name: '显示名称'})).not.toBeInTheDocument()
		expect(screen.queryByRole('combobox', {name: '参数类型'})).not.toBeInTheDocument()
		expect(screen.queryByRole('button', {name: /删除动态参数/})).not.toBeInTheDocument()
		await user.selectOptions(screen.getByLabelText('选择分类'), 'issue')
		expect(search).toHaveValue('API')
		expect(screen.queryByRole('checkbox', {name: '缺陷单'})).not.toBeInTheDocument()
		await user.clear(search)
		await user.click(screen.getByRole('checkbox', {name: '缺陷单'}))
		expect(screen.queryByText('git · API 服务')).not.toBeInTheDocument()
		expect(screen.queryByText('issue · 缺陷单')).not.toBeInTheDocument()
		expect(screen.getByText('API 服务')).toBeInTheDocument()
		expect(screen.getAllByText('缺陷单')).not.toHaveLength(0)
		await user.click(screen.getByRole('button', {name: '创建'}))

		await waitFor(() => expect(bindings.CreateTaskWithExtraInfo).toHaveBeenCalledWith('关联 API', '', expect.any(String), expect.arrayContaining([
			expect.objectContaining({
				informationId: 'api-info', templateId: 'git-template', catalogue: 'git',
				parameters: [expect.objectContaining({key: 'branch', value: 'main'})],
			}),
			expect.objectContaining({informationId: 'issue-info', templateId: 'issue-template', catalogue: 'issue'}),
		])))
	})

	it('任务中的复选框动态参数以 true 或 false 保存', async () => {
		const user = userEvent.setup()
		bindings.ListExtraInfoTemplates.mockResolvedValue([{
			id: 'git-template', catalogue: 'git', displayName: 'Git', builtIn: true,
			fields: [{key: 'name', displayName: '项目名称'}, {key: 'repository', displayName: '仓库地址'}],
			parameters: [{key: 'branch', displayName: '仓库分支', required: false, inputType: 'checkbox'}],
		}])
		bindings.ListExtraInfos.mockResolvedValue([{
			id: 'api-info', templateId: 'git-template', catalogue: 'git', fields: [
				{key: 'name', displayName: '项目名称', value: 'API 服务'},
				{key: 'repository', displayName: '仓库地址', value: 'git@example.com:team/api.git'},
			], parameters: [],
		}])
		bindings.CreateTaskWithExtraInfo.mockResolvedValue({
			id: 'task-2', title: '发布 API', description: '', status: 'pending', color: '#4f46e5', createdAt: '2026-07-22T00:00:00Z', extraInfo: [],
		})
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '新建任务'}))
		await user.type(screen.getByRole('textbox', {name: '标题'}), '发布 API')
		await user.click(screen.getByRole('checkbox', {name: 'API 服务'}))
		const branch = screen.getByRole('checkbox', {name: '仓库分支'})
		expect(branch).not.toBeChecked()
		await user.click(branch)
		expect(screen.queryByRole('button', {name: '新增动态参数'})).not.toBeInTheDocument()
		await user.click(screen.getByRole('button', {name: '创建'}))

		await waitFor(() => expect(bindings.CreateTaskWithExtraInfo).toHaveBeenCalledWith('发布 API', '', expect.any(String), expect.arrayContaining([
			expect.objectContaining({
				parameters: expect.arrayContaining([
					expect.objectContaining({key: 'branch', inputType: 'checkbox', required: false, value: 'true'}),
				]),
			}),
		])))
	})

	it('编辑任务时保留已删除信息快照并允许修改动态参数', async () => {
		const user = userEvent.setup()
		bindings.ListTasks.mockResolvedValue([{
			id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
			extraInfo: [{
				id: 'snapshot-old', informationId: 'removed-info', templateId: 'git-template', catalogue: 'git', displayName: '旧 API 仓库',
				fields: [
					{key: 'name', displayName: '项目名称', value: '旧 API 仓库'},
					{key: 'repository', displayName: '仓库地址', value: 'git@example.com:team/old-api.git'},
				],
				parameters: [{key: 'branch', displayName: '仓库分支', required: true, value: 'main'}],
			}],
		}])
		bindings.UpdateTaskWithExtraInfo.mockResolvedValue({
			id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z', extraInfo: [],
		})
		render(<App/>)

		await user.click(await screen.findByRole('tab', {name: /执行中/}))
		await user.click(screen.getByRole('button', {name: '任务操作'}))
		await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
		expect(screen.getByText('旧 API 仓库（已删除）')).toBeInTheDocument()
		expect(screen.queryByRole('textbox', {name: '项目名称'})).not.toBeInTheDocument()
		const branch = screen.getByRole('textbox', {name: '仓库分支'})
		await user.clear(branch)
		await user.type(branch, 'release/1.0')
		await user.click(screen.getByRole('button', {name: '保存'}))

		await waitFor(() => expect(bindings.UpdateTaskWithExtraInfo).toHaveBeenCalledWith('task-1', '清理临时文件', '', '#4f46e5', expect.arrayContaining([
			expect.objectContaining({id: 'snapshot-old', parameters: [expect.objectContaining({key: 'branch', value: 'release/1.0'})]}),
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

  it('暗色模式分隔条使用快门波普描边色', async () => {
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'dark', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
    })
    render(<App/>)

    const divider = await screen.findByRole('separator', {name: '调整任务树宽度'})
    // jsdom 无法计算 Tailwind 的 color-mix 透明度，故不断言计算色，改断言快门波普描边工具类已应用
    expect(divider).toHaveClass('bg-snap-outline/25')
  })

  it('亮色模式根容器标注 data-color-scheme=light 且文档根不附 dark 类', async () => {
    render(<App/>)
    await screen.findByRole('img', {name: '任务 AI 图标'})
    const root = document.querySelector('.taskai-app') as HTMLElement
    expect(root).toHaveAttribute('data-color-scheme', 'light')
    expect(document.documentElement).not.toHaveClass('dark')
  })

  it('暗色模式根容器标注 data-color-scheme=dark 并在文档根附 dark 类以激活暗色令牌', async () => {
    bindings.GetSettings.mockResolvedValue({
      workspaceRoot: '/tmp/workspaces', taskTreeWidth: 360, colorScheme: 'dark', shellPath: '/bin/sh', taskMenuItems: fixedTaskMenuItems,
    })
    render(<App/>)
    await screen.findByRole('img', {name: '任务 AI 图标'})
    const root = document.querySelector('.taskai-app') as HTMLElement
    expect(root).toHaveAttribute('data-color-scheme', 'dark')
    expect(document.documentElement).toHaveClass('dark')
  })

  it('在工作台标题中展示任务 AI 图标', async () => {
    render(<App/>)

    const icon = await screen.findByRole('img', {name: '任务 AI 图标'})
    expect(icon).toHaveClass('taskai-brand-mark')
    expect(icon).toHaveAttribute('src', expect.stringContaining('task-ai-mark.svg'))
  })

  it('工作台样式不包含松林夜跑的斜纹装饰', () => {
    expect(appStyles).not.toContain('repeating-linear-gradient')
    expect(appStyles).not.toContain('#b6e338')
  })

  it('选中任务不显示容易误解为状态的斜纹尾标', () => {
    expect(appStyles).not.toContain('.taskai-task-row[data-task-selected="true"]::after')
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

  it('更新任务启动命令链后立即关闭编辑窗，并按事件展示后续步骤', async () => {
    const user = userEvent.setup()
    let lifecycleEventListener: ((task: TaskRecord) => void) | undefined
    runtime.EventsOn.mockImplementation((eventName, listener) => {
      if (eventName === 'task-lifecycle:event') {
        lifecycleEventListener = listener
      }
    })
    bindings.UpdateTask.mockResolvedValue({
      id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
      lifecycleExecution: {
        runId: 'update-run-1', revision: 1, hook: 'updateTask', chainId: 'update-chain',
        currentCommandId: 'prepare', currentCommandName: '准备环境', currentIndex: 1, commandCount: 2, state: 'running',
      },
    })
    render(<App/>)

    await user.click(await screen.findByRole('tab', {name: /执行中/}))
    await user.click(screen.getByRole('button', {name: '任务操作'}))
    await user.click(screen.getByRole('menuitem', {name: '编辑任务'}))
    await user.click(screen.getByRole('button', {name: '保存'}))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.getByText('更新后 · 准备环境 1/2')).toBeInTheDocument()
    expect(screen.getByRole('button', {name: '任务操作'})).toBeDisabled()
    expect(screen.queryByRole('button', {name: '重试命令链'})).not.toBeInTheDocument()

    if (!lifecycleEventListener) {
      throw new Error('未注册生命周期事件监听器')
    }
    act(() => lifecycleEventListener?.({
      id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
      lifecycleExecution: {
        runId: 'update-run-1', revision: 2, hook: 'updateTask', chainId: 'update-chain',
        currentCommandId: 'deploy', currentCommandName: '部署服务', currentIndex: 2, commandCount: 2, state: 'running',
      },
    }))

    expect(await screen.findByText('更新后 · 部署服务 2/2')).toBeInTheDocument()
  })

  it('开始前命令链完成后自动切换到执行中任务标签', async () => {
    const user = userEvent.setup()
    let lifecycleEventListener: ((task: TaskRecord) => void) | undefined
		let resolveStartTask: ((task: TaskRecord) => void) | undefined
    runtime.EventsOn.mockImplementation((eventName, listener) => {
      if (eventName === 'task-lifecycle:event') {
        lifecycleEventListener = listener
      }
    })
    bindings.ListTasks.mockResolvedValue([{
      id: 'task-1', title: '清理临时文件', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
    }])
    bindings.StartTask.mockImplementation(() => new Promise<TaskRecord>((resolve) => { resolveStartTask = resolve }))
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '执行'}))
		await waitFor(() => expect(bindings.StartTask).toHaveBeenCalledWith('task-1'))

    if (!lifecycleEventListener) {
      throw new Error('未注册生命周期事件监听器')
    }
    act(() => lifecycleEventListener?.({
      id: 'task-1', title: '清理临时文件', description: '', status: 'running', createdAt: '2026-07-22T00:00:00Z',
    }))
		if (!resolveStartTask) {
			throw new Error('开始任务绑定未等待返回')
		}
		const startTaskResolver = resolveStartTask
		startTaskResolver({
			id: 'task-1', title: '清理临时文件', description: '', status: 'pending', createdAt: '2026-07-22T00:00:00Z',
			lifecycleExecution: {
				runId: 'before-start-run', revision: 1, hook: 'beforeStart', chainId: 'start-chain',
				currentCommandId: 'prepare', currentCommandName: '准备环境', currentIndex: 1, commandCount: 1, state: 'running',
			},
		})

    await waitFor(() => expect(screen.getByRole('tab', {name: /执行中/})).toHaveAttribute('aria-selected', 'true'))
    const startedTask = within(screen.getByRole('navigation', {name: '任务和终端'})).getByText('清理临时文件').closest('[data-task-id]')
    expect(startedTask).toHaveAttribute('data-task-start-feedback', 'flash')
    expect(startedTask).not.toHaveTextContent('开始前')
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

		const help = await screen.findByRole('dialog', {name: 'HTTP 状态接口使用说明'})
		expect(help).toHaveTextContent('TASKAI_STATUS_API')
		expect(help).toHaveTextContent('本机 HTTP 服务正在监听时额外获得 API 地址')
		expect(help).toHaveTextContent('无终端后台命令以及前置、后置脚本仅注入 TASKAI_TASK_ID')
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
    const help = await screen.findByRole('dialog', {name: 'HTTP 状态接口使用说明'})
    expect(help).toHaveTextContent('本机 HTTP 服务正在监听时，之后新建的终端会获得 API 地址')
    expect(help).toHaveTextContent('仅本机 HTTP 服务正在监听时注入')
    await user.click(screen.getByRole('button', {name: '关闭'}))
    await waitFor(() => expect(screen.queryByRole('dialog', {name: 'HTTP 状态接口使用说明'})).not.toBeInTheDocument())
    await user.click(screen.getByRole('button', {name: '保存'}))

    await waitFor(() => expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining({
      statusManagementMode: 'title-change', statusManagementHTTPPort: 38562, httpServiceEnabled: true,
    })))
  })

  it('在设置中配置终端输出状态管理并说明静默规则', async () => {
    const user = userEvent.setup()
    render(<App/>)

    await user.click(await screen.findByRole('button', {name: '设置'}))
    await user.click(screen.getByRole('tab', {name: '实时状态'}))
    await user.click(screen.getByLabelText('状态管理方式'))
    await user.click(screen.getByRole('option', {name: '根据终端输出变化'}))

    expect(screen.getByText('任意非空终端输出会在 1.5 秒内显示为工作中，未选中的终端静默后显示为未读。')).toBeInTheDocument()
    expect(screen.getByRole('switch', {name: '启用本机 HTTP 服务'})).not.toBeDisabled()
    await user.click(screen.getByRole('button', {name: '保存'}))

    await waitFor(() => expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining({
      statusManagementMode: 'output-change', httpServiceEnabled: false,
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

  it('右键新增终端时保留创建返回前到达的标题并直接路由输出', async () => {
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

    expect(terminalSessionRegistry.handleTerminalEvent).toHaveBeenCalledWith({taskId: 'task-1', terminalId: 'terminal-1', type: 'output', data: output})

    resolveTerminal!({id: 'terminal-1', taskId: 'task-1', state: 'active'})

    await waitFor(() => expect(screen.getAllByText('启动标题')).toHaveLength(2))
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
    await user.click(screen.getByRole('button', {name: '结束任务'}))
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
    await user.click(screen.getByRole('button', {name: '结束任务'}))
    await waitFor(() => expect(bindings.FinishTask).toHaveBeenCalledWith('task-1'))

    await act(async () => {
      resolveCommand!({terminal: {id: 'terminal-codex', taskId: 'task-1', state: 'active'}})
      await Promise.resolve()
    })

    expect(screen.queryByText('终端视图')).not.toBeInTheDocument()
  })

  it('显示终端的自定义命令时保留创建返回前到达的标题并直接路由输出', async () => {
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

    expect(terminalSessionRegistry.handleTerminalEvent).toHaveBeenCalledWith({taskId: 'task-1', terminalId: 'terminal-codex', type: 'output', data: output})

    resolveCommand!({terminal: {id: 'terminal-codex', taskId: 'task-1', state: 'active'}})

    await waitFor(() => expect(screen.getAllByText('命令标题')).toHaveLength(2))
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
    await user.click(screen.getByRole('tab', {name: '菜单管理'}))
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
  }, 10_000)

	it('将任务菜单配置显示为菜单管理', async () => {
		const user = userEvent.setup()
		render(<App/>)

		await user.click(await screen.findByRole('button', {name: '设置'}))
		expect(screen.queryByRole('tab', {name: '任务操作'})).not.toBeInTheDocument()
		await user.click(screen.getByRole('tab', {name: '菜单管理'}))
		expect(screen.getByText('右键菜单与“任务操作”下拉菜单共用此顺序。系统项仅可调序。')).toBeInTheDocument()
	})
})
