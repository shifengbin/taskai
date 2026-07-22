export type TaskStatus = 'pending' | 'running' | 'completed'
export type TerminalState = 'active' | 'exited'

export interface TaskRecord {
  id: string
  title: string
  description: string
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
}

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
