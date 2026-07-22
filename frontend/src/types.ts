export type TaskStatus = 'pending' | 'running' | 'completed'
export type TerminalState = 'active' | 'exited'
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
}

export interface TerminalRecord {
  id: string
  taskId: string
  state: TerminalState
  output?: string
}

export interface TerminalEvent {
  taskId: string
  terminalId: string
  type: 'output' | 'exited' | 'error'
  data?: string
}

export interface SettingsRecord {
	workspaceRoot: string
	taskTreeWidth: number
	colorScheme: ColorScheme
	shellPath: string
	taskMenuItems: TaskMenuItem[]
}

export interface TaskMenuItem {
  id: string
  kind: TaskMenuItemKind
  name: string
  command?: string
  arguments?: string[]
  showTerminal: boolean
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

export function clampTaskTreeWidth(width: number, viewportWidth = window.innerWidth): number {
  return Math.round(Math.min(Math.max(280, width), Math.max(280, viewportWidth - 360)))
}
