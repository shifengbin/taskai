import {act, cleanup, render, screen, waitFor, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {afterEach, describe, expect, it, vi} from 'vitest'

import {UpdateControl, type UpdateClient} from './UpdateControl'
import type {UpdateState} from '../types'
import {TooltipProvider} from './ui'

afterEach(cleanup)

describe('UpdateControl state synchronization', () => {
	it('queries the current state before subscribing and cleans up on unmount', async () => {
		let resolveState: ((state: UpdateState) => void) | undefined
		const initialState = new Promise<UpdateState>((resolve) => { resolveState = resolve })
		const unsubscribe = vi.fn()
		const client = updateClient({
			getUpdateState: vi.fn(() => initialState),
			onUpdateState: vi.fn(() => unsubscribe),
		})

		const view = renderUpdateControl(client)
		expect(client.getUpdateState).toHaveBeenCalledOnce()
		expect(client.onUpdateState).not.toHaveBeenCalled()

		resolveState?.({status: 'available', version: 'v1.2.3', releaseUrl: 'https://github.com/shifengbin/taskai/releases/tag/v1.2.3'})
		expect(await screen.findByRole('button', {name: '下载新版本 v1.2.3'})).toHaveTextContent('new')
		expect(client.onUpdateState).toHaveBeenCalledOnce()

		view.unmount()
		expect(unsubscribe).toHaveBeenCalledOnce()
	})

	it.each([
		{state: {status: 'idle'} as UpdateState, label: undefined, text: undefined, disabled: false},
		{state: {status: 'available', version: 'v1.2.3'} as UpdateState, label: '下载新版本 v1.2.3', text: 'new', disabled: false},
		{state: {status: 'downloading', version: 'v1.2.3'} as UpdateState, label: '正在下载 v1.2.3', text: '下载中', disabled: true},
		{state: {status: 'download_failed', version: 'v1.2.3'} as UpdateState, label: '下载失败 v1.2.3', text: '下载失败', disabled: false},
		{state: {status: 'downloaded', version: 'v1.2.3'} as UpdateState, label: '安装已下载版本 v1.2.3', text: '已下载', disabled: false},
	])('renders $state.status without resizing the header control', async ({state, label, text, disabled}) => {
		const client = updateClient({getUpdateState: vi.fn(async () => state)})
		renderUpdateControl(client)

		if (!label) {
			await waitFor(() => expect(client.onUpdateState).toHaveBeenCalledOnce())
			expect(screen.queryByTestId('update-control')).not.toBeInTheDocument()
			return
		}
		const button = await screen.findByRole('button', {name: label})
		expect(button).toHaveTextContent(text ?? '')
		expect(button).toHaveClass('w-[86px]')
		if (disabled) {
			expect(button).toBeDisabled()
		} else {
			expect(button).toBeEnabled()
		}
	})
})

describe('UpdateControl download interactions', () => {
	it('keeps the failed state while offering cancel, retry, and manual download', async () => {
		const user = userEvent.setup()
		const client = updateClient({
			getUpdateState: vi.fn(async () => updateState({status: 'download_failed', version: 'v1.2.3', error: 'network unavailable'})),
		})
		renderUpdateControl(client)

		await user.click(await screen.findByRole('button', {name: '下载失败 v1.2.3'}))
		let dialog = screen.getByRole('dialog', {name: '更新下载失败'})
		expect(within(dialog).getByText('network unavailable')).toBeInTheDocument()
		await user.click(within(dialog).getByRole('button', {name: '取消'}))
		expect(screen.queryByRole('dialog', {name: '更新下载失败'})).not.toBeInTheDocument()
		expect(screen.getByRole('button', {name: '下载失败 v1.2.3'})).toBeEnabled()

		await user.click(screen.getByRole('button', {name: '下载失败 v1.2.3'}))
		dialog = screen.getByRole('dialog', {name: '更新下载失败'})
		await user.click(within(dialog).getByRole('button', {name: '重新下载'}))
		expect(client.startUpdateDownload).toHaveBeenCalledOnce()
		expect(await screen.findByRole('button', {name: '正在下载 v1.2.3'})).toBeDisabled()
	})

	it('opens the exact release page and reports browser launch errors without clearing failure state', async () => {
		const user = userEvent.setup()
		const onError = vi.fn()
		const client = updateClient({
			getUpdateState: vi.fn(async () => updateState({status: 'download_failed', version: 'v1.2.3'})),
			openUpdateReleasePage: vi.fn(async () => { throw new Error('browser unavailable') }),
		})
		renderUpdateControl(client, onError)

		await user.click(await screen.findByRole('button', {name: '下载失败 v1.2.3'}))
		const dialog = screen.getByRole('dialog', {name: '更新下载失败'})
		await user.click(within(dialog).getByRole('button', {name: '手动下载'}))
		await waitFor(() => expect(onError).toHaveBeenCalledWith('browser unavailable'))
		expect(client.openUpdateReleasePage).toHaveBeenCalledOnce()
		expect(screen.getByRole('dialog', {name: '更新下载失败'})).toBeInTheDocument()
		await user.click(within(screen.getByRole('dialog', {name: '更新下载失败'})).getByRole('button', {name: '取消'}))
		expect(screen.getByRole('button', {name: '下载失败 v1.2.3'})).toBeEnabled()
	})

	it('opens installation confirmation on download completion and reopens it after choosing later', async () => {
		const user = userEvent.setup()
		let listener: ((state: UpdateState) => void) | undefined
		const client = updateClient({
			getUpdateState: vi.fn(async () => updateState({status: 'available', version: 'v1.2.3'})),
			onUpdateState: vi.fn((next) => {
				listener = next
				return () => undefined
			}),
		})
		renderUpdateControl(client)
		await waitFor(() => expect(listener).toBeTypeOf('function'))

		act(() => listener?.({status: 'downloaded', version: 'v1.2.3', installPath: '/tmp/taskai.deb'}))
		let dialog = await screen.findByRole('dialog', {name: '安装更新 v1.2.3'})
		await user.click(within(dialog).getByRole('button', {name: '稍后安装'}))
		expect(screen.queryByRole('dialog', {name: '安装更新 v1.2.3'})).not.toBeInTheDocument()

		await user.click(screen.getByRole('button', {name: '安装已下载版本 v1.2.3'}))
		dialog = screen.getByRole('dialog', {name: '安装更新 v1.2.3'})
		expect(within(dialog).getByRole('button', {name: '立即安装'})).toBeEnabled()
	})
})

describe('UpdateControl installation interactions', () => {
	it('launches the installer before preparing and quitting when no task is running', async () => {
		const user = userEvent.setup()
		const calls: string[] = []
		const client = updateClient({
			getUpdateState: vi.fn(async () => updateState({status: 'downloaded', version: 'v1.2.3'})),
			hasRunningTasks: vi.fn(async () => { calls.push('has-running-tasks'); return false }),
			launchDownloadedUpdate: vi.fn(async () => { calls.push('launch-installer') }),
			prepareQuit: vi.fn(async () => { calls.push('prepare-quit') }),
			quit: vi.fn(() => { calls.push('quit') }),
		})
		renderUpdateControl(client)

		await user.click(await screen.findByRole('button', {name: '安装已下载版本 v1.2.3'}))
		await user.click(within(screen.getByRole('dialog', {name: '安装更新 v1.2.3'})).getByRole('button', {name: '立即安装'}))
		await waitFor(() => expect(calls).toEqual(['has-running-tasks', 'launch-installer', 'prepare-quit', 'quit']))
	})

	it('requires explicit confirmation for running tasks and preserves the downloaded state on cancel', async () => {
		const user = userEvent.setup()
		const calls: string[] = []
		const client = updateClient({
			getUpdateState: vi.fn(async () => updateState({status: 'downloaded', version: 'v1.2.3'})),
			hasRunningTasks: vi.fn(async () => true),
			launchDownloadedUpdate: vi.fn(async () => { calls.push('launch-installer') }),
			prepareQuit: vi.fn(async () => { calls.push('prepare-quit') }),
			quit: vi.fn(() => { calls.push('quit') }),
		})
		renderUpdateControl(client)

		await user.click(await screen.findByRole('button', {name: '安装已下载版本 v1.2.3'}))
		await user.click(within(screen.getByRole('dialog', {name: '安装更新 v1.2.3'})).getByRole('button', {name: '立即安装'}))
		let dialog = await screen.findByRole('dialog', {name: '仍有执行中的任务'})
		expect(within(dialog).getByText(/不会改变任务状态或删除工作目录/)).toBeInTheDocument()
		await user.click(within(dialog).getByRole('button', {name: '取消'}))
		expect(calls).toEqual([])
		expect(screen.getByRole('button', {name: '安装已下载版本 v1.2.3'})).toBeEnabled()

		await user.click(screen.getByRole('button', {name: '安装已下载版本 v1.2.3'}))
		await user.click(within(screen.getByRole('dialog', {name: '安装更新 v1.2.3'})).getByRole('button', {name: '立即安装'}))
		dialog = await screen.findByRole('dialog', {name: '仍有执行中的任务'})
		await user.click(within(dialog).getByRole('button', {name: '关闭终端并安装'}))
		await waitFor(() => expect(calls).toEqual(['launch-installer', 'prepare-quit', 'quit']))
	})

	it('does not prepare or quit when installer launch fails', async () => {
		const user = userEvent.setup()
		const onError = vi.fn()
		const client = updateClient({
			getUpdateState: vi.fn(async () => updateState({status: 'downloaded', version: 'v1.2.3'})),
			launchDownloadedUpdate: vi.fn(async () => { throw new Error('installer unavailable') }),
		})
		renderUpdateControl(client, onError)

		await user.click(await screen.findByRole('button', {name: '安装已下载版本 v1.2.3'}))
		await user.click(within(screen.getByRole('dialog', {name: '安装更新 v1.2.3'})).getByRole('button', {name: '立即安装'}))
		await waitFor(() => expect(onError).toHaveBeenCalledWith('installer unavailable'))
		expect(client.prepareQuit).not.toHaveBeenCalled()
		expect(client.quit).not.toHaveBeenCalled()
		expect(screen.getByRole('dialog', {name: '安装更新 v1.2.3'})).toBeInTheDocument()
	})

	it('does not quit when terminal preparation fails after a successful installer launch', async () => {
		const user = userEvent.setup()
		const onError = vi.fn()
		const client = updateClient({
			getUpdateState: vi.fn(async () => updateState({status: 'downloaded', version: 'v1.2.3'})),
			prepareQuit: vi.fn(async () => { throw new Error('terminal shutdown failed') }),
		})
		renderUpdateControl(client, onError)

		await user.click(await screen.findByRole('button', {name: '安装已下载版本 v1.2.3'}))
		await user.click(within(screen.getByRole('dialog', {name: '安装更新 v1.2.3'})).getByRole('button', {name: '立即安装'}))
		await waitFor(() => expect(onError).toHaveBeenCalledWith('terminal shutdown failed'))
		expect(client.launchDownloadedUpdate).toHaveBeenCalledOnce()
		expect(client.quit).not.toHaveBeenCalled()
	})
})

function renderUpdateControl(client: UpdateClient, onError?: (message: string) => void) {
	return render(<TooltipProvider><UpdateControl client={client} onError={onError}/></TooltipProvider>)
}

function updateClient(overrides: Partial<UpdateClient> = {}): UpdateClient {
	return {
		getUpdateState: vi.fn(async () => updateState({status: 'idle'})),
		onUpdateState: vi.fn(() => () => undefined),
		startUpdateDownload: vi.fn(async () => undefined),
		openUpdateReleasePage: vi.fn(async () => undefined),
		launchDownloadedUpdate: vi.fn(async () => undefined),
		hasRunningTasks: vi.fn(async () => false),
		prepareQuit: vi.fn(async () => undefined),
		quit: vi.fn(),
		...overrides,
	}
}

function updateState(state: UpdateState): UpdateState {
	return state
}
