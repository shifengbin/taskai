package application

import (
	"taskai/internal/settings"
	"taskai/internal/task"
	"taskai/internal/terminal"
)

type TaskBinding interface {
	CreateTask(title, description, color string) (task.Task, error)
	CreateTaskWithExtraInfo(title, description, color string, extraInfo []task.TaskExtraInfo) (task.Task, error)
	CreateTaskWithExtraInfoAndTemplateFields(title, description, color string, extraInfo []task.TaskExtraInfo, templateFields map[string]any) (task.Task, error)
	CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(title, description, color string, extraInfo []task.TaskExtraInfo, templateFields map[string]any, chains map[task.LifecycleHook]string) (task.Task, error)
	ListTasks() ([]task.Task, error)
	DeleteCompletedTasks(taskIDs []string) ([]task.Task, error)
	ReorderTasks(status task.Status, taskIDs []string) ([]task.Task, error)
	SetTaskShelved(taskID string, shelved bool) ([]task.Task, error)
	UpdateTask(taskID, title, description, color string) (task.Task, error)
	UpdateTaskWithExtraInfo(taskID, title, description, color string, extraInfo []task.TaskExtraInfo) (task.Task, error)
	UpdateTaskWithExtraInfoAndTemplateFields(taskID, title, description, color string, extraInfo []task.TaskExtraInfo, templateFields map[string]any) (task.Task, error)
	UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(taskID, title, description, color string, extraInfo []task.TaskExtraInfo, templateFields map[string]any, chains map[task.LifecycleHook]string) (task.Task, error)
	StartTask(taskID string) (task.Task, error)
	FinishTask(taskID string) (task.Task, error)
}

type ExtraInfoBinding interface {
	ListExtraInfoCatalogues() ([]string, error)
	SaveExtraInfoCatalogue(name string) (string, error)
	DeleteExtraInfoCatalogue(name string) error
	ListExtraInfoTemplates() ([]task.ExtraInfoTemplate, error)
	SaveExtraInfoTemplate(template task.ExtraInfoTemplate) (task.ExtraInfoTemplate, error)
	DeleteExtraInfoTemplate(templateID string) error
	ListExtraInfos() ([]task.ExtraInfo, error)
	SaveExtraInfo(info task.ExtraInfo) (task.ExtraInfo, error)
	DeleteExtraInfo(infoID string) error
}

type TerminalBinding interface {
	CreateTerminal(taskID string, columns, rows uint16) (terminal.Info, error)
	CreateCommandTerminal(taskID, command string, arguments []string, columns, rows uint16) (terminal.Info, error)
	ExecuteTaskMenuCommand(taskID, itemID string, columns, rows uint16) (TaskMenuCommandResult, error)
	RunTaskCommand(taskID, command string, arguments []string) error
	WriteTerminal(taskID, terminalID, data string) error
	ResizeTerminal(taskID, terminalID string, columns, rows uint16) error
	CloseTerminal(taskID, terminalID string) error
}

type TaskMenuCommandResult struct {
	Terminal *terminal.Info `json:"terminal,omitempty"`
}

type SettingsBinding interface {
	GetSettings() (settings.Settings, error)
	SaveSettings(next settings.Settings) (settings.Settings, error)
	DetectShells() []string
}

type LifecycleConfigurationBinding interface {
	ListLifecycleCommands() ([]settings.LifecycleCommand, error)
	SaveLifecycleCommand(command settings.LifecycleCommand) (settings.LifecycleCommand, error)
	DeleteLifecycleCommand(commandID string) error
	ListLifecycleCommandChains() ([]settings.LifecycleCommandChain, error)
	SaveLifecycleCommandChain(chain settings.LifecycleCommandChain) (settings.LifecycleCommandChain, error)
	CopyLifecycleCommandChain(chainID string) (settings.LifecycleCommandChain, error)
	DeleteLifecycleCommandChain(chainID string) error
	ListLifecyclePresets() ([]settings.LifecyclePreset, error)
	SaveLifecyclePreset(preset settings.LifecyclePreset) (settings.LifecyclePreset, error)
	CopyLifecyclePreset(presetID string) (settings.LifecyclePreset, error)
	DeleteLifecyclePreset(presetID string) error
	SaveDefaultLifecyclePreset(presetID string) (settings.Settings, error)
}

type EventPublisher interface {
	PublishTerminalEvent(event terminal.Event)
}

type RealtimeStatusBinding interface {
	ReportTerminalTitleActivity(taskID, terminalID string) bool
	SelectTerminal(taskID, terminalID string)
	ClearSelectedTerminal()
}
