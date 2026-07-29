import {
  ClearSelectedTerminal,
  CloseTerminal,
	CreateTask,
	CreateTaskWithExtraInfo,
	CreateCommandTerminal,
	CreateTerminal,
	DetectShells,
	ExecuteTaskMenuCommand,
  FinishTask,
	GetSettings,
	DeleteExtraInfoCatalogue,
	DeleteExtraInfo,
	DeleteExtraInfoTemplate,
  HasRunningTasks,
	ListTasks,
	ListExtraInfoCatalogues,
	ListExtraInfos,
	ListExtraInfoTemplates,
	OpenTaskFolder,
  PrepareQuit,
  ReorderTasks,
	ReportTerminalTitleActivity,
  ResizeTerminal,
  RunTaskCommand,
	SaveSettings,
	SaveExtraInfoCatalogue,
	SaveExtraInfo,
	SaveExtraInfoTemplate,
	SelectTerminal,
	StartTask,
	UpdateTask,
	UpdateTaskWithExtraInfo,
  WriteTerminal,
} from '../wailsjs/go/main/App'
import {settings as settingsModel} from '../wailsjs/go/models'
import {task as taskModel} from '../wailsjs/go/models'
import {EventsOff, EventsOn, Quit} from '../wailsjs/runtime/runtime'

import type {ExtraInfo, ExtraInfoTemplate, RealtimeStatusEvent, SettingsRecord, TaskExtraInfo, TaskMenuCommandResult, TaskRecord, TerminalEvent, TerminalRecord} from './types'

export const api = {
  createTask: (title: string, description: string, color: string) => CreateTask(title, description, color) as Promise<TaskRecord>,
	createTaskWithExtraInfo: (title: string, description: string, color: string, extraInfo: TaskExtraInfo[]) => CreateTaskWithExtraInfo(title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item))) as Promise<TaskRecord>,
  updateTask: (taskID: string, title: string, description: string, color: string) => UpdateTask(taskID, title, description, color) as Promise<TaskRecord>,
	updateTaskWithExtraInfo: (taskID: string, title: string, description: string, color: string, extraInfo: TaskExtraInfo[]) => UpdateTaskWithExtraInfo(taskID, title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item))) as Promise<TaskRecord>,
  listTasks: () => ListTasks() as Promise<TaskRecord[]>,
	listExtraInfoTemplates: async () => {
		const templates = await ListExtraInfoTemplates()
		return Array.isArray(templates) ? templates as unknown as ExtraInfoTemplate[] : []
	},
	listExtraInfoCatalogues: async () => {
		const catalogues = await ListExtraInfoCatalogues()
		return Array.isArray(catalogues) ? catalogues as string[] : []
	},
	saveExtraInfoCatalogue: (name: string) => SaveExtraInfoCatalogue(name),
	deleteExtraInfoCatalogue: (name: string) => DeleteExtraInfoCatalogue(name),
	saveExtraInfoTemplate: (template: ExtraInfoTemplate) => SaveExtraInfoTemplate(taskModel.ExtraInfoTemplate.createFrom(template)) as unknown as Promise<ExtraInfoTemplate>,
	deleteExtraInfoTemplate: (templateID: string) => DeleteExtraInfoTemplate(templateID),
	listExtraInfos: async () => {
		const infos = await ListExtraInfos()
		return Array.isArray(infos) ? infos as ExtraInfo[] : []
	},
	saveExtraInfo: (info: ExtraInfo) => SaveExtraInfo(taskModel.ExtraInfo.createFrom(info)) as unknown as Promise<ExtraInfo>,
	deleteExtraInfo: (infoID: string) => DeleteExtraInfo(infoID),
	reorderTasks: (status: TaskRecord['status'], taskIDs: string[]) => ReorderTasks(status, taskIDs) as Promise<TaskRecord[]>,
  startTask: (taskID: string) => StartTask(taskID) as Promise<TaskRecord>,
  finishTask: (taskID: string) => FinishTask(taskID) as Promise<TaskRecord>,
  getSettings: () => GetSettings() as Promise<SettingsRecord>,
	saveSettings: (settings: SettingsRecord) => SaveSettings(settingsModel.Settings.createFrom(settings)) as Promise<SettingsRecord>,
	detectShells: () => DetectShells() as Promise<string[]>,
  createTerminal: (taskID: string, columns: number, rows: number) => CreateTerminal(taskID, columns, rows) as Promise<TerminalRecord>,
  createCommandTerminal: (taskID: string, command: string, arguments_: string[], columns: number, rows: number) => CreateCommandTerminal(taskID, command, arguments_, columns, rows) as Promise<TerminalRecord>,
	executeTaskMenuCommand: (taskID: string, itemID: string, columns: number, rows: number) => ExecuteTaskMenuCommand(taskID, itemID, columns, rows) as Promise<TaskMenuCommandResult>,
  openTaskFolder: (taskID: string) => OpenTaskFolder(taskID),
  runTaskCommand: (taskID: string, command: string, arguments_: string[]) => RunTaskCommand(taskID, command, arguments_),
  writeTerminal: (taskID: string, terminalID: string, data: string) => WriteTerminal(taskID, terminalID, data),
  resizeTerminal: (taskID: string, terminalID: string, columns: number, rows: number) => ResizeTerminal(taskID, terminalID, columns, rows),
  closeTerminal: (taskID: string, terminalID: string) => CloseTerminal(taskID, terminalID),
	reportTerminalTitleActivity: (taskID: string, terminalID: string) => ReportTerminalTitleActivity(taskID, terminalID),
	selectTerminal: (taskID: string, terminalID: string) => SelectTerminal(taskID, terminalID),
	clearSelectedTerminal: () => ClearSelectedTerminal(),
  hasRunningTasks: () => HasRunningTasks(),
  prepareQuit: () => PrepareQuit(),
  quit: () => Quit(),
  onTerminalEvent(listener: (event: TerminalEvent) => void) {
    EventsOn('task-terminal:event', listener)
    return () => EventsOff('task-terminal:event')
  },
	onRealtimeStatusEvent(listener: (event: RealtimeStatusEvent) => void) {
		EventsOn('task-realtime-status:event', listener)
		return () => EventsOff('task-realtime-status:event')
	},
	onRealtimeStatusError(listener: (message: string) => void) {
		EventsOn('task-realtime-status:error', listener)
		return () => EventsOff('task-realtime-status:error')
	},
  onCloseRequested(listener: () => void) {
	EventsOn('application:close-requested', listener)
	return () => EventsOff('application:close-requested')
  },
	onTaskScriptError(listener: (message: string) => void) {
		EventsOn('task-script:error', listener)
		return () => EventsOff('task-script:error')
	},
}
