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

	"taskai/internal/lifecycle"
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

func TestAppKeepsPendingTaskWhenBeforeStartChainFails(t *testing.T) {
	app := newApp(t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.LifecycleCommands = append(current.LifecycleCommands, settings.LifecycleCommand{
		ID: "fail", Kind: settings.LifecycleCommandKindCustom, Name: "失败前置", Command: "fail",
	})
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: "fail-before-start", Name: "失败前置链", CommandIDs: []string{"fail"},
	})
	current.LifecycleDefaultChains[task.LifecycleHookBeforeStart] = "fail-before-start"
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		return lifecycle.CommandResult{StandardError: []byte("拒绝开始")}, errors.New("exit status 1")
	}))

	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	updated, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if updated.Status != task.StatusPending || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookBeforeStart || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("beforeStart 失败后的任务 = %#v", updated)
	}
}

func TestAppPostStartFailureKeepsTaskRunning(t *testing.T) {
	app := newApp(t.TempDir())
	configureLifecycleFailure(t, app, task.LifecycleHookPostStart)
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	updated, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if updated.Status != task.StatusRunning || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookPostStart || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("postStart 失败后的任务 = %#v", updated)
	}
}

func TestAppBeforeEndFailureKeepsTaskRunning(t *testing.T) {
	app := newApp(t.TempDir())
	configureLifecycleFailure(t, app, task.LifecycleHookBeforeEnd)
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	updated, err := app.FinishTask(created.ID)
	if err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	if updated.Status != task.StatusRunning || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookBeforeEnd || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("beforeEnd 失败后的任务 = %#v", updated)
	}
}

func TestAppPostEndFailureKeepsTaskCompleted(t *testing.T) {
	app := newApp(t.TempDir())
	configureLifecycleFailure(t, app, task.LifecycleHookPostEnd)
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	updated, err := app.FinishTask(created.ID)
	if err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	if updated.Status != task.StatusCompleted || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookPostEnd || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("postEnd 失败后的任务 = %#v", updated)
	}
}

func TestAppUpdateTaskFailureDoesNotRollbackSavedDetails(t *testing.T) {
	app := newApp(t.TempDir())
	configureLifecycleFailure(t, app, task.LifecycleHookUpdateTask)
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	updated, err := app.UpdateTask(created.ID, "已保存的标题", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Title != "已保存的标题" || updated.Status != task.StatusRunning || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookUpdateTask || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("updateTask 失败后的任务 = %#v", updated)
	}
}

func TestAppUsesHookSpecificDirectoryAndHTTPCommandInput(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.StatusManagementHTTPPort = availableLoopbackPort(t)
	current.HTTPServiceEnabled = true
	current.LifecycleCommands = append(current.LifecycleCommands,
		settings.LifecycleCommand{ID: "before-command", Kind: settings.LifecycleCommandKindCustom, Name: "开始前检查", Command: "before"},
		settings.LifecycleCommand{ID: "post-command", Kind: settings.LifecycleCommandKindCustom, Name: "结束后检查", Command: "post"},
	)
	current.LifecycleChains = append(current.LifecycleChains,
		settings.LifecycleCommandChain{ID: "before-chain", Name: "开始前链", CommandIDs: []string{settings.LifecycleCommandCreateWorkspaceID, "before-command"}},
		settings.LifecycleCommandChain{ID: "post-chain", Name: "结束后链", CommandIDs: []string{"post-command"}},
	)
	current.LifecycleDefaultChains[task.LifecycleHookBeforeStart] = "before-chain"
	current.LifecycleDefaultChains[task.LifecycleHookPostEnd] = "post-chain"
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	directories := make(map[string]string)
	baseURL := ""
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(invocation lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		directories[invocation.Command] = invocation.Directory
		if invocation.Command == "before" {
			if _, err := os.Stat(invocation.Directory); err != nil {
				t.Fatalf("beforeStart 自定义命令启动时工作目录不存在: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(invocation.Input, &payload); err != nil {
				t.Fatalf("解析生命周期命令输入: %v", err)
			}
			baseURL, _ = payload["baseURL"].(string)
		}
		return lifecycle.CommandResult{Output: []byte("ok")}, nil
	}))
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if got, want := directories["before"], started.WorkspacePath; got != want {
		t.Fatalf("beforeStart 工作目录 = %q，期望 %q", got, want)
	}
	if baseURL == "" || baseURL != app.statusHTTP.APIURL() {
		t.Fatalf("beforeStart baseURL = %q，HTTP 服务 = %q", baseURL, app.statusHTTP.APIURL())
	}
	if _, err := app.FinishTask(created.ID); err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	if got, want := directories["post"], started.WorkspaceRoot; got != want {
		t.Fatalf("postEnd 工作目录 = %q，期望根目录 %q", got, want)
	}
}

func TestLifecycleChainAppendsReferenceArgumentsToCommandArguments(t *testing.T) {
	_, commands, err := lifecycleChain(settings.Settings{
		LifecycleCommands: []settings.LifecycleCommand{{
			ID: "prepare", Kind: settings.LifecycleCommandKindCustom, Name: "准备", Command: "prepare", Arguments: []string{"--verbose"},
		}},
		LifecycleChains: []settings.LifecycleCommandChain{{
			ID: "chain", Name: "准备链", Commands: []settings.LifecycleCommandReference{{CommandID: "prepare", Arguments: []string{"--profile", "dev"}}},
		}},
	}, "chain")
	if err != nil {
		t.Fatalf("lifecycleChain() error = %v", err)
	}
	if len(commands) != 1 || !reflect.DeepEqual(commands[0].Arguments, []string{"--verbose", "--profile", "dev"}) {
		t.Fatalf("合并后的命令参数 = %#v", commands)
	}
}

func configureLifecycleFailure(t *testing.T, app *App, hook task.LifecycleHook) {
	t.Helper()
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	commandID := "fail-" + string(hook)
	chainID := "chain-" + string(hook)
	current.LifecycleCommands = append(current.LifecycleCommands, settings.LifecycleCommand{
		ID: commandID, Kind: settings.LifecycleCommandKindCustom, Name: "失败命令", Command: "fail",
	})
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: chainID, Name: "失败链", CommandIDs: []string{commandID},
	})
	current.LifecycleDefaultChains[hook] = chainID
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		return lifecycle.CommandResult{StandardError: []byte("失败")}, errors.New("exit status 1")
	}))
}

func TestAppRunsUpdateTaskHookOnlyForRunningTask(t *testing.T) {
	app := newApp(t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.LifecycleCommands = append(current.LifecycleCommands, settings.LifecycleCommand{
		ID: "record-update", Kind: settings.LifecycleCommandKindCustom, Name: "记录更新", Command: "record",
	})
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: "update-chain", Name: "更新链", CommandIDs: []string{"record-update"},
	})
	current.LifecycleDefaultChains[task.LifecycleHookUpdateTask] = "update-chain"
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	calls := 0
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		calls++
		return lifecycle.CommandResult{Output: []byte(`{"ok":true}`)}, nil
	}))
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := app.UpdateTask(created.ID, "未执行任务", "", task.DefaultColor); err != nil {
		t.Fatalf("更新未执行任务: %v", err)
	}
	if calls != 0 {
		t.Fatalf("未执行任务更新命令次数 = %d，期望 0", calls)
	}

	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if _, err := app.UpdateTask(created.ID, "执行中任务", "", task.DefaultColor); err != nil {
		t.Fatalf("更新执行中任务: %v", err)
	}
	if calls != 1 {
		t.Fatalf("执行中任务更新命令次数 = %d，期望 1", calls)
	}

	if _, err := app.FinishTask(created.ID); err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	if _, err := app.UpdateTask(created.ID, "已完成任务", "", task.DefaultColor); err != nil {
		t.Fatalf("更新已完成任务: %v", err)
	}
	if calls != 1 {
		t.Fatalf("已完成任务更新命令次数 = %d，期望 1", calls)
	}
}

func TestAppRetriesFailedLifecycleChainFromFirstCommand(t *testing.T) {
	app := newApp(t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.LifecycleCommands = append(current.LifecycleCommands, settings.LifecycleCommand{
		ID: "retryable", Kind: settings.LifecycleCommandKindCustom, Name: "可重试命令", Command: "retryable",
	})
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: "retry-before-start", Name: "可重试开始链", CommandIDs: []string{"retryable"},
	})
	current.LifecycleDefaultChains[task.LifecycleHookBeforeStart] = "retry-before-start"
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	fail := true
	calls := 0
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		calls++
		if fail {
			return lifecycle.CommandResult{StandardError: []byte("第一次失败")}, errors.New("exit status 1")
		}
		return lifecycle.CommandResult{Output: []byte("ok")}, nil
	}))
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	failed, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if failed.LifecycleExecution == nil || failed.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("失败记录 = %#v", failed.LifecycleExecution)
	}
	if _, err := app.StartTask(created.ID); err == nil {
		t.Fatal("失败中的任务再次开始执行 error = nil，期望被锁定")
	}

	fail = false
	retried, err := app.RetryTaskLifecycleCommandChain(created.ID)
	if err != nil {
		t.Fatalf("RetryTaskLifecycleCommandChain() error = %v", err)
	}
	if retried.Status != task.StatusRunning || retried.LifecycleExecution != nil {
		t.Fatalf("重试后的任务 = %#v", retried)
	}
	if calls != 2 {
		t.Fatalf("命令执行次数 = %d，期望从第一个命令重试后为 2", calls)
	}
}

func TestAppUpdatesLifecycleChainSelectionsOnlyForPendingTasks(t *testing.T) {
	app := newApp(t.TempDir())
	selected := map[task.LifecycleHook]string{
		task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
	}
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("任务", "", task.DefaultColor, nil, selected)
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndLifecycleChains() error = %v", err)
	}
	if !reflect.DeepEqual(created.LifecycleChains, selected) {
		t.Fatalf("创建绑定的生命周期命令链 = %#v，期望 %#v", created.LifecycleChains, selected)
	}
	updated, err := app.UpdateTaskWithExtraInfoAndLifecycleChains(created.ID, "任务", "", task.DefaultColor, nil, map[task.LifecycleHook]string{
		task.LifecycleHookPostEnd: settings.LifecycleChainDeleteWorkspaceID,
	})
	if err != nil {
		t.Fatalf("未执行任务更新命令链 error = %v", err)
	}
	if got := updated.LifecycleChains; !reflect.DeepEqual(got, map[task.LifecycleHook]string{task.LifecycleHookPostEnd: settings.LifecycleChainDeleteWorkspaceID}) {
		t.Fatalf("未执行任务命令链 = %#v", got)
	}

	started, err := app.StartTask(updated.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if _, err := app.UpdateTaskWithExtraInfoAndLifecycleChains(started.ID, "执行中任务", "", task.DefaultColor, nil, selected); err == nil {
		t.Fatal("执行中任务更新命令链 error = nil")
	}
	completed, err := app.FinishTask(started.ID)
	if err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	if _, err := app.UpdateTaskWithExtraInfoAndLifecycleChains(completed.ID, "已完成任务", "", task.DefaultColor, nil, selected); err == nil {
		t.Fatal("已完成任务更新命令链 error = nil")
	}
}

func TestAppHTTPTaskDetailIncludesLifecycleConfiguration(t *testing.T) {
	app := newApp(t.TempDir())
	selected := map[task.LifecycleHook]string{
		task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
	}
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("任务", "", task.DefaultColor, nil, selected)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	resource, found, err := app.httpTask(created.ID)
	if err != nil || !found {
		t.Fatalf("httpTask() = (%#v, %t, %v)", resource, found, err)
	}
	if !reflect.DeepEqual(resource.LifecycleChains, selected) {
		t.Fatalf("HTTP 生命周期选择 = %#v，期望 %#v", resource.LifecycleChains, selected)
	}
	input, err := lifecycle.BuildCommandInput(resource, "")
	if err != nil {
		t.Fatalf("BuildCommandInput() error = %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(input, &payload); err != nil {
		t.Fatalf("解析命令输入: %v", err)
	}
	if got := payload["lifecycleChains"].(map[string]any)[string(task.LifecycleHookBeforeStart)]; got != settings.LifecycleChainCreateWorkspaceID {
		t.Fatalf("命令输入 lifecycleChains = %#v", payload["lifecycleChains"])
	}
}

func TestAppExposesLifecycleCommandAndChainManagementBindings(t *testing.T) {
	app := newApp(t.TempDir())
	commands, err := app.ListLifecycleCommands()
	if err != nil || len(commands) < 2 {
		t.Fatalf("ListLifecycleCommands() = %#v, %v", commands, err)
	}
	command, err := app.SaveLifecycleCommand(settings.LifecycleCommand{Name: "转换输入", Command: "transform", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookPostStart}})
	if err != nil {
		t.Fatalf("SaveLifecycleCommand() error = %v", err)
	}
	chain, err := app.SaveLifecycleCommandChain(settings.LifecycleCommandChain{Name: "转换链", CommandIDs: []string{command.ID}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookPostStart}})
	if err != nil {
		t.Fatalf("SaveLifecycleCommandChain() error = %v", err)
	}
	if _, err := app.SaveLifecycleDefaultChain(task.LifecycleHookBeforeStart, chain.ID); err == nil {
		t.Fatal("SaveLifecycleDefaultChain() error = nil，期望拒绝不适用默认链")
	}
	if current, err := app.SaveLifecycleDefaultChain(task.LifecycleHookPostStart, chain.ID); err != nil || current.LifecycleDefaultChains[task.LifecycleHookPostStart] != chain.ID {
		t.Fatalf("SaveLifecycleDefaultChain() = (%#v, %v)", current, err)
	}
	copy, err := app.CopyLifecycleCommandChain(chain.ID)
	if err != nil {
		t.Fatalf("CopyLifecycleCommandChain() error = %v", err)
	}
	if err := app.DeleteLifecycleCommand(command.ID); err == nil {
		t.Fatal("删除被命令链引用的命令 error = nil")
	}
	if err := app.DeleteLifecycleCommandChain(chain.ID); err != nil {
		t.Fatalf("DeleteLifecycleCommandChain() error = %v", err)
	}
	if err := app.DeleteLifecycleCommandChain(copy.ID); err != nil {
		t.Fatalf("DeleteLifecycleCommandChain(副本) error = %v", err)
	}
	if err := app.DeleteLifecycleCommand(command.ID); err != nil {
		t.Fatalf("DeleteLifecycleCommand() error = %v", err)
	}
}

func TestAppRejectsTaskActionsWhileLifecycleChainIsLocked(t *testing.T) {
	app := newApp(t.TempDir())
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	if _, err := app.tasks.UpdateLifecycleExecution(created.ID, &task.LifecycleExecution{
		Hook: task.LifecycleHookBeforeStart, ChainID: settings.LifecycleChainCreateWorkspaceID,
		CurrentCommandID: settings.LifecycleCommandCreateWorkspaceID, CurrentIndex: 1, CommandCount: 1,
		State: task.LifecycleExecutionFailed, Error: "失败",
	}); err != nil {
		t.Fatalf("UpdateLifecycleExecution() error = %v", err)
	}
	if _, err := app.StartTask(created.ID); err == nil {
		t.Fatal("锁定任务开始执行 error = nil")
	}
	if _, err := app.UpdateTask(created.ID, "新名称", "", task.DefaultColor); err == nil {
		t.Fatal("锁定任务修改 error = nil")
	}
	if _, err := app.ReorderTasks(task.StatusPending, []string{created.ID}); err == nil {
		t.Fatal("锁定任务排序 error = nil")
	}
	shelver, ok := any(app).(interface {
		SetTaskShelved(taskID string, shelved bool) ([]task.Task, error)
	})
	if !ok {
		t.Fatal("App 缺少 SetTaskShelved()")
	}
	if _, err := shelver.SetTaskShelved(created.ID, true); err == nil {
		t.Fatal("锁定任务切换搁置状态 error = nil")
	}
}

func TestAppSetsRunningTaskShelved(t *testing.T) {
	app := newApp(t.TempDir())
	created, err := app.CreateTask("执行中任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}

	shelver, ok := any(app).(interface {
		SetTaskShelved(taskID string, shelved bool) ([]task.Task, error)
	})
	if !ok {
		t.Fatal("App 缺少 SetTaskShelved()")
	}
	updated, err := shelver.SetTaskShelved(created.ID, true)
	if err != nil {
		t.Fatalf("SetTaskShelved() error = %v", err)
	}
	if len(updated) != 1 || !updated[0].Shelved || updated[0].Status != task.StatusRunning {
		t.Fatalf("SetTaskShelved() 返回任务 = %#v", updated)
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

func TestAppManagesExtraInfoTemplatesThroughBindings(t *testing.T) {
	app := newApp(t.TempDir())
	created, err := app.SaveExtraInfoTemplate(task.ExtraInfoTemplate{
		Catalogue: "deployment", DisplayName: "部署信息", Fields: []task.ExtraInfoField{{Key: "environment", DisplayName: "环境", DefaultValue: "test"}},
		Parameters: []task.ExtraInfoParameterDefinition{{Key: "region", DisplayName: "区域", Required: true}},
	})
	if err != nil {
		t.Fatalf("保存额外信息模板: %v", err)
	}
	if created.ID == "" {
		t.Fatal("保存额外信息模板未生成 ID")
	}

	listed, err := app.ListExtraInfoTemplates()
	if err != nil {
		t.Fatalf("列出额外信息模板: %v", err)
	}
	if len(listed) != 2 || listed[0].Catalogue != "git" || listed[1].ID != created.ID || listed[1].Fields[0].Key != "name" {
		t.Fatalf("额外信息模板列表 = %#v，期望内置 Git 和自定义模板", listed)
	}
	if err := app.DeleteExtraInfoTemplate(listed[0].ID); err == nil {
		t.Fatal("删除内置 Git 模板未失败")
	}
	if err := app.DeleteExtraInfoTemplate(created.ID); err != nil {
		t.Fatalf("删除额外信息模板: %v", err)
	}
	listed, err = app.ListExtraInfoTemplates()
	if err != nil {
		t.Fatalf("删除后列出额外信息模板: %v", err)
	}
	if len(listed) != 1 || listed[0].Catalogue != "git" {
		t.Fatalf("删除后额外信息模板列表 = %#v，期望只保留内置 Git", listed)
	}
}

func TestAppCreatesAndUpdatesTaskExtraInfoSnapshots(t *testing.T) {
	app := newApp(t.TempDir())
	template := gitTemplateForTest(t, app)
	info, err := task.NewExtraInfo(template, map[string]string{
		"name":       "API 服务",
		"repository": "git@example.com:team/api.git",
	})
	if err != nil {
		t.Fatalf("创建额外信息: %v", err)
	}
	info, err = app.SaveExtraInfo(info)
	if err != nil {
		t.Fatalf("保存额外信息: %v", err)
	}
	requested := task.TaskExtraInfo{InformationID: info.ID, Parameters: []task.ExtraInfoParameter{
		{Key: "branch", DisplayName: "仓库分支", Required: false, Value: "main"},
		{Key: "tag", DisplayName: "发布标签", Value: "v1.0.0"},
	}}
	created, err := app.CreateTaskWithExtraInfo("关联仓库", "", task.DefaultColor, []task.TaskExtraInfo{requested})
	if err != nil {
		t.Fatalf("创建带附加信息任务: %v", err)
	}
	if len(created.ExtraInfo) != 1 || created.ExtraInfo[0].InformationID != info.ID || created.ExtraInfo[0].Fields[0].Value != "API 服务" || created.ExtraInfo[0].Fields[1].Value != "git@example.com:team/api.git" {
		t.Fatalf("创建任务的附加信息 = %#v", created.ExtraInfo)
	}
	updatedSnapshot := created.ExtraInfo[0]
	updatedSnapshot.Parameters[0].Value = "release/1.0"
	updated, err := app.UpdateTaskWithExtraInfo(created.ID, "关联仓库", "已更新分支", task.DefaultColor, []task.TaskExtraInfo{updatedSnapshot})
	if err != nil {
		t.Fatalf("更新任务附加信息: %v", err)
	}
	if updated.Description != "已更新分支" || updated.ExtraInfo[0].Parameters[0].Value != "release/1.0" {
		t.Fatalf("更新任务 = %#v", updated)
	}

	if err := app.DeleteExtraInfo(info.ID); err != nil {
		t.Fatalf("删除源信息: %v", err)
	}
	updatedSnapshot = updated.ExtraInfo[0]
	updatedSnapshot.Parameters[0].Value = "release/1.1"
	updated, err = app.UpdateTaskWithExtraInfo(created.ID, "关联仓库", "来源已删除", task.DefaultColor, []task.TaskExtraInfo{updatedSnapshot})
	if err != nil {
		t.Fatalf("删除来源后更新任务动态参数: %v", err)
	}
	loaded, err := app.ListTasks()
	if err != nil {
		t.Fatalf("删除模板后列出任务: %v", err)
	}
	if !reflect.DeepEqual(loaded[0].ExtraInfo, updated.ExtraInfo) {
		t.Fatalf("模板删除影响了任务快照 = %#v", loaded[0].ExtraInfo)
	}
	invalid := updated.ExtraInfo[0]
	invalid.Fields[1].Value = "git@example.com:tampered/repository.git"
	if _, err := app.UpdateTaskWithExtraInfo(created.ID, "关联仓库", "", task.DefaultColor, []task.TaskExtraInfo{invalid}); err == nil {
		t.Fatal("修改任务固定字段未失败")
	}
}

func TestAppUsesInformationParametersAsProtectedTaskDefaults(t *testing.T) {
	app := newApp(t.TempDir())
	template := gitTemplateForTest(t, app)
	info, err := app.SaveExtraInfo(task.ExtraInfo{
		TemplateID: template.ID,
		Catalogue:  template.Catalogue,
		Fields: []task.ExtraInfoField{
			{Key: "name", Value: "API 服务"},
			{Key: "repository", Value: "git@example.com:team/api.git"},
		},
		Parameters: []task.ExtraInfoParameter{{Key: "environment", DisplayName: "环境", Required: true, Value: "production"}},
	})
	if err != nil {
		t.Fatalf("保存包含信息级参数的信息: %v", err)
	}

	created, err := app.CreateTaskWithExtraInfo("关联 API", "", task.DefaultColor, []task.TaskExtraInfo{{
		InformationID: info.ID,
		Parameters: []task.ExtraInfoParameter{
			{Key: "branch", DisplayName: "仓库分支", Required: false, Value: "main"},
			{Key: "environment", DisplayName: "环境", Required: true, Value: "staging"},
			{Key: "tag", DisplayName: "发布标签", Required: false, Value: "v1.2.0"},
		},
	}})
	if err != nil {
		t.Fatalf("创建带信息级参数的任务: %v", err)
	}
	if got := taskParameterValue(created.ExtraInfo[0].Parameters, "environment"); got != "staging" {
		t.Fatalf("任务信息级参数值 = %q，期望任务填写值 staging", got)
	}
	if got := taskParameterValue(created.ExtraInfo[0].Parameters, "tag"); got != "v1.2.0" {
		t.Fatalf("任务级动态参数值 = %q，期望保留", got)
	}

	_, err = app.CreateTaskWithExtraInfo("伪造参数", "", task.DefaultColor, []task.TaskExtraInfo{{
		InformationID: info.ID,
		Parameters: []task.ExtraInfoParameter{
			{Key: "branch", DisplayName: "仓库分支", Required: false, Value: "main"},
			{Key: "environment", DisplayName: "篡改环境", Required: true, Value: "production"},
		},
	}})
	if err == nil {
		t.Fatal("伪造信息级参数定义未失败")
	}
}

func TestAppHTTPTaskDetailFlattensExtraInfoByCatalogue(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: t.TempDir(), TaskTreeWidth: settings.DefaultTaskTreeWidth,
		StatusManagementMode: settings.StatusManagementModeTitleChange,
		HTTPServiceEnabled:   true, StatusManagementHTTPPort: availableLoopbackPort(t),
	}); err != nil {
		t.Fatalf("启用任务 HTTP 服务: %v", err)
	}
	template := gitTemplateForTest(t, app)
	first, err := app.SaveExtraInfo(task.ExtraInfo{TemplateID: template.ID, Catalogue: template.Catalogue, Fields: []task.ExtraInfoField{
		{Key: "name", Value: "API 服务"},
		{Key: "repository", Value: "git@example.com:team/api.git"},
	}})
	if err != nil {
		t.Fatalf("保存第一个信息: %v", err)
	}
	second, err := app.SaveExtraInfo(task.ExtraInfo{TemplateID: template.ID, Catalogue: template.Catalogue, Fields: []task.ExtraInfoField{
		{Key: "name", Value: "Web 服务"},
		{Key: "repository", Value: "git@example.com:team/web.git"},
	}})
	if err != nil {
		t.Fatalf("保存第二个信息: %v", err)
	}
	created, err := app.CreateTaskWithExtraInfo("查询附加信息", "", task.DefaultColor, []task.TaskExtraInfo{
		{InformationID: first.ID, Parameters: []task.ExtraInfoParameter{{Key: "branch", DisplayName: "仓库分支", Value: "main"}}},
		{InformationID: second.ID, Parameters: []task.ExtraInfoParameter{{Key: "branch", DisplayName: "仓库分支", Value: ""}}},
	})
	if err != nil {
		t.Fatalf("创建带附加信息任务: %v", err)
	}

	response, err := http.Get(app.statusHTTP.APIURL() + "/tasks/" + created.ID)
	if err != nil {
		t.Fatalf("查询任务详情: %v", err)
	}
	defer response.Body.Close()
	var detail struct {
		ExtraInfo map[string][]map[string]string `json:"extraInfo"`
	}
	if err := json.NewDecoder(response.Body).Decode(&detail); err != nil {
		t.Fatalf("解析任务详情: %v", err)
	}
	want := map[string][]map[string]string{
		"git": {
			{"name": "API 服务", "repository": "git@example.com:team/api.git", "branch": "main"},
			{"name": "Web 服务", "repository": "git@example.com:team/web.git", "branch": ""},
		},
	}
	if !reflect.DeepEqual(detail.ExtraInfo, want) {
		t.Fatalf("任务详情附加信息 = %#v，期望 %#v", detail.ExtraInfo, want)
	}
}

func TestAppHTTPTaskDetailIncludesActiveTerminalDetails(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	backend := &activeTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	t.Cleanup(func() { _ = app.terminals.CloseAll() })
	configured, err := app.GetSettings()
	if err != nil {
		t.Fatalf("读取默认设置: %v", err)
	}
	configured.WorkspaceRoot = t.TempDir()
	configured.HTTPServiceEnabled = true
	configured.StatusManagementHTTPPort = availableLoopbackPort(t)
	configured, err = app.SaveSettings(configured)
	if err != nil {
		t.Fatalf("启用任务 HTTP 服务: %v", err)
	}
	created, err := app.CreateTask("终端详情", "", task.DefaultColor)
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
	command, err := app.CreateCommandTerminal(started.ID, "codex", []string{"--full-auto"}, 100, 32)
	if err != nil {
		t.Fatalf("创建命令终端: %v", err)
	}
	if !app.realtime.SetTerminalStatus(started.ID, normal.ID, realtime.StatusWorking) {
		t.Fatal("设置普通终端状态失败")
	}
	if !app.realtime.SetTerminalStatus(started.ID, command.ID, realtime.StatusIdle) {
		t.Fatal("设置命令终端状态失败")
	}

	response, err := http.Get(app.statusHTTP.APIURL() + "/tasks/" + started.ID)
	if err != nil {
		t.Fatalf("查询任务详情: %v", err)
	}
	defer response.Body.Close()
	var detail struct {
		Terminals []struct {
			ID      string          `json:"id"`
			Command string          `json:"command"`
			Status  realtime.Status `json:"status"`
		} `json:"terminals"`
	}
	if err := json.NewDecoder(response.Body).Decode(&detail); err != nil {
		t.Fatalf("解析任务详情: %v", err)
	}
	if len(detail.Terminals) != 2 {
		t.Fatalf("任务详情终端 = %#v，期望两个活动终端", detail.Terminals)
	}
	terminalsByID := make(map[string]struct {
		Command string
		Status  realtime.Status
	}, len(detail.Terminals))
	for _, item := range detail.Terminals {
		terminalsByID[item.ID] = struct {
			Command string
			Status  realtime.Status
		}{Command: item.Command, Status: item.Status}
	}
	if got := terminalsByID[normal.ID]; got.Command != configured.ShellPath || got.Status != realtime.StatusWorking {
		t.Fatalf("普通终端详情 = %#v，期望命令 %q、状态 working", got, configured.ShellPath)
	}
	if got := terminalsByID[command.ID]; got.Command != "codex" || got.Status != realtime.StatusIdle {
		t.Fatalf("命令终端详情 = %#v，期望命令 codex、状态 idle", got)
	}

	response, err = http.Get(app.statusHTTP.APIURL() + "/tasks")
	if err != nil {
		t.Fatalf("查询任务列表: %v", err)
	}
	defer response.Body.Close()
	var listed []map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		t.Fatalf("解析任务列表: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("任务列表 = %#v，期望一个任务", listed)
	}
	if _, ok := listed[0]["terminals"]; ok {
		t.Fatalf("任务列表不应包含 terminals: %#v", listed[0])
	}

	withoutTerminal, err := app.CreateTask("无终端详情", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建无终端任务: %v", err)
	}
	response, err = http.Get(app.statusHTTP.APIURL() + "/tasks/" + withoutTerminal.ID)
	if err != nil {
		t.Fatalf("查询无终端任务详情: %v", err)
	}
	defer response.Body.Close()
	var emptyDetail struct {
		Terminals []json.RawMessage `json:"terminals"`
	}
	if err := json.NewDecoder(response.Body).Decode(&emptyDetail); err != nil {
		t.Fatalf("解析无终端任务详情: %v", err)
	}
	if emptyDetail.Terminals == nil || len(emptyDetail.Terminals) != 0 {
		t.Fatalf("无终端任务详情 = %#v，期望 terminals 为空数组", emptyDetail.Terminals)
	}
}

func TestAppHTTPTaskDetailReturnsCheckboxParameterAsBoolean(t *testing.T) {
	app := newApp(t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: t.TempDir(), TaskTreeWidth: settings.DefaultTaskTreeWidth,
		StatusManagementMode: settings.StatusManagementModeTitleChange,
		HTTPServiceEnabled:   true, StatusManagementHTTPPort: availableLoopbackPort(t),
	}); err != nil {
		t.Fatalf("启用任务 HTTP 服务: %v", err)
	}
	template := gitTemplateForTest(t, app)
	info, err := app.SaveExtraInfo(task.ExtraInfo{TemplateID: template.ID, Catalogue: template.Catalogue, Fields: []task.ExtraInfoField{
		{Key: "name", Value: "部署服务"},
		{Key: "repository", Value: "git@example.com:team/deploy.git"},
	}})
	if err != nil {
		t.Fatalf("保存 Git 信息: %v", err)
	}
	created, err := app.CreateTaskWithExtraInfo("查询复选框参数", "", task.DefaultColor, []task.TaskExtraInfo{{
		InformationID: info.ID,
		Parameters: []task.ExtraInfoParameter{
			{Key: "branch", DisplayName: "仓库分支", Value: "main"},
			{Key: "deploy", DisplayName: "允许部署", InputType: task.ExtraInfoParameterInputCheckbox, Value: "true"},
			{Key: "reviewed", DisplayName: "已复核", InputType: task.ExtraInfoParameterInputCheckbox, Value: "false"},
		},
	}})
	if err != nil {
		t.Fatalf("创建带复选框参数的任务: %v", err)
	}

	response, err := http.Get(app.statusHTTP.APIURL() + "/tasks/" + created.ID)
	if err != nil {
		t.Fatalf("查询任务详情: %v", err)
	}
	defer response.Body.Close()
	var detail struct {
		ExtraInfo map[string][]map[string]any `json:"extraInfo"`
	}
	if err := json.NewDecoder(response.Body).Decode(&detail); err != nil {
		t.Fatalf("解析任务详情: %v", err)
	}
	values := detail.ExtraInfo["git"][0]
	if got, ok := values["branch"].(string); !ok || got != "main" {
		t.Fatalf("文本动态参数 = %#v，期望字符串 main", values["branch"])
	}
	if got, ok := values["deploy"].(bool); !ok || !got {
		t.Fatalf("复选框动态参数 = %#v，期望布尔值 true", values["deploy"])
	}
	if got, ok := values["reviewed"].(bool); !ok || got {
		t.Fatalf("复选框动态参数 = %#v，期望布尔值 false", values["reviewed"])
	}
}

func gitTemplateForTest(t *testing.T, app *App) task.ExtraInfoTemplate {
	t.Helper()
	templates, err := app.ListExtraInfoTemplates()
	if err != nil {
		t.Fatalf("列出 Git 模板: %v", err)
	}
	for _, template := range templates {
		if template.Catalogue == "git" && template.BuiltIn {
			return template
		}
	}
	t.Fatal("未找到内置 Git 模板")
	return task.ExtraInfoTemplate{}
}

func taskParameterValue(parameters []task.ExtraInfoParameter, key string) string {
	for _, parameter := range parameters {
		if parameter.Key == key {
			return parameter.Value
		}
	}
	return ""
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

type activeTerminalBackend struct{}

func (activeTerminalBackend) Start(request terminal.StartRequest) (terminal.Session, error) {
	reader, writer := io.Pipe()
	return &activeTerminalSession{id: request.ID, reader: reader, writer: writer}, nil
}

type activeTerminalSession struct {
	id     string
	reader *io.PipeReader
	writer *io.PipeWriter
}

func (session *activeTerminalSession) ID() string { return session.id }
func (session *activeTerminalSession) Read(data []byte) (int, error) {
	return session.reader.Read(data)
}
func (session *activeTerminalSession) Write(data []byte) (int, error) { return len(data), nil }
func (session *activeTerminalSession) Resize(uint16, uint16) error    { return nil }
func (session *activeTerminalSession) Wait() error                    { return nil }
func (session *activeTerminalSession) Close() error {
	_ = session.writer.Close()
	return session.reader.Close()
}

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
