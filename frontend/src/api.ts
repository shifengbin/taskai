import {
  CloseTerminal,
	CreateTask,
	CreateCommandTerminal,
	CreateTerminal,
	DetectShells,
  FinishTask,
  GetSettings,
  HasRunningTasks,
	ListTasks,
	OpenTaskFolder,
  PrepareQuit,
	ResizeTerminal,
	RunTaskCommand,
	SaveSettings,
	StartTask,
	UpdateTask,
  WriteTerminal,
} from '../wailsjs/go/main/App'
import {settings as settingsModel} from '../wailsjs/go/models'
import {EventsOff, EventsOn, Quit} from '../wailsjs/runtime/runtime'

import type {SettingsRecord, TaskRecord, TerminalEvent, TerminalRecord} from './types'

export const api = {
  createTask: (title: string, description: string, color: string) => CreateTask(title, description, color) as Promise<TaskRecord>,
  updateTask: (taskID: string, title: string, description: string, color: string) => UpdateTask(taskID, title, description, color) as Promise<TaskRecord>,
  listTasks: () => ListTasks() as Promise<TaskRecord[]>,
  startTask: (taskID: string) => StartTask(taskID) as Promise<TaskRecord>,
  finishTask: (taskID: string) => FinishTask(taskID) as Promise<TaskRecord>,
  getSettings: () => GetSettings() as Promise<SettingsRecord>,
	saveSettings: (settings: SettingsRecord) => SaveSettings(settingsModel.Settings.createFrom(settings)) as Promise<SettingsRecord>,
	detectShells: () => DetectShells() as Promise<string[]>,
  createTerminal: (taskID: string, columns: number, rows: number) => CreateTerminal(taskID, columns, rows) as Promise<TerminalRecord>,
  createCommandTerminal: (taskID: string, command: string, arguments_: string[], columns: number, rows: number) => CreateCommandTerminal(taskID, command, arguments_, columns, rows) as Promise<TerminalRecord>,
  openTaskFolder: (taskID: string) => OpenTaskFolder(taskID),
  runTaskCommand: (taskID: string, command: string, arguments_: string[]) => RunTaskCommand(taskID, command, arguments_),
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
