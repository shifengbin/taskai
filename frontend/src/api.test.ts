import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

import {api} from './api'
import type {LifecyclePreset, SettingsRecord} from './types'

describe('api.saveSettings', () => {
	const saveSettings = vi.fn()

	beforeEach(() => {
		vi.clearAllMocks()
		saveSettings.mockResolvedValue({})
		Object.assign(window, {go: {main: {App: {SaveSettings: saveSettings}}}})
	})

	afterEach(() => {
		delete (window as {go?: unknown}).go
	})

	it('不向普通设置保存传递生命周期配置', async () => {
		const settings: SettingsRecord = {
			workspaceRoot: '/tmp/workspaces',
			taskTreeWidth: 360,
			colorScheme: 'light',
			shellPath: '/bin/sh',
			taskMenuItems: [],
			activeTaskStatus: 'pending',
			statusManagementMode: 'title-change',
			statusManagementHTTPPort: 0,
			httpServiceEnabled: false,
			lifecycleCommands: [{id: 'prepare', kind: 'custom', name: '准备', arguments: [], chainArgumentMode: 'disabled', applicableHooks: ['beforeStart']}],
			lifecycleChains: [{id: 'prepare-chain', name: '准备链', commands: [{commandId: 'prepare', arguments: []}], applicableHooks: ['beforeStart']}],
			lifecyclePresets: [{id: 'prepare', name: '准备预设', chains: {beforeStart: 'prepare-chain'}}],
			defaultLifecyclePresetId: 'prepare',
		}

		await api.saveSettings(settings)

		const payload = saveSettings.mock.calls[0][0] as Record<string, unknown>
		expect(payload.workspaceRoot).toBe(settings.workspaceRoot)
		expect(payload).not.toHaveProperty('lifecycleCommands')
		expect(payload).not.toHaveProperty('lifecycleChains')
		expect(payload).not.toHaveProperty('lifecyclePresets')
		expect(payload).not.toHaveProperty('defaultLifecyclePresetId')
		expect(payload).not.toHaveProperty('taskTemplates')
		expect(payload).not.toHaveProperty('activeTaskTemplateId')
	})

	it('当前模板标识缺少模板集合时不伪造空模板快照', async () => {
		const settings: SettingsRecord = {
			workspaceRoot: '/tmp/workspaces',
			taskTreeWidth: 360,
			colorScheme: 'light',
			shellPath: '/bin/sh',
			taskMenuItems: [],
			activeTaskStatus: 'pending',
			statusManagementMode: 'title-change',
			statusManagementHTTPPort: 0,
			httpServiceEnabled: false,
			activeTaskTemplateId: '',
		}

		await api.saveSettings(settings)

		const payload = saveSettings.mock.calls[0][0] as Record<string, unknown>
		expect(payload.taskTemplates).toBeUndefined()
		expect(payload.activeTaskTemplateId).toBeUndefined()
	})

	it('为缺少主题的旧设置发送完整暗色终端主题', async () => {
		const settings: SettingsRecord = {
			workspaceRoot: '/tmp/workspaces',
			taskTreeWidth: 360,
			colorScheme: 'light',
			shellPath: '/bin/sh',
			taskMenuItems: [],
			activeTaskStatus: 'pending',
			statusManagementMode: 'title-change',
			statusManagementHTTPPort: 0,
			httpServiceEnabled: false,
		}

		await api.saveSettings(settings)

		expect(saveSettings.mock.calls[0][0]).toMatchObject({
			terminalTheme: {
				background: '#070A16',
				foreground: '#E8ECFF',
				brightWhite: '#E8ECFF',
			},
		})
	})
})

describe('api.lifecyclePresets', () => {
	const listLifecyclePresets = vi.fn()
	const saveLifecyclePreset = vi.fn()
	const copyLifecyclePreset = vi.fn()
	const deleteLifecyclePreset = vi.fn()
	const saveDefaultLifecyclePreset = vi.fn()

	beforeEach(() => {
		vi.clearAllMocks()
		listLifecyclePresets.mockResolvedValue([])
		saveLifecyclePreset.mockImplementation(async (preset) => preset)
		copyLifecyclePreset.mockResolvedValue({id: 'preset-copy', name: '部署预设 副本', chains: {}})
		deleteLifecyclePreset.mockResolvedValue(undefined)
		saveDefaultLifecyclePreset.mockResolvedValue({defaultLifecyclePresetId: 'preset-1'})
		Object.assign(window, {go: {main: {App: {
			ListLifecyclePresets: listLifecyclePresets,
			SaveLifecyclePreset: saveLifecyclePreset,
			CopyLifecyclePreset: copyLifecyclePreset,
			DeleteLifecyclePreset: deleteLifecyclePreset,
			SaveDefaultLifecyclePreset: saveDefaultLifecyclePreset,
		}}}})
	})

	afterEach(() => {
		delete (window as {go?: unknown}).go
	})

	it('通过专用绑定管理生命周期预设', async () => {
		const preset: LifecyclePreset = {id: 'preset-1', name: '部署预设', chains: {beforeStart: 'prepare-chain'}}

		await api.listLifecyclePresets()
		await api.saveLifecyclePreset(preset)
		await api.copyLifecyclePreset(preset.id)
		await api.deleteLifecyclePreset(preset.id)
		await api.saveDefaultLifecyclePreset(preset.id)

		expect(listLifecyclePresets).toHaveBeenCalledOnce()
		expect(saveLifecyclePreset).toHaveBeenCalledWith(preset)
		expect(copyLifecyclePreset).toHaveBeenCalledWith(preset.id)
		expect(deleteLifecyclePreset).toHaveBeenCalledWith(preset.id)
		expect(saveDefaultLifecyclePreset).toHaveBeenCalledWith(preset.id)
	})
})

describe('api.deleteTasks', () => {
	const deleteTasks = vi.fn()

	beforeEach(() => {
		vi.clearAllMocks()
		deleteTasks.mockResolvedValue([])
		Object.assign(window, {go: {main: {App: {DeleteTasks: deleteTasks}}}})
	})

	afterEach(() => {
		delete (window as {go?: unknown}).go
	})

	it('转发全部待删除任务 ID', async () => {
		const deleteTaskRecords = (api as typeof api & {deleteTasks?: (taskIDs: string[]) => Promise<unknown>}).deleteTasks

		expect(deleteTaskRecords).toEqual(expect.any(Function))
		await deleteTaskRecords?.(['pending-1', 'completed-1'])

		expect(deleteTasks).toHaveBeenCalledWith(['pending-1', 'completed-1'])
	})
})
