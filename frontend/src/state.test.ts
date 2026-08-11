import {describe, expect, it} from 'vitest'

import {
	applyRealtimeStatusEvent,
	applyTerminalEvent,
  bufferPendingRealtimeStatusEvent,
  bufferPendingTerminalEvent,
  clearTaskTerminalTracking,
	mergeLifecycleTask,
  mergePendingTerminalEvents,
  parseTerminalEventTitle,
	registerTerminal,
	shouldReportTerminalTitleActivity,
  terminalEventKey,
	updateTerminalAlias,
} from './state'
import {clampTaskTreeWidth, type TaskRecord, type TerminalRecord} from './types'

describe('终端状态路由', () => {
	 it('保留同一命令链运行批次中版本更高的进度，避免旧绑定返回覆盖事件', () => {
		const current: TaskRecord[] = [{
			id: 'task-a', title: '发布服务', description: '', status: 'running', createdAt: '2026-07-30T00:00:00Z',
			lifecycleExecution: {
				runId: 'run-1', revision: 2, hook: 'updateTask', chainId: 'update-chain',
				currentCommandId: 'deploy', currentCommandName: '部署服务', currentIndex: 2, commandCount: 3, state: 'running',
			},
		}]
		const staleBindingResult: TaskRecord = {
			...current[0],
			lifecycleExecution: {
				runId: 'run-1', revision: 1, hook: 'updateTask', chainId: 'update-chain',
				currentCommandId: 'prepare', currentCommandName: '准备环境', currentIndex: 1, commandCount: 3, state: 'running',
			},
		}

		expect(mergeLifecycleTask(current, staleBindingResult)[0].lifecycleExecution).toMatchObject({
			runId: 'run-1', revision: 2, currentCommandName: '部署服务', currentIndex: 2,
		})
	})

	it('命令链已清空后仍忽略同一运行的延迟快照', () => {
		const current: TaskRecord[] = [{
			id: 'task-a', title: '发布服务', description: '', status: 'running', createdAt: '2026-07-30T00:00:00Z',
			lifecycleExecution: {
				runId: 'run-1', revision: 2, hook: 'updateTask', chainId: 'update-chain',
				currentCommandId: 'deploy', currentCommandName: '部署服务', currentIndex: 2, commandCount: 2, state: 'running',
			},
		}]
		const cleared = mergeLifecycleTask(current, {...current[0], lifecycleExecution: undefined})
		const staleBindingResult: TaskRecord = {
			...current[0],
			lifecycleExecution: {
				runId: 'run-1', revision: 1, hook: 'updateTask', chainId: 'update-chain',
				currentCommandId: 'prepare', currentCommandName: '准备环境', currentIndex: 1, commandCount: 2, state: 'running',
			},
		}

		expect(mergeLifecycleTask(cleared, staleBindingResult)[0].lifecycleExecution).toBeUndefined()
	})

	it('新的重试运行可以替换旧运行的失败快照', () => {
		const current: TaskRecord[] = [{
			id: 'task-a', title: '发布服务', description: '', status: 'running', createdAt: '2026-07-30T00:00:00Z',
			lifecycleExecution: {
				runId: 'failed-run', revision: 2, hook: 'updateTask', chainId: 'update-chain',
				currentCommandId: 'deploy', currentCommandName: '部署服务', currentIndex: 2, commandCount: 2, state: 'failed', error: '退出码 1',
			},
		}]
		const retrying: TaskRecord = {
			...current[0],
			lifecycleExecution: {
				runId: 'retry-run', revision: 1, hook: 'updateTask', chainId: 'update-chain',
				currentCommandId: 'prepare', currentCommandName: '准备环境', currentIndex: 1, commandCount: 2, state: 'running',
			},
		}

		expect(mergeLifecycleTask(current, retrying)[0].lifecycleExecution).toMatchObject({
			runId: 'retry-run', state: 'running', currentCommandName: '准备环境',
		})
	})

  it('输出事件不累积到 React 终端记录，退出状态只应用到对应终端', () => {
    const terminals: TerminalRecord[] = [
      {id: 'one', taskId: 'task-a', state: 'active'},
      {id: 'two', taskId: 'task-b', state: 'active'},
    ]
    const afterOutput = applyTerminalEvent(terminals, {
      taskId: 'task-a', terminalId: 'one', type: 'output', data: 'hello',
    })
    const afterExit = applyTerminalEvent(afterOutput, {taskId: 'task-b', terminalId: 'two', type: 'exited'})

    expect(afterExit).toEqual([
      {id: 'one', taskId: 'task-a', state: 'active'},
      {id: 'two', taskId: 'task-b', state: 'exited'},
    ])
  })

  it('只为收到输出的对应终端更新实时标题', () => {
    const terminals: TerminalRecord[] = [
      {id: 'one', taskId: 'task-a', state: 'active', title: '旧标题'},
      {id: 'two', taskId: 'task-b', state: 'active', title: '其他标题'},
    ]

    const updated = applyTerminalEvent(terminals, {
      taskId: 'task-a', terminalId: 'one', type: 'output', data: '新增输出',
    }, '实时标题')

    expect(updated).toEqual([
      {id: 'one', taskId: 'task-a', state: 'active', title: '实时标题'},
      {id: 'two', taskId: 'task-b', state: 'active', title: '其他标题'},
    ])
  })

  it('实时标题更新保留别名，但终端退出后清除别名', () => {
    const terminals: TerminalRecord[] = [
      {id: 'one', taskId: 'task-a', state: 'active', title: '初始标题', alias: '前端调试'},
    ]

    const afterTitle = applyTerminalEvent(terminals, {
      taskId: 'task-a', terminalId: 'one', type: 'output', data: '新增输出',
    }, '实时标题')
    const afterExit = applyTerminalEvent(afterTitle, {taskId: 'task-a', terminalId: 'one', type: 'exited'})

    expect(afterTitle[0]).toMatchObject({title: '实时标题', alias: '前端调试'})
    expect(afterExit[0]).toEqual({id: 'one', taskId: 'task-a', state: 'exited', title: '实时标题'})
  })

  it('将关闭异常终端的实时状态事件投影到对应任务和终端', () => {
    const tasks: TaskRecord[] = [{
      id: 'task-a', title: '任务 A', description: '', status: 'running', createdAt: '2026-07-27T00:00:00Z',
    }]
    const terminals: TerminalRecord[] = [{id: 'terminal-a', taskId: 'task-a', state: 'exited'}]

    const updated = applyRealtimeStatusEvent(tasks, terminals, {
      version: 3,
      taskId: 'task-a',
      taskStatus: 'error',
      terminalId: 'terminal-a',
      terminalStatus: 'error',
    })

    expect(updated.tasks[0].realtimeStatus).toBe('error')
    expect(updated.terminals[0].realtimeStatus).toBe('error')

    const removed = applyRealtimeStatusEvent(updated.tasks, updated.terminals, {
      version: 4,
      taskId: 'task-a',
      taskStatus: 'idle',
      terminalId: 'terminal-a',
      terminalRemoved: true,
    })
    expect(removed.tasks[0].realtimeStatus).toBe('idle')
    expect(removed.terminals).toEqual([])
  })

  it('只在终端标题的值实际变化时上报活动', () => {
    const terminal: TerminalRecord = {id: 'terminal-a', taskId: 'task-a', state: 'active', title: '旧标题'}

    expect(shouldReportTerminalTitleActivity(terminal, '旧标题')).toBe(false)
    expect(shouldReportTerminalTitleActivity(terminal, '新标题')).toBe(true)
  })

  it('更新别名不会改写实际标题或实时状态', () => {
    const terminals: TerminalRecord[] = [
      {id: 'terminal-a', taskId: 'task-a', state: 'active', title: '实际标题', realtimeStatus: 'idle'},
      {id: 'terminal-b', taskId: 'task-a', state: 'active', title: '其他标题', realtimeStatus: 'working'},
    ]

    const renamed = updateTerminalAlias(terminals, 'task-a', 'terminal-a', ' 前端调试 ')

    expect(renamed).toEqual([
      {id: 'terminal-a', taskId: 'task-a', state: 'active', title: '实际标题', alias: '前端调试', realtimeStatus: 'idle'},
      terminals[1],
    ])
    expect(shouldReportTerminalTitleActivity(renamed[0], '实际标题')).toBe(false)
    expect(shouldReportTerminalTitleActivity(renamed[0], '新实际标题')).toBe(true)
    expect(updateTerminalAlias(renamed, 'task-a', 'terminal-a', '  ')[0]).not.toHaveProperty('alias')
  })

  it('终端退出后清理其未完成标题的解析状态', () => {
    const parserStates = new Map()
    const terminal = {taskId: 'task-a', terminalId: 'terminal-1'}

    expect(parseTerminalEventTitle(parserStates, {...terminal, type: 'output', data: '\x1b]0;构建'})).toBeUndefined()
    expect(parserStates.size).toBe(1)

    parseTerminalEventTitle(parserStates, {...terminal, type: 'exited'})

    expect(parserStates.size).toBe(0)
    expect(parseTerminalEventTitle(parserStates, {...terminal, type: 'output', data: '完成\x07'})).toBeUndefined()
  })

  it('合并创建前缓存的最新标题和退出状态后清理缓存', () => {
    const pendingEvents = new Map()
    const terminal = {id: 'terminal-1', taskId: 'task-a', state: 'active' as const}

    bufferPendingTerminalEvent(pendingEvents, {...terminal, terminalId: terminal.id, type: 'output', data: '初始输出'}, '初始标题')
    bufferPendingTerminalEvent(pendingEvents, {...terminal, terminalId: terminal.id, type: 'output', data: '后续输出'}, '最新标题')
    bufferPendingTerminalEvent(pendingEvents, {...terminal, terminalId: terminal.id, type: 'exited'})

    expect(mergePendingTerminalEvents(pendingEvents, terminal)).toEqual({
      ...terminal,
      title: '最新标题',
      state: 'exited',
    })
    expect(pendingEvents.size).toBe(0)
  })

  it('合并终端创建前到达的实时状态事件', () => {
    const pendingEvents = new Map()
    const terminal: TerminalRecord = {id: 'terminal-1', taskId: 'task-a', state: 'active'}

    bufferPendingRealtimeStatusEvent(pendingEvents, {
      version: 1, taskId: terminal.taskId, taskStatus: 'unread', terminalId: terminal.id, terminalStatus: 'unread',
    })

    expect(mergePendingTerminalEvents(pendingEvents, terminal)).toEqual({...terminal, realtimeStatus: 'unread'})
  })

  it('合并待处理事件时保留终端启动命令以支持快捷键生效范围', () => {
    const pendingEvents = new Map()
    const terminal: TerminalRecord = {id: 'terminal-1', taskId: 'task-a', state: 'active', command: 'codex'}

    bufferPendingTerminalEvent(pendingEvents, {...terminal, terminalId: terminal.id, type: 'output', data: '输出'}, '终端标题')

    expect(mergePendingTerminalEvents(pendingEvents, terminal)).toEqual({
      ...terminal,
      title: '终端标题',
      command: 'codex',
    })
  })

  it('登记已退出的终端，避免后续事件重新成为创建前缓存', () => {
    const registeredTerminalKeys = new Set<string>()
    const terminal = {id: 'terminal-1', taskId: 'task-a', state: 'exited' as const}

    registerTerminal(registeredTerminalKeys, terminal)

    expect(registeredTerminalKeys).toEqual(new Set([terminalEventKey('task-a', 'terminal-1')]))
  })

  it('结束任务时仅清理该任务的解析、缓存和登记状态', () => {
    const parserStates = new Map()
    const pendingEvents = new Map()
    const registeredTerminalKeys = new Set<string>()
    const taskATerminal = {taskId: 'task-a', terminalId: 'terminal-a'}
    const taskBTerminal = {taskId: 'task-b', terminalId: 'terminal-b'}

    parseTerminalEventTitle(parserStates, {...taskATerminal, type: 'output', data: '\x1b]2;任务 A'})
    parseTerminalEventTitle(parserStates, {...taskBTerminal, type: 'output', data: '\x1b]2;任务 B'})
    bufferPendingTerminalEvent(pendingEvents, {...taskATerminal, type: 'output', data: '任务 A 输出'}, '任务 A')
    bufferPendingTerminalEvent(pendingEvents, {...taskBTerminal, type: 'output', data: '任务 B 输出'}, '任务 B')
    registerTerminal(registeredTerminalKeys, {id: 'terminal-a', taskId: 'task-a', state: 'active'})
    registerTerminal(registeredTerminalKeys, {id: 'terminal-b', taskId: 'task-b', state: 'exited'})

    clearTaskTerminalTracking('task-a', parserStates, pendingEvents, registeredTerminalKeys)

    expect([...parserStates.keys()]).toEqual([terminalEventKey('task-b', 'terminal-b')])
    expect([...pendingEvents.keys()]).toEqual([terminalEventKey('task-b', 'terminal-b')])
    expect(registeredTerminalKeys).toEqual(new Set([terminalEventKey('task-b', 'terminal-b')]))
  })

  it('拖拽宽度遵循最小值和右侧最小空间', () => {
    expect(clampTaskTreeWidth(120, 1000)).toBe(280)
    expect(clampTaskTreeWidth(900, 1000)).toBe(640)
  })
})
