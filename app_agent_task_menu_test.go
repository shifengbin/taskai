package main

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"taskai/internal/settings"
	"taskai/internal/task"
	"taskai/internal/terminal"
)

func TestStartupPersistsDetectedAgentTaskMenusOnce(t *testing.T) {
	app := newApp(t.TempDir())
	app.agentCommandDetector = func() settings.DetectedAgentCommands {
		return settings.DetectedAgentCommands{Codex: true, Claude: true}
	}
	saveCalls := 0
	app.agentMenuSynchronizer = func(detected settings.DetectedAgentCommands) (settings.Settings, bool, error) {
		current, changed, err := app.repository.MergeDetectedAgentTaskMenuItems(detected)
		if changed {
			saveCalls++
		}
		return current, changed, err
	}

	app.startup(nil)
	app.startup(nil)

	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if saveCalls != 1 {
		t.Fatalf("代理菜单设置保存次数 = %d，期望 1", saveCalls)
	}
	want := []settings.TaskMenuItem{
		{ID: settings.TaskMenuItemDetectedCodexID, Kind: settings.TaskMenuItemKindCommand, Name: "codex", Command: "codex", Arguments: []string{"--yolo"}, ShowTerminal: true},
		{ID: settings.TaskMenuItemDetectedClaudeID, Kind: settings.TaskMenuItemKindCommand, Name: "claude", Command: "claude", Arguments: []string{"--dangerously-skip-permissions"}, ShowTerminal: true},
	}
	if got := current.TaskMenuItems[len(current.TaskMenuItems)-2:]; !reflect.DeepEqual(got, want) {
		t.Fatalf("启动后的代理菜单项 = %#v，期望 %#v", got, want)
	}
}

func TestStartupKeepsSettingsAtomicWhenAgentMenuSaveFails(t *testing.T) {
	app := newApp(t.TempDir())
	app.agentCommandDetector = func() settings.DetectedAgentCommands {
		return settings.DetectedAgentCommands{Codex: true, Claude: true}
	}
	app.agentMenuSynchronizer = func(settings.DetectedAgentCommands) (settings.Settings, bool, error) {
		return settings.Settings{}, false, errors.New("磁盘已满")
	}
	var published string
	app.startupErrorPublisher = func(message string) {
		published = message
	}

	app.startup(nil)

	if !strings.Contains(published, "磁盘已满") {
		t.Fatalf("启动错误通知 = %q，期望包含保存错误", published)
	}
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	for _, item := range current.TaskMenuItems {
		if item.ID == settings.TaskMenuItemDetectedCodexID || item.ID == settings.TaskMenuItemDetectedClaudeID {
			t.Fatalf("保存失败后出现部分代理菜单项 = %#v", current.TaskMenuItems)
		}
	}
	if app.ctx != nil {
		t.Fatal("使用 nil 上下文启动后 app.ctx 应保持 nil，但启动必须正常返回")
	}
}

func TestStartupDoesNotSaveUnavailableAgentMenus(t *testing.T) {
	app := newApp(t.TempDir())
	app.agentCommandDetector = func() settings.DetectedAgentCommands {
		return settings.DetectedAgentCommands{}
	}
	saveCalls := 0
	app.agentMenuSynchronizer = func(detected settings.DetectedAgentCommands) (settings.Settings, bool, error) {
		current, changed, err := app.repository.MergeDetectedAgentTaskMenuItems(detected)
		if changed {
			saveCalls++
		}
		return current, changed, err
	}

	app.startup(nil)

	if saveCalls != 0 {
		t.Fatalf("没有检测到代理时保存次数 = %d，期望 0", saveCalls)
	}
}

func TestExecuteDetectedAgentTaskMenusUsesVisibleTerminalAndTaskDirectory(t *testing.T) {
	items, changed := settings.MergeDetectedAgentTaskMenuItems(nil, settings.DetectedAgentCommands{Codex: true, Claude: true})
	if !changed {
		t.Fatal("测试准备未生成代理菜单项")
	}
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	configured, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	configured.WorkspaceRoot = t.TempDir()
	configured.TaskMenuItems = items
	if _, err := app.SaveSettings(configured); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	created, err := app.CreateTask("代理菜单", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)

	wants := []struct {
		itemID    string
		command   string
		arguments []string
	}{
		{itemID: settings.TaskMenuItemDetectedCodexID, command: "codex", arguments: []string{"--yolo"}},
		{itemID: settings.TaskMenuItemDetectedClaudeID, command: "claude", arguments: []string{"--dangerously-skip-permissions"}},
	}
	for _, want := range wants {
		result, err := app.ExecuteTaskMenuCommand(started.ID, want.itemID, 100, 32)
		if err != nil {
			t.Fatalf("ExecuteTaskMenuCommand(%q) error = %v", want.itemID, err)
		}
		if result.Terminal == nil {
			t.Fatalf("ExecuteTaskMenuCommand(%q) 未返回显示终端", want.itemID)
		}
		request := backend.request(result.Terminal.ID)
		if request.Directory != started.WorkspacePath || request.Command != want.command || !reflect.DeepEqual(request.Arguments, want.arguments) {
			t.Fatalf("%s 启动请求 = %#v，期望目录 %q、命令 %q、参数 %#v", want.itemID, request, started.WorkspacePath, want.command, want.arguments)
		}
	}
}

func TestGetSettingsWaitsForStartupAgentMenuSynchronization(t *testing.T) {
	app := newApp(t.TempDir())
	app.startupReady = make(chan struct{})
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	app.agentMenuSynchronizer = func(settings.DetectedAgentCommands) (settings.Settings, bool, error) {
		once.Do(func() { close(started) })
		<-release
		data, err := app.repository.Load()
		return data.Settings, false, err
	}

	startupDone := make(chan struct{})
	go func() {
		app.startup(nil)
		close(startupDone)
	}()
	<-started

	settingsResult := make(chan error, 1)
	go func() {
		_, err := app.GetSettings()
		settingsResult <- err
	}()
	select {
	case err := <-settingsResult:
		t.Fatalf("启动同步完成前 GetSettings() 提前返回: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case err := <-settingsResult:
		if err != nil {
			t.Fatalf("GetSettings() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("启动同步完成后 GetSettings() 未返回")
	}
	select {
	case <-startupDone:
	case <-time.After(time.Second):
		t.Fatal("startup() 未返回")
	}
}

func TestSaveSettingsWaitsForStartupAgentMenuSynchronization(t *testing.T) {
	app := newApp(t.TempDir())
	app.startupReady = make(chan struct{})
	started := make(chan struct{})
	release := make(chan struct{})
	app.agentMenuSynchronizer = func(settings.DetectedAgentCommands) (settings.Settings, bool, error) {
		close(started)
		<-release
		data, err := app.repository.Load()
		return data.Settings, false, err
	}

	go app.startup(nil)
	<-started
	next, err := app.loadSettings()
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}
	saveResult := make(chan error, 1)
	go func() {
		_, err := app.SaveSettings(next)
		saveResult <- err
	}()
	select {
	case err := <-saveResult:
		t.Fatalf("启动同步完成前 SaveSettings() 提前返回: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-saveResult:
		if err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("启动同步完成后 SaveSettings() 未返回")
	}
}
