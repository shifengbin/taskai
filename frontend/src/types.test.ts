import {describe, expect, it} from 'vitest'

import {settings as wailsSettings, task as wailsTask} from '../wailsjs/go/models'
import {clampTaskTreeWidth, defaultTaskMenuItems} from './types'

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

	it('Wails 模型转换生命周期预设和默认预设标识', () => {
		const settings = wailsSettings.Settings.createFrom({
			lifecyclePresets: [{id: 'deploy', name: '部署', chains: {beforeStart: 'prepare'}}],
			defaultLifecyclePresetId: 'deploy',
		})

		expect(settings.lifecyclePresets).toHaveLength(1)
		expect(settings.lifecyclePresets[0].chains).toEqual({beforeStart: 'prepare'})
		expect(settings.defaultLifecyclePresetId).toBe('deploy')
	})
})
