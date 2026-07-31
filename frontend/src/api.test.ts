import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'

import {api} from './api'
import type {SettingsRecord} from './types'

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
			lifecycleDefaultChains: {beforeStart: 'prepare-chain'},
		}

		await api.saveSettings(settings)

		const payload = saveSettings.mock.calls[0][0] as Record<string, unknown>
		expect(payload.workspaceRoot).toBe(settings.workspaceRoot)
		expect(payload).not.toHaveProperty('lifecycleCommands')
		expect(payload).not.toHaveProperty('lifecycleChains')
		expect(payload).not.toHaveProperty('lifecycleDefaultChains')
	})
})
