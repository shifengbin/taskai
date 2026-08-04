import {
  ClearSelectedTerminal,
  CloseTerminal,
	CreateTask,
	CreateTaskWithExtraInfo,
	CreateTaskWithExtraInfoAndLifecycleChains,
	CreateTaskWithExtraInfoAndTemplateFields,
	CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains,
	CopyLifecycleCommandChain,
	CreateCommandTerminal,
	CreateTerminal,
	DeleteCompletedTasks,
	DetectShells,
	ExecuteTaskMenuCommand,
  FinishTask,
	GetSettings,
	GetLifecycleCommandInput,
	DeleteExtraInfoCatalogue,
	DeleteExtraInfo,
	DeleteExtraInfoTemplate,
	DeleteLifecycleCommand,
	DeleteLifecycleCommandChain,
  HasRunningTasks,
	ListTasks,
	ListExtraInfoCatalogues,
	ListExtraInfos,
	ListExtraInfoTemplates,
	ListLifecycleCommandChains,
	ListLifecycleCommands,
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
	SaveLifecycleCommand,
	SaveLifecycleCommandChain,
	SaveLifecycleDefaultChain,
	SelectTerminal,
	SetTaskShelved,
	StartTask,
	RetryTaskLifecycleCommandChain,
	UpdateTask,
	UpdateTaskWithExtraInfo,
	UpdateTaskWithExtraInfoAndLifecycleChains,
	UpdateTaskWithExtraInfoAndTemplateFields,
	UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains,
  WriteTerminal,
} from '../wailsjs/go/main/App'
import {settings as settingsModel} from '../wailsjs/go/models'
import {task as taskModel} from '../wailsjs/go/models'
import {EventsOff, EventsOn, Quit} from '../wailsjs/runtime/runtime'

import type {ExtraInfo, ExtraInfoTemplate, LifecycleCommand, LifecycleCommandChain, LifecycleHook, RealtimeStatusEvent, SettingsRecord, TaskExtraInfo, TaskMenuCommandResult, TaskRecord, TaskTemplateValues, TerminalEvent, TerminalRecord} from './types'

export const api = {
  createTask: (title: string, description: string, color: string) => CreateTask(title, description, color) as Promise<TaskRecord>,
	createTaskWithExtraInfo: (title: string, description: string, color: string, extraInfo: TaskExtraInfo[]) => CreateTaskWithExtraInfo(title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item))) as Promise<TaskRecord>,
	createTaskWithExtraInfoAndLifecycleChains: (title: string, description: string, color: string, extraInfo: TaskExtraInfo[], chains: Partial<Record<LifecycleHook, string>>) => CreateTaskWithExtraInfoAndLifecycleChains(title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item)), chains) as Promise<TaskRecord>,
	createTaskWithExtraInfoAndTemplateFields: (title: string, description: string, color: string, extraInfo: TaskExtraInfo[], templateFields: TaskTemplateValues) => CreateTaskWithExtraInfoAndTemplateFields(title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item)), templateFields) as Promise<TaskRecord>,
	createTaskWithExtraInfoTemplateFieldsAndLifecycleChains: (title: string, description: string, color: string, extraInfo: TaskExtraInfo[], templateFields: TaskTemplateValues, chains: Partial<Record<LifecycleHook, string>>) => CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item)), templateFields, chains) as Promise<TaskRecord>,
  updateTask: (taskID: string, title: string, description: string, color: string) => UpdateTask(taskID, title, description, color) as Promise<TaskRecord>,
	updateTaskWithExtraInfo: (taskID: string, title: string, description: string, color: string, extraInfo: TaskExtraInfo[]) => UpdateTaskWithExtraInfo(taskID, title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item))) as Promise<TaskRecord>,
	updateTaskWithExtraInfoAndLifecycleChains: (taskID: string, title: string, description: string, color: string, extraInfo: TaskExtraInfo[], chains: Partial<Record<LifecycleHook, string>>) => UpdateTaskWithExtraInfoAndLifecycleChains(taskID, title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item)), chains) as Promise<TaskRecord>,
	updateTaskWithExtraInfoAndTemplateFields: (taskID: string, title: string, description: string, color: string, extraInfo: TaskExtraInfo[], templateFields: TaskTemplateValues) => UpdateTaskWithExtraInfoAndTemplateFields(taskID, title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item)), templateFields) as Promise<TaskRecord>,
	updateTaskWithExtraInfoTemplateFieldsAndLifecycleChains: (taskID: string, title: string, description: string, color: string, extraInfo: TaskExtraInfo[], templateFields: TaskTemplateValues, chains: Partial<Record<LifecycleHook, string>>) => UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(taskID, title, description, color, extraInfo.map((item) => taskModel.TaskExtraInfo.createFrom(item)), templateFields, chains) as Promise<TaskRecord>,
  listTasks: () => ListTasks() as Promise<TaskRecord[]>,
	deleteCompletedTasks: (taskIDs: string[]) => DeleteCompletedTasks(taskIDs) as Promise<TaskRecord[]>,
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
	listLifecycleCommands: async () => {
		const commands = await ListLifecycleCommands()
		return Array.isArray(commands) ? commands as LifecycleCommand[] : []
	},
	saveLifecycleCommand: (command: LifecycleCommand) => SaveLifecycleCommand(command) as Promise<LifecycleCommand>,
	deleteLifecycleCommand: (commandID: string) => DeleteLifecycleCommand(commandID),
	listLifecycleCommandChains: async () => {
		const chains = await ListLifecycleCommandChains()
		return Array.isArray(chains) ? chains as LifecycleCommandChain[] : []
	},
	saveLifecycleCommandChain: (chain: LifecycleCommandChain) => SaveLifecycleCommandChain(settingsModel.LifecycleCommandChain.createFrom(chain)) as Promise<LifecycleCommandChain>,
	copyLifecycleCommandChain: (chainID: string) => CopyLifecycleCommandChain(chainID) as Promise<LifecycleCommandChain>,
	deleteLifecycleCommandChain: (chainID: string) => DeleteLifecycleCommandChain(chainID),
	saveLifecycleDefaultChain: (hook: LifecycleHook, chainID: string) => SaveLifecycleDefaultChain(hook, chainID) as Promise<SettingsRecord>,
	reorderTasks: (status: TaskRecord['status'], taskIDs: string[]) => ReorderTasks(status, taskIDs) as Promise<TaskRecord[]>,
	setTaskShelved: (taskID: string, shelved: boolean) => SetTaskShelved(taskID, shelved) as Promise<TaskRecord[]>,
	startTask: (taskID: string) => StartTask(taskID) as Promise<TaskRecord>,
	retryTaskLifecycleCommandChain: (taskID: string) => RetryTaskLifecycleCommandChain(taskID) as Promise<TaskRecord>,
  finishTask: (taskID: string) => FinishTask(taskID) as Promise<TaskRecord>,
	getSettings: () => GetSettings() as Promise<SettingsRecord>,
	getLifecycleCommandInput: (taskID: string) => GetLifecycleCommandInput(taskID),
	saveSettings: (settings: SettingsRecord) => {
		const payload = settingsModel.Settings.createFrom(settings)
		delete (payload as {lifecycleCommands?: unknown}).lifecycleCommands
		delete (payload as {lifecycleChains?: unknown}).lifecycleChains
		delete (payload as {lifecycleDefaultChains?: unknown}).lifecycleDefaultChains
		if (settings.taskTemplates !== undefined) {
			Object.assign(payload, {
				taskTemplates: settings.taskTemplates,
				activeTaskTemplateId: settings.activeTaskTemplateId ?? '',
			})
		} else {
			delete (payload as {taskTemplates?: unknown}).taskTemplates
			delete (payload as {activeTaskTemplateId?: unknown}).activeTaskTemplateId
		}
		return SaveSettings(payload) as Promise<SettingsRecord>
	},
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
	onLifecycleEvent(listener: (task: TaskRecord) => void) {
		EventsOn('task-lifecycle:event', listener)
		return () => EventsOff('task-lifecycle:event')
	},
}
