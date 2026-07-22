package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"taskai/internal/lifecycle"
	"taskai/internal/settings"
	"taskai/internal/storage"
	"taskai/internal/task"
	"taskai/internal/terminal"
)

type App struct {
	mu         sync.RWMutex
	ctx        context.Context
	allowClose bool

	repository      storage.Repository
	tasks           *lifecycle.Service
	terminals       *terminal.Manager
	directoryOpener func(string) error
}

func NewApp() *App {
	return newApp(defaultDataDirectory())
}

func newApp(dataDirectory string) *App {
	defaults := settings.Default(dataDirectory)
	repository := storage.New(filepath.Join(dataDirectory, "tasks.json"), defaults)
	app := &App{repository: repository, directoryOpener: openDirectory}
	app.terminals = terminal.NewManager(terminal.NewBackend(), app.publishTerminalEvent)
	app.tasks = lifecycle.New(repository, app.terminals, time.Now)
	return app
}

func defaultDataDirectory() string {
	configurationDirectory, err := os.UserConfigDir()
	if err == nil {
		return filepath.Join(configurationDirectory, "taskai")
	}
	return filepath.Join(os.TempDir(), "taskai")
}

func (app *App) startup(ctx context.Context) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.ctx = ctx
}

func (app *App) shutdown(context.Context) {
	_ = app.terminals.CloseAll()
}

func (app *App) beforeClose(ctx context.Context) bool {
	app.mu.RLock()
	allowClose := app.allowClose
	app.mu.RUnlock()
	if allowClose || !app.HasRunningTasks() {
		return false
	}
	runtime.EventsEmit(ctx, "application:close-requested")
	return true
}

func (app *App) CreateTask(title, description, color string) (task.Task, error) {
	return app.tasks.CreateTask(title, description, color)
}

func (app *App) ListTasks() ([]task.Task, error) {
	return app.tasks.ListTasks()
}

func (app *App) StartTask(taskID string) (task.Task, error) {
	return app.tasks.StartTask(taskID)
}

func (app *App) FinishTask(taskID string) (task.Task, error) {
	return app.tasks.FinishTask(taskID)
}

func (app *App) GetSettings() (settings.Settings, error) {
	data, err := app.repository.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	return data.Settings, nil
}

func (app *App) SaveSettings(next settings.Settings) (settings.Settings, error) {
	validated, err := settings.Validate(next)
	if err != nil {
		return settings.Settings{}, err
	}
	return app.repository.SaveSettings(validated)
}

func (app *App) DetectShells() []string {
	return settings.DetectShells()
}

func (app *App) CreateTerminal(taskID string, columns, rows uint16) (terminal.Info, error) {
	running, shellPath, err := app.runningTask(taskID)
	if err != nil {
		return terminal.Info{}, err
	}
	return app.terminals.Create(taskID, running.WorkspacePath, shellPath, columns, rows)
}

func (app *App) OpenTaskFolder(taskID string) error {
	running, _, err := app.runningTask(taskID)
	if err != nil {
		return err
	}
	return app.directoryOpener(running.WorkspacePath)
}

func (app *App) WriteTerminal(taskID, terminalID, data string) error {
	return app.terminals.Write(taskID, terminalID, data)
}

func (app *App) ResizeTerminal(taskID, terminalID string, columns, rows uint16) error {
	return app.terminals.Resize(taskID, terminalID, columns, rows)
}

func (app *App) CloseTerminal(taskID, terminalID string) error {
	return app.terminals.Close(taskID, terminalID)
}

func (app *App) HasRunningTasks() bool {
	tasks, err := app.tasks.ListTasks()
	if err != nil {
		return true
	}
	for _, current := range tasks {
		if current.Status == task.StatusRunning {
			return true
		}
	}
	return false
}

// PrepareQuit closes only live PTY sessions. It intentionally preserves task status and workspaces.
func (app *App) PrepareQuit() error {
	if err := app.terminals.CloseAll(); err != nil {
		return err
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	app.allowClose = true
	return nil
}

func (app *App) runningTask(taskID string) (task.Task, string, error) {
	data, err := app.repository.Load()
	if err != nil {
		return task.Task{}, "", err
	}
	for _, current := range data.Tasks {
		if current.ID != taskID {
			continue
		}
		if current.Status != task.StatusRunning {
			return task.Task{}, "", fmt.Errorf("仅执行中的任务可以创建终端")
		}
		if current.WorkspacePath == "" {
			return task.Task{}, "", fmt.Errorf("任务缺少工作目录")
		}
		return current, data.Settings.ShellPath, nil
	}
	return task.Task{}, "", fmt.Errorf("任务不存在")
}

func (app *App) publishTerminalEvent(event terminal.Event) {
	app.mu.RLock()
	ctx := app.ctx
	app.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "task-terminal:event", event)
	}
}

func openDirectory(directory string) error {
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("检查任务工作目录: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("任务工作目录不是目录")
	}

	var command *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		command = exec.Command("open", directory)
	case "windows":
		command = exec.Command("explorer.exe", directory)
	default:
		command = exec.Command("xdg-open", directory)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("打开任务工作目录: %w", err)
	}
	return nil
}
