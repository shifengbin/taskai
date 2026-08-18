import {describe, expect, it} from 'vitest'

import {settings as wailsSettings, task as wailsTask, terminal as wailsTerminal} from '../wailsjs/go/models'
import {clampTaskTreeWidth, defaultTaskMenuItems, terminalActualName, terminalAliasDetails, terminalDisplayName} from './types'

describe('clampTaskTreeWidth', () => {
  it('将拖拽产生的小数宽度归一化为整数', () => {
    expect(clampTaskTreeWidth(311.25390625, 1280)).toBe(311)
  })

  it('默认任务菜单包含具有两种名称的搁置切换', () => {
    expect(defaultTaskMenuItems).toContainEqual({
      id: 'system.toggle-shelved',
      kind: 'toggle-shelved',
      name: '搁置任务',
      unshelveName: '取消搁置',
      showTerminal: false,
    })
  })

  it('保留生命周期命令引用参数和系统使用说明', () => {
    const command = wailsSettings.LifecycleCommand.createFrom({
      id: 'system.lifecycle.git-clone',
      kind: 'git-clone',
      name: 'Git 仓库克隆',
      arguments: [],
			chainArgumentMode: 'enabled',
			documentation: '参数可留空；填写时使用 dir=<相对目录>',
      applicableHooks: ['beforeStart', 'beforeEnd', 'updateTask'],
    })
    const chain = wailsSettings.LifecycleCommandChain.createFrom({
      id: 'clone-repositories',
      name: '克隆仓库',
      commands: [{commandId: command.id, arguments: ['dir=repositories']}],
      applicableHooks: ['beforeStart'],
    })

		expect(command.documentation).toContain('参数可留空')
		expect(command.documentation).toContain('dir=<相对目录>')
		expect(command.chainArgumentMode).toBe('enabled')
		expect(chain.commands).toEqual([{commandId: 'system.lifecycle.git-clone', arguments: ['dir=repositories']}])
  })

	it('Wails 模型保留任务模板定义和任务字段原生值', () => {
		const settings = wailsSettings.Settings.createFrom({
			taskTemplates: [{
				id: 'release', name: '发布任务', fields: [
					{key: 'environment', displayName: '环境', inputType: 'string', required: true, defaultValue: 'production', injectEnvironment: true},
					{key: 'deploy', displayName: '立即部署', inputType: 'bool', required: false, defaultValue: false, injectEnvironment: false},
				],
			}],
			activeTaskTemplateId: 'release',
		})
		const task = wailsTask.Task.createFrom({id: 'task-1', templateFields: {environment: 'production', deploy: true}})

		expect(settings.activeTaskTemplateId).toBe('release')
		expect(settings.taskTemplates[0].fields[1].defaultValue).toBe(false)
		expect(task.templateFields).toEqual({environment: 'production', deploy: true})
	})

	it('Wails 模型保留目录字段约束和独立目录数组', () => {
		const settings = wailsSettings.Settings.createFrom({
			taskTemplates: [{id: 'directories', name: '项目目录', fields: [{
				key: 'sources', displayName: '源码目录', inputType: 'directories', required: true,
				defaultValue: [], injectEnvironment: false, multiple: true, updatable: false,
			}]}],
		})
		const task = wailsTask.Task.createFrom({id: 'task-1', templateFields: {sources: ['/tmp/project-a/src', '/tmp/project-b/src']}})

		expect(settings.taskTemplates[0].fields[0].multiple).toBe(true)
		expect(settings.taskTemplates[0].fields[0].updatable).toBe(false)
		expect(task.templateFields.sources).toEqual(['/tmp/project-a/src', '/tmp/project-b/src'])
	})

	it('Wails 模型转换生命周期预设和默认预设标识', () => {
		const settings = wailsSettings.Settings.createFrom({
			lifecyclePresets: [{id: 'deploy', name: '部署', chains: {beforeStart: 'prepare'}}],
			defaultLifecyclePresetId: 'deploy',
		})

		expect(settings.lifecyclePresets).toHaveLength(1)
		expect(settings.lifecyclePresets[0].chains).toEqual({beforeStart: 'prepare'})
		expect(settings.defaultLifecyclePresetId).toBe('deploy')
	})

	it('Wails 模型保留开始前任务目录归属快照', () => {
		const execution = wailsTask.LifecycleExecution.createFrom({
			hook: 'beforeStart', chainId: 'prepare', currentIndex: 1, commandCount: 2, state: 'failed',
			workspaceRoot: '/tmp/workspaces', workspacePath: '/tmp/workspaces/task-1', workspaceOwnership: 'created',
			workspaceToken: 'ownership-token',
		})

		expect(execution.workspaceRoot).toBe('/tmp/workspaces')
		expect(execution.workspacePath).toBe('/tmp/workspaces/task-1')
		expect(execution.workspaceOwnership).toBe('created')
		expect(execution.workspaceToken).toBe('ownership-token')
	})

	it('Wails 模型转换终端备注模板', () => {
		const settings = wailsSettings.Settings.createFrom({
			terminalNoteTemplate: {originalPrefix: '原文：', notePrefix: '备注：', listSuffix: '请处理'},
		})

		expect(settings.terminalNoteTemplate).toEqual({originalPrefix: '原文：', notePrefix: '备注：', listSuffix: '请处理'})
	})

	it('Wails 菜单和终端模型忽略已删除的鼠标策略字段', () => {
		const menuItem = wailsSettings.TaskMenuItem.createFrom({
			id: 'legacy', kind: 'command', name: 'Legacy', command: 'legacy', showTerminal: true, disableTaskAIMouseClipboard: true,
		} as never)
		const terminal = wailsTerminal.Info.createFrom({
			id: 'terminal-1', taskId: 'task-1', state: 'active', disableTaskAIMouseClipboard: true,
		} as never)

		expect(menuItem).not.toHaveProperty('disableTaskAIMouseClipboard')
		expect(terminal).not.toHaveProperty('disableTaskAIMouseClipboard')
	})

	it('别名优先显示，空白别名恢复实际终端名称', () => {
		const terminal = {title: ' npm run dev ', alias: ' 前端调试 '}

		expect(terminalActualName(terminal)).toBe('npm run dev')
		expect(terminalDisplayName(terminal)).toBe('前端调试')
		expect(terminalDisplayName({...terminal, alias: '  '})).toBe('npm run dev')
		expect(terminalActualName({})).toBe('终端')
	})

	it('别名提示始终显示实际名称和启动命令', () => {
		expect(terminalAliasDetails({title: '当前标题', command: ' zsh '})).toEqual({
			actualName: '当前标题',
			command: 'zsh',
		})
		expect(terminalAliasDetails({})).toEqual({
			actualName: '终端',
			command: '未提供启动命令',
		})
	})
})
