package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"taskai/internal/application"
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

	repository           storage.Repository
	tasks                *lifecycle.Service
	terminals            *terminal.Manager
	directoryOpener      func(string) error
	commandRunner        func(string, string, string, []string) error
	commandStarter       func(string, string, string, []string) (commandWaiter, error)
	scriptRunner         func(string, string, string, []string, []byte) error
	scriptStarter        func(string, string, string, []string, []byte) (commandWaiter, error)
	scriptErrorPublisher func(string, string)
	endingTasks          map[string]bool
}

type commandWaiter interface {
	Wait() error
}

type taskCommandInvocation struct {
	taskID    string
	directory string
	shellPath string
	command   string
	arguments []string
}

type taskCommandScriptPayload struct {
	TaskID    string   `json:"taskId"`
	Directory string   `json:"directory"`
	Command   string   `json:"command"`
	Arguments []string `json:"arguments"`
}

func NewApp() *App {
	return newApp(defaultDataDirectory())
}

func newApp(dataDirectory string) *App {
	defaults := settings.Default(dataDirectory)
	repository := storage.New(filepath.Join(dataDirectory, "tasks.json"), defaults)
	app := &App{
		repository:      repository,
		directoryOpener: openDirectory,
		commandRunner:   runTaskCommand,
		commandStarter:  startTaskCommand,
		scriptRunner:    runTaskScript,
		scriptStarter:   startTaskScript,
		endingTasks:     make(map[string]bool),
	}
	app.terminals = terminal.NewManager(terminal.NewBackend(), app.publishTerminalEvent)
	app.tasks = lifecycle.New(repository, app.terminals, time.Now)
	app.scriptErrorPublisher = app.publishTaskScriptError
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

func (app *App) ReorderTasks(status task.Status, taskIDs []string) ([]task.Task, error) {
	return app.tasks.ReorderTasks(status, taskIDs)
}

func (app *App) UpdateTask(taskID, title, description, color string) (task.Task, error) {
	return app.tasks.UpdateTask(taskID, title, description, color)
}

func (app *App) StartTask(taskID string) (task.Task, error) {
	return app.tasks.StartTask(taskID)
}

func (app *App) FinishTask(taskID string) (task.Task, error) {
	app.mu.Lock()
	app.endingTasks[taskID] = true
	app.mu.Unlock()
	finished, err := app.tasks.FinishTask(taskID)
	app.mu.Lock()
	delete(app.endingTasks, taskID)
	app.mu.Unlock()
	return finished, err
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

func (app *App) CreateCommandTerminal(taskID, command string, arguments []string, columns, rows uint16) (terminal.Info, error) {
	running, shellPath, err := app.runningTask(taskID)
	if err != nil {
		return terminal.Info{}, err
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return terminal.Info{}, fmt.Errorf("任务命令不能为空")
	}
	return app.terminals.CreateCommand(taskID, running.WorkspacePath, shellPath, command, arguments, columns, rows)
}

func (app *App) RunTaskCommand(taskID, command string, arguments []string) error {
	running, shellPath, err := app.runningTask(taskID)
	if err != nil {
		return err
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("任务命令不能为空")
	}
	return app.commandRunner(running.WorkspacePath, shellPath, command, append([]string(nil), arguments...))
}

func (app *App) ExecuteTaskMenuCommand(taskID, itemID string, columns, rows uint16) (application.TaskMenuCommandResult, error) {
	invocation, item, err := app.taskMenuCommand(taskID, itemID)
	if err != nil {
		return application.TaskMenuCommandResult{}, err
	}
	if err := app.runTaskScript(invocation, item.BeforeScript); err != nil {
		return application.TaskMenuCommandResult{}, fmt.Errorf("执行前置脚本: %w", err)
	}
	if item.ShowTerminal {
		created, err := app.terminals.CreateCommand(taskID, invocation.directory, invocation.shellPath, invocation.command, invocation.arguments, columns, rows)
		if err != nil {
			return application.TaskMenuCommandResult{}, err
		}
		afterScript := cloneTaskScript(item.AfterScript)
		app.terminals.OnExit(taskID, created.ID, func() { app.runAfterScript(invocation, afterScript) })
		return application.TaskMenuCommandResult{Terminal: &created}, nil
	}
	process, err := app.commandStarter(invocation.directory, invocation.shellPath, invocation.command, invocation.arguments)
	if err != nil {
		return application.TaskMenuCommandResult{}, err
	}
	afterScript := cloneTaskScript(item.AfterScript)
	go func() {
		_ = process.Wait()
		app.runAfterScript(invocation, afterScript)
	}()
	return application.TaskMenuCommandResult{}, nil
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

func (app *App) taskMenuCommand(taskID, itemID string) (taskCommandInvocation, settings.TaskMenuItem, error) {
	running, shellPath, err := app.runningTask(taskID)
	if err != nil {
		return taskCommandInvocation{}, settings.TaskMenuItem{}, err
	}
	data, err := app.repository.Load()
	if err != nil {
		return taskCommandInvocation{}, settings.TaskMenuItem{}, err
	}
	for _, item := range data.Settings.TaskMenuItems {
		if item.ID != itemID {
			continue
		}
		if item.Kind != settings.TaskMenuItemKindCommand {
			return taskCommandInvocation{}, settings.TaskMenuItem{}, fmt.Errorf("任务菜单项不是自定义命令")
		}
		return taskCommandInvocation{
			taskID: taskID, directory: running.WorkspacePath, shellPath: shellPath, command: item.Command, arguments: append([]string(nil), item.Arguments...),
		}, item, nil
	}
	return taskCommandInvocation{}, settings.TaskMenuItem{}, fmt.Errorf("任务菜单项不存在")
}

func (app *App) runTaskScript(invocation taskCommandInvocation, script *settings.TaskScript) error {
	if script == nil {
		return nil
	}
	payload, err := taskScriptInput(invocation)
	if err != nil {
		return err
	}
	return app.scriptRunner(invocation.directory, invocation.shellPath, script.Script, append([]string(nil), script.Arguments...), payload)
}

func (app *App) runAfterScript(invocation taskCommandInvocation, script *settings.TaskScript) {
	if script == nil {
		return
	}

	// The finish marker and process start share this lock so an ending task cannot
	// start a post script after its workspace cleanup begins.
	app.mu.Lock()
	if app.endingTasks[invocation.taskID] {
		app.mu.Unlock()
		return
	}
	if _, _, err := app.runningTask(invocation.taskID); err != nil {
		app.mu.Unlock()
		return
	}
	process, err := app.startTaskScript(invocation, script)
	app.mu.Unlock()
	if err != nil {
		app.scriptErrorPublisher(invocation.taskID, fmt.Sprintf("执行后置脚本: %v", err))
		return
	}
	go func() {
		if err := process.Wait(); err != nil {
			app.scriptErrorPublisher(invocation.taskID, fmt.Sprintf("执行后置脚本: %v", err))
		}
	}()
}

func (app *App) startTaskScript(invocation taskCommandInvocation, script *settings.TaskScript) (commandWaiter, error) {
	payload, err := taskScriptInput(invocation)
	if err != nil {
		return nil, err
	}
	return app.scriptStarter(invocation.directory, invocation.shellPath, script.Script, append([]string(nil), script.Arguments...), payload)
}

func taskScriptInput(invocation taskCommandInvocation) ([]byte, error) {
	payload, err := json.Marshal(taskCommandScriptPayload{
		TaskID: invocation.taskID, Directory: invocation.directory, Command: invocation.command, Arguments: append([]string{}, invocation.arguments...),
	})
	if err != nil {
		return nil, fmt.Errorf("编码任务上下文: %w", err)
	}
	return payload, nil
}

func cloneTaskScript(script *settings.TaskScript) *settings.TaskScript {
	if script == nil {
		return nil
	}
	return &settings.TaskScript{Script: script.Script, Arguments: append([]string(nil), script.Arguments...)}
}

func (app *App) publishTerminalEvent(event terminal.Event) {
	app.mu.RLock()
	ctx := app.ctx
	app.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "task-terminal:event", event)
	}
}

func (app *App) publishTaskScriptError(_ string, message string) {
	app.mu.RLock()
	ctx := app.ctx
	app.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "task-script:error", message)
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

func runTaskCommand(directory, shellPath, command string, arguments []string) error {
	process, err := startTaskCommand(directory, shellPath, command, arguments)
	if err != nil {
		return err
	}
	go func() { _ = process.Wait() }()
	return nil
}

func startTaskCommand(directory, shellPath, command string, arguments []string) (commandWaiter, error) {
	process := commandProcess(shellPath, command, arguments)
	configureCommandProcess(process, directory)
	if err := process.Start(); err != nil {
		return nil, fmt.Errorf("启动任务命令: %w", err)
	}
	return process, nil
}

func runTaskScript(directory, shellPath, script string, arguments []string, input []byte) error {
	process, err := startTaskScript(directory, shellPath, script, arguments, input)
	if err != nil {
		return err
	}
	if err := process.Wait(); err != nil {
		return fmt.Errorf("执行脚本: %w", err)
	}
	return nil
}

func startTaskScript(directory, shellPath, script string, arguments []string, input []byte) (commandWaiter, error) {
	process := commandProcess(shellPath, script, arguments)
	configureCommandProcess(process, directory)
	process.Stdin = bytes.NewReader(input)
	if err := process.Start(); err != nil {
		return nil, fmt.Errorf("启动脚本: %w", err)
	}
	return process, nil
}

func commandProcess(shellPath, command string, arguments []string) *exec.Cmd {
	return commandProcessForPlatform(goruntime.GOOS, shellPath, command, arguments)
}

func commandProcessForPlatform(platform, shellPath, command string, arguments []string) *exec.Cmd {
	if shellPath == "" {
		return exec.Command(command, arguments...)
	}

	switch {
	case platform == "windows" && isPowerShell(shellPath):
		encodedArguments, _ := json.Marshal(append([]string{}, arguments...))
		process := exec.Command(shellPath, "-NoLogo", "-Command", `$taskaiArguments = ConvertFrom-Json -InputObject $env:TASKAI_EXEC_ARGUMENTS; & $env:TASKAI_EXEC_COMMAND @($taskaiArguments)`)
		process.Env = append(os.Environ(), "TASKAI_EXEC_COMMAND="+command, "TASKAI_EXEC_ARGUMENTS="+string(encodedArguments))
		return process
	case platform == "windows" && shellName(shellPath) == "cmd":
		shellArguments := append([]string{"/C", command}, arguments...)
		return exec.Command(shellPath, shellArguments...)
	case platform != "windows" && shellName(shellPath) == "fish":
		shellArguments := append([]string{"-ic", "exec $argv[2..-1]", shellPath, command}, arguments...)
		return exec.Command(shellPath, shellArguments...)
	default:
		shellArguments := append([]string{"-ic", `exec "$@"`, shellPath, command}, arguments...)
		return exec.Command(shellPath, shellArguments...)
	}
}

func configureCommandProcess(process *exec.Cmd, directory string) {
	process.Dir = directory
	if process.Env == nil {
		process.Env = os.Environ()
	}
}

func isPowerShell(shellPath string) bool {
	name := shellName(shellPath)
	return name == "powershell" || name == "pwsh"
}

func shellName(shellPath string) string {
	if index := strings.LastIndexAny(shellPath, `/\\`); index >= 0 {
		shellPath = shellPath[index+1:]
	}
	return strings.TrimSuffix(strings.ToLower(shellPath), ".exe")
}
