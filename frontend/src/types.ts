export type TaskStatus = 'pending' | 'running' | 'completed'
export type TerminalState = 'active' | 'exited'
export type RealtimeStatus = 'idle' | 'working' | 'unread' | 'error'
export type StatusManagementMode = 'title-change' | 'http'
export type ColorScheme = 'light' | 'dark'
export type TaskMenuItemKind = 'edit-task' | 'create-terminal' | 'open-folder' | 'command'
export const defaultTaskColor = '#4f46e5'

export interface TaskRecord {
  id: string
  title: string
  description: string
  color?: string
  status: TaskStatus
  createdAt: string
  completedAt?: string
  workspaceRoot?: string
  workspacePath?: string
	extraInfo?: TaskExtraInfo[]
  realtimeStatus?: RealtimeStatus
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
  output?: string
  title?: string
  realtimeStatus?: RealtimeStatus
}

export function terminalDisplayName(terminal: Pick<TerminalRecord, 'title'>): string {
  return terminal.title?.trim() || '终端'
}

export interface TerminalEvent {
  taskId: string
  terminalId: string
  type: 'output' | 'exited' | 'error'
  data?: string
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
	taskTreeWidth: number
	colorScheme: ColorScheme
	shellPath: string
	taskMenuItems: TaskMenuItem[]
	activeTaskStatus: TaskStatus
	statusManagementMode: StatusManagementMode
	statusManagementHTTPPort: number
	httpServiceEnabled: boolean
}

export interface TaskMenuItem {
  id: string
  kind: TaskMenuItemKind
  name: string
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
