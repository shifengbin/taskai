import {type FormEvent, useEffect, useMemo, useRef, useState} from 'react'
import {AlertTriangle, GitFork, Loader2} from 'lucide-react'

import {api} from '../api'
import type {GitLabCloneURLMode, GitLabImportResult, GitLabProject} from '../types'
import {
	Alert,
	Button,
	Checkbox,
	Chip,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Field,
	Input,
} from './ui'

const selectClass = 'w-full border-2 border-snap-outline rounded-snap bg-snap-surface px-3 py-2 text-sm text-snap-ink shadow-snap-sm outline-none transition-[box-shadow,border-color] focus-visible:border-snap-cobalt focus-visible:ring-[3px] focus-visible:ring-snap-cobalt disabled:cursor-not-allowed disabled:opacity-60'

type GitLabImportDialogProps = {
	open: boolean
	onOpenChange: (open: boolean) => void
	onImported: (result: GitLabImportResult) => Promise<void> | void
}

export function GitLabImportDialog({open, onOpenChange, onImported}: GitLabImportDialogProps) {
	const [address, setAddress] = useState('')
	const [username, setUsername] = useState('')
	const [token, setToken] = useState('')
	const [projects, setProjects] = useState<GitLabProject[]>([])
	const [loaded, setLoaded] = useState(false)
	const [filter, setFilter] = useState('')
	const [selectedIDs, setSelectedIDs] = useState<Set<number>>(new Set())
	const [mode, setMode] = useState<GitLabCloneURLMode>('ssh')
	const [loading, setLoading] = useState(false)
	const [savingDefaults, setSavingDefaults] = useState(false)
	const [importing, setImporting] = useState(false)
	const [error, setError] = useState('')
	const loadRequestID = useRef(0)
	const defaultsRequestID = useRef(0)
	const formRevision = useRef(0)
	const loadingRef = useRef(false)
	const savingDefaultsRef = useRef(false)
	const importingRef = useRef(false)

	useEffect(() => {
		if (!open) {
			return
		}
		const requestID = ++defaultsRequestID.current
		const revision = formRevision.current
		void api.getGitLabImportDefaults().then((defaults) => {
			if (requestID !== defaultsRequestID.current || revision !== formRevision.current) {
				return
			}
			setAddress(defaults.address ?? '')
			setUsername(defaults.username ?? '')
			setToken(defaults.token ?? '')
		}).catch((currentError) => {
			if (requestID === defaultsRequestID.current && revision === formRevision.current) {
				setError(errorMessage(currentError))
			}
		})
	}, [open])

	const visibleProjects = useMemo(() => {
		const search = filter.trim().toLocaleLowerCase()
		if (!search) {
			return projects
		}
		return projects.filter((project) => project.name.toLocaleLowerCase().includes(search) || project.pathWithNamespace.toLocaleLowerCase().includes(search))
	}, [filter, projects])
	const selectableCount = projects.filter((project) => !project.imported).length
	const selectedCount = selectedIDs.size
	const usesPlainHTTP = /^http:\/\//i.test(address.trim())

	const reset = () => {
		loadRequestID.current++
		defaultsRequestID.current++
		formRevision.current = 0
		setAddress('')
		setUsername('')
		setToken('')
		setProjects([])
		setLoaded(false)
		setFilter('')
		setSelectedIDs(new Set())
		setMode('ssh')
		loadingRef.current = false
		savingDefaultsRef.current = false
		importingRef.current = false
		setLoading(false)
		setSavingDefaults(false)
		setImporting(false)
		setError('')
	}
	const changeAddress = (value: string) => {
		if (loadingRef.current || importingRef.current) {
			return
		}
		formRevision.current++
		setAddress(value)
	}
	const changeUsername = (value: string) => {
		if (loadingRef.current || importingRef.current) {
			return
		}
		formRevision.current++
		setUsername(value)
	}
	const changeToken = (value: string) => {
		if (loadingRef.current || importingRef.current) {
			return
		}
		formRevision.current++
		setToken(value)
	}
	const changeFilter = (value: string) => {
		if (!importingRef.current) {
			setFilter(value)
		}
	}
	const changeMode = (value: GitLabCloneURLMode) => {
		if (!importingRef.current) {
			setMode(value)
		}
	}

	const close = (force = false) => {
		if ((savingDefaultsRef.current || importingRef.current) && !force) {
			return
		}
		reset()
		onOpenChange(false)
	}

	const loadProjects = async (event: FormEvent) => {
		event.preventDefault()
		if (loadingRef.current || importingRef.current || !address.trim() || !username.trim() || !token) {
			return
		}
		const nextDefaults = {address: address.trim(), username: username.trim(), token}
		const requestID = ++loadRequestID.current
		loadingRef.current = true
		setLoading(true)
		setError('')
		setLoaded(false)
		setProjects([])
		setSelectedIDs(new Set())
		try {
			const result = await api.listGitLabProjects(nextDefaults.address, nextDefaults.username, nextDefaults.token)
			if (requestID !== loadRequestID.current) {
				return
			}
			savingDefaultsRef.current = true
			setSavingDefaults(true)
			await api.saveGitLabImportDefaults(nextDefaults.address, nextDefaults.username, nextDefaults.token)
			if (requestID !== loadRequestID.current) {
				return
			}
			setProjects(result.projects ?? [])
			setLoaded(true)
			setFilter('')
			setMode('ssh')
		} catch (currentError) {
			if (requestID === loadRequestID.current) {
				setError(errorMessage(currentError))
			}
		} finally {
			if (requestID === loadRequestID.current) {
				loadingRef.current = false
				savingDefaultsRef.current = false
				setLoading(false)
				setSavingDefaults(false)
			}
		}
	}

	const selectCurrent = () => {
		if (importingRef.current) {
			return
		}
		setSelectedIDs((current) => {
			const next = new Set(current)
			visibleProjects.forEach((project) => {
				if (!project.imported) {
					next.add(project.id)
				}
			})
			return next
		})
	}

	const deselectCurrent = () => {
		if (importingRef.current) {
			return
		}
		setSelectedIDs((current) => {
			const next = new Set(current)
			visibleProjects.forEach((project) => next.delete(project.id))
			return next
		})
	}

	const toggleProject = (project: GitLabProject, selected: boolean) => {
		if (project.imported || importingRef.current) {
			return
		}
		setSelectedIDs((current) => {
			const next = new Set(current)
			if (selected) {
				next.add(project.id)
			} else {
				next.delete(project.id)
			}
			return next
		})
	}

	const importSelected = async () => {
		if (loadingRef.current || savingDefaultsRef.current || importingRef.current) {
			return
		}
		const selected = projects.filter((project) => selectedIDs.has(project.id) && !project.imported)
		if (selected.length === 0) {
			return
		}
		importingRef.current = true
		setImporting(true)
		setError('')
		try {
			const result = await api.importGitLabProjects(selected, mode)
			await onImported(result)
			close(true)
		} catch (currentError) {
			setError(errorMessage(currentError))
			importingRef.current = false
			setImporting(false)
		}
	}

	return <Dialog open={open} onOpenChange={(next) => { if (!next) close() }}>
		<DialogContent className="max-w-4xl" showClose={false}>
			<DialogHeader>
				<div className="flex items-center gap-3">
					<span className="grid h-10 w-10 place-items-center rounded-snap border-2 border-snap-outline bg-snap-surface-2 text-snap-cobalt shadow-snap-sm"><GitFork className="h-5 w-5" strokeWidth={2.25}/></span>
					<div>
						<DialogTitle>从 GitLab 导入</DialogTitle>
						<DialogDescription>连接私有 GitLab，筛选并批量加入内置 Git 分类。</DialogDescription>
					</div>
				</div>
			</DialogHeader>
			<form className="grid gap-3" onSubmit={(event) => void loadProjects(event)}>
				<div className="grid gap-3 sm:grid-cols-2">
					<Field label="GitLab 地址" className="sm:col-span-2"><Input autoFocus placeholder="https://gitlab.example.com" value={address} disabled={loading || importing} onChange={(event) => changeAddress(event.target.value)}/></Field>
					<Field label="GitLab 用户名"><Input autoComplete="off" value={username} disabled={loading || importing} onChange={(event) => changeUsername(event.target.value)}/></Field>
					<Field label="个人访问令牌"><Input type="password" autoComplete="off" value={token} disabled={loading || importing} onChange={(event) => changeToken(event.target.value)}/></Field>
				</div>
				<Alert severity="info">GitLab 地址、用户名和访问令牌将以未加密形式保存在本机。</Alert>
				{usesPlainHTTP && <Alert severity="warning" icon={<AlertTriangle className="h-5 w-5"/>}>访问令牌将通过未加密的 HTTP 连接发送。</Alert>}
				<div className="flex justify-end">
					<Button type="submit" variant="primary" disabled={loading || importing || !address.trim() || !username.trim() || !token}>
						{loading && <Loader2 className="h-4 w-4 animate-spin"/>}{savingDefaults ? '正在保存连接' : loading ? '正在获取项目' : '获取项目'}
					</Button>
				</div>
			</form>
			{error && <Alert severity="error">{error}</Alert>}
			{loaded && <div className="grid gap-3 min-h-0">
				<div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_180px]">
					<Field label="筛选 GitLab 项目"><Input placeholder="按名称或 namespace/project 筛选" value={filter} disabled={importing} onChange={(event) => changeFilter(event.target.value)}/></Field>
					<Field label="仓库地址格式"><select className={selectClass} value={mode} disabled={importing} onChange={(event) => changeMode(event.target.value as GitLabCloneURLMode)}><option value="ssh">SSH</option><option value="https">HTTPS</option></select></Field>
				</div>
				<div className="flex flex-wrap items-center justify-between gap-2">
					<div className="flex flex-wrap items-center gap-2 text-xs text-snap-muted">
						<span>{`显示 ${visibleProjects.length} / ${projects.length} 个项目`}</span><span>{`可选 ${selectableCount} 个`}</span><SnapSelectedCount count={selectedCount}/>
					</div>
					<div className="flex flex-wrap gap-2"><Button type="button" variant="secondary" size="sm" disabled={importing} onClick={selectCurrent}>全选当前结果</Button><Button type="button" variant="ghost" size="sm" disabled={importing} onClick={deselectCurrent}>取消选择当前结果</Button></div>
				</div>
				<div className="max-h-[42vh] overflow-y-auto rounded-snap border-2 border-snap-outline bg-snap-surface snap-no-scrollbar">
					{visibleProjects.length === 0 ? <div className="px-4 py-8 text-center text-sm text-snap-muted">未找到匹配的项目。</div> : visibleProjects.map((project) => {
						const preview = mode === 'ssh' ? project.sshUrl : project.httpUrl
						return <label key={project.id} className="grid grid-cols-[auto_minmax(0,1fr)] gap-3 border-b-2 border-snap-outline px-3 py-3 last:border-b-0 hover:bg-snap-surface-2">
							<Checkbox aria-label={`${project.name} ${project.pathWithNamespace}`} checked={selectedIDs.has(project.id)} disabled={project.imported || importing} onCheckedChange={(checked) => toggleProject(project, checked === true)}/>
							<span className="grid min-w-0 gap-1">
								<span className="flex flex-wrap items-center gap-2"><span className="font-display text-sm font-bold text-snap-ink">{project.name}</span><Chip variant="muted">{project.visibility || '未知可见性'}</Chip>{project.archived && <Chip variant="amber">已归档</Chip>}{project.imported && <Chip variant="info">已导入</Chip>}</span>
								<span className="text-xs font-bold text-snap-muted">{project.pathWithNamespace}</span>
								<span className="truncate font-mono text-xs text-snap-muted" title={preview}>{preview}</span>
							</span>
						</label>
					})}
				</div>
			</div>}
			<DialogFooter>
				<Button type="button" variant="secondary" disabled={savingDefaults || importing} onClick={() => close()}>取消</Button>
				<Button type="button" variant="primary" disabled={!loaded || selectedCount === 0 || importing} onClick={() => void importSelected()}>{importing ? '正在导入' : `导入 ${selectedCount} 个项目`}</Button>
			</DialogFooter>
		</DialogContent>
	</Dialog>
}

function SnapSelectedCount({count}: {count: number}) {
	return <span className="font-bold text-snap-ink">{'已选择 ' + count + ' 项'}</span>
}

function errorMessage(error: unknown): string {
	return error instanceof Error ? error.message : String(error)
}
