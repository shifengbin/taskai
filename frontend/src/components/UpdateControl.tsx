import {useEffect, useRef, useState} from 'react'
import {CircleAlert, Download, LoaderCircle, PackageCheck} from 'lucide-react'

import {api} from '../api'
import type {UpdateState} from '../types'
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from './ui'

export interface UpdateClient {
	getUpdateState(): Promise<UpdateState>
	onUpdateState(listener: (state: UpdateState) => void): () => void
	startUpdateDownload(): Promise<void>
	openUpdateReleasePage(): Promise<void>
	launchDownloadedUpdate(): Promise<void>
	hasRunningTasks(): Promise<boolean>
	prepareQuit(): Promise<void>
	quit(): void
}

interface UpdateControlProps {
	client?: UpdateClient
	onError?: (message: string) => void
}

const idleState: UpdateState = {status: 'idle'}

export function UpdateControl({client = api, onError}: UpdateControlProps) {
	const [state, setState] = useState<UpdateState>(idleState)
	const [downloadFailureOpen, setDownloadFailureOpen] = useState(false)
	const [installDialogOpen, setInstallDialogOpen] = useState(false)
	const [runningTasksOpen, setRunningTasksOpen] = useState(false)
	const [actionBusy, setActionBusy] = useState(false)
	const onErrorRef = useRef(onError)
	onErrorRef.current = onError

	useEffect(() => {
		let active = true
		let eventRevision = 0
		const applyState = (next: UpdateState) => {
			setState(next?.status ? next : idleState)
		}
		const initialState = client.getUpdateState()
		const unsubscribe = client.onUpdateState((next) => {
			if (!active) return
			eventRevision += 1
			applyState(next)
			if (next.status === 'download_failed') {
				setDownloadFailureOpen(true)
			}
			if (next.status === 'downloaded') {
				setDownloadFailureOpen(false)
				setInstallDialogOpen(true)
			}
		})
		void (async () => {
			try {
				const initial = await initialState
				if (active && eventRevision === 0) applyState(initial)
			} catch (error) {
				if (active) onErrorRef.current?.(errorMessage(error))
			}
			const revisionBeforeRecheck = eventRevision
			try {
				const current = await client.getUpdateState()
				if (active && eventRevision === revisionBeforeRecheck) applyState(current)
			} catch (error) {
				if (active) onErrorRef.current?.(errorMessage(error))
			}
		})()
		return () => {
			active = false
			unsubscribe()
		}
	}, [client])

	const version = state.version ?? ''
	const presentation = updatePresentation(state)
	const startDownload = async () => {
		if (state.status !== 'available' && state.status !== 'download_failed') return
		setDownloadFailureOpen(false)
		setState({...state, status: 'downloading', error: undefined})
		try {
			await client.startUpdateDownload()
		} catch (error) {
			const message = errorMessage(error)
			setState({...state, status: 'download_failed', error: message})
			setDownloadFailureOpen(true)
		}
	}
	const openManualDownload = async (closeDialog?: () => void) => {
		setActionBusy(true)
		try {
			await client.openUpdateReleasePage()
			closeDialog?.()
		} catch (error) {
			onErrorRef.current?.(errorMessage(error))
		} finally {
			setActionBusy(false)
		}
	}
	const launchAndQuit = async () => {
		setActionBusy(true)
		try {
			await client.launchDownloadedUpdate()
			await client.prepareQuit()
			client.quit()
		} catch (error) {
			onErrorRef.current?.(errorMessage(error))
		} finally {
			setActionBusy(false)
		}
	}
	const requestInstall = async () => {
		setActionBusy(true)
		try {
			if (await client.hasRunningTasks()) {
				setInstallDialogOpen(false)
				setRunningTasksOpen(true)
				return
			}
		} catch (error) {
			onErrorRef.current?.(errorMessage(error))
			return
		} finally {
			setActionBusy(false)
		}
		await launchAndQuit()
	}
	const openStateAction = () => {
		switch (state.status) {
		case 'available':
			void startDownload()
			break
		case 'download_failed':
			setDownloadFailureOpen(true)
			break
		case 'downloaded':
			setInstallDialogOpen(true)
			break
		}
	}

	return (
		<>
			{state.status === 'idle' && state.currentVersion && (
				<span className="text-xs font-medium text-snap-ink/60" data-testid="app-version">
					{state.currentVersion}
				</span>
			)}
			{state.status !== 'idle' && (
				<Tooltip>
					<TooltipTrigger asChild>
						<span className="inline-flex h-8 w-[86px] shrink-0" data-testid="update-control">
							<Button
								aria-label={`${presentation.ariaLabel}${version ? ` ${version}` : ''}`}
								className={`h-8 w-[86px] px-2 text-xs ${presentation.className}`}
								disabled={state.status === 'downloading'}
								onClick={openStateAction}
								size="sm"
								variant={presentation.variant}
							>
								<presentation.Icon className={`h-3.5 w-3.5 shrink-0 ${state.status === 'downloading' ? 'animate-spin' : ''}`} strokeWidth={2.4}/>
								<span>{presentation.label}</span>
							</Button>
						</span>
					</TooltipTrigger>
					<TooltipContent>{presentation.tooltip}{version ? ` ${version}` : ''}</TooltipContent>
				</Tooltip>
			)}

			<Dialog open={downloadFailureOpen} onOpenChange={setDownloadFailureOpen}>
				<DialogContent className="max-w-sm" showClose={false}>
					<DialogHeader><DialogTitle>更新下载失败</DialogTitle></DialogHeader>
					<DialogDescription asChild>
						<div className="space-y-2">
							<p>无法完成 {version || '目标版本'} 的安装包下载。</p>
							{state.error && <p>{state.error}</p>}
						</div>
					</DialogDescription>
					<DialogFooter>
						<Button variant="ghost" disabled={actionBusy} onClick={() => setDownloadFailureOpen(false)}>取消</Button>
						<Button variant="secondary" disabled={actionBusy} onClick={() => void openManualDownload(() => setDownloadFailureOpen(false))}>手动下载</Button>
						<Button variant="primary" disabled={actionBusy} onClick={() => void startDownload()}>重新下载</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			<Dialog open={installDialogOpen} onOpenChange={setInstallDialogOpen}>
				<DialogContent className="max-w-sm" showClose={false}>
					<DialogHeader><DialogTitle>安装更新 {version}</DialogTitle></DialogHeader>
					<DialogDescription>安装程序成功启动后，TaskAI 会关闭全部终端并退出。</DialogDescription>
					<DialogFooter>
						<Button variant="ghost" disabled={actionBusy} onClick={() => setInstallDialogOpen(false)}>稍后安装</Button>
						<Button variant="secondary" disabled={actionBusy} onClick={() => void openManualDownload()}>手动下载</Button>
						<Button variant="primary" disabled={actionBusy} onClick={() => void requestInstall()}>立即安装</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			<Dialog open={runningTasksOpen} onOpenChange={setRunningTasksOpen}>
				<DialogContent className="max-w-sm" showClose={false}>
					<DialogHeader><DialogTitle>仍有执行中的任务</DialogTitle></DialogHeader>
					<DialogDescription>
						安装会关闭全部终端，但不会改变任务状态或删除工作目录。下次启动后这些任务仍显示为执行中。
					</DialogDescription>
					<DialogFooter>
						<Button variant="ghost" disabled={actionBusy} onClick={() => setRunningTasksOpen(false)}>取消</Button>
						<Button variant="primary" disabled={actionBusy} onClick={() => void launchAndQuit()}>关闭终端并安装</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	)
}

function updatePresentation(state: UpdateState) {
	switch (state.status) {
	case 'available':
		return {Icon: Download, label: 'new', ariaLabel: '下载新版本', tooltip: '发现新版本', variant: 'primary' as const, className: ''}
	case 'downloading':
		return {Icon: LoaderCircle, label: '下载中', ariaLabel: '正在下载', tooltip: '正在下载安装包', variant: 'secondary' as const, className: ''}
	case 'download_failed':
		return {Icon: CircleAlert, label: '下载失败', ariaLabel: '下载失败', tooltip: '下载安装包失败', variant: 'danger' as const, className: ''}
	case 'downloaded':
		return {Icon: PackageCheck, label: '已下载', ariaLabel: '安装已下载版本', tooltip: '安装包已下载', variant: 'secondary' as const, className: 'text-snap-cobalt'}
	default:
		return {Icon: Download, label: '', ariaLabel: '', tooltip: '', variant: 'secondary' as const, className: ''}
	}
}

function errorMessage(error: unknown): string {
	return error instanceof Error ? error.message : String(error)
}
