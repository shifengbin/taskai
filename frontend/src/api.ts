import {
  CloseTerminal,
  CreateTask,
  CreateTerminal,
  FinishTask,
  GetSettings,
  HasRunningTasks,
  ListTasks,
  PrepareQuit,
  ResizeTerminal,
  SaveSettings,
  StartTask,
  WriteTerminal,
} from '../wailsjs/go/main/App'
import {EventsOff, EventsOn, Quit} from '../wailsjs/runtime/runtime'

import type {SettingsRecord, TaskRecord, TerminalEvent, TerminalRecord} from './types'

export const api = {
  createTask: (title: string, description: string) => CreateTask(title, description) as Promise<TaskRecord>,
  listTasks: () => ListTasks() as Promise<TaskRecord[]>,
  startTask: (taskID: string) => StartTask(taskID) as Promise<TaskRecord>,
  finishTask: (taskID: string) => FinishTask(taskID) as Promise<TaskRecord>,
  getSettings: () => GetSettings() as Promise<SettingsRecord>,
  saveSettings: (settings: SettingsRecord) => SaveSettings(settings) as Promise<SettingsRecord>,
  createTerminal: (taskID: string, columns: number, rows: number) => CreateTerminal(taskID, columns, rows) as Promise<TerminalRecord>,
  writeTerminal: (taskID: string, terminalID: string, data: string) => WriteTerminal(taskID, terminalID, data),
  resizeTerminal: (taskID: string, terminalID: string, columns: number, rows: number) => ResizeTerminal(taskID, terminalID, columns, rows),
  closeTerminal: (taskID: string, terminalID: string) => CloseTerminal(taskID, terminalID),
  hasRunningTasks: () => HasRunningTasks(),
  prepareQuit: () => PrepareQuit(),
  quit: () => Quit(),
  onTerminalEvent(listener: (event: TerminalEvent) => void) {
    EventsOn('task-terminal:event', listener)
    return () => EventsOff('task-terminal:event')
  },
  onCloseRequested(listener: () => void) {
    EventsOn('application:close-requested', listener)
    return () => EventsOff('application:close-requested')
  },
}
