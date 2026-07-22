package application

import (
	"taskai/internal/settings"
	"taskai/internal/task"
	"taskai/internal/terminal"
)

type TaskBinding interface {
	CreateTask(title, description, color string) (task.Task, error)
	ListTasks() ([]task.Task, error)
	UpdateTask(taskID, title, description, color string) (task.Task, error)
	StartTask(taskID string) (task.Task, error)
	FinishTask(taskID string) (task.Task, error)
}

type TerminalBinding interface {
	CreateTerminal(taskID string, columns, rows uint16) (terminal.Info, error)
	CreateCommandTerminal(taskID, command string, arguments []string, columns, rows uint16) (terminal.Info, error)
	RunTaskCommand(taskID, command string, arguments []string) error
	WriteTerminal(taskID, terminalID, data string) error
	ResizeTerminal(taskID, terminalID string, columns, rows uint16) error
	CloseTerminal(taskID, terminalID string) error
}

type SettingsBinding interface {
	GetSettings() (settings.Settings, error)
	SaveSettings(next settings.Settings) (settings.Settings, error)
	DetectShells() []string
}

type EventPublisher interface {
	PublishTerminalEvent(event terminal.Event)
}
