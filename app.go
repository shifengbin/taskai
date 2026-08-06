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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"taskai/internal/application"
	"taskai/internal/lifecycle"
	"taskai/internal/realtime"
	"taskai/internal/settings"
	"taskai/internal/storage"
	"taskai/internal/task"
	"taskai/internal/terminal"
)

type App struct {
	mu               sync.RWMutex
	ctx              context.Context
	allowClose       bool
	lifecycleLockMu  sync.Mutex
	lifecycleTaskMux map[string]*sync.Mutex

	repository             *storage.Repository
	tasks                  *lifecycle.Service
	terminals              *terminal.Manager
	realtime               *realtime.Service
	statusHTTP             *realtime.HTTPServer
	directoryOpener        func(string) error
	homeDirectory          func() (string, error)
	commandRunner          func(string, string, string, []string, []string) error
	commandStarter         func(string, string, string, []string, []string) (commandWaiter, error)
	scriptRunner           func(string, string, string, []string, []byte, []string) error
	scriptStarter          func(string, string, string, []string, []byte, []string) (commandWaiter, error)
	scriptErrorPublisher   func(string, string)
	lifecycleCommandRunner *lifecycle.CommandChainRunner
	endingTasks            map[string]bool
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
		repository:       repository,
		directoryOpener:  openDirectory,
		homeDirectory:    os.UserHomeDir,
		commandRunner:    runTaskCommand,
		commandStarter:   startTaskCommand,
		scriptRunner:     runTaskScript,
		scriptStarter:    startTaskScript,
		endingTasks:      make(map[string]bool),
		lifecycleTaskMux: make(map[string]*sync.Mutex),
	}
	app.realtime = realtime.New(realtime.Options{Publish: app.publishRealtimeStatusEvent})
	app.statusHTTP = realtime.NewHTTPServer(realtime.HTTPServerOptions{
		Service:     app.realtime,
		ResolveTask: app.realtimeTaskState,
		TaskCatalog: realtime.TaskCatalog{
			List: app.httpTasks,
			Get:  app.httpTask,
		},
	})
	app.terminals = terminal.NewManager(terminal.NewBackend(), app.publishTerminalEvent)
	app.tasks = lifecycle.New(repository, app.terminals, time.Now)
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.NewShellCommandExecutor())
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
	app.ctx = ctx
	app.mu.Unlock()
	if current, err := app.GetSettings(); err == nil {
		if err := app.applyStatusSettings(current); err != nil {
			app.publishRealtimeStatusError(err.Error())
		}
	}
	app.registerRunningRealtimeTasks()
}

func (app *App) shutdown(context.Context) {
	_ = app.statusHTTP.Close()
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

func (app *App) CreateTaskWithExtraInfo(title, description, color string, extraInfo []task.TaskExtraInfo) (task.Task, error) {
	return app.tasks.CreateTaskWithExtraInfo(title, description, color, extraInfo)
}

func (app *App) CreateTaskWithExtraInfoAndTemplateFields(title, description, color string, extraInfo []task.TaskExtraInfo, templateFields map[string]any) (task.Task, error) {
	return app.tasks.CreateTaskWithExtraInfoAndTemplateFields(title, description, color, extraInfo, templateFields)
}

func (app *App) CreateTaskWithExtraInfoAndLifecycleChains(title, description, color string, extraInfo []task.TaskExtraInfo, chains map[task.LifecycleHook]string) (task.Task, error) {
	return app.tasks.CreateTaskWithExtraInfoAndLifecycleChains(title, description, color, extraInfo, chains)
}

func (app *App) CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(title, description, color string, extraInfo []task.TaskExtraInfo, templateFields map[string]any, chains map[task.LifecycleHook]string) (task.Task, error) {
	return app.tasks.CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(title, description, color, extraInfo, templateFields, chains)
}

func (app *App) ListTasks() ([]task.Task, error) {
	return app.tasks.ListTasks()
}

func (app *App) DeleteCompletedTasks(taskIDs []string) ([]task.Task, error) {
	unlock := app.lockLifecycleTasks(taskIDs)
	defer unlock()

	remaining, err := app.tasks.DeleteCompletedTasks(taskIDs)
	if err != nil {
		return nil, err
	}
	for _, taskID := range taskIDs {
		app.realtime.RemoveTask(taskID)
	}
	return remaining, nil
}

func (app *App) GetLifecycleCommandInput(taskID string) (string, error) {
	data, err := app.repository.Load()
	if err != nil {
		return "", err
	}
	for _, current := range data.Tasks {
		if current.ID != taskID {
			continue
		}
		input, err := app.lifecycleCommandInput(current, data.Settings)
		if err != nil {
			return "", err
		}
		return string(input), nil
	}
	return "", fmt.Errorf("任务不存在")
}

func (app *App) ReorderTasks(status task.Status, taskIDs []string) ([]task.Task, error) {
	items, err := app.tasks.ListTasks()
	if err != nil {
		return nil, err
	}
	for _, current := range items {
		if current.Status == status && current.IsLifecycleLocked() {
			return nil, fmt.Errorf("任务正在执行命令链，暂不能调整排序")
		}
	}
	return app.tasks.ReorderTasks(status, taskIDs)
}

func (app *App) SetTaskShelved(taskID string, shelved bool) ([]task.Task, error) {
	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	if current.Status != task.StatusRunning {
		return nil, fmt.Errorf("仅执行中任务可以切换搁置状态")
	}
	if current.IsLifecycleLocked() {
		return nil, fmt.Errorf("任务正在执行命令链，暂不能切换搁置状态")
	}
	return app.tasks.SetTaskShelved(taskID, shelved)
}

func (app *App) UpdateTask(taskID, title, description, color string) (task.Task, error) {
	unlock := app.lockLifecycleTask(taskID)
	defer unlock()

	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if current.IsLifecycleLocked() {
		return task.Task{}, fmt.Errorf("任务正在执行命令链，暂不能修改")
	}
	updated, err := app.tasks.UpdateTask(taskID, title, description, color)
	if err != nil {
		return task.Task{}, err
	}
	if updated.Status != task.StatusRunning {
		return updated, nil
	}
	return app.scheduleLifecycleHookLocked(updated, task.LifecycleHookUpdateTask)
}

func (app *App) UpdateTaskWithExtraInfo(taskID, title, description, color string, extraInfo []task.TaskExtraInfo) (task.Task, error) {
	unlock := app.lockLifecycleTask(taskID)
	defer unlock()

	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if current.IsLifecycleLocked() {
		return task.Task{}, fmt.Errorf("任务正在执行命令链，暂不能修改")
	}
	updated, err := app.tasks.UpdateTaskWithExtraInfo(taskID, title, description, color, extraInfo)
	if err != nil {
		return task.Task{}, err
	}
	if updated.Status != task.StatusRunning {
		return updated, nil
	}
	return app.scheduleLifecycleHookLocked(updated, task.LifecycleHookUpdateTask)
}

func (app *App) UpdateTaskWithExtraInfoAndTemplateFields(taskID, title, description, color string, extraInfo []task.TaskExtraInfo, templateFields map[string]any) (task.Task, error) {
	unlock := app.lockLifecycleTask(taskID)
	defer unlock()

	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if current.IsLifecycleLocked() {
		return task.Task{}, fmt.Errorf("任务正在执行命令链，暂不能修改")
	}
	updated, err := app.tasks.UpdateTaskWithExtraInfoAndTemplateFields(taskID, title, description, color, extraInfo, templateFields)
	if err != nil {
		return task.Task{}, err
	}
	if updated.Status != task.StatusRunning {
		return updated, nil
	}
	return app.scheduleLifecycleHookLocked(updated, task.LifecycleHookUpdateTask)
}

func (app *App) UpdateTaskWithExtraInfoAndLifecycleChains(taskID, title, description, color string, extraInfo []task.TaskExtraInfo, chains map[task.LifecycleHook]string) (task.Task, error) {
	unlock := app.lockLifecycleTask(taskID)
	defer unlock()

	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if current.IsLifecycleLocked() {
		return task.Task{}, fmt.Errorf("任务正在执行命令链，暂不能修改")
	}
	if current.Status != task.StatusPending {
		return task.Task{}, fmt.Errorf("仅未执行任务可以修改生命周期命令链")
	}
	return app.tasks.UpdateTaskWithExtraInfoAndLifecycleChains(taskID, title, description, color, extraInfo, chains)
}

func (app *App) UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(taskID, title, description, color string, extraInfo []task.TaskExtraInfo, templateFields map[string]any, chains map[task.LifecycleHook]string) (task.Task, error) {
	unlock := app.lockLifecycleTask(taskID)
	defer unlock()

	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if current.IsLifecycleLocked() {
		return task.Task{}, fmt.Errorf("任务正在执行命令链，暂不能修改")
	}
	if current.Status != task.StatusPending {
		return task.Task{}, fmt.Errorf("仅未执行任务可以修改生命周期命令链")
	}
	return app.tasks.UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(taskID, title, description, color, extraInfo, templateFields, chains)
}

func (app *App) ListExtraInfoTemplates() ([]task.ExtraInfoTemplate, error) {
	return app.repository.ListExtraInfoTemplates()
}

func (app *App) ListExtraInfoCatalogues() ([]string, error) {
	return app.repository.ListExtraInfoCatalogues()
}

func (app *App) SaveExtraInfoCatalogue(name string) (string, error) {
	return app.repository.SaveExtraInfoCatalogue(name)
}

func (app *App) DeleteExtraInfoCatalogue(name string) error {
	return app.repository.DeleteExtraInfoCatalogue(name)
}

func (app *App) SaveExtraInfoTemplate(template task.ExtraInfoTemplate) (task.ExtraInfoTemplate, error) {
	return app.repository.SaveExtraInfoTemplate(template)
}

func (app *App) DeleteExtraInfoTemplate(templateID string) error {
	return app.repository.DeleteExtraInfoTemplate(templateID)
}

func (app *App) ListExtraInfos() ([]task.ExtraInfo, error) {
	return app.repository.ListExtraInfos()
}

func (app *App) SaveExtraInfo(info task.ExtraInfo) (task.ExtraInfo, error) {
	return app.repository.SaveExtraInfo(info)
}

func (app *App) DeleteExtraInfo(infoID string) error {
	return app.repository.DeleteExtraInfo(infoID)
}

func (app *App) ListLifecycleCommands() ([]settings.LifecycleCommand, error) {
	return app.repository.ListLifecycleCommands()
}

func (app *App) SaveLifecycleCommand(command settings.LifecycleCommand) (settings.LifecycleCommand, error) {
	return app.repository.SaveLifecycleCommand(command)
}

func (app *App) DeleteLifecycleCommand(commandID string) error {
	return app.repository.DeleteLifecycleCommand(commandID)
}

func (app *App) ListLifecycleCommandChains() ([]settings.LifecycleCommandChain, error) {
	return app.repository.ListLifecycleCommandChains()
}

func (app *App) SaveLifecycleCommandChain(chain settings.LifecycleCommandChain) (settings.LifecycleCommandChain, error) {
	return app.repository.SaveLifecycleCommandChain(chain)
}

func (app *App) CopyLifecycleCommandChain(chainID string) (settings.LifecycleCommandChain, error) {
	return app.repository.CopyLifecycleCommandChain(chainID)
}

func (app *App) DeleteLifecycleCommandChain(chainID string) error {
	return app.repository.DeleteLifecycleCommandChain(chainID)
}

func (app *App) ListLifecyclePresets() ([]settings.LifecyclePreset, error) {
	return app.repository.ListLifecyclePresets()
}

func (app *App) SaveLifecyclePreset(preset settings.LifecyclePreset) (settings.LifecyclePreset, error) {
	return app.repository.SaveLifecyclePreset(preset)
}

func (app *App) CopyLifecyclePreset(presetID string) (settings.LifecyclePreset, error) {
	return app.repository.CopyLifecyclePreset(presetID)
}

func (app *App) DeleteLifecyclePreset(presetID string) error {
	return app.repository.DeleteLifecyclePreset(presetID)
}

func (app *App) SaveDefaultLifecyclePreset(presetID string) (settings.Settings, error) {
	return app.repository.SaveDefaultLifecyclePreset(presetID)
}

type lifecycleRun struct {
	task      task.Task
	hook      task.LifecycleHook
	execution task.LifecycleExecution
	request   lifecycle.CommandChainRequest
}

func (app *App) StartTask(taskID string) (task.Task, error) {
	unlock := app.lockLifecycleTask(taskID)
	defer unlock()

	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if current.IsLifecycleLocked() {
		return task.Task{}, fmt.Errorf("任务正在执行命令链，暂不能开始执行")
	}
	prepared, err := app.tasks.PrepareStartTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if app.hasLifecycleChain(prepared, task.LifecycleHookBeforeStart) {
		return app.scheduleLifecycleHookLocked(prepared, task.LifecycleHookBeforeStart)
	}
	return app.commitStartAndSchedulePostLocked(prepared)
}

func (app *App) FinishTask(taskID string) (task.Task, error) {
	unlock := app.lockLifecycleTask(taskID)
	defer unlock()

	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if current.IsLifecycleLocked() {
		return task.Task{}, fmt.Errorf("任务正在执行命令链，暂不能结束执行")
	}
	if app.hasLifecycleChain(current, task.LifecycleHookBeforeEnd) {
		return app.scheduleLifecycleHookLocked(current, task.LifecycleHookBeforeEnd)
	}
	return app.finishTaskAndSchedulePostLocked(taskID)
}

func (app *App) RetryTaskLifecycleCommandChain(taskID string) (task.Task, error) {
	unlock := app.lockLifecycleTask(taskID)
	defer unlock()

	current, err := app.tasks.GetTask(taskID)
	if err != nil {
		return task.Task{}, err
	}
	if current.LifecycleExecution == nil || current.LifecycleExecution.State != task.LifecycleExecutionFailed {
		return task.Task{}, fmt.Errorf("任务没有可重试的命令链")
	}
	hook := current.LifecycleExecution.Hook
	if !app.hasLifecycleChain(current, hook) {
		return task.Task{}, fmt.Errorf("任务没有可重试的命令链")
	}

	switch hook {
	case task.LifecycleHookBeforeStart:
		if current.Status != task.StatusPending {
			return task.Task{}, fmt.Errorf("开始前命令链只能在未执行任务上重试")
		}
		prepared, err := app.tasks.PrepareStartTask(taskID)
		if err != nil {
			return task.Task{}, err
		}
		return app.scheduleLifecycleHookLocked(prepared, hook)
	case task.LifecycleHookPostStart:
		if current.Status != task.StatusRunning {
			return task.Task{}, fmt.Errorf("开始后命令链只能在执行中任务上重试")
		}
	case task.LifecycleHookBeforeEnd:
		if current.Status != task.StatusRunning {
			return task.Task{}, fmt.Errorf("结束前命令链只能在执行中任务上重试")
		}
	case task.LifecycleHookPostEnd:
		if current.Status != task.StatusCompleted {
			return task.Task{}, fmt.Errorf("结束后命令链只能在已完成任务上重试")
		}
	case task.LifecycleHookUpdateTask:
		if current.Status != task.StatusRunning {
			return task.Task{}, fmt.Errorf("更新命令链只能在执行中任务上重试")
		}
	default:
		return task.Task{}, fmt.Errorf("不支持的命令链钩子: %q", hook)
	}
	return app.scheduleLifecycleHookLocked(current, hook)
}

func (app *App) commitStartAndSchedulePostLocked(prepared task.Task) (task.Task, error) {
	started, err := app.tasks.CommitStartTask(prepared)
	if err != nil {
		return task.Task{}, err
	}
	app.realtime.RegisterTask(started.ID)
	if app.hasLifecycleChain(started, task.LifecycleHookPostStart) {
		return app.scheduleLifecycleHookLocked(started, task.LifecycleHookPostStart)
	}
	app.publishLifecycleTask(started)
	return started, nil
}

func (app *App) finishTaskAndSchedulePostLocked(taskID string) (task.Task, error) {
	app.mu.Lock()
	app.endingTasks[taskID] = true
	app.mu.Unlock()
	finished, err := app.tasks.FinishTask(taskID)
	if err == nil {
		app.realtime.RemoveTask(taskID)
	}
	app.mu.Lock()
	delete(app.endingTasks, taskID)
	app.mu.Unlock()
	if err != nil {
		return task.Task{}, err
	}
	if app.hasLifecycleChain(finished, task.LifecycleHookPostEnd) {
		return app.scheduleLifecycleHookLocked(finished, task.LifecycleHookPostEnd)
	}
	app.publishLifecycleTask(finished)
	return finished, nil
}

func (app *App) scheduleLifecycleHookLocked(current task.Task, hook task.LifecycleHook) (task.Task, error) {
	chainID := strings.TrimSpace(current.LifecycleChains[hook])
	if chainID == "" {
		return current, nil
	}
	data, err := app.repository.Load()
	if err != nil {
		return task.Task{}, err
	}
	execution := task.LifecycleExecution{
		RunID:        task.NewLifecycleExecutionRunID(),
		Revision:     1,
		Hook:         hook,
		ChainID:      chainID,
		CurrentIndex: 1,
		CommandCount: 1,
		State:        task.LifecycleExecutionRunning,
	}
	chain, commands, chainErr := lifecycleChain(data.Settings, chainID)
	if chainErr == nil {
		execution.ChainID = chain.ID
		execution.CommandCount = len(commands)
		execution.CurrentCommandID = commands[0].ID
		execution.CurrentCommandName = commands[0].Name
	}
	updated, err := app.tasks.UpdateLifecycleExecution(current.ID, &execution)
	if err != nil {
		return task.Task{}, err
	}
	app.publishLifecycleTask(updated)
	if chainErr != nil {
		return app.failLifecycleRun(current.ID, execution, chainErr)
	}

	inputTask, err := app.tasks.GetTask(current.ID)
	if err != nil {
		return app.failLifecycleRun(current.ID, execution, err)
	}
	activeTemplate := data.Settings.ActiveTaskTemplate()
	templateFields, err := lifecycleTemplateFields(activeTemplate, inputTask.TemplateFields)
	if err != nil {
		return app.failLifecycleRun(current.ID, execution, err)
	}
	templateEnvironment, err := task.TaskTemplateEnvironment(activeTemplate, templateFields)
	if err != nil {
		return app.failLifecycleRun(current.ID, execution, err)
	}
	input, err := app.lifecycleCommandInput(inputTask, data.Settings)
	if err != nil {
		return app.failLifecycleRun(current.ID, execution, err)
	}
	directory := current.WorkspacePath
	if hook == task.LifecycleHookPostEnd {
		directory = current.WorkspaceRoot
	}
	run := lifecycleRun{
		task:      current,
		hook:      hook,
		execution: execution,
		request: lifecycle.CommandChainRequest{
			Task:           inputTask,
			TemplateFields: templateFields,
			Directory:      directory,
			WorkspaceRoot:  current.WorkspaceRoot,
			WorkspacePath:  current.WorkspacePath,
			ShellPath:      data.Settings.ShellPath,
			Environment:    append(app.taskCommandEnvironment(current.ID), templateEnvironment...),
			Input:          append([]byte(nil), input...),
			Commands:       copyLifecycleCommands(commands),
		},
	}
	go app.executeLifecycleRun(run)
	return updated, nil
}

func (app *App) executeLifecycleRun(run lifecycleRun) {
	execution := run.execution
	request := run.request
	request.OnProgress = func(index, count int, command settings.LifecycleCommand) {
		next := execution
		next.Revision++
		next.CurrentIndex = index
		next.CommandCount = count
		next.CurrentCommandID = command.ID
		next.CurrentCommandName = command.Name
		updated, applied, err := app.tasks.UpdateLifecycleExecutionIfNewer(run.task.ID, &next)
		if err != nil || !applied {
			return
		}
		execution = next
		app.publishLifecycleTask(updated)
	}
	_, err := app.lifecycleCommandRunner.Run(request)
	if err != nil {
		_, _ = app.failLifecycleRun(run.task.ID, execution, err)
		return
	}
	app.continueLifecycleRun(run, execution)
}

func (app *App) continueLifecycleRun(run lifecycleRun, execution task.LifecycleExecution) {
	unlock := app.lockLifecycleTask(run.task.ID)
	defer unlock()

	current, err := app.tasks.GetTask(run.task.ID)
	if err != nil || !isCurrentLifecycleExecution(current, execution) {
		return
	}

	switch run.hook {
	case task.LifecycleHookBeforeStart:
		started, err := app.tasks.CommitStartTask(run.task)
		if err != nil {
			_, _ = app.failLifecycleRun(run.task.ID, execution, err)
			return
		}
		app.realtime.RegisterTask(started.ID)
		app.scheduleNextLifecycleHookLocked(started, task.LifecycleHookPostStart, execution)
	case task.LifecycleHookBeforeEnd:
		finished, err := app.finishTaskAndSchedulePostStateLocked(run.task.ID)
		if err != nil {
			_, _ = app.failLifecycleRun(run.task.ID, execution, err)
			return
		}
		app.scheduleNextLifecycleHookLocked(finished, task.LifecycleHookPostEnd, execution)
	default:
		app.clearLifecycleRun(run.task.ID, execution)
	}
}

func (app *App) finishTaskAndSchedulePostStateLocked(taskID string) (task.Task, error) {
	app.mu.Lock()
	app.endingTasks[taskID] = true
	app.mu.Unlock()
	finished, err := app.tasks.FinishTask(taskID)
	if err == nil {
		app.realtime.RemoveTask(taskID)
	}
	app.mu.Lock()
	delete(app.endingTasks, taskID)
	app.mu.Unlock()
	return finished, err
}

func (app *App) scheduleNextLifecycleHookLocked(current task.Task, hook task.LifecycleHook, previous task.LifecycleExecution) {
	if app.hasLifecycleChain(current, hook) {
		if _, err := app.scheduleLifecycleHookLocked(current, hook); err != nil {
			_, _ = app.failLifecycleRun(current.ID, previous, err)
		}
		return
	}
	app.clearLifecycleRun(current.ID, previous)
}

func (app *App) failLifecycleRun(taskID string, execution task.LifecycleExecution, cause error) (task.Task, error) {
	failed := execution
	failed.Revision++
	failed.State = task.LifecycleExecutionFailed
	failed.Error = cause.Error()
	updated, applied, err := app.tasks.UpdateLifecycleExecutionIfNewer(taskID, &failed)
	if err != nil {
		return task.Task{}, err
	}
	if applied {
		app.publishLifecycleTask(updated)
	}
	return updated, nil
}

func (app *App) clearLifecycleRun(taskID string, execution task.LifecycleExecution) {
	updated, applied, err := app.tasks.ClearLifecycleExecutionIfCurrent(taskID, execution.RunID, execution.Revision)
	if err == nil && applied {
		app.publishLifecycleTask(updated)
	}
}

func (app *App) hasLifecycleChain(current task.Task, hook task.LifecycleHook) bool {
	return strings.TrimSpace(current.LifecycleChains[hook]) != ""
}

func (app *App) lockLifecycleTask(taskID string) func() {
	app.lifecycleLockMu.Lock()
	lock := app.lifecycleTaskMux[taskID]
	if lock == nil {
		lock = &sync.Mutex{}
		app.lifecycleTaskMux[taskID] = lock
	}
	app.lifecycleLockMu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func (app *App) lockLifecycleTasks(taskIDs []string) func() {
	ordered := append([]string(nil), taskIDs...)
	sort.Strings(ordered)
	unlockers := make([]func(), 0, len(ordered))
	seen := make(map[string]bool, len(ordered))
	for _, taskID := range ordered {
		if seen[taskID] {
			continue
		}
		seen[taskID] = true
		unlockers = append(unlockers, app.lockLifecycleTask(taskID))
	}
	return func() {
		for index := len(unlockers) - 1; index >= 0; index-- {
			unlockers[index]()
		}
	}
}

func isCurrentLifecycleExecution(current task.Task, execution task.LifecycleExecution) bool {
	return current.LifecycleExecution != nil && current.LifecycleExecution.RunID == execution.RunID && current.LifecycleExecution.Revision == execution.Revision
}

func lifecycleTemplateFields(template *task.TaskTemplate, values map[string]any) (map[string]any, error) {
	if template == nil {
		return map[string]any{}, nil
	}
	resolved, err := task.ResolveTaskTemplateFields(*template, values)
	if err != nil {
		return nil, err
	}
	frozen := make(map[string]any, len(resolved))
	for key, value := range resolved {
		frozen[key] = value
	}
	return frozen, nil
}

func copyLifecycleCommands(commands []settings.LifecycleCommand) []settings.LifecycleCommand {
	copy := make([]settings.LifecycleCommand, len(commands))
	for index, command := range commands {
		copy[index] = command
		copy[index].Arguments = append([]string(nil), command.Arguments...)
		copy[index].ApplicableHooks = append([]settings.LifecycleHook(nil), command.ApplicableHooks...)
	}
	return copy
}

func lifecycleChain(current settings.Settings, chainID string) (settings.LifecycleCommandChain, []settings.LifecycleCommand, error) {
	var chain settings.LifecycleCommandChain
	for _, candidate := range current.LifecycleChains {
		if candidate.ID == chainID {
			chain = candidate
			break
		}
	}
	if chain.ID == "" {
		return settings.LifecycleCommandChain{}, nil, fmt.Errorf("命令链已删除")
	}
	commandsByID := make(map[string]settings.LifecycleCommand, len(current.LifecycleCommands))
	for _, command := range current.LifecycleCommands {
		commandsByID[command.ID] = command
	}
	commands := make([]settings.LifecycleCommand, 0, len(chain.Commands))
	for _, reference := range chain.Commands {
		command, found := commandsByID[reference.CommandID]
		if !found {
			return settings.LifecycleCommandChain{}, nil, fmt.Errorf("命令链引用的命令已删除")
		}
		command.Arguments = append(append([]string(nil), command.Arguments...), reference.Arguments...)
		commands = append(commands, command)
	}
	return chain, commands, nil
}

func (app *App) publishLifecycleTask(current task.Task) {
	app.mu.RLock()
	ctx := app.ctx
	app.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "task-lifecycle:event", current)
	}
}

func (app *App) GetSettings() (settings.Settings, error) {
	data, err := app.repository.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	return data.Settings, nil
}

func (app *App) SaveSettings(next settings.Settings) (settings.Settings, error) {
	previous, err := app.GetSettings()
	if err != nil {
		return settings.Settings{}, err
	}
	if next.TaskTemplates == nil {
		next.TaskTemplates = append([]task.TaskTemplate(nil), previous.TaskTemplates...)
		next.ActiveTaskTemplateID = previous.ActiveTaskTemplateID
	}
	validated, err := settings.Validate(next)
	if err != nil {
		return settings.Settings{}, err
	}
	if statusHTTPSettingsChanged(previous, validated) {
		if err := app.applyStatusSettings(validated); err != nil {
			return settings.Settings{}, err
		}
	}
	saved, err := app.repository.SaveSettings(validated)
	if err != nil {
		if statusHTTPSettingsChanged(previous, validated) {
			_ = app.applyStatusSettings(previous)
		}
		return settings.Settings{}, err
	}
	return saved, nil
}

func statusHTTPSettingsChanged(previous, current settings.Settings) bool {
	return previous.HTTPServiceEnabled != current.HTTPServiceEnabled ||
		previous.StatusManagementMode != current.StatusManagementMode ||
		previous.StatusManagementHTTPPort != current.StatusManagementHTTPPort
}

func (app *App) DetectShells() []string {
	return settings.DetectShells()
}

func (app *App) CreateTerminal(taskID string, columns, rows uint16) (terminal.Info, error) {
	_, directory, shellPath, err := app.taskOperationContext(taskID)
	if err != nil {
		return terminal.Info{}, err
	}
	environment, unregister := app.terminalStatusEnvironmentBuilder(taskID)
	created, err := app.terminals.CreateWithEnvironmentBuilder(taskID, directory, shellPath, environment, columns, rows)
	if err != nil {
		unregister()
		return terminal.Info{}, err
	}
	return created, nil
}

func (app *App) CreateCommandTerminal(taskID, command string, arguments []string, columns, rows uint16) (terminal.Info, error) {
	_, directory, shellPath, err := app.taskOperationContext(taskID)
	if err != nil {
		return terminal.Info{}, err
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return terminal.Info{}, fmt.Errorf("任务命令不能为空")
	}
	environment, unregister := app.terminalStatusEnvironmentBuilder(taskID)
	created, err := app.terminals.CreateCommandWithEnvironmentBuilder(taskID, directory, shellPath, command, arguments, environment, columns, rows)
	if err != nil {
		unregister()
		return terminal.Info{}, err
	}
	return created, nil
}

func (app *App) RunTaskCommand(taskID, command string, arguments []string) error {
	_, directory, shellPath, err := app.taskOperationContext(taskID)
	if err != nil {
		return err
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("任务命令不能为空")
	}
	return app.commandRunner(directory, shellPath, command, append([]string(nil), arguments...), app.taskCommandEnvironment(taskID))
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
		environment, unregister := app.terminalStatusEnvironmentBuilder(taskID)
		created, err := app.terminals.CreateCommandWithEnvironmentBuilder(taskID, invocation.directory, invocation.shellPath, invocation.command, invocation.arguments, environment, columns, rows)
		if err != nil {
			unregister()
			return application.TaskMenuCommandResult{}, err
		}
		afterScript := cloneTaskScript(item.AfterScript)
		app.terminals.OnExit(taskID, created.ID, func() { app.runAfterScript(invocation, afterScript) })
		return application.TaskMenuCommandResult{Terminal: &created}, nil
	}
	process, err := app.commandStarter(invocation.directory, invocation.shellPath, invocation.command, invocation.arguments, app.taskCommandEnvironment(invocation.taskID))
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
	_, directory, _, err := app.taskOperationContext(taskID)
	if err != nil {
		return err
	}
	return app.directoryOpener(directory)
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

func (app *App) ReportTerminalTitleActivity(taskID, terminalID string) bool {
	return app.realtime.ReportTitleActivity(taskID, terminalID)
}

func (app *App) SelectTerminal(taskID, terminalID string) {
	app.realtime.SelectTerminal(taskID, terminalID)
}

func (app *App) ClearSelectedTerminal() {
	app.realtime.ClearSelection()
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
		if current.IsLifecycleLocked() {
			return task.Task{}, "", fmt.Errorf("任务正在执行命令链，暂不能执行此操作")
		}
		if current.WorkspacePath == "" {
			return task.Task{}, "", fmt.Errorf("任务缺少工作目录")
		}
		return current, data.Settings.ShellPath, nil
	}
	return task.Task{}, "", fmt.Errorf("任务不存在")
}

func (app *App) taskOperationDirectory(current task.Task) (string, error) {
	if len(current.LifecycleChains) != 0 {
		return current.WorkspacePath, nil
	}
	homeDirectory := app.homeDirectory
	if homeDirectory == nil {
		homeDirectory = os.UserHomeDir
	}
	home, err := homeDirectory()
	if err != nil {
		return "", fmt.Errorf("获取用户 Home 目录失败: %w", err)
	}
	info, err := os.Stat(home)
	if err != nil {
		return "", fmt.Errorf("检查用户 Home 目录: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("用户 Home 目录不是目录")
	}
	return home, nil
}

func (app *App) taskOperationContext(taskID string) (task.Task, string, string, error) {
	running, shellPath, err := app.runningTask(taskID)
	if err != nil {
		return task.Task{}, "", "", err
	}
	directory, err := app.taskOperationDirectory(running)
	if err != nil {
		return task.Task{}, "", "", err
	}
	return running, directory, shellPath, nil
}

func (app *App) taskMenuCommand(taskID, itemID string) (taskCommandInvocation, settings.TaskMenuItem, error) {
	_, directory, shellPath, err := app.taskOperationContext(taskID)
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
			taskID: taskID, directory: directory, shellPath: shellPath, command: item.Command, arguments: append([]string(nil), item.Arguments...),
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
	return app.scriptRunner(invocation.directory, invocation.shellPath, script.Script, append([]string(nil), script.Arguments...), payload, app.taskCommandEnvironment(invocation.taskID))
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
	return app.scriptStarter(invocation.directory, invocation.shellPath, script.Script, append([]string(nil), script.Arguments...), payload, app.taskCommandEnvironment(invocation.taskID))
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
	if event.Type == "output" && event.Data != "" {
		app.realtime.ReportOutputActivity(event.TaskID, event.TerminalID)
	}
	if event.Type == "exited" {
		if event.ExitReason == terminal.ExitReasonUnexpected {
			app.realtime.RegisterTerminal(event.TaskID, event.TerminalID)
			app.realtime.SetTerminalStatus(event.TaskID, event.TerminalID, realtime.StatusError)
		} else {
			app.realtime.RemoveTerminal(event.TaskID, event.TerminalID)
		}
	}
	app.mu.RLock()
	ctx := app.ctx
	app.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "task-terminal:event", event)
	}
}

func (app *App) publishRealtimeStatusEvent(event realtime.Event) {
	app.mu.RLock()
	ctx := app.ctx
	app.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "task-realtime-status:event", event)
	}
}

func (app *App) publishRealtimeStatusError(message string) {
	app.mu.RLock()
	ctx := app.ctx
	app.mu.RUnlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "task-realtime-status:error", message)
	}
}

func (app *App) applyStatusSettings(current settings.Settings) error {
	shouldRunHTTP := current.HTTPServiceEnabled || current.StatusManagementMode == settings.StatusManagementModeHTTP
	if shouldRunHTTP {
		if err := app.statusHTTP.Configure(current.StatusManagementHTTPPort); err != nil {
			return err
		}
	} else if err := app.statusHTTP.Close(); err != nil {
		return err
	}

	switch current.StatusManagementMode {
	case settings.StatusManagementModeHTTP:
		app.realtime.SetMode(realtime.ModeHTTP)
	case settings.StatusManagementModeTitleChange:
		app.realtime.SetMode(realtime.ModeTitleChange)
	case settings.StatusManagementModeOutputChange:
		app.realtime.SetMode(realtime.ModeOutputChange)
	default:
		return fmt.Errorf("不支持的状态管理方式: %q", current.StatusManagementMode)
	}
	return nil
}

func (app *App) terminalStatusEnvironment(taskID, terminalID string) []string {
	environment := []string{
		"TASKAI_TASK_ID=" + taskID,
		"TASKAI_TERMINAL_ID=" + terminalID,
	}
	apiURL := app.statusHTTP.APIURL()
	if apiURL == "" {
		return environment
	}
	return append([]string{
		"TASKAI_STATUS_API=" + apiURL,
	}, environment...)
}

func (app *App) taskCommandEnvironment(taskID string) []string {
	return []string{"TASKAI_TASK_ID=" + taskID}
}

func (app *App) terminalStatusEnvironmentBuilder(taskID string) (terminal.TerminalEnvironmentBuilder, func()) {
	registeredTerminalID := ""
	builder := func(terminalID string) []string {
		registeredTerminalID = terminalID
		app.realtime.RegisterTask(taskID)
		app.realtime.RegisterTerminal(taskID, terminalID)
		return app.terminalStatusEnvironment(taskID, terminalID)
	}
	return builder, func() {
		if registeredTerminalID != "" {
			app.realtime.RemoveTerminal(taskID, registeredTerminalID)
		}
	}
}

func (app *App) realtimeTaskState(taskID string) realtime.TaskState {
	data, err := app.repository.Load()
	if err != nil {
		return realtime.TaskStateMissing
	}
	for _, current := range data.Tasks {
		if current.ID != taskID {
			continue
		}
		if current.Status == task.StatusRunning {
			return realtime.TaskStateRunning
		}
		return realtime.TaskStateEnded
	}
	return realtime.TaskStateMissing
}

func (app *App) httpTasks() ([]realtime.TaskResource, error) {
	data, err := app.repository.Load()
	if err != nil {
		return nil, err
	}

	tasks := make([]realtime.TaskResource, 0, len(data.Tasks))
	for _, current := range data.Tasks {
		tasks = append(tasks, app.realtimeTaskResource(current, data.Settings, false))
	}
	return tasks, nil
}

func (app *App) httpTask(taskID string) (realtime.TaskResource, bool, error) {
	data, err := app.repository.Load()
	if err != nil {
		return realtime.TaskResource{}, false, err
	}

	for _, current := range data.Tasks {
		if current.ID == taskID {
			return app.realtimeTaskResource(current, data.Settings, true), true, nil
		}
	}

	return realtime.TaskResource{}, false, nil
}

func (app *App) realtimeTaskResource(current task.Task, configured settings.Settings, includeExtraInfo bool) realtime.TaskResource {
	resource := realtime.TaskResource{
		ID:                 current.ID,
		Title:              current.Title,
		Description:        current.Description,
		Color:              current.Color,
		Status:             string(current.Status),
		CreatedAt:          current.CreatedAt,
		CompletedAt:        current.CompletedAt,
		WorkspaceRoot:      current.WorkspaceRoot,
		WorkspacePath:      current.WorkspacePath,
		LifecycleChains:    copyTaskLifecycleChains(current.LifecycleChains),
		LifecycleExecution: copyTaskLifecycleExecution(current.LifecycleExecution),
		TemplateFields:     taskTemplateFieldsForResource(current, configured),
	}
	if includeExtraInfo {
		extraInfo := httpExtraInfo(current.ExtraInfo)
		resource.ExtraInfo = &extraInfo
		terminals := app.httpTerminals(current.ID)
		resource.Terminals = &terminals
	}
	return resource
}

func (app *App) lifecycleCommandInput(current task.Task, configured settings.Settings) ([]byte, error) {
	return lifecycle.BuildCommandInput(app.realtimeTaskResource(current, configured, true), app.statusHTTP.APIURL())
}

func taskTemplateFieldsForResource(current task.Task, configured settings.Settings) map[string]any {
	activeTemplate := configured.ActiveTaskTemplate()
	if activeTemplate == nil {
		return map[string]any{}
	}
	resolved, err := task.ResolveTaskTemplateFields(*activeTemplate, current.TemplateFields)
	if err != nil {
		return map[string]any{}
	}
	return resolved
}

func copyTaskLifecycleChains(chains map[task.LifecycleHook]string) map[task.LifecycleHook]string {
	copy := make(map[task.LifecycleHook]string, len(chains))
	for hook, chainID := range chains {
		copy[hook] = chainID
	}
	return copy
}

func copyTaskLifecycleExecution(execution *task.LifecycleExecution) *task.LifecycleExecution {
	if execution == nil {
		return nil
	}
	copy := *execution
	return &copy
}

func (app *App) httpTerminals(taskID string) []realtime.TerminalResource {
	sessions := app.terminals.ListActive(taskID)
	terminals := make([]realtime.TerminalResource, 0, len(sessions))
	for _, session := range sessions {
		terminals = append(terminals, realtime.TerminalResource{
			ID:      session.ID,
			Command: session.Command,
			Status:  app.realtime.TerminalStatus(taskID, session.ID),
		})
	}
	return terminals
}

func httpExtraInfo(items []task.TaskExtraInfo) map[string][]map[string]any {
	grouped := make(map[string][]map[string]any)
	for _, item := range items {
		values := make(map[string]any, len(item.Fields)+len(item.Parameters))
		for _, field := range item.Fields {
			values[field.Key] = field.Value
		}
		for _, parameter := range item.Parameters {
			if task.NormalizeExtraInfoParameterInputType(parameter.InputType) == task.ExtraInfoParameterInputCheckbox {
				values[parameter.Key] = parameter.Value == "true"
				continue
			}
			values[parameter.Key] = parameter.Value
		}
		grouped[item.Catalogue] = append(grouped[item.Catalogue], values)
	}
	return grouped
}

func (app *App) registerRunningRealtimeTasks() {
	items, err := app.tasks.ListTasks()
	if err != nil {
		return
	}
	for _, current := range items {
		if current.Status == task.StatusRunning {
			app.realtime.RegisterTask(current.ID)
		}
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

func runTaskCommand(directory, shellPath, command string, arguments, environment []string) error {
	process, err := startTaskCommand(directory, shellPath, command, arguments, environment)
	if err != nil {
		return err
	}
	go func() { _ = process.Wait() }()
	return nil
}

func startTaskCommand(directory, shellPath, command string, arguments, environment []string) (commandWaiter, error) {
	process := commandProcess(shellPath, command, arguments)
	configureCommandProcess(process, directory, environment)
	if err := process.Start(); err != nil {
		return nil, fmt.Errorf("启动任务命令: %w", err)
	}
	return process, nil
}

func runTaskScript(directory, shellPath, script string, arguments []string, input []byte, environment []string) error {
	process, err := startTaskScript(directory, shellPath, script, arguments, input, environment)
	if err != nil {
		return err
	}
	if err := process.Wait(); err != nil {
		return fmt.Errorf("执行脚本: %w", err)
	}
	return nil
}

func startTaskScript(directory, shellPath, script string, arguments []string, input []byte, environment []string) (commandWaiter, error) {
	process := commandProcess(shellPath, script, arguments)
	configureCommandProcess(process, directory, environment)
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

func configureCommandProcess(process *exec.Cmd, directory string, environment []string) {
	process.Dir = directory
	if process.Env == nil {
		process.Env = os.Environ()
	}
	process.Env = append(process.Env, environment...)
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
