package main

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"taskai/internal/realtime"
	"taskai/internal/settings"
	"taskai/internal/task"
	"taskai/internal/terminal"
)

func TestDefaultDataDirectoryUsesApplicationName(t *testing.T) {
	configurationDirectory, err := os.UserConfigDir()
	if err == nil {
		if got, want := defaultDataDirectory(), filepath.Join(configurationDirectory, "taskai"); got != want {
			t.Fatalf("默认数据目录 = %q，期望 %q", got, want)
		}
		return
	}

	if got, want := defaultDataDirectory(), filepath.Join(os.TempDir(), "taskai"); got != want {
		t.Fatalf("默认数据目录 = %q，期望 %q", got, want)
	}
}

func TestAppExposesTaskAndSettingsBindings(t *testing.T) {
	t.Parallel()

	app := newApp(t.TempDir())
	created, err := app.CreateTask("整理文档", "在工作目录中完成", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	if created.Status != task.StatusPending {
		t.Fatalf("新任务状态 = %q，期望未执行", created.Status)
	}

	updatedSettings, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: filepath.Join(t.TempDir(), "workspaces"),
		TaskTreeWidth: 460,
	})
	if err != nil {
		t.Fatalf("保存设置: %v", err)
	}
	if updatedSettings.TaskTreeWidth != 460 {
		t.Fatalf("任务树宽度 = %d，期望 460", updatedSettings.TaskTreeWidth)
	}

	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("开始任务: %v", err)
	}
	if started.Status != task.StatusRunning || started.WorkspaceRoot != updatedSettings.WorkspaceRoot {
		t.Fatalf("开始任务快照错误: %#v", started)
	}
	if !app.HasRunningTasks() {
		t.Fatal("执行中任务未被退出检查识别")
	}

	if err := app.PrepareQuit(); err != nil {
		t.Fatalf("准备退出: %v", err)
	}
	listed, err := app.ListTasks()
	if err != nil {
		t.Fatalf("列出任务: %v", err)
	}
	if listed[0].Status != task.StatusRunning {
		t.Fatalf("退出清理不应改变任务状态: %#v", listed[0])
	}
}

func TestAppRegistersRunningTaskAndClearsRealtimeStatusWhenFinished(t *testing.T) {
	app := newApp(t.TempDir())
	created, err := app.CreateTask("实时状态任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("开始任务: %v", err)
	}
	if got := app.realtime.Snapshot(); len(got.Tasks) != 1 || got.Tasks[0].TaskID != started.ID {
		t.Fatalf("开始任务后的实时状态 = %#v", got)
	}

	if _, err := app.FinishTask(started.ID); err != nil {
		t.Fatalf("结束任务: %v", err)
	}
	if got := app.realtime.Snapshot(); len(got.Tasks) != 0 {
		t.Fatalf("结束任务后的实时状态 = %#v，期望清理", got)
	}
}

func TestAppMapsTerminalExitReasonsToRealtimeStatus(t *testing.T) {
	app := newApp(t.TempDir())
	app.realtime.RegisterTerminal("task-1", "terminal-1")
	app.publishTerminalEvent(terminal.Event{TaskID: "task-1", TerminalID: "terminal-1", Type: "exited", ExitReason: terminal.ExitReasonUnexpected})
	if got := app.realtime.TerminalStatus("task-1", "terminal-1"); got != realtime.StatusError {
		t.Fatalf("异常退出终端状态 = %q，期望 %q", got, realtime.StatusError)
	}

	app.publishTerminalEvent(terminal.Event{TaskID: "task-1", TerminalID: "terminal-1", Type: "exited", ExitReason: terminal.ExitReasonClosed})
	if got := app.realtime.TerminalPresence("task-1", "terminal-1"); got != realtime.TerminalRemoved {
		t.Fatalf("主动关闭终端状态记录 = %q，期望 %q", got, realtime.TerminalRemoved)
	}
}

func TestAppConfiguresHTTPStatusServiceAtomically(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	initialPort := availableLoopbackPort(t)
	initial, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            t.TempDir(),
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		StatusManagementMode:     settings.StatusManagementModeHTTP,
		StatusManagementHTTPPort: initialPort,
	})
	if err != nil {
		t.Fatalf("保存 HTTP 状态设置: %v", err)
	}
	initialURL := app.statusHTTP.APIURL()
	if initialURL == "" {
		t.Fatal("保存 HTTP 状态设置后未启动服务")
	}

	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("占用测试端口: %v", err)
	}
	defer occupied.Close()
	failedPort := occupied.Addr().(*net.TCPAddr).Port
	_, err = app.SaveSettings(settings.Settings{
		WorkspaceRoot:            initial.WorkspaceRoot,
		TaskTreeWidth:            initial.TaskTreeWidth,
		StatusManagementMode:     settings.StatusManagementModeHTTP,
		StatusManagementHTTPPort: failedPort,
	})
	if err == nil {
		t.Fatal("保存被占用 HTTP 端口 error = nil，期望错误")
	}
	if app.statusHTTP.APIURL() != initialURL {
		t.Errorf("保存失败后的 HTTP 服务 = %q，期望保留 %q", app.statusHTTP.APIURL(), initialURL)
	}
	saved, err := app.GetSettings()
	if err != nil {
		t.Fatalf("读取保存设置: %v", err)
	}
	if saved.StatusManagementHTTPPort != initialPort {
		t.Errorf("保存失败后的 HTTP 端口 = %d，期望 %d", saved.StatusManagementHTTPPort, initialPort)
	}
}

func TestAppConfiguresIndependentHTTPServiceAndKeepsStatusHTTPAutomatic(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	port := availableLoopbackPort(t)
	base := settings.Settings{
		WorkspaceRoot: t.TempDir(), TaskTreeWidth: settings.DefaultTaskTreeWidth,
		StatusManagementMode: settings.StatusManagementModeTitleChange, StatusManagementHTTPPort: port,
	}

	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: base.WorkspaceRoot, TaskTreeWidth: base.TaskTreeWidth,
		StatusManagementMode: base.StatusManagementMode, StatusManagementHTTPPort: base.StatusManagementHTTPPort, HTTPServiceEnabled: true,
	}); err != nil {
		t.Fatalf("保存独立 HTTP 服务设置: %v", err)
	}
	if app.statusHTTP.APIURL() == "" {
		t.Fatal("独立 HTTP 服务未启动")
	}

	if _, err := app.SaveSettings(base); err != nil {
		t.Fatalf("关闭独立 HTTP 服务: %v", err)
	}
	if app.statusHTTP.APIURL() != "" {
		t.Errorf("关闭独立 HTTP 服务后 API 地址 = %q，期望为空", app.statusHTTP.APIURL())
	}

	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: base.WorkspaceRoot, TaskTreeWidth: base.TaskTreeWidth,
		StatusManagementMode: settings.StatusManagementModeHTTP, StatusManagementHTTPPort: port, HTTPServiceEnabled: false,
	}); err != nil {
		t.Fatalf("保存 HTTP 状态管理设置: %v", err)
	}
	if app.statusHTTP.APIURL() == "" {
		t.Fatal("HTTP 状态管理未自动启动 HTTP 服务")
	}
}

func TestAppKeepsHTTPServiceWhenSavingActiveTaskStatus(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	initial := settings.Settings{
		WorkspaceRoot:            t.TempDir(),
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		ActiveTaskStatus:         settings.TaskStatusPending,
		StatusManagementMode:     settings.StatusManagementModeHTTP,
		StatusManagementHTTPPort: availableLoopbackPort(t),
	}
	if _, err := app.SaveSettings(initial); err != nil {
		t.Fatalf("保存 HTTP 状态设置: %v", err)
	}
	previousURL := app.statusHTTP.APIURL()

	initial.ActiveTaskStatus = settings.TaskStatusRunning
	if _, err := app.SaveSettings(initial); err != nil {
		t.Fatalf("仅保存任务标签时不应重启 HTTP 服务: %v", err)
	}
	if app.statusHTTP.APIURL() != previousURL {
		t.Errorf("保存任务标签后的 API 地址 = %q，期望保持 %q", app.statusHTTP.APIURL(), previousURL)
	}
}

func TestAppHTTPServiceListsTasksByStatusAndReturnsTaskDetails(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: t.TempDir(), TaskTreeWidth: settings.DefaultTaskTreeWidth,
		StatusManagementMode: settings.StatusManagementModeTitleChange,
		HTTPServiceEnabled:   true, StatusManagementHTTPPort: availableLoopbackPort(t),
	}); err != nil {
		t.Fatalf("启用任务 HTTP 服务: %v", err)
	}
	pending, err := app.CreateTask("待执行任务", "等待执行", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建待执行任务: %v", err)
	}
	running, err := app.CreateTask("执行中任务", "正在执行", "#22c55e")
	if err != nil {
		t.Fatalf("创建执行中任务: %v", err)
	}
	if _, err := app.StartTask(running.ID); err != nil {
		t.Fatalf("开始执行中任务: %v", err)
	}
	completed, err := app.CreateTask("已完成任务", "已经完成", "#f97316")
	if err != nil {
		t.Fatalf("创建已完成任务: %v", err)
	}
	if _, err := app.StartTask(completed.ID); err != nil {
		t.Fatalf("开始已完成任务: %v", err)
	}
	if _, err := app.FinishTask(completed.ID); err != nil {
		t.Fatalf("结束已完成任务: %v", err)
	}

	response, err := http.Get(app.statusHTTP.APIURL() + "/tasks?status=pending")
	if err != nil {
		t.Fatalf("查询待执行任务: %v", err)
	}
	defer response.Body.Close()
	var listed []realtime.TaskResource
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		t.Fatalf("解析待执行任务: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != pending.ID || listed[0].Status != string(task.StatusPending) {
		t.Fatalf("待执行任务列表 = %#v", listed)
	}

	response, err = http.Get(app.statusHTTP.APIURL() + "/tasks/" + running.ID)
	if err != nil {
		t.Fatalf("查询任务详情: %v", err)
	}
	defer response.Body.Close()
	var detail realtime.TaskResource
	if err := json.NewDecoder(response.Body).Decode(&detail); err != nil {
		t.Fatalf("解析任务详情: %v", err)
	}
	if detail.ID != running.ID || detail.Description != "正在执行" || detail.WorkspacePath == "" {
		t.Fatalf("执行中任务详情 = %#v", detail)
	}
}

func TestAppBuildsTerminalEnvironmentForEveryStatusMode(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	created, err := app.CreateTask("终端环境", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("开始任务: %v", err)
	}
	titleChangeEnvironment := app.terminalStatusEnvironment(started.ID, "terminal-1")
	if !containsEnvironmentValue(titleChangeEnvironment, "TASKAI_TASK_ID="+started.ID) || !containsEnvironmentValue(titleChangeEnvironment, "TASKAI_TERMINAL_ID=terminal-1") {
		t.Fatalf("标题变化模式终端环境 = %#v，期望包含任务和终端 ID", titleChangeEnvironment)
	}
	if containsEnvironmentValue(titleChangeEnvironment, "TASKAI_STATUS_API="+app.statusHTTP.APIURL()) {
		t.Fatalf("标题变化模式终端环境不应包含 HTTP 地址: %#v", titleChangeEnvironment)
	}

	port := availableLoopbackPort(t)
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            started.WorkspaceRoot,
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		StatusManagementMode:     settings.StatusManagementModeHTTP,
		StatusManagementHTTPPort: port,
	}); err != nil {
		t.Fatalf("保存 HTTP 状态设置: %v", err)
	}
	environment := app.terminalStatusEnvironment(started.ID, "terminal-1")
	if !containsEnvironmentValue(environment, "TASKAI_TASK_ID="+started.ID) || !containsEnvironmentValue(environment, "TASKAI_TERMINAL_ID=terminal-1") || !containsEnvironmentValue(environment, "TASKAI_STATUS_API="+app.statusHTTP.APIURL()) {
		t.Fatalf("HTTP 模式终端环境 = %#v", environment)
	}
}

func TestAppInjectsTaskIDIntoBackgroundCommandsAndScripts(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "background-command", Kind: settings.TaskMenuItemKindCommand, Name: "后台执行", Command: "main-command",
		BeforeScript: &settings.TaskScript{Script: "prepare"},
		AfterScript:  &settings.TaskScript{Script: "cleanup"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	mainWaiter := &controlledCommandWaiter{done: make(chan error, 1)}
	afterStarted := make(chan struct{})
	var beforeEnvironment, commandEnvironment, afterEnvironment []string
	app.scriptRunner = func(_ string, _ string, script string, _ []string, _ []byte, environment []string) error {
		if script != "prepare" {
			t.Fatalf("前置脚本 = %q，期望 prepare", script)
		}
		beforeEnvironment = append([]string(nil), environment...)
		return nil
	}
	app.commandStarter = func(_ string, _ string, _ string, _ []string, environment []string) (commandWaiter, error) {
		commandEnvironment = append([]string(nil), environment...)
		return mainWaiter, nil
	}
	app.scriptStarter = func(_ string, _ string, script string, _ []string, _ []byte, environment []string) (commandWaiter, error) {
		if script != "cleanup" {
			t.Fatalf("后置脚本 = %q，期望 cleanup", script)
		}
		afterEnvironment = append([]string(nil), environment...)
		close(afterStarted)
		return commandWaiterFunc(func() error { return nil }), nil
	}

	if _, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32); err != nil {
		t.Fatalf("执行后台命令: %v", err)
	}
	assertTaskOnlyEnvironment(t, beforeEnvironment, started.ID)
	assertTaskOnlyEnvironment(t, commandEnvironment, started.ID)
	mainWaiter.done <- nil
	select {
	case <-afterStarted:
		assertTaskOnlyEnvironment(t, afterEnvironment, started.ID)
	case <-time.After(time.Second):
		t.Fatal("后置脚本未启动")
	}
}

func TestAppInjectsHTTPStatusEnvironmentForNormalAndCommandTerminals(t *testing.T) {
	port := availableLoopbackPort(t)
	item := settings.TaskMenuItem{
		ID: "custom-status-command", Kind: settings.TaskMenuItemKindCommand, Name: "状态命令", Command: "status-command", ShowTerminal: true,
	}
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            t.TempDir(),
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		TaskMenuItems:            []settings.TaskMenuItem{item},
		StatusManagementMode:     settings.StatusManagementModeHTTP,
		StatusManagementHTTPPort: port,
	}); err != nil {
		t.Fatalf("保存 HTTP 状态设置: %v", err)
	}
	created, err := app.CreateTask("状态环境", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("开始任务: %v", err)
	}

	normal, err := app.CreateTerminal(started.ID, 100, 32)
	if err != nil {
		t.Fatalf("创建普通终端: %v", err)
	}
	command, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32)
	if err != nil {
		t.Fatalf("创建显示终端命令: %v", err)
	}
	if command.Terminal == nil {
		t.Fatal("显示终端命令未返回终端")
	}

	initialNormalEnvironment := backend.request(normal.ID).Environment
	assertStatusEnvironment(t, initialNormalEnvironment, app.statusHTTP.APIURL(), started.ID, normal.ID)
	assertStatusEnvironment(t, backend.request(command.Terminal.ID).Environment, app.statusHTTP.APIURL(), started.ID, command.Terminal.ID)
	if request := backend.request(command.Terminal.ID); request.Command != "status-command" {
		t.Fatalf("显示终端命令 = %q，期望 status-command", request.Command)
	}

	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            started.WorkspaceRoot,
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		TaskMenuItems:            []settings.TaskMenuItem{item},
		StatusManagementMode:     settings.StatusManagementModeTitleChange,
		StatusManagementHTTPPort: port,
	}); err != nil {
		t.Fatalf("切换状态管理方式: %v", err)
	}
	if got := backend.request(normal.ID).Environment; !reflect.DeepEqual(got, initialNormalEnvironment) {
		t.Fatalf("切换状态方式后既有进程环境 = %#v，期望保持 %#v", got, initialNormalEnvironment)
	}
}

func TestAppRegistersRealtimeTerminalBeforeStartingProcess(t *testing.T) {
	app := newApp(t.TempDir())
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	created, err := app.CreateTask("启动注册", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("开始任务: %v", err)
	}
	backend.onStart = func(request terminal.StartRequest) {
		if got := app.realtime.TerminalPresence(request.TaskID, request.ID); got != realtime.TerminalActive {
			t.Fatalf("进程启动前的终端状态记录 = %q，期望 %q", got, realtime.TerminalActive)
		}
	}

	if _, err := app.CreateTerminal(started.ID, 100, 32); err != nil {
		t.Fatalf("创建终端: %v", err)
	}
}

func TestAppReordersTasksWithinStatus(t *testing.T) {
	app := newApp(t.TempDir())
	first, err := app.CreateTask("第一个待办", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建第一个任务: %v", err)
	}
	second, err := app.CreateTask("第二个待办", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建第二个任务: %v", err)
	}

	reordered, err := app.ReorderTasks(task.StatusPending, []string{second.ID, first.ID})

	if err != nil {
		t.Fatalf("重排任务: %v", err)
	}
	if len(reordered) != 2 || reordered[0].ID != second.ID || reordered[1].ID != first.ID {
		t.Fatalf("重排任务结果 = %#v", reordered)
	}
}

func TestOpenTaskFolderUsesRunningTaskWorkspace(t *testing.T) {
	app := newApp(t.TempDir())
	created, err := app.CreateTask("打开目录", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("开始任务: %v", err)
	}

	var openedPath string
	app.directoryOpener = func(path string) error {
		openedPath = path
		return nil
	}
	if err := app.OpenTaskFolder(started.ID); err != nil {
		t.Fatalf("打开任务目录: %v", err)
	}
	if openedPath != started.WorkspacePath {
		t.Fatalf("打开目录 = %q，期望 %q", openedPath, started.WorkspacePath)
	}
}

func TestRunTaskCommandUsesRunningTaskWorkspace(t *testing.T) {
	app := newApp(t.TempDir())
	created, err := app.CreateTask("运行命令", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("开始任务: %v", err)
	}

	var directory, shellPath, command string
	var arguments, environment []string
	app.commandRunner = func(nextDirectory, nextShellPath, nextCommand string, nextArguments, nextEnvironment []string) error {
		directory = nextDirectory
		shellPath = nextShellPath
		command = nextCommand
		arguments = nextArguments
		environment = nextEnvironment
		return nil
	}
	if err := app.RunTaskCommand(started.ID, "code", []string{"."}); err != nil {
		t.Fatalf("运行任务命令: %v", err)
	}
	if directory != started.WorkspacePath || shellPath == "" || command != "code" || len(arguments) != 1 || arguments[0] != "." {
		t.Fatalf("运行任务命令参数 = directory:%q shell:%q command:%q arguments:%#v", directory, shellPath, command, arguments)
	}
	assertTaskOnlyEnvironment(t, environment, started.ID)
}

func TestExecuteTaskMenuCommandRunsScriptsAroundBackgroundCommand(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "custom-command", Kind: settings.TaskMenuItemKindCommand, Name: "执行", Command: "main-command", Arguments: []string{"--first", "第二个参数"},
		BeforeScript: &settings.TaskScript{Script: "prepare", Arguments: []string{"--before"}},
		AfterScript:  &settings.TaskScript{Script: "cleanup", Arguments: []string{"--after"}},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	events := make(chan string, 3)
	waiter := &controlledCommandWaiter{done: make(chan error, 1)}
	var beforeInput []byte
	app.scriptRunner = func(_ string, _ string, script string, _ []string, input []byte, _ []string) error {
		if script != "prepare" {
			t.Fatalf("前置阶段运行脚本 = %q，期望 prepare", script)
		}
		beforeInput = append([]byte(nil), input...)
		events <- "script:" + script
		return nil
	}
	app.scriptStarter = func(_ string, _ string, script string, _ []string, _ []byte, _ []string) (commandWaiter, error) {
		events <- "script:" + script
		return commandWaiterFunc(func() error { return nil }), nil
	}
	app.commandStarter = func(directory, shellPath, command string, arguments []string, _ []string) (commandWaiter, error) {
		if directory != started.WorkspacePath || shellPath == "" || command != "main-command" || len(arguments) != 2 || arguments[0] != "--first" || arguments[1] != "第二个参数" {
			t.Fatalf("主命令启动参数 = directory:%q shell:%q command:%q arguments:%#v", directory, shellPath, command, arguments)
		}
		events <- "main"
		return waiter, nil
	}

	result, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32)
	if err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	if result.Terminal != nil {
		t.Fatalf("后台命令返回终端 = %#v，期望 nil", result.Terminal)
	}
	if event := receiveCommandEvent(t, events); event != "script:prepare" {
		t.Fatalf("第一个执行事件 = %q，期望前置脚本", event)
	}
	if event := receiveCommandEvent(t, events); event != "main" {
		t.Fatalf("第二个执行事件 = %q，期望主命令", event)
	}
	var payload map[string]any
	if err := json.Unmarshal(beforeInput, &payload); err != nil {
		t.Fatalf("解析前置脚本输入: %v", err)
	}
	if payload["taskId"] != started.ID || payload["directory"] != started.WorkspacePath || payload["command"] != "main-command" {
		t.Fatalf("脚本 JSON 上下文 = %#v", payload)
	}
	if arguments, ok := payload["arguments"].([]any); !ok || len(arguments) != 2 || arguments[0] != "--first" || arguments[1] != "第二个参数" {
		t.Fatalf("脚本 JSON 参数 = %#v", payload["arguments"])
	}

	waiter.done <- nil
	if event := receiveCommandEvent(t, events); event != "script:cleanup" {
		t.Fatalf("主命令退出后的事件 = %q，期望后置脚本", event)
	}
}

func TestExecuteTaskMenuCommandStopsAfterFailingBeforeScript(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "custom-command", Kind: settings.TaskMenuItemKindCommand, Name: "执行", Command: "main-command",
		BeforeScript: &settings.TaskScript{Script: "prepare"},
		AfterScript:  &settings.TaskScript{Script: "cleanup"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	app.scriptRunner = func(_ string, _ string, script string, _ []string, _ []byte, _ []string) error {
		if script != "prepare" {
			t.Fatalf("不应执行脚本 %q", script)
		}
		return errors.New("准备失败")
	}
	app.commandStarter = func(string, string, string, []string, []string) (commandWaiter, error) {
		t.Fatal("前置脚本失败后不应启动主命令")
		return nil, nil
	}

	if _, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32); err == nil {
		t.Fatal("执行菜单命令错误 = nil，期望前置脚本错误")
	}
}

func TestExecuteTaskMenuCommandSkipsAfterScriptWhenTaskFinishes(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "custom-command", Kind: settings.TaskMenuItemKindCommand, Name: "执行", Command: "main-command",
		AfterScript: &settings.TaskScript{Script: "cleanup"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	waiter := &controlledCommandWaiter{done: make(chan error, 1)}
	scriptCalls := make(chan string, 1)
	app.scriptStarter = func(_ string, _ string, script string, _ []string, _ []byte, _ []string) (commandWaiter, error) {
		scriptCalls <- script
		return commandWaiterFunc(func() error { return nil }), nil
	}
	app.commandStarter = func(string, string, string, []string, []string) (commandWaiter, error) { return waiter, nil }

	if _, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32); err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	if _, err := app.FinishTask(started.ID); err != nil {
		t.Fatalf("结束任务: %v", err)
	}
	waiter.done <- nil
	select {
	case script := <-scriptCalls:
		t.Fatalf("任务结束后执行了后置脚本 %q", script)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestFinishTaskWaitsForInProgressAfterScriptStart(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "custom-command", Kind: settings.TaskMenuItemKindCommand, Name: "执行", Command: "main-command",
		AfterScript: &settings.TaskScript{Script: "cleanup"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	mainWaiter := &controlledCommandWaiter{done: make(chan error, 1)}
	scriptStartEntered := make(chan struct{})
	allowScriptStart := make(chan struct{})
	app.commandStarter = func(string, string, string, []string, []string) (commandWaiter, error) { return mainWaiter, nil }
	app.scriptStarter = func(_ string, _ string, script string, _ []string, _ []byte, _ []string) (commandWaiter, error) {
		if script != "cleanup" {
			t.Fatalf("后置阶段运行脚本 = %q，期望 cleanup", script)
		}
		close(scriptStartEntered)
		<-allowScriptStart
		return commandWaiterFunc(func() error { return nil }), nil
	}

	if _, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32); err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	mainWaiter.done <- nil
	select {
	case <-scriptStartEntered:
	case <-time.After(time.Second):
		t.Fatal("后置脚本未进入启动边界")
	}

	finishResult := make(chan error, 1)
	go func() {
		_, err := app.FinishTask(started.ID)
		finishResult <- err
	}()
	select {
	case err := <-finishResult:
		t.Fatalf("脚本启动边界尚未完成时结束任务返回: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(allowScriptStart)
	select {
	case err := <-finishResult:
		if err != nil {
			t.Fatalf("结束任务: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("脚本启动边界释放后结束任务未完成")
	}
}

func TestExecuteTaskMenuCommandReportsFailingAfterScript(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "custom-command", Kind: settings.TaskMenuItemKindCommand, Name: "执行", Command: "main-command",
		AfterScript: &settings.TaskScript{Script: "cleanup"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	waiter := &controlledCommandWaiter{done: make(chan error, 1)}
	errorMessages := make(chan string, 1)
	app.scriptStarter = func(_ string, _ string, script string, _ []string, _ []byte, _ []string) (commandWaiter, error) {
		if script != "cleanup" {
			t.Fatalf("后置阶段运行脚本 = %q", script)
		}
		return commandWaiterFunc(func() error { return errors.New("清理失败") }), nil
	}
	app.scriptErrorPublisher = func(_ string, message string) { errorMessages <- message }
	app.commandStarter = func(string, string, string, []string, []string) (commandWaiter, error) { return waiter, nil }

	if _, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32); err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	waiter.done <- nil
	select {
	case message := <-errorMessages:
		if message == "" {
			t.Fatal("后置脚本失败提示为空")
		}
	case <-time.After(time.Second):
		t.Fatal("未收到后置脚本失败提示")
	}
}

func TestExecuteTaskMenuCommandRunsAfterScriptWhenTerminalExits(t *testing.T) {
	t.Setenv("TASKAI_MENU_COMMAND_HELPER", "1")
	item := settings.TaskMenuItem{
		ID: "custom-command", Kind: settings.TaskMenuItemKindCommand, Name: "执行", Command: os.Args[0], Arguments: []string{"-test.run=TestTaskMenuCommandProcessHelper", "--"}, ShowTerminal: true,
		AfterScript: &settings.TaskScript{Script: "cleanup"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	scriptCalls := make(chan string, 1)
	app.scriptStarter = func(_ string, _ string, script string, _ []string, _ []byte, _ []string) (commandWaiter, error) {
		scriptCalls <- script
		return commandWaiterFunc(func() error { return nil }), nil
	}

	result, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32)
	if err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	if result.Terminal == nil {
		t.Fatal("显示终端的菜单命令未返回终端")
	}
	select {
	case script := <-scriptCalls:
		if script != "cleanup" {
			t.Fatalf("终端退出后脚本 = %q，期望 cleanup", script)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("命令终端退出后未执行后置脚本")
	}
}

func TestRunTaskScriptWritesInputToStandardInput(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "script-input.json")
	t.Setenv("TASKAI_SCRIPT_PROCESS_HELPER", "1")
	t.Setenv("TASKAI_SCRIPT_PROCESS_OUTPUT", outputPath)
	input := []byte(`{"taskId":"task-a","directory":"/tmp/task-a","command":"codex","arguments":["--full-auto"]}`)

	if err := runTaskScript(t.TempDir(), "", os.Args[0], []string{"-test.run=TestTaskScriptProcessHelper", "--"}, input, nil); err != nil {
		t.Fatalf("运行脚本: %v", err)
	}
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("读取脚本标准输入: %v", err)
	}
	if string(output) != string(input) {
		t.Fatalf("脚本标准输入 = %q，期望 %q", output, input)
	}
}

func TestExecuteTaskMenuCommandKeepsEmptyMainArgumentsAsJSONArray(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "custom-command", Kind: settings.TaskMenuItemKindCommand, Name: "执行", Command: "main-command",
		BeforeScript: &settings.TaskScript{Script: "prepare"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	var input []byte
	app.scriptRunner = func(_ string, _ string, _ string, _ []string, nextInput []byte, _ []string) error {
		input = append([]byte(nil), nextInput...)
		return nil
	}
	app.commandStarter = func(string, string, string, []string, []string) (commandWaiter, error) {
		return commandWaiterFunc(func() error { return nil }), nil
	}

	if _, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32); err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	var payload struct {
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(input, &payload); err != nil {
		t.Fatalf("解析脚本标准输入: %v", err)
	}
	if got, want := string(payload.Arguments), "[]"; got != want {
		t.Fatalf("无主命令参数的 JSON = %s，期望 %s", got, want)
	}
}

func TestExecuteTaskMenuCommandRunsAfterScriptWhenMainCommandFails(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "custom-command", Kind: settings.TaskMenuItemKindCommand, Name: "执行", Command: "main-command",
		AfterScript: &settings.TaskScript{Script: "cleanup"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	waiter := &controlledCommandWaiter{done: make(chan error, 1)}
	scriptCalls := make(chan string, 1)
	app.commandStarter = func(string, string, string, []string, []string) (commandWaiter, error) { return waiter, nil }
	app.scriptStarter = func(_ string, _ string, script string, _ []string, _ []byte, _ []string) (commandWaiter, error) {
		scriptCalls <- script
		return commandWaiterFunc(func() error { return nil }), nil
	}

	if _, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32); err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	waiter.done <- errors.New("主命令失败")
	if script := receiveCommandEvent(t, scriptCalls); script != "cleanup" {
		t.Fatalf("主命令非零退出后的脚本 = %q，期望 cleanup", script)
	}
}

func TestCommandProcessForPlatformUsesConfiguredShell(t *testing.T) {
	const powerShellInvocation = "$taskaiArguments = ConvertFrom-Json -InputObject $env:TASKAI_EXEC_ARGUMENTS; & $env:TASKAI_EXEC_COMMAND @($taskaiArguments)"
	tests := []struct {
		name      string
		platform  string
		shellPath string
		arguments []string
		wantArgs  []string
	}{
		{
			name: "POSIX Shell", platform: "linux", shellPath: "/bin/zsh", arguments: []string{"--full-auto"},
			wantArgs: []string{"-ic", `exec "$@"`, "/bin/zsh", "codex", "--full-auto"},
		},
		{
			name: "fish", platform: "linux", shellPath: "/usr/bin/fish", arguments: []string{"--full-auto"},
			wantArgs: []string{"-ic", "exec $argv[2..-1]", "/usr/bin/fish", "codex", "--full-auto"},
		},
		{
			name: "PowerShell", platform: "windows", shellPath: `C:\\Program Files\\PowerShell\\7\\pwsh.exe`, arguments: []string{"--full-auto"},
			wantArgs: []string{"-NoLogo", "-Command", powerShellInvocation},
		},
		{
			name: "cmd", platform: "windows", shellPath: `C:\\Windows\\System32\\cmd.exe`, arguments: []string{"--full-auto"},
			wantArgs: []string{"/C", "codex", "--full-auto"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			process := commandProcessForPlatform(test.platform, test.shellPath, "codex", test.arguments)
			if process.Path != test.shellPath {
				t.Fatalf("进程路径 = %q，期望 %q", process.Path, test.shellPath)
			}
			if got, want := process.Args, append([]string{test.shellPath}, test.wantArgs...); !reflect.DeepEqual(got, want) {
				t.Fatalf("进程参数 = %#v，期望 %#v", got, want)
			}
			if test.name == "PowerShell" && (!containsEnvironmentValue(process.Env, "TASKAI_EXEC_COMMAND=codex") || !containsEnvironmentValue(process.Env, `TASKAI_EXEC_ARGUMENTS=["--full-auto"]`)) {
				t.Fatalf("PowerShell 命令环境 = %#v，未保留独立命令和参数", process.Env)
			}
		})
	}
}

func TestTaskMenuCommandProcessHelper(t *testing.T) {
	if os.Getenv("TASKAI_MENU_COMMAND_HELPER") != "1" {
		return
	}
}

func TestTaskScriptProcessHelper(t *testing.T) {
	if os.Getenv("TASKAI_SCRIPT_PROCESS_HELPER") != "1" {
		return
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatalf("读取标准输入: %v", err)
	}
	if err := os.WriteFile(os.Getenv("TASKAI_SCRIPT_PROCESS_OUTPUT"), input, 0o600); err != nil {
		t.Fatalf("写入脚本输入: %v", err)
	}
}

func runningAppWithTaskMenuItem(t *testing.T, item settings.TaskMenuItem) (*App, task.Task) {
	t.Helper()
	app := newApp(t.TempDir())
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: t.TempDir(), TaskTreeWidth: settings.DefaultTaskTreeWidth, TaskMenuItems: []settings.TaskMenuItem{item},
	}); err != nil {
		t.Fatalf("保存菜单设置: %v", err)
	}
	created, err := app.CreateTask("运行命令", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("开始任务: %v", err)
	}
	return app, started
}

type controlledCommandWaiter struct {
	done chan error
}

func (waiter *controlledCommandWaiter) Wait() error { return <-waiter.done }

type commandWaiterFunc func() error

func (waiter commandWaiterFunc) Wait() error { return waiter() }

func containsEnvironmentValue(environment []string, want string) bool {
	for _, value := range environment {
		if value == want {
			return true
		}
	}
	return false
}

func assertStatusEnvironment(t *testing.T, environment []string, apiURL, taskID, terminalID string) {
	t.Helper()
	if !containsEnvironmentValue(environment, "TASKAI_STATUS_API="+apiURL) ||
		!containsEnvironmentValue(environment, "TASKAI_TASK_ID="+taskID) ||
		!containsEnvironmentValue(environment, "TASKAI_TERMINAL_ID="+terminalID) {
		t.Fatalf("终端环境 = %#v", environment)
	}
}

func assertTaskOnlyEnvironment(t *testing.T, environment []string, taskID string) {
	t.Helper()
	if !reflect.DeepEqual(environment, []string{"TASKAI_TASK_ID=" + taskID}) {
		t.Fatalf("任务环境 = %#v，期望仅包含任务 ID", environment)
	}
}

type capturingTerminalBackend struct {
	requests map[string]terminal.StartRequest
	onStart  func(terminal.StartRequest)
}

func (backend *capturingTerminalBackend) Start(request terminal.StartRequest) (terminal.Session, error) {
	if backend.requests == nil {
		backend.requests = make(map[string]terminal.StartRequest)
	}
	backend.requests[request.ID] = request
	if backend.onStart != nil {
		backend.onStart(request)
	}
	return capturingTerminalSession{id: request.ID}, nil
}

func (backend *capturingTerminalBackend) request(terminalID string) terminal.StartRequest {
	return backend.requests[terminalID]
}

type capturingTerminalSession struct {
	id string
}

func (session capturingTerminalSession) ID() string             { return session.id }
func (capturingTerminalSession) Read([]byte) (int, error)       { return 0, io.EOF }
func (capturingTerminalSession) Write(data []byte) (int, error) { return len(data), nil }
func (capturingTerminalSession) Close() error                   { return nil }
func (capturingTerminalSession) Resize(uint16, uint16) error    { return nil }
func (capturingTerminalSession) Wait() error                    { return nil }

func availableLoopbackPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("申请测试端口: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func receiveCommandEvent(t *testing.T, events <-chan string) string {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("等待命令执行事件超时")
		return ""
	}
}
