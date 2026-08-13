import type {TerminalTheme} from './terminal-theme'

export type TaskStatus = 'pending' | 'running' | 'completed'
export type TaskGitAction = 'none' | 'commit' | 'publish' | 'sync'
export type TerminalState = 'active' | 'exited'
export type TerminalExitReason = 'normal' | 'unexpected' | 'closed' | 'task-ended' | 'application-shutdown'
export type RealtimeStatus = 'idle' | 'working' | 'unread' | 'error'
export type StatusManagementMode = 'title-change' | 'output-change' | 'http'
export type ColorScheme = 'light' | 'dark'
export type TaskMenuItemKind = 'edit-task' | 'create-terminal' | 'open-folder' | 'toggle-shelved' | 'command'
export type LifecycleHook = 'beforeStart' | 'postStart' | 'beforeEnd' | 'postEnd' | 'updateTask'
export type LifecycleExecutionState = 'running' | 'failed'
export type LifecycleCommandKind = 'custom' | 'create-workspace' | 'delete-workspace' | 'git-clone' | 'git-clone-repository' | 'manifest-file' | 'update-default-branch'
export type LifecycleCommandChainArgumentMode = 'enabled' | 'disabled'
export type TaskTemplateFieldInputType = 'string' | 'bool'
export type TaskTemplateValues = Record<string, string | boolean>
export type TerminalShortcutStep = {kind: 'text', text: string} | {kind: 'key', key: string, modifiers?: string[]} | {kind: 'enter'}

export interface TerminalShortcut {
  id: string
  shortcut: string
  steps: TerminalShortcutStep[]
	includePrograms?: string[]
}

export interface TerminalNoteTemplate {
	originalPrefix: string
	notePrefix: string
	listSuffix: string
}
export const defaultTaskColor = '#4f46e5'
export const taskColorOptions = ['#ef4444', '#f97316', '#eab308', '#22c55e', '#14b8a6', '#3b82f6', '#6366f1', '#a855f7', '#ec4899'] as const

export function randomTaskColor(): string {
  return taskColorOptions[Math.floor(Math.random() * taskColorOptions.length)]
}

export const lifecycleHooks: Array<{id: LifecycleHook, label: string}> = [
  {id: 'beforeStart', label: '开始前'},
  {id: 'postStart', label: '开始后'},
  {id: 'beforeEnd', label: '结束前'},
  {id: 'postEnd', label: '结束后'},
  {id: 'updateTask', label: '更新任务后'},
]

export interface LifecycleExecution {
  runId?: string
  revision?: number
  hook: LifecycleHook
  chainId: string
  currentCommandId?: string
  currentCommandName?: string
  currentIndex: number
  commandCount: number
  state: LifecycleExecutionState
  error?: string
}

export interface LifecycleExecutionWatermark {
  runId: string
  revision: number
}

export interface LifecycleCommand {
  id: string
  kind: LifecycleCommandKind
  name: string
  command?: string
  arguments: string[]
  chainArgumentMode: LifecycleCommandChainArgumentMode
  documentation?: string
  applicableHooks: LifecycleHook[]
}

export interface LifecycleCommandReference {
  commandId: string
  arguments: string[]
}

export interface LifecycleCommandChain {
  id: string
  name: string
  commands: LifecycleCommandReference[]
  applicableHooks: LifecycleHook[]
}

export interface LifecyclePreset {
  id: string
  name: string
  chains: Partial<Record<LifecycleHook, string>>
}

export interface TaskRecord {
  id: string
  title: string
  description: string
  color?: string
  status: TaskStatus
  shelved?: boolean
  createdAt: string
  completedAt?: string
  workspaceRoot?: string
	workspacePath?: string
	extraInfo?: TaskExtraInfo[]
	taskTemplateId?: string
  templateFields?: TaskTemplateValues
  lifecycleChains?: Partial<Record<LifecycleHook, string>>
  lifecycleExecution?: LifecycleExecution
  lifecycleExecutionWatermark?: LifecycleExecutionWatermark
  realtimeStatus?: RealtimeStatus
}

export interface TaskGitRepository {
	path: string
	branch?: string
	remote?: string
	notice?: string
	dirty: boolean
	hasUpstream?: boolean
	remoteBranchExists?: boolean
	synchronized?: boolean
	action: TaskGitAction
}

export interface TaskTemplateField {
	key: string
	displayName: string
	inputType: TaskTemplateFieldInputType
	required: boolean
	defaultValue: string | boolean
	injectEnvironment: boolean
}

export interface TaskTemplate {
	id: string
	name: string
	fields: TaskTemplateField[]
}

export interface ExtraInfoParameterDefinition {
	key: string
	displayName: string
	required: boolean
	inputType?: ExtraInfoParameterInputType
}

export type ExtraInfoParameterInputType = 'text' | 'checkbox'

export interface ExtraInfoField {
	key: string
	displayName: string
	value?: string
	defaultValue?: string
}

export interface ExtraInfoTemplate {
	id: string
	catalogue: string
	displayName?: string
	fields: ExtraInfoField[]
	parameters: ExtraInfoParameterDefinition[]
	builtIn: boolean
}

export interface ExtraInfo {
	id: string
	templateId: string
	catalogue: string
	fields: ExtraInfoField[]
	parameters: ExtraInfoParameter[]
}

export interface QuickInput {
	id: string
	name: string
	content: string
}

export interface ExtraInfoParameter extends ExtraInfoParameterDefinition {
	value: string
}

export type TaskExtraInfoParameter = ExtraInfoParameter

export interface TaskExtraInfo {
	id: string
	informationId?: string
	templateId?: string
	catalogue: string
	displayName?: string
	fields: ExtraInfoField[]
	parameters: TaskExtraInfoParameter[]
}

export interface TerminalRecord {
  id: string
  taskId: string
  state: TerminalState
  title?: string
  alias?: string
  command?: string
  realtimeStatus?: RealtimeStatus
}

export function terminalActualName(terminal: Pick<TerminalRecord, 'title'>): string {
  return terminal.title?.trim() || '终端'
}

export function terminalDisplayName(terminal: Pick<TerminalRecord, 'title' | 'alias'>): string {
  return terminal.alias?.trim() || terminalActualName(terminal)
}

export function terminalAliasDetails(terminal: Pick<TerminalRecord, 'title' | 'command'>): {actualName: string, command: string} {
  return {
    actualName: terminalActualName(terminal),
    command: terminal.command?.trim() || '未提供启动命令',
  }
}

export interface TerminalEvent {
  taskId: string
  terminalId: string
  type: 'output' | 'exited' | 'error'
  data?: string
  exitCode?: number
  exitReason?: TerminalExitReason
}

export interface RealtimeStatusEvent {
  version: number
  taskId: string
  taskStatus: RealtimeStatus
  terminalId?: string
  terminalStatus?: RealtimeStatus
  terminalRemoved?: boolean
}

export interface SettingsRecord {
	workspaceRoot: string
	gitScanDepth?: number
	taskTreeWidth: number
	colorScheme: ColorScheme
	terminalFontFamily?: string
	terminalFontSize?: number
	terminalTheme?: TerminalTheme
	terminalShortcuts?: TerminalShortcut[]
	terminalNoteTemplate?: TerminalNoteTemplate
	windowMaximized?: boolean
	shellPath: string
	taskMenuItems: TaskMenuItem[]
	activeTaskStatus: TaskStatus
	statusManagementMode: StatusManagementMode
	statusManagementHTTPPort: number
	httpServiceEnabled: boolean
  lifecycleCommands?: LifecycleCommand[]
  lifecycleChains?: LifecycleCommandChain[]
  lifecyclePresets?: LifecyclePreset[]
  defaultLifecyclePresetId?: string
	taskTemplates?: TaskTemplate[]
	activeTaskTemplateId?: string
}

export interface TerminalFontCandidate {
	family: string
	spacing: 'mono' | 'dual' | 'unavailable'
}

export interface TaskMenuItem {
  id: string
  kind: TaskMenuItemKind
  name: string
  unshelveName?: string
  command?: string
  arguments?: string[]
  showTerminal: boolean
  beforeScript?: TaskScript
  afterScript?: TaskScript
}

export interface TaskScript {
  script: string
  arguments?: string[]
}

export interface TaskMenuCommandResult {
  terminal?: TerminalRecord
}

export const defaultTaskMenuItems: TaskMenuItem[] = [
  {id: 'system.edit-task', kind: 'edit-task', name: '编辑任务', showTerminal: false},
  {id: 'system.create-terminal', kind: 'create-terminal', name: '新增终端', showTerminal: false},
  {id: 'system.open-folder', kind: 'open-folder', name: '打开任务文件夹', showTerminal: false},
  {id: 'system.toggle-shelved', kind: 'toggle-shelved', name: '搁置任务', unshelveName: '取消搁置', showTerminal: false},
]

export const taskStatusLabel: Record<TaskStatus, string> = {
  pending: '未执行',
  running: '执行中',
  completed: '已完成',
}

export const terminalStatusLabel: Record<TerminalState, string> = {
  active: '运行中',
  exited: '已退出',
}

export const realtimeStatusLabel: Record<RealtimeStatus, string> = {
  idle: '空闲',
  working: '工作中',
  unread: '未读',
  error: '异常',
}

export function terminalRealtimeStatus(terminal: Pick<TerminalRecord, 'realtimeStatus' | 'state'>): RealtimeStatus {
  return terminal.realtimeStatus ?? (terminal.state === 'exited' ? 'error' : 'idle')
}

export function clampTaskTreeWidth(width: number, viewportWidth = window.innerWidth): number {
  return Math.round(Math.min(Math.max(280, width), Math.max(280, viewportWidth - 360)))
}
