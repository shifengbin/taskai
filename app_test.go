package main

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"taskai/internal/lifecycle"
	"taskai/internal/quickinput"
	"taskai/internal/realtime"
	"taskai/internal/repositorygit"
	"taskai/internal/settings"
	"taskai/internal/task"
	"taskai/internal/terminal"
	"taskai/internal/workspace"
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

func newAppWithoutActiveTaskTemplate(t *testing.T, dataDirectory string) *App {
	t.Helper()
	app := newApp(dataDirectory)
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.TaskTemplates = []task.TaskTemplate{}
	current.ActiveTaskTemplateID = ""
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if _, err := app.SaveDefaultLifecyclePreset(settings.DefaultLifecyclePresetID); err != nil {
		t.Fatalf("SaveDefaultLifecyclePreset() error = %v", err)
	}
	return app
}

func TestAppExposesTaskAndSettingsBindings(t *testing.T) {
	t.Parallel()

	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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

	started := startTaskAndWait(t, app, created.ID)
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

func TestAppManagesGitRepositoriesOnlyInsideTaskWorkspace(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.WorkspaceRoot = filepath.Join(t.TempDir(), "workspaces")
	current.DefaultLifecyclePresetID = ""
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	created, err := app.CreateTask("管理仓库", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	if err := os.MkdirAll(started.WorkspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	runGitInDirectory(t, started.WorkspacePath, "init", "--initial-branch=main")
	runGitInDirectory(t, started.WorkspacePath, "config", "user.name", "测试用户")
	runGitInDirectory(t, started.WorkspacePath, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(started.WorkspacePath, "README.md"), []byte("new\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	repositories, err := app.ListTaskGitRepositories(started.ID)
	if err != nil {
		t.Fatalf("ListTaskGitRepositories() error = %v", err)
	}
	if len(repositories) != 1 || repositories[0].Action != repositorygit.ActionCommit {
		t.Fatalf("ListTaskGitRepositories() = %#v", repositories)
	}
	if _, err := app.CommitTaskGitRepository(started.ID, "../outside", "不应执行"); err == nil || !strings.Contains(err.Error(), "仓库路径无效") {
		t.Fatalf("CommitTaskGitRepository() error = %v", err)
	}
	committed, err := app.CommitTaskGitRepository(started.ID, ".", "初始提交")
	if err != nil {
		t.Fatalf("CommitTaskGitRepository() error = %v", err)
	}
	if committed.Dirty || committed.Action != repositorygit.ActionSync {
		t.Fatalf("CommitTaskGitRepository() = %#v", committed)
	}
}

func TestAppListsGitRepositoriesForCompletedTaskWorkspace(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.WorkspaceRoot = filepath.Join(t.TempDir(), "workspaces")
	current.DefaultLifecyclePresetID = ""
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	created, err := app.CreateTask("已完成仓库", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	finished := finishTaskAndWait(t, app, started.ID)
	if finished.Status != task.StatusCompleted {
		t.Fatalf("FinishTask() = %#v", finished)
	}
	if err := os.MkdirAll(started.WorkspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	runGitInDirectory(t, started.WorkspacePath, "init", "--initial-branch=main")

	repositories, err := app.ListTaskGitRepositories(finished.ID)
	if err != nil {
		t.Fatalf("ListTaskGitRepositories() error = %v", err)
	}
	if len(repositories) != 1 || repositories[0].Path != "." {
		t.Fatalf("ListTaskGitRepositories() = %#v", repositories)
	}
}

func TestAppListsGitRepositoriesWithConfiguredScanDepth(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.WorkspaceRoot = filepath.Join(t.TempDir(), "workspaces")
	current.DefaultLifecyclePresetID = ""
	current.GitScanDepth = 2
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	created, err := app.CreateTask("扫描深度", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	childPath := filepath.Join(started.WorkspacePath, "child")
	grandchildPath := filepath.Join(childPath, "grandchild")
	if err := os.MkdirAll(grandchildPath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	for _, directory := range []string{childPath, grandchildPath} {
		runGitInDirectory(t, directory, "init", "--initial-branch=main")
	}

	shallow, err := app.ListTaskGitRepositories(started.ID)
	if err != nil {
		t.Fatalf("ListTaskGitRepositories() depth 2 error = %v", err)
	}
	if got := gitRepositoryPaths(shallow); !reflect.DeepEqual(got, []string{"child"}) {
		t.Fatalf("depth 2 paths = %#v", got)
	}

	current.GitScanDepth = 3
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() depth 3 error = %v", err)
	}
	deep, err := app.ListTaskGitRepositories(started.ID)
	if err != nil {
		t.Fatalf("ListTaskGitRepositories() depth 3 error = %v", err)
	}
	if got := gitRepositoryPaths(deep); !reflect.DeepEqual(got, []string{"child", filepath.Join("child", "grandchild")}) {
		t.Fatalf("depth 3 paths = %#v", got)
	}
}

func TestAppReportsTaskGitWorkspaceOnlyAfterDirectoryExists(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("未创建目录", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndLifecycleChains() error = %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	if started.WorkspacePath == "" {
		t.Fatal("任务缺少预期工作目录路径")
	}
	if app.HasTaskGitWorkspace(started.ID) {
		t.Fatal("目录尚未创建时 HasTaskGitWorkspace() = true")
	}
	if err := os.MkdirAll(started.WorkspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if !app.HasTaskGitWorkspace(started.ID) {
		t.Fatal("目录已创建时 HasTaskGitWorkspace() = false")
	}
}

func TestAppExposesLifecyclePresetBindings(t *testing.T) {
	app := newApp(t.TempDir())
	preset, err := app.SaveLifecyclePreset(settings.LifecyclePreset{Name: "空预设", Chains: map[task.LifecycleHook]string{}})
	if err != nil {
		t.Fatalf("SaveLifecyclePreset() error = %v", err)
	}
	listed, err := app.ListLifecyclePresets()
	if err != nil || len(listed) != 3 {
		t.Fatalf("ListLifecyclePresets() = (%#v, %v)", listed, err)
	}
	copy, err := app.CopyLifecyclePreset(preset.ID)
	if err != nil || copy.ID == preset.ID {
		t.Fatalf("CopyLifecyclePreset() = (%#v, %v)", copy, err)
	}
	if _, err := app.SaveDefaultLifecyclePreset(preset.ID); err != nil {
		t.Fatalf("SaveDefaultLifecyclePreset() error = %v", err)
	}
	if err := app.DeleteLifecyclePreset(preset.ID); err != nil {
		t.Fatalf("DeleteLifecyclePreset() error = %v", err)
	}
}

func TestAppManagesQuickInputsThroughBindings(t *testing.T) {
	app := newApp(t.TempDir())
	first, err := app.SaveQuickInput(quickinput.QuickInput{Name: "部署", Content: "git status"})
	if err != nil {
		t.Fatalf("SaveQuickInput() first error = %v", err)
	}
	second, err := app.SaveQuickInput(quickinput.QuickInput{Name: "部署", Content: "git push origin main"})
	if err != nil {
		t.Fatalf("SaveQuickInput() second error = %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("SaveQuickInput() generated duplicate IDs: %q", first.ID)
	}

	ordered, err := app.ReorderQuickInputs([]string{second.ID, first.ID})
	if err != nil {
		t.Fatalf("ReorderQuickInputs() error = %v", err)
	}
	if len(ordered) != 2 || ordered[0].ID != second.ID || ordered[1].ID != first.ID {
		t.Fatalf("ReorderQuickInputs() = %#v", ordered)
	}

	if err := app.DeleteQuickInput(second.ID); err != nil {
		t.Fatalf("DeleteQuickInput() error = %v", err)
	}
	listed, err := app.ListQuickInputs()
	if err != nil {
		t.Fatalf("ListQuickInputs() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != first.ID {
		t.Fatalf("ListQuickInputs() = %#v", listed)
	}
}

func TestAppSaveSettingsWithoutTemplateSnapshotKeepsSavedTaskTemplates(t *testing.T) {
	app := newApp(t.TempDir())
	before, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() before save error = %v", err)
	}

	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: filepath.Join(t.TempDir(), "workspaces"),
		TaskTreeWidth: settings.DefaultTaskTreeWidth + 20,
	}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	after, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() after save error = %v", err)
	}
	if !reflect.DeepEqual(after.TaskTemplates, before.TaskTemplates) || after.ActiveTaskTemplateID != before.ActiveTaskTemplateID {
		t.Fatalf("普通设置保存后的任务模板 = %#v, 当前模板 = %q; want %#v, %q", after.TaskTemplates, after.ActiveTaskTemplateID, before.TaskTemplates, before.ActiveTaskTemplateID)
	}
}

func TestAppRegistersRunningTaskAndClearsRealtimeStatusWhenFinished(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTask("实时状态任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	if got := app.realtime.Snapshot(); len(got.Tasks) != 1 || got.Tasks[0].TaskID != started.ID {
		t.Fatalf("开始任务后的实时状态 = %#v", got)
	}

	finishTaskAndWait(t, app, started.ID)
	if got := app.realtime.Snapshot(); len(got.Tasks) != 0 {
		t.Fatalf("结束任务后的实时状态 = %#v，期望清理", got)
	}
}

func TestAppDeletesPendingTasksAndClearsRealtimeStatus(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTask("待删除记录", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	workspacePath := filepath.Join(t.TempDir(), "retained-workspace")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data, err := app.repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Tasks[0].WorkspacePath = workspacePath
	if err := app.repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	app.realtime.RegisterTask(created.ID)

	remaining, err := app.DeleteTasks([]string{created.ID})
	if err != nil {
		t.Fatalf("DeleteTasks() error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("DeleteTasks() tasks = %#v, want empty", remaining)
	}
	if got := app.realtime.Snapshot(); len(got.Tasks) != 0 {
		t.Fatalf("DeleteTasks() realtime status = %#v", got)
	}
	if _, err := os.Stat(workspacePath); err != nil {
		t.Fatalf("DeleteTasks() removed workspace: %v", err)
	}
}

func TestAppKeepsPendingTaskWhenBeforeStartChainFails(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	current.LifecyclePresets[0].Chains[task.LifecycleHookBeforeStart] = "fail-before-start"
	saveSettingsWithLifecycleConfiguration(t, app, current)
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		return lifecycle.CommandResult{StandardError: []byte("拒绝开始")}, errors.New("exit status 1")
	}))

	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	updated := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusPending && current.LifecycleExecution != nil && current.LifecycleExecution.Hook == task.LifecycleHookBeforeStart && current.LifecycleExecution.State == task.LifecycleExecutionFailed
	})
	if updated.Status != task.StatusPending || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookBeforeStart || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("beforeStart 失败后的任务 = %#v", updated)
	}
}

func TestAppPreservesSavedLifecycleChainWhenTaskStartsAfterStaleSettingsSave(t *testing.T) {
	app := newApp(t.TempDir())
	stale, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	command, err := app.SaveLifecycleCommand(settings.LifecycleCommand{
		Name: "初始化项目", Command: "prepare", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommand() error = %v", err)
	}
	chain, err := app.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
		Name: "项目初始化", Commands: []settings.LifecycleCommandReference{{CommandID: command.ID}}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommandChain() error = %v", err)
	}
	created, err := app.CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains("初始化任务", "", task.DefaultColor, nil, map[string]any{"branch": "main"}, map[task.LifecycleHook]string{
		task.LifecycleHookBeforeStart: chain.ID,
	})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndLifecycleChains() error = %v", err)
	}
	stale.ActiveTaskStatus = settings.TaskStatusPending
	if _, err := app.SaveSettings(stale); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	chains, err := app.ListLifecycleCommandChains()
	if err != nil {
		t.Fatalf("ListLifecycleCommandChains() error = %v", err)
	}
	foundChain := false
	for _, current := range chains {
		if current.ID == chain.ID {
			foundChain = true
			break
		}
	}
	if !foundChain {
		t.Fatalf("任务开始前命令链已丢失: %#v", chains)
	}

	calls := 0
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		calls++
		return lifecycle.CommandResult{Output: []byte("准备完成")}, nil
	}))
	started := startTaskAndWait(t, app, created.ID)
	if started.Status != task.StatusRunning || calls != 1 {
		t.Fatalf("命令链未执行并开始任务: task=%#v, calls=%d", started, calls)
	}
}

func TestAppPostStartFailureKeepsTaskRunning(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	configureLifecycleFailure(t, app, task.LifecycleHookPostStart)
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	updated := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusRunning && current.LifecycleExecution != nil && current.LifecycleExecution.Hook == task.LifecycleHookPostStart && current.LifecycleExecution.State == task.LifecycleExecutionFailed
	})
	if updated.Status != task.StatusRunning || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookPostStart || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("postStart 失败后的任务 = %#v", updated)
	}
}

func TestAppBeforeEndFailureKeepsTaskRunning(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	configureLifecycleFailure(t, app, task.LifecycleHookBeforeEnd)
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	startTaskAndWait(t, app, created.ID)
	if _, err := app.FinishTask(created.ID); err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	updated := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusRunning && current.LifecycleExecution != nil && current.LifecycleExecution.Hook == task.LifecycleHookBeforeEnd && current.LifecycleExecution.State == task.LifecycleExecutionFailed
	})
	if updated.Status != task.StatusRunning || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookBeforeEnd || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("beforeEnd 失败后的任务 = %#v", updated)
	}
}

func TestAppPostEndFailureKeepsTaskCompleted(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	configureLifecycleFailure(t, app, task.LifecycleHookPostEnd)
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	startTaskAndWait(t, app, created.ID)
	if _, err := app.FinishTask(created.ID); err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	updated := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusCompleted && current.LifecycleExecution != nil && current.LifecycleExecution.Hook == task.LifecycleHookPostEnd && current.LifecycleExecution.State == task.LifecycleExecutionFailed
	})
	if updated.Status != task.StatusCompleted || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookPostEnd || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("postEnd 失败后的任务 = %#v", updated)
	}
}

func TestAppUpdateTaskFailureDoesNotRollbackSavedDetails(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	configureLifecycleFailure(t, app, task.LifecycleHookUpdateTask)
	created, err := app.CreateTask("任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	startTaskAndWait(t, app, created.ID)
	if _, err := app.UpdateTask(created.ID, "已保存的标题", "", task.DefaultColor); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	updated := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Title == "已保存的标题" && current.Status == task.StatusRunning && current.LifecycleExecution != nil && current.LifecycleExecution.Hook == task.LifecycleHookUpdateTask && current.LifecycleExecution.State == task.LifecycleExecutionFailed
	})
	if updated.Title != "已保存的标题" || updated.Status != task.StatusRunning || updated.LifecycleExecution == nil || updated.LifecycleExecution.Hook != task.LifecycleHookUpdateTask || updated.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("updateTask 失败后的任务 = %#v", updated)
	}
}

func TestAppUsesHookSpecificDirectoryAndHTTPCommandInput(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	current.LifecyclePresets[0].Chains[task.LifecycleHookBeforeStart] = "before-chain"
	current.LifecyclePresets[0].Chains[task.LifecycleHookPostEnd] = "post-chain"
	saveSettingsWithLifecycleConfiguration(t, app, current)

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
	started := startTaskAndWait(t, app, created.ID)
	if got, want := directories["before"], started.WorkspacePath; got != want {
		t.Fatalf("beforeStart 工作目录 = %q，期望 %q", got, want)
	}
	if baseURL == "" || baseURL != app.statusHTTP.APIURL() {
		t.Fatalf("beforeStart baseURL = %q，HTTP 服务 = %q", baseURL, app.statusHTTP.APIURL())
	}
	finishTaskAndWait(t, app, created.ID)
	if got, want := directories["post"], started.WorkspaceRoot; got != want {
		t.Fatalf("postEnd 工作目录 = %q，期望根目录 %q", got, want)
	}
}

func TestAppPassesCurrentTemplateFieldsToLifecycleCommandInput(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.TaskTemplates = []task.TaskTemplate{{
		ID:   "release",
		Name: "发布任务",
		Fields: []task.TaskTemplateField{
			{Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, Required: true, DefaultValue: "development"},
			{Key: "dryRun", DisplayName: "演练", InputType: task.TaskTemplateFieldInputBool, DefaultValue: true},
		},
	}}
	current.ActiveTaskTemplateID = "release"
	current.LifecycleCommands = append(current.LifecycleCommands, settings.LifecycleCommand{
		ID: "template-input", Kind: settings.LifecycleCommandKindCustom, Name: "读取模板输入", Command: "capture", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: "template-input", Name: "模板输入", Commands: []settings.LifecycleCommandReference{{CommandID: "template-input", Arguments: []string{}}}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	current.LifecyclePresets[0].Chains[task.LifecycleHookBeforeStart] = "template-input"
	saveSettingsWithLifecycleConfiguration(t, app, current)

	payloads := make(chan map[string]any, 1)
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(invocation lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		payload := map[string]any{}
		if err := json.Unmarshal(invocation.Input, &payload); err != nil {
			return lifecycle.CommandResult{}, err
		}
		payloads <- payload
		return lifecycle.CommandResult{Output: []byte("ok")}, nil
	}))
	created, err := app.CreateTaskWithExtraInfoAndTemplateFields("发布", "", task.DefaultColor, nil, map[string]any{"environment": "production"})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	startTaskAndWait(t, app, created.ID)

	select {
	case payload := <-payloads:
		if got, want := payload["templateFields"], map[string]any{"environment": "production", "dryRun": true}; !reflect.DeepEqual(got, want) {
			t.Fatalf("生命周期命令输入模板字段 = %#v，期望 %#v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("生命周期命令未收到模板字段输入")
	}
}

func TestAppFreezesTaskTemplateBranchForSpecifiedRepositoryClone(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.TaskTemplates = []task.TaskTemplate{{
		ID: "release", Name: "发布任务", Fields: []task.TaskTemplateField{{
			Key: "branch", DisplayName: "模板分支", InputType: task.TaskTemplateFieldInputString, DefaultValue: "main",
		}},
	}}
	current.ActiveTaskTemplateID = "release"
	repository := filepath.Join(t.TempDir(), "template")
	runGitTestCommand(t, "init", repository)
	runGitTestCommand(t, "-C", repository, "config", "user.email", "taskai@example.test")
	runGitTestCommand(t, "-C", repository, "config", "user.name", "TaskAI Test")
	if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("template\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runGitTestCommand(t, "-C", repository, "add", "README.md")
	runGitTestCommand(t, "-C", repository, "commit", "-m", "initial")
	runGitTestCommand(t, "-C", repository, "branch", "-M", "remote-default")
	runGitTestCommand(t, "-C", repository, "checkout", "-b", "release/1.2")
	remoteRepository := filepath.Join(t.TempDir(), "template.git")
	runGitTestCommand(t, "clone", "--bare", repository, remoteRepository)
	runGitTestCommand(t, "-C", remoteRepository, "symbolic-ref", "HEAD", "refs/heads/remote-default")
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: "clone-template", Name: "初始化模板", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
		Commands: []settings.LifecycleCommandReference{
			{CommandID: settings.LifecycleCommandUpdateDefaultBranchID, Arguments: []string{}},
			{CommandID: settings.LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
			{CommandID: settings.LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + remoteRepository}},
		},
	})
	current.LifecyclePresets[0].Chains[task.LifecycleHookBeforeStart] = "clone-template"
	saveSettingsWithLifecycleConfiguration(t, app, current)

	created, err := app.CreateTaskWithExtraInfoAndTemplateFields("发布", "", task.DefaultColor, nil, map[string]any{"branch": "release/1.2"})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	if started.Status != task.StatusRunning {
		t.Fatalf("任务状态 = %q，期望执行中", started.Status)
	}
	branch := runGitTestCommand(t, "-C", started.WorkspacePath, "branch", "--show-current")
	if branch != "release/1.2" {
		t.Fatalf("指定仓库克隆分支 = %q，期望 release/1.2", branch)
	}
}

func TestLifecycleTemplateFieldsFreezeVisibleValues(t *testing.T) {
	template := &task.TaskTemplate{
		ID: "release", Name: "发布任务", Fields: []task.TaskTemplateField{
			{Key: "branch", DisplayName: "模板分支", InputType: task.TaskTemplateFieldInputString},
			{Key: "deploy", DisplayName: "立即部署", InputType: task.TaskTemplateFieldInputBool, DefaultValue: false},
		},
	}
	fields, err := lifecycleTemplateFields(template, map[string]any{"branch": "android2.45-0727", "retired": "忽略"})
	if err != nil {
		t.Fatalf("lifecycleTemplateFields() error = %v", err)
	}
	if want := map[string]any{"branch": "android2.45-0727", "deploy": false}; !reflect.DeepEqual(fields, want) {
		t.Fatalf("冻结的模板字段 = %#v，期望 %#v", fields, want)
	}
}

func TestAppManifestFilePreservesLifecycleFailureSemantics(t *testing.T) {
	t.Run("开始前失败阻止启动，重试后写入模板分支", func(t *testing.T) {
		app := newManifestLifecycleApp(t, task.LifecycleHookBeforeStart, true)
		created := createManifestLifecycleTask(t, app, "开始前清单")
		workspacePath := prepareManifestTargetAsDirectory(t, app, created.ID)

		if _, err := app.StartTask(created.ID); err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		failed := waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusPending && lifecycleHookFailed(current, task.LifecycleHookBeforeStart)
		})
		if failed.Status != task.StatusPending {
			t.Fatalf("开始前清单失败后的任务状态 = %q，期望未执行", failed.Status)
		}
		if err := os.Remove(filepath.Join(workspacePath, "manifest.yaml")); err != nil {
			t.Fatalf("移除阻塞目标: %v", err)
		}
		if _, err := app.RetryTaskLifecycleCommandChain(created.ID); err != nil {
			t.Fatalf("RetryTaskLifecycleCommandChain() error = %v", err)
		}
		started := waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && current.LifecycleExecution == nil
		})
		manifest := decodeAppManifestFile(t, filepath.Join(started.WorkspacePath, "manifest.yaml"))
		if len(manifest.Repositories) != 1 || manifest.Repositories[0].Branch != "android2.45-0727" {
			t.Fatalf("开始前重试后的清单分支 = %#v", manifest.Repositories)
		}
	})

	t.Run("开始后失败保持执行中并可重试", func(t *testing.T) {
		app := newManifestLifecycleApp(t, task.LifecycleHookPostStart, false)
		created := createManifestLifecycleTask(t, app, "开始后清单")
		workspacePath := prepareManifestTargetAsDirectory(t, app, created.ID)

		if _, err := app.StartTask(created.ID); err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		failed := waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && lifecycleHookFailed(current, task.LifecycleHookPostStart)
		})
		if failed.Status != task.StatusRunning {
			t.Fatalf("开始后清单失败后的任务状态 = %q，期望执行中", failed.Status)
		}
		if err := os.Remove(filepath.Join(workspacePath, "manifest.yaml")); err != nil {
			t.Fatalf("移除阻塞目标: %v", err)
		}
		if _, err := app.RetryTaskLifecycleCommandChain(created.ID); err != nil {
			t.Fatalf("RetryTaskLifecycleCommandChain() error = %v", err)
		}
		waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && current.LifecycleExecution == nil
		})
		if _, err := os.Stat(filepath.Join(workspacePath, "manifest.yaml")); err != nil {
			t.Fatalf("开始后重试未生成清单文件: %v", err)
		}
	})

	t.Run("更新失败保留已提交内容并可重试", func(t *testing.T) {
		app := newManifestLifecycleApp(t, task.LifecycleHookUpdateTask, false)
		created := createManifestLifecycleTask(t, app, "更新前清单")
		started := startTaskAndWait(t, app, created.ID)
		if err := os.Mkdir(filepath.Join(started.WorkspacePath, "manifest.yaml"), 0o700); err != nil {
			t.Fatalf("创建阻塞目标: %v", err)
		}

		if _, err := app.UpdateTask(created.ID, "更新后清单", "已保存", task.DefaultColor); err != nil {
			t.Fatalf("UpdateTask() error = %v", err)
		}
		failed := waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && current.Title == "更新后清单" && lifecycleHookFailed(current, task.LifecycleHookUpdateTask)
		})
		if failed.Title != "更新后清单" {
			t.Fatalf("更新清单失败后标题 = %q，期望保留已提交内容", failed.Title)
		}
		if err := os.Remove(filepath.Join(started.WorkspacePath, "manifest.yaml")); err != nil {
			t.Fatalf("移除阻塞目标: %v", err)
		}
		if _, err := app.RetryTaskLifecycleCommandChain(created.ID); err != nil {
			t.Fatalf("RetryTaskLifecycleCommandChain() error = %v", err)
		}
		waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && current.LifecycleExecution == nil
		})
		manifest := decodeAppManifestFile(t, filepath.Join(started.WorkspacePath, "manifest.yaml"))
		if manifest.Iteration != "更新后清单" || manifest.Description != "已保存" {
			t.Fatalf("更新重试后的清单 = %#v", manifest)
		}
	})
}

func newManifestLifecycleApp(t *testing.T, hook task.LifecycleHook, includeCreateWorkspace bool) *App {
	t.Helper()
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.TaskTemplates = []task.TaskTemplate{{
		ID: "release", Name: "发布任务", Fields: []task.TaskTemplateField{{
			Key: "branch", DisplayName: "分支", InputType: task.TaskTemplateFieldInputString,
		}},
	}}
	current.ActiveTaskTemplateID = "release"
	chainID := "manifest-" + string(hook)
	commands := []settings.LifecycleCommandReference{{CommandID: settings.LifecycleCommandUpdateDefaultBranchID}}
	if includeCreateWorkspace {
		commands = append(commands, settings.LifecycleCommandReference{CommandID: settings.LifecycleCommandCreateWorkspaceID})
	}
	commands = append(commands, settings.LifecycleCommandReference{CommandID: settings.LifecycleCommandManifestFileID})
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: chainID, Name: "生成清单文件", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHook(hook)}, Commands: commands,
	})
	current.LifecyclePresets[0].Chains[settings.LifecycleHook(hook)] = chainID
	saveSettingsWithLifecycleConfiguration(t, app, current)
	return app
}

func createManifestLifecycleTask(t *testing.T, app *App, title string) task.Task {
	t.Helper()
	gitTemplate := task.BuiltInGitTemplate()
	information, err := app.SaveExtraInfo(task.ExtraInfo{
		TemplateID: gitTemplate.ID,
		Catalogue:  gitTemplate.Catalogue,
		Fields: []task.ExtraInfoField{
			{Key: "name", Value: "istudy-v2"},
			{Key: "repository", Value: "git@gitlab.jiandan100.cn:webdev/istudy-v2.git"},
		},
	})
	if err != nil {
		t.Fatalf("SaveExtraInfo() error = %v", err)
	}
	created, err := app.CreateTaskWithExtraInfoAndTemplateFields(title, "任务描述", task.DefaultColor, []task.TaskExtraInfo{{
		InformationID: information.ID,
		Parameters:    []task.ExtraInfoParameter{{Key: "branch", Value: ""}},
	}}, map[string]any{"branch": "android2.45-0727"})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	return created
}

func prepareManifestTargetAsDirectory(t *testing.T, app *App, taskID string) string {
	t.Helper()
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	_, workspacePath, err := workspace.TaskPath(current.WorkspaceRoot, taskID)
	if err != nil {
		t.Fatalf("TaskPath() error = %v", err)
	}
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("创建任务工作目录: %v", err)
	}
	if err := os.Mkdir(filepath.Join(workspacePath, "manifest.yaml"), 0o700); err != nil {
		t.Fatalf("创建阻塞目标: %v", err)
	}
	return workspacePath
}

func lifecycleHookFailed(current task.Task, hook task.LifecycleHook) bool {
	return current.LifecycleExecution != nil && current.LifecycleExecution.Hook == hook && current.LifecycleExecution.State == task.LifecycleExecutionFailed
}

type appManifestFile struct {
	Iteration    string `yaml:"iteration"`
	Description  string `yaml:"desc"`
	Repositories []struct {
		Branch string `yaml:"branch"`
	} `yaml:"repos"`
}

func decodeAppManifestFile(t *testing.T, path string) appManifestFile {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var manifest appManifestFile
	if err := yaml.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", contents, err)
	}
	return manifest
}

func TestAppSpecifiedRepositoryClonePreservesLifecycleFailureSemantics(t *testing.T) {
	t.Run("开始前失败阻止任务启动", func(t *testing.T) {
		app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
		current, err := app.GetSettings()
		if err != nil {
			t.Fatalf("GetSettings() error = %v", err)
		}
		current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
			ID: "clone-before-start", Name: "开始前初始化", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
			Commands: []settings.LifecycleCommandReference{{
				CommandID: settings.LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + filepath.Join(t.TempDir(), "missing.git")},
			}},
		})
		current.LifecyclePresets[0].Chains[task.LifecycleHookBeforeStart] = "clone-before-start"
		saveSettingsWithLifecycleConfiguration(t, app, current)

		created, err := app.CreateTask("初始化失败", "", task.DefaultColor)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		if _, err := app.StartTask(created.ID); err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		failed := waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusPending && current.LifecycleExecution != nil && current.LifecycleExecution.Hook == task.LifecycleHookBeforeStart && current.LifecycleExecution.State == task.LifecycleExecutionFailed
		})
		if failed.Status != task.StatusPending {
			t.Fatalf("开始前克隆失败后任务状态 = %q，期望未执行", failed.Status)
		}
	})

	t.Run("开始后失败可重试", func(t *testing.T) {
		app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
		current, err := app.GetSettings()
		if err != nil {
			t.Fatalf("GetSettings() error = %v", err)
		}
		remoteRepository := filepath.Join(t.TempDir(), "template.git")
		current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
			ID: "clone-post-start", Name: "开始后初始化", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookPostStart},
			Commands: []settings.LifecycleCommandReference{{
				CommandID: settings.LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + remoteRepository},
			}},
		})
		current.LifecyclePresets[0].Chains[task.LifecycleHookPostStart] = "clone-post-start"
		saveSettingsWithLifecycleConfiguration(t, app, current)

		created, err := app.CreateTask("可重试初始化", "", task.DefaultColor)
		if err != nil {
			t.Fatalf("CreateTask() error = %v", err)
		}
		if _, err := app.StartTask(created.ID); err != nil {
			t.Fatalf("StartTask() error = %v", err)
		}
		failed := waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && current.LifecycleExecution != nil && current.LifecycleExecution.Hook == task.LifecycleHookPostStart && current.LifecycleExecution.State == task.LifecycleExecutionFailed
		})
		if failed.Status != task.StatusRunning {
			t.Fatalf("开始后克隆失败后任务状态 = %q，期望执行中", failed.Status)
		}
		runGitTestCommand(t, "init", "--bare", remoteRepository)
		if _, err := app.RetryTaskLifecycleCommandChain(created.ID); err != nil {
			t.Fatalf("RetryTaskLifecycleCommandChain() error = %v", err)
		}
		retried := waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && current.LifecycleExecution == nil
		})
		if _, err := os.Stat(filepath.Join(retried.WorkspacePath, ".git")); err != nil {
			t.Fatalf("重试后指定仓库未直接克隆到工作目录: %v", err)
		}
	})
}

func TestAppInjectsTemplateFieldsOnlyIntoCustomLifecycleCommands(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.TaskTemplates = []task.TaskTemplate{{
		ID:   "release",
		Name: "发布任务",
		Fields: []task.TaskTemplateField{
			{Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, DefaultValue: "", InjectEnvironment: true},
			{Key: "deploy", DisplayName: "立即部署", InputType: task.TaskTemplateFieldInputBool, DefaultValue: false, InjectEnvironment: true},
			{Key: "privateNote", DisplayName: "内部备注", InputType: task.TaskTemplateFieldInputString, DefaultValue: "hidden"},
		},
	}}
	current.ActiveTaskTemplateID = "release"
	current.HTTPServiceEnabled = true
	current.StatusManagementHTTPPort = availableLoopbackPort(t)
	current.LifecycleCommands = append(current.LifecycleCommands, settings.LifecycleCommand{
		ID: "capture-template-environment", Kind: settings.LifecycleCommandKindCustom, Name: "读取模板变量", Command: "capture", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: "capture-template-environment", Name: "模板变量", Commands: []settings.LifecycleCommandReference{
			{CommandID: settings.LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
			{CommandID: "capture-template-environment", Arguments: []string{}},
		}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	current.LifecyclePresets[0].Chains[task.LifecycleHookBeforeStart] = "capture-template-environment"
	saveSettingsWithLifecycleConfiguration(t, app, current)
	if app.statusHTTP.APIURL() == "" {
		t.Fatal("独立 HTTP 服务未启动")
	}

	environments := make(chan []string, 1)
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(invocation lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		if invocation.Command != "capture" {
			t.Fatalf("内置命令不应调用 Shell 执行器: %#v", invocation)
		}
		environments <- append([]string(nil), invocation.Environment...)
		return lifecycle.CommandResult{Output: []byte("ok")}, nil
	}))
	created, err := app.CreateTaskWithExtraInfoAndTemplateFields("发布", "", task.DefaultColor, nil, map[string]any{"environment": "", "deploy": true})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	startTaskAndWait(t, app, created.ID)

	select {
	case environment := <-environments:
		want := []string{"TASKAI_TASK_ID=" + created.ID, "TASKAI_ENVIRONMENT=", "TASKAI_DEPLOY=true"}
		if !reflect.DeepEqual(environment, want) {
			t.Fatalf("自定义生命周期命令环境 = %#v，期望 %#v", environment, want)
		}
		if containsEnvironmentValue(environment, "TASKAI_PRIVATE_NOTE=hidden") {
			t.Fatalf("未标记字段不应注入环境变量: %#v", environment)
		}
	case <-time.After(time.Second):
		t.Fatal("自定义生命周期命令未收到环境变量")
	}
}

func TestLifecycleChainAppendsSavedReferenceArgumentsWhenChainArgumentsAreDisabled(t *testing.T) {
	_, commands, err := lifecycleChain(settings.Settings{
		LifecycleCommands: []settings.LifecycleCommand{{
			ID: "prepare", Kind: settings.LifecycleCommandKindCustom, Name: "准备", Command: "prepare", Arguments: []string{"--verbose"}, ChainArgumentMode: settings.LifecycleCommandChainArgumentModeDisabled,
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

func TestLifecyclePresetChainsResolveWithConfiguredParameters(t *testing.T) {
	current := settings.Default(t.TempDir())

	iterations, iterationCommands, err := lifecycleChain(current, settings.LifecycleChainIterationsAIID)
	if err != nil {
		t.Fatalf("lifecycleChain(iterations-ai) error = %v", err)
	}
	if iterations.Name != "iterations-ai" || !reflect.DeepEqual(iterations.ApplicableHooks, []settings.LifecycleHook{settings.LifecycleHookBeforeStart}) {
		t.Fatalf("iterations-ai 链 = %#v", iterations)
	}
	if got := lifecycleCommandIDsAndArguments(iterationCommands); !reflect.DeepEqual(got, []struct {
		ID        string
		Arguments []string
	}{
		{ID: settings.LifecycleCommandUpdateDefaultBranchID, Arguments: []string{}},
		{ID: settings.LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
		{ID: settings.LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + settings.IterationsAIRepository}},
		{ID: settings.LifecycleCommandManifestFileID, Arguments: []string{}},
		{ID: settings.LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
	}) {
		t.Fatalf("iterations-ai 命令 = %#v", got)
	}

	updateRepositories, updateCommands, err := lifecycleChain(current, settings.LifecycleChainUpdateRepositoriesID)
	if err != nil {
		t.Fatalf("lifecycleChain(更新仓库) error = %v", err)
	}
	if updateRepositories.Name != "更新框架仓库" || !reflect.DeepEqual(updateRepositories.ApplicableHooks, []settings.LifecycleHook{settings.LifecycleHookUpdateTask}) {
		t.Fatalf("更新框架仓库链 = %#v", updateRepositories)
	}
	if got := lifecycleCommandIDsAndArguments(updateCommands); !reflect.DeepEqual(got, []struct {
		ID        string
		Arguments []string
	}{
		{ID: settings.LifecycleCommandUpdateDefaultBranchID, Arguments: []string{}},
		{ID: settings.LifecycleCommandManifestFileID, Arguments: []string{}},
		{ID: settings.LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
	}) {
		t.Fatalf("更新仓库命令 = %#v", got)
	}
}

func lifecycleCommandIDsAndArguments(commands []settings.LifecycleCommand) []struct {
	ID        string
	Arguments []string
} {
	values := make([]struct {
		ID        string
		Arguments []string
	}, len(commands))
	for index, command := range commands {
		arguments := command.Arguments
		if arguments == nil {
			arguments = []string{}
		}
		values[index] = struct {
			ID        string
			Arguments []string
		}{ID: command.ID, Arguments: arguments}
	}
	return values
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
	current.LifecyclePresets[0].Chains[hook] = chainID
	saveSettingsWithLifecycleConfiguration(t, app, current)
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		return lifecycle.CommandResult{StandardError: []byte("失败")}, errors.New("exit status 1")
	}))
}

func TestAppRunsUpdateTaskHookOnlyForRunningTask(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	current.LifecyclePresets[0].Chains[task.LifecycleHookUpdateTask] = "update-chain"
	saveSettingsWithLifecycleConfiguration(t, app, current)

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

	startTaskAndWait(t, app, created.ID)
	if _, err := app.UpdateTask(created.ID, "执行中任务", "", task.DefaultColor); err != nil {
		t.Fatalf("更新执行中任务: %v", err)
	}
	waitForTask(t, app, created.ID, func(current task.Task) bool { return current.LifecycleExecution == nil })
	if calls != 1 {
		t.Fatalf("执行中任务更新命令次数 = %d，期望 1", calls)
	}

	finishTaskAndWait(t, app, created.ID)
	if _, err := app.UpdateTask(created.ID, "已完成任务", "", task.DefaultColor); err != nil {
		t.Fatalf("更新已完成任务: %v", err)
	}
	if calls != 1 {
		t.Fatalf("已完成任务更新命令次数 = %d，期望 1", calls)
	}
}

func TestAppRetriesFailedLifecycleChainFromFirstCommand(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	current.LifecyclePresets[0].Chains[task.LifecycleHookBeforeStart] = "retry-before-start"
	saveSettingsWithLifecycleConfiguration(t, app, current)

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
	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	failed := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.LifecycleExecution != nil && current.LifecycleExecution.State == task.LifecycleExecutionFailed
	})
	if failed.LifecycleExecution == nil || failed.LifecycleExecution.State != task.LifecycleExecutionFailed {
		t.Fatalf("失败记录 = %#v", failed.LifecycleExecution)
	}
	if _, err := app.StartTask(created.ID); err == nil {
		t.Fatal("失败中的任务再次开始执行 error = nil，期望被锁定")
	}

	fail = false
	if _, err := app.RetryTaskLifecycleCommandChain(created.ID); err != nil {
		t.Fatalf("RetryTaskLifecycleCommandChain() error = %v", err)
	}
	retried := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusRunning && current.LifecycleExecution == nil
	})
	if retried.Status != task.StatusRunning || retried.LifecycleExecution != nil {
		t.Fatalf("重试后的任务 = %#v", retried)
	}
	if calls != 2 {
		t.Fatalf("命令执行次数 = %d，期望从第一个命令重试后为 2", calls)
	}
}

func TestAppSchedulesLifecycleCommandChainsInBackground(t *testing.T) {
	t.Run("开始任务", func(t *testing.T) {
		app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
		chainID, entered, release := configureBlockingLifecycleHook(t, app, task.LifecycleHookBeforeStart)
		created := createTaskWithLifecycleChain(t, app, task.LifecycleHookBeforeStart, chainID)

		returned := invokeWhileLifecycleCommandBlocks(t, func() (task.Task, error) {
			return app.StartTask(created.ID)
		}, entered, release)
		if returned.Status != task.StatusPending || returned.LifecycleExecution == nil || returned.LifecycleExecution.State != task.LifecycleExecutionRunning || returned.LifecycleExecution.RunID == "" || returned.LifecycleExecution.Revision < 1 {
			t.Fatalf("开始任务应立即返回运行记录: %#v", returned)
		}
		waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && current.LifecycleExecution == nil
		})
	})

	t.Run("更新任务", func(t *testing.T) {
		app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
		chainID, entered, release := configureBlockingLifecycleHook(t, app, task.LifecycleHookUpdateTask)
		created := createTaskWithLifecycleChain(t, app, task.LifecycleHookUpdateTask, chainID)
		if _, err := app.tasks.StartTask(created.ID); err != nil {
			t.Fatalf("直接开始任务: %v", err)
		}

		returned := invokeWhileLifecycleCommandBlocks(t, func() (task.Task, error) {
			return app.UpdateTask(created.ID, "已保存的新标题", "", task.DefaultColor)
		}, entered, release)
		if returned.Title != "已保存的新标题" || returned.LifecycleExecution == nil || returned.LifecycleExecution.Hook != task.LifecycleHookUpdateTask || returned.LifecycleExecution.State != task.LifecycleExecutionRunning {
			t.Fatalf("更新任务应在命令结束前返回已保存运行快照: %#v", returned)
		}
		waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Title == "已保存的新标题" && current.LifecycleExecution == nil
		})
	})

	t.Run("结束任务", func(t *testing.T) {
		app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
		chainID, entered, release := configureBlockingLifecycleHook(t, app, task.LifecycleHookBeforeEnd)
		created := createTaskWithLifecycleChain(t, app, task.LifecycleHookBeforeEnd, chainID)
		if _, err := app.tasks.StartTask(created.ID); err != nil {
			t.Fatalf("直接开始任务: %v", err)
		}

		returned := invokeWhileLifecycleCommandBlocks(t, func() (task.Task, error) {
			return app.FinishTask(created.ID)
		}, entered, release)
		if returned.Status != task.StatusRunning || returned.LifecycleExecution == nil || returned.LifecycleExecution.Hook != task.LifecycleHookBeforeEnd {
			t.Fatalf("结束任务应在命令结束前返回运行快照: %#v", returned)
		}
		waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusCompleted && current.LifecycleExecution == nil
		})
	})

	t.Run("重试失败链", func(t *testing.T) {
		app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
		chainID, entered, release := configureRetryLifecycleHook(t, app, task.LifecycleHookBeforeStart)
		created := createTaskWithLifecycleChain(t, app, task.LifecycleHookBeforeStart, chainID)
		if _, err := app.StartTask(created.ID); err != nil {
			t.Fatalf("首次开始任务: %v", err)
		}
		failed := waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.LifecycleExecution != nil && current.LifecycleExecution.State == task.LifecycleExecutionFailed
		})

		returned := invokeWhileLifecycleCommandBlocks(t, func() (task.Task, error) {
			return app.RetryTaskLifecycleCommandChain(created.ID)
		}, entered, release)
		if returned.LifecycleExecution == nil || returned.LifecycleExecution.State != task.LifecycleExecutionRunning || returned.LifecycleExecution.RunID == failed.LifecycleExecution.RunID || returned.LifecycleExecution.CurrentIndex != 1 {
			t.Fatalf("重试应创建新的首步运行记录: %#v", returned.LifecycleExecution)
		}
		waitForTask(t, app, created.ID, func(current task.Task) bool {
			return current.Status == task.StatusRunning && current.LifecycleExecution == nil
		})
	})
}

func TestAppRecordsDeletedLifecycleChainAsFailed(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTask("缺失链任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	data, err := app.repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	chains := make([]settings.LifecycleCommandChain, 0, len(data.Settings.LifecycleChains)-1)
	for _, chain := range data.Settings.LifecycleChains {
		if chain.ID != settings.LifecycleChainCreateWorkspaceID {
			chains = append(chains, chain)
		}
	}
	data.Settings.LifecycleChains = chains
	delete(data.Settings.LifecyclePresets[0].Chains, settings.LifecycleHookBeforeStart)
	if err := app.repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	failed, err := app.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	execution := failed.LifecycleExecution
	if failed.Status != task.StatusPending || execution == nil || execution.State != task.LifecycleExecutionFailed || execution.ChainID != settings.LifecycleChainCreateWorkspaceID || execution.Error == "" || execution.RunID == "" || execution.Revision < 2 {
		t.Fatalf("删除链后的失败记录 = %#v", failed)
	}
}

func configureBlockingLifecycleHook(t *testing.T, app *App, hook task.LifecycleHook) (string, <-chan struct{}, chan<- error) {
	t.Helper()
	chainID := "blocking-" + string(hook)
	commandID := "blocking-command-" + string(hook)
	configureLifecycleTestChain(t, app, hook, chainID, commandID)
	entered := make(chan struct{}, 1)
	release := make(chan error, 1)
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		entered <- struct{}{}
		if err := <-release; err != nil {
			return lifecycle.CommandResult{StandardError: []byte("失败")}, err
		}
		return lifecycle.CommandResult{Output: []byte("完成")}, nil
	}))
	return chainID, entered, release
}

func configureRetryLifecycleHook(t *testing.T, app *App, hook task.LifecycleHook) (string, <-chan struct{}, chan<- error) {
	t.Helper()
	chainID := "retry-blocking-" + string(hook)
	commandID := "retry-blocking-command-" + string(hook)
	configureLifecycleTestChain(t, app, hook, chainID, commandID)
	entered := make(chan struct{}, 1)
	release := make(chan error, 1)
	attempts := 0
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		attempts++
		if attempts == 1 {
			return lifecycle.CommandResult{StandardError: []byte("首次失败")}, errors.New("exit status 1")
		}
		entered <- struct{}{}
		if err := <-release; err != nil {
			return lifecycle.CommandResult{StandardError: []byte("失败")}, err
		}
		return lifecycle.CommandResult{Output: []byte("完成")}, nil
	}))
	return chainID, entered, release
}

func saveSettingsWithLifecycleConfiguration(t *testing.T, app *App, next settings.Settings) settings.Settings {
	t.Helper()
	desired, err := settings.NormalizeLifecycle(next)
	if err != nil {
		t.Fatalf("NormalizeLifecycle() error = %v", err)
	}
	next.LifecycleCommands = nil
	next.LifecycleChains = nil
	next.LifecyclePresets = nil
	next.DefaultLifecyclePresetID = ""
	if _, err := app.SaveSettings(next); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	commands, err := app.ListLifecycleCommands()
	if err != nil {
		t.Fatalf("ListLifecycleCommands() error = %v", err)
	}
	commandsByID := make(map[string]settings.LifecycleCommand, len(commands))
	for _, command := range commands {
		commandsByID[command.ID] = command
	}
	for _, command := range desired.LifecycleCommands {
		if command.Kind != settings.LifecycleCommandKindCustom || reflect.DeepEqual(commandsByID[command.ID], command) {
			continue
		}
		if _, err := app.SaveLifecycleCommand(command); err != nil {
			t.Fatalf("SaveLifecycleCommand() error = %v", err)
		}
	}

	chains, err := app.ListLifecycleCommandChains()
	if err != nil {
		t.Fatalf("ListLifecycleCommandChains() error = %v", err)
	}
	chainsByID := make(map[string]settings.LifecycleCommandChain, len(chains))
	for _, chain := range chains {
		chainsByID[chain.ID] = chain
	}
	for _, chain := range desired.LifecycleChains {
		if reflect.DeepEqual(chainsByID[chain.ID], chain) {
			continue
		}
		if _, err := app.SaveLifecycleCommandChain(chain); err != nil {
			t.Fatalf("SaveLifecycleCommandChain() error = %v", err)
		}
	}

	for _, preset := range desired.LifecyclePresets {
		if _, err := app.SaveLifecyclePreset(preset); err != nil {
			t.Fatalf("SaveLifecyclePreset() error = %v", err)
		}
	}
	if _, err := app.SaveDefaultLifecyclePreset(desired.DefaultLifecyclePresetID); err != nil {
		t.Fatalf("SaveDefaultLifecyclePreset() error = %v", err)
	}
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	return current
}

func configureLifecycleTestChain(t *testing.T, app *App, hook task.LifecycleHook, chainID, commandID string) {
	t.Helper()
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.LifecycleCommands = append(current.LifecycleCommands, settings.LifecycleCommand{
		ID: commandID, Kind: settings.LifecycleCommandKindCustom, Name: "阻塞命令", Command: "blocking", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHook(hook)},
	})
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: chainID, Name: "阻塞链", Commands: []settings.LifecycleCommandReference{{CommandID: commandID, Arguments: []string{}}}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHook(hook)},
	})
	saveSettingsWithLifecycleConfiguration(t, app, current)
}

func createTaskWithLifecycleChain(t *testing.T, app *App, hook task.LifecycleHook, chainID string) task.Task {
	t.Helper()
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("异步生命周期任务", "", task.DefaultColor, nil, map[task.LifecycleHook]string{hook: chainID})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndLifecycleChains() error = %v", err)
	}
	return created
}

func invokeWhileLifecycleCommandBlocks(t *testing.T, invoke func() (task.Task, error), entered <-chan struct{}, release chan<- error) task.Task {
	t.Helper()
	type result struct {
		task task.Task
		err  error
	}
	results := make(chan result, 1)
	go func() {
		current, err := invoke()
		results <- result{task: current, err: err}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("生命周期命令未开始执行")
	}
	select {
	case returned := <-results:
		if returned.err != nil {
			t.Fatalf("绑定调用 error = %v", returned.err)
		}
		release <- nil
		return returned.task
	case <-time.After(100 * time.Millisecond):
		release <- nil
		<-results
		t.Fatal("绑定调用等待了生命周期命令完成")
		return task.Task{}
	}
}

func waitForTask(t *testing.T, app *App, taskID string, matches func(task.Task) bool) task.Task {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, err := app.tasks.GetTask(taskID)
		if err == nil && matches(current) {
			return current
		}
		time.Sleep(5 * time.Millisecond)
	}
	current, err := app.tasks.GetTask(taskID)
	t.Fatalf("等待任务最终状态超时: task=%#v, err=%v", current, err)
	return task.Task{}
}

func waitForRealtimeTerminalStatus(t *testing.T, app *App, taskID, terminalID string, expected realtime.Status) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if got := app.realtime.TerminalStatus(taskID, terminalID); got == expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := app.realtime.TerminalStatus(taskID, terminalID); got != expected {
		t.Fatalf("等待终端实时状态超时: got=%q, want=%q", got, expected)
	}
}

func startTaskAndWait(t *testing.T, app *App, taskID string) task.Task {
	t.Helper()
	if _, err := app.StartTask(taskID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	return waitForTask(t, app, taskID, func(current task.Task) bool {
		return current.Status == task.StatusRunning && current.LifecycleExecution == nil
	})
}

func finishTaskAndWait(t *testing.T, app *App, taskID string) task.Task {
	t.Helper()
	if _, err := app.FinishTask(taskID); err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	return waitForTask(t, app, taskID, func(current task.Task) bool {
		return current.Status == task.StatusCompleted && current.LifecycleExecution == nil
	})
}

func runGitTestCommand(t *testing.T, arguments ...string) string {
	t.Helper()
	output, err := exec.Command("git", arguments...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v error = %v: %s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}

func runGitInDirectory(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v error = %v: %s", arguments, err, output)
	}
}

func gitRepositoryPaths(repositories []repositorygit.Repository) []string {
	paths := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		paths = append(paths, repository.Path)
	}
	return paths
}

func TestAppRecordsFailingLifecycleCommandDetails(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.LifecycleCommands = append(current.LifecycleCommands,
		settings.LifecycleCommand{ID: "first", Kind: settings.LifecycleCommandKindCustom, Name: "创建工作区", Command: "first"},
		settings.LifecycleCommand{ID: "second", Kind: settings.LifecycleCommandKindCustom, Name: "安装依赖", Command: "second"},
	)
	current.LifecycleChains = append(current.LifecycleChains, settings.LifecycleCommandChain{
		ID: "two-steps", Name: "两步准备", CommandIDs: []string{"first", "second"},
	})
	current.LifecyclePresets[0].Chains[task.LifecycleHookBeforeStart] = "two-steps"
	saveSettingsWithLifecycleConfiguration(t, app, current)
	app.lifecycleCommandRunner = lifecycle.NewCommandChainRunner(lifecycle.CommandExecutorFunc(func(invocation lifecycle.CommandInvocation) (lifecycle.CommandResult, error) {
		if invocation.Command == "second" {
			return lifecycle.CommandResult{StandardError: []byte("失败")}, errors.New("exit status 1")
		}
		return lifecycle.CommandResult{Output: []byte("ok")}, nil
	}))
	created, err := app.CreateTask("两步任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	failed := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.LifecycleExecution != nil && current.LifecycleExecution.State == task.LifecycleExecutionFailed
	})
	execution := failed.LifecycleExecution
	if execution == nil || execution.State != task.LifecycleExecutionFailed || execution.CurrentCommandID != "second" || execution.CurrentCommandName != "安装依赖" || execution.CurrentIndex != 2 || execution.CommandCount != 2 {
		t.Fatalf("失败记录未定位到第二条命令: %#v", execution)
	}
}

func TestAppUpdatesLifecycleChainSelectionsOnlyForPendingTasks(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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

	started := startTaskAndWait(t, app, updated.ID)
	if _, err := app.UpdateTaskWithExtraInfoAndLifecycleChains(started.ID, "执行中任务", "", task.DefaultColor, nil, selected); err == nil {
		t.Fatal("执行中任务更新命令链 error = nil")
	}
	if _, err := app.FinishTask(started.ID); err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	completed := waitForTask(t, app, started.ID, func(current task.Task) bool {
		return current.Status == task.StatusCompleted
	})
	if _, err := app.UpdateTaskWithExtraInfoAndLifecycleChains(completed.ID, "已完成任务", "", task.DefaultColor, nil, selected); err == nil {
		t.Fatal("已完成任务更新命令链 error = nil")
	}
}

func TestAppCreatesAndUpdatesTaskTemplateFieldsWithExtraInfoAndLifecycleChains(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.TaskTemplates = []task.TaskTemplate{{
		ID:   "release",
		Name: "发布任务",
		Fields: []task.TaskTemplateField{
			{Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, Required: true, DefaultValue: "development"},
			{Key: "deploy", DisplayName: "立即部署", InputType: task.TaskTemplateFieldInputBool, DefaultValue: false},
		},
	}}
	current.ActiveTaskTemplateID = "release"
	saveSettingsWithLifecycleConfiguration(t, app, current)

	created, err := app.CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(
		"发布", "", task.DefaultColor, nil,
		map[string]any{"environment": "staging"},
		map[task.LifecycleHook]string{task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID},
	)
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains() error = %v", err)
	}
	if got, want := created.TemplateFields, map[string]any{"environment": "staging", "deploy": false}; !reflect.DeepEqual(got, want) {
		t.Fatalf("创建任务模板字段 = %#v，期望 %#v", got, want)
	}
	if got := created.LifecycleChains[task.LifecycleHookBeforeStart]; got != settings.LifecycleChainCreateWorkspaceID {
		t.Fatalf("创建任务生命周期命令链 = %q", got)
	}

	updated, err := app.UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(
		created.ID, "发布", "准备完成", task.DefaultColor, nil,
		map[string]any{"environment": "production", "deploy": true},
		map[task.LifecycleHook]string{task.LifecycleHookPostEnd: settings.LifecycleChainDeleteWorkspaceID},
	)
	if err != nil {
		t.Fatalf("UpdateTaskWithExtraInfoTemplateFieldsAndLifecycleChains() error = %v", err)
	}
	if got, want := updated.TemplateFields, map[string]any{"environment": "production", "deploy": true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("更新任务模板字段 = %#v，期望 %#v", got, want)
	}
	if got := updated.LifecycleChains[task.LifecycleHookPostEnd]; got != settings.LifecycleChainDeleteWorkspaceID {
		t.Fatalf("更新任务生命周期命令链 = %q", got)
	}
}

func TestAppHTTPTaskDetailIncludesLifecycleConfiguration(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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

func TestAppGetsCurrentLifecycleCommandInput(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	backend := &activeTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	t.Cleanup(func() { _ = app.terminals.CloseAll() })

	configured, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	configured.WorkspaceRoot = t.TempDir()
	configured.HTTPServiceEnabled = true
	configured.StatusManagementHTTPPort = availableLoopbackPort(t)
	configured.TaskTemplates = []task.TaskTemplate{{
		ID: "release", Name: "发布任务",
		Fields: []task.TaskTemplateField{{Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, DefaultValue: "development"}},
	}}
	configured.ActiveTaskTemplateID = "release"
	configured, err = app.SaveSettings(configured)
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	template := gitTemplateForTest(t, app)
	info, err := app.SaveExtraInfo(task.ExtraInfo{
		TemplateID: template.ID,
		Catalogue:  template.Catalogue,
		Fields: []task.ExtraInfoField{
			{Key: "name", Value: "API 服务"},
			{Key: "repository", Value: "git@example.com:team/api.git"},
		},
	})
	if err != nil {
		t.Fatalf("SaveExtraInfo() error = %v", err)
	}
	created, err := app.CreateTaskWithExtraInfoAndTemplateFields("发布 API", "准备部署", task.DefaultColor, []task.TaskExtraInfo{{
		InformationID: info.ID,
		Parameters:    []task.ExtraInfoParameter{{Key: "branch", DisplayName: "仓库分支", Value: "main"}},
	}}, map[string]any{"environment": "production"})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	terminalInfo, err := app.CreateTerminal(started.ID, 100, 32)
	if err != nil {
		t.Fatalf("CreateTerminal() error = %v", err)
	}
	if !app.realtime.SetTerminalStatus(started.ID, terminalInfo.ID, realtime.StatusWorking) {
		t.Fatal("设置终端实时状态失败")
	}

	got, err := app.GetLifecycleCommandInput(started.ID)
	if err != nil {
		t.Fatalf("GetLifecycleCommandInput() error = %v", err)
	}
	expectedResource, found, err := app.httpTask(started.ID)
	if err != nil || !found {
		t.Fatalf("httpTask() = (%#v, %t, %v)", expectedResource, found, err)
	}
	want, err := lifecycle.BuildCommandInput(expectedResource, app.statusHTTP.APIURL())
	if err != nil {
		t.Fatalf("BuildCommandInput() error = %v", err)
	}
	if got != string(want) {
		t.Fatalf("当前命令链输入 = %s，期望 %s", got, want)
	}

	if err := app.statusHTTP.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	withoutHTTP, err := app.GetLifecycleCommandInput(started.ID)
	if err != nil {
		t.Fatalf("关闭 HTTP 后 GetLifecycleCommandInput() error = %v", err)
	}
	if strings.Contains(withoutHTTP, `"baseURL"`) {
		t.Fatalf("未监听 HTTP 服务时输入不应包含 baseURL: %s", withoutHTTP)
	}
	if _, err := app.GetLifecycleCommandInput("missing"); err == nil {
		t.Fatal("不存在任务的 GetLifecycleCommandInput() error = nil")
	}
}

func TestAppHTTPTaskResourcesExposeOnlyCurrentTemplateFields(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.TaskTemplates = []task.TaskTemplate{{
		ID:   "release",
		Name: "发布任务",
		Fields: []task.TaskTemplateField{
			{Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, Required: true, DefaultValue: "development"},
			{Key: "dryRun", DisplayName: "演练", InputType: task.TaskTemplateFieldInputBool, DefaultValue: false},
		},
	}}
	current.ActiveTaskTemplateID = "release"
	saveSettingsWithLifecycleConfiguration(t, app, current)
	created, err := app.CreateTaskWithExtraInfoAndTemplateFields("发布", "", task.DefaultColor, nil, map[string]any{"environment": "staging"})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	data, err := app.repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Tasks[0].TemplateFields["removedField"] = "preserved"
	if err := app.repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	detail, found, err := app.httpTask(created.ID)
	if err != nil || !found {
		t.Fatalf("httpTask() = (%#v, %t, %v)", detail, found, err)
	}
	want := map[string]any{"environment": "staging", "dryRun": false}
	if got := detail.TemplateFields; !reflect.DeepEqual(got, want) {
		t.Fatalf("HTTP 任务详情模板字段 = %#v，期望 %#v", got, want)
	}
	listed, err := app.httpTasks()
	if err != nil {
		t.Fatalf("httpTasks() error = %v", err)
	}
	if got := listed[0].TemplateFields; !reflect.DeepEqual(got, want) {
		t.Fatalf("HTTP 任务列表模板字段 = %#v，期望 %#v", got, want)
	}

	current.ActiveTaskTemplateID = ""
	saveSettingsWithLifecycleConfiguration(t, app, current)
	detail, found, err = app.httpTask(created.ID)
	if err != nil || !found {
		t.Fatalf("停用模板后的 httpTask() = (%#v, %t, %v)", detail, found, err)
	}
	if got := detail.TemplateFields; !reflect.DeepEqual(got, map[string]any{}) {
		t.Fatalf("未启用模板时 HTTP 模板字段 = %#v，期望空对象", got)
	}
}

func TestAppExposesLifecycleCommandChainAndPresetManagementBindings(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	if _, err := app.SaveLifecyclePreset(settings.LifecyclePreset{Name: "不适用预设", Chains: map[task.LifecycleHook]string{task.LifecycleHookBeforeStart: chain.ID}}); err == nil {
		t.Fatal("SaveLifecyclePreset() error = nil，期望拒绝不适用链")
	}
	preset, err := app.SaveLifecyclePreset(settings.LifecyclePreset{Name: "转换预设", Chains: map[task.LifecycleHook]string{task.LifecycleHookPostStart: chain.ID}})
	if err != nil {
		t.Fatalf("SaveLifecyclePreset() error = %v", err)
	}
	if current, err := app.SaveDefaultLifecyclePreset(preset.ID); err != nil || current.DefaultLifecyclePresetID != preset.ID {
		t.Fatalf("SaveDefaultLifecyclePreset() = (%#v, %v)", current, err)
	}
	copy, err := app.CopyLifecycleCommandChain(chain.ID)
	if err != nil {
		t.Fatalf("CopyLifecycleCommandChain() error = %v", err)
	}
	if err := app.DeleteLifecycleCommand(command.ID); err == nil {
		t.Fatal("删除被命令链引用的命令 error = nil")
	}
	if err := app.DeleteLifecycleCommandChain(chain.ID); err == nil {
		t.Fatal("删除被预设引用的命令链 error = nil")
	}
	if err := app.DeleteLifecyclePreset(preset.ID); err != nil {
		t.Fatalf("DeleteLifecyclePreset() error = %v", err)
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTask("执行中任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	startTaskAndWait(t, app, created.ID)

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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	app.realtime.RegisterTerminal("task-1", "terminal-normal")
	app.publishTerminalEvent(terminal.Event{TaskID: "task-1", TerminalID: "terminal-normal", Type: "exited", ExitReason: terminal.ExitReasonNormal})
	if got := app.realtime.TerminalPresence("task-1", "terminal-normal"); got != realtime.TerminalRemoved {
		t.Fatalf("正常退出终端状态记录 = %q，期望 %q", got, realtime.TerminalRemoved)
	}

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

func TestAppCloseTerminalClearsRealtimeErrorForExitedTerminal(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	app.terminals = terminal.NewManager(exitedTerminalBackend{exitResult: terminal.ExitResult{ExitCode: intPointer(1)}}, app.publishTerminalEvent)

	created, err := app.CreateTask("关闭异常终端", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	createdTerminal, err := app.CreateTerminal(started.ID, 100, 32)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	waitForRealtimeTerminalStatus(t, app, started.ID, createdTerminal.ID, realtime.StatusError)

	if err := app.CloseTerminal(started.ID, createdTerminal.ID); err != nil {
		t.Fatalf("关闭已退出终端: %v", err)
	}
	if got := app.realtime.TerminalPresence(started.ID, createdTerminal.ID); got != realtime.TerminalRemoved {
		t.Fatalf("已关闭异常终端状态记录 = %q，期望 %q", got, realtime.TerminalRemoved)
	}
	if got := app.realtime.TaskStatus(started.ID); got != realtime.StatusIdle {
		t.Fatalf("关闭唯一异常终端后的任务状态 = %q，期望 %q", got, realtime.StatusIdle)
	}
}

func TestAppCloseTerminalKeepsRemainingRealtimeStatus(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	app.terminals = terminal.NewManager(exitedTerminalBackend{exitResult: terminal.ExitResult{ExitCode: intPointer(1)}}, app.publishTerminalEvent)

	created, err := app.CreateTask("保留其他终端状态", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	createdTerminal, err := app.CreateTerminal(started.ID, 100, 32)
	if err != nil {
		t.Fatalf("创建异常终端: %v", err)
	}
	waitForRealtimeTerminalStatus(t, app, started.ID, createdTerminal.ID, realtime.StatusError)

	const remainingTerminalID = "remaining-terminal"
	app.realtime.RegisterTerminal(started.ID, remainingTerminalID)
	if !app.realtime.SetTerminalStatus(started.ID, remainingTerminalID, realtime.StatusWorking) {
		t.Fatal("设置剩余终端实时状态失败")
	}

	if err := app.CloseTerminal(started.ID, createdTerminal.ID); err != nil {
		t.Fatalf("关闭已退出异常终端: %v", err)
	}
	if got := app.realtime.TaskStatus(started.ID); got != realtime.StatusWorking {
		t.Fatalf("关闭异常终端后的任务状态 = %q，期望剩余终端状态 %q", got, realtime.StatusWorking)
	}
}

func TestAppCloseTerminalKeepsActiveCloseIdempotent(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	app.terminals = terminal.NewManager(activeTerminalBackend{}, app.publishTerminalEvent)

	created, err := app.CreateTask("关闭活动终端", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	createdTerminal, err := app.CreateTerminal(started.ID, 100, 32)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	if err := app.CloseTerminal(started.ID, createdTerminal.ID); err != nil {
		t.Fatalf("关闭活动终端: %v", err)
	}
	if got := app.realtime.TerminalPresence(started.ID, createdTerminal.ID); got != realtime.TerminalRemoved {
		t.Fatalf("活动终端关闭后的状态记录 = %q，期望 %q", got, realtime.TerminalRemoved)
	}
	if err := app.CloseTerminal(started.ID, createdTerminal.ID); err != nil {
		t.Fatalf("重复关闭活动终端: %v", err)
	}
}

func TestAppCloseTerminalRetainsRealtimeStatusWhenCloseFails(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	backend := &failingCloseTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)

	created, err := app.CreateTask("关闭失败终端", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	createdTerminal, err := app.CreateTerminal(started.ID, 100, 32)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	t.Cleanup(func() { backend.stop() })

	if err := app.CloseTerminal(started.ID, createdTerminal.ID); err == nil {
		t.Fatal("关闭失败终端 error = nil")
	}
	if got := app.realtime.TerminalPresence(started.ID, createdTerminal.ID); got != realtime.TerminalActive {
		t.Fatalf("关闭失败终端的状态记录 = %q，期望 %q", got, realtime.TerminalActive)
	}
}

func TestAppClosingExitedUnexpectedTerminalRemovesRealtimeStatus(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	exited := make(chan struct{}, 1)
	app.terminals = terminal.NewManager(exitedTerminalBackend{exitResult: terminal.ExitResult{ExitCode: intPointer(1)}}, func(event terminal.Event) {
		app.publishTerminalEvent(event)
		if event.Type == "exited" {
			exited <- struct{}{}
		}
	})
	created, err := app.terminals.Create("task-1", t.TempDir(), "/bin/sh", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("等待异常终端退出超时")
	}
	if got := app.realtime.TerminalStatus("task-1", created.ID); got != realtime.StatusError {
		t.Fatalf("异常退出终端状态 = %q，期望 %q", got, realtime.StatusError)
	}

	if err := app.CloseTerminal("task-1", created.ID); err != nil {
		t.Fatalf("关闭已退出异常终端: %v", err)
	}
	if got := app.realtime.TerminalPresence("task-1", created.ID); got != realtime.TerminalRemoved {
		t.Fatalf("关闭后终端状态记录 = %q，期望 %q", got, realtime.TerminalRemoved)
	}
}

func TestAppReportsTerminalVisualActivityOnlyInOutputChangeMode(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	app.realtime.RegisterTerminal("task-1", "terminal-1")

	if app.ReportTerminalVisualActivity("task-1", "terminal-1") {
		t.Fatal("标题方式下的画面活动被接受")
	}
	if got := app.realtime.TerminalStatus("task-1", "terminal-1"); got != realtime.StatusIdle {
		t.Fatalf("标题方式画面活动后的状态 = %q，期望 %q", got, realtime.StatusIdle)
	}

	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.StatusManagementMode = settings.StatusManagementModeOutputChange
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("保存输出状态管理设置: %v", err)
	}

	if !app.ReportTerminalVisualActivity("task-1", "terminal-1") {
		t.Fatal("输出方式下的画面活动未被接受")
	}
	if got := app.realtime.TerminalStatus("task-1", "terminal-1"); got != realtime.StatusWorking {
		t.Fatalf("输出方式画面活动后的状态 = %q，期望 %q", got, realtime.StatusWorking)
	}

	current.StatusManagementMode = settings.StatusManagementModeHTTP
	current.StatusManagementHTTPPort = availableLoopbackPort(t)
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("保存 HTTP 状态管理设置: %v", err)
	}
	if app.ReportTerminalVisualActivity("task-1", "terminal-1") {
		t.Fatal("HTTP 方式下的画面活动被接受")
	}
	if got := app.realtime.TerminalStatus("task-1", "terminal-1"); got != realtime.StatusIdle {
		t.Fatalf("HTTP 方式画面活动后的状态 = %q，期望 %q", got, realtime.StatusIdle)
	}
}

func TestAppRawTerminalOutputDoesNotReportActivity(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	app.realtime.RegisterTerminal("task-1", "terminal-1")

	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.StatusManagementMode = settings.StatusManagementModeOutputChange
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("保存输出状态管理设置: %v", err)
	}

	app.publishTerminalEvent(terminal.Event{TaskID: "task-1", TerminalID: "terminal-1", Type: "output", Data: "原始 PTY 输出"})
	if got := app.realtime.TerminalStatus("task-1", "terminal-1"); got != realtime.StatusIdle {
		t.Fatalf("原始输出后的状态 = %q，期望 %q", got, realtime.StatusIdle)
	}
}

func TestAppConfiguresHTTPStatusServiceAtomically(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	startTaskAndWait(t, app, running.ID)
	completed, err := app.CreateTask("已完成任务", "已经完成", "#f97316")
	if err != nil {
		t.Fatalf("创建已完成任务: %v", err)
	}
	startTaskAndWait(t, app, completed.ID)
	finishTaskAndWait(t, app, completed.ID)

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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	started := startTaskAndWait(t, app, created.ID)
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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

func TestAppBuildsTerminalEnvironmentWhenHTTPServiceIsListening(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	created, err := app.CreateTask("终端环境", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	titleChangeEnvironment := app.terminalStatusEnvironment(started.ID, "terminal-1")
	if !containsEnvironmentValue(titleChangeEnvironment, "TASKAI_TASK_ID="+started.ID) || !containsEnvironmentValue(titleChangeEnvironment, "TASKAI_TERMINAL_ID=terminal-1") {
		t.Fatalf("标题变化模式终端环境 = %#v，期望包含任务和终端 ID", titleChangeEnvironment)
	}
	if containsEnvironmentValue(titleChangeEnvironment, "TASKAI_STATUS_API=") {
		t.Fatalf("标题变化模式终端环境不应包含 HTTP 地址: %#v", titleChangeEnvironment)
	}

	port := availableLoopbackPort(t)
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            started.WorkspaceRoot,
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		StatusManagementMode:     settings.StatusManagementModeTitleChange,
		HTTPServiceEnabled:       true,
		StatusManagementHTTPPort: port,
	}); err != nil {
		t.Fatalf("保存独立 HTTP 服务设置: %v", err)
	}
	independentHTTPEnvironment := app.terminalStatusEnvironment(started.ID, "terminal-independent-http")
	assertStatusEnvironment(t, independentHTTPEnvironment, app.statusHTTP.APIURL(), started.ID, "terminal-independent-http")

	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            started.WorkspaceRoot,
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		StatusManagementMode:     settings.StatusManagementModeOutputChange,
		HTTPServiceEnabled:       true,
		StatusManagementHTTPPort: port,
	}); err != nil {
		t.Fatalf("保存输出方式独立 HTTP 服务设置: %v", err)
	}
	outputChangeEnvironment := app.terminalStatusEnvironment(started.ID, "terminal-output-change")
	assertStatusEnvironment(t, outputChangeEnvironment, app.statusHTTP.APIURL(), started.ID, "terminal-output-change")

	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            started.WorkspaceRoot,
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		StatusManagementMode:     settings.StatusManagementModeTitleChange,
		StatusManagementHTTPPort: port,
	}); err != nil {
		t.Fatalf("关闭独立 HTTP 服务: %v", err)
	}
	closedServiceEnvironment := app.terminalStatusEnvironment(started.ID, "terminal-service-closed")
	if containsEnvironmentValue(closedServiceEnvironment, "TASKAI_STATUS_API=") {
		t.Fatalf("服务关闭后的新终端环境不应包含 HTTP 地址: %#v", closedServiceEnvironment)
	}

	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            started.WorkspaceRoot,
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		StatusManagementMode:     settings.StatusManagementModeHTTP,
		StatusManagementHTTPPort: port,
	}); err != nil {
		t.Fatalf("保存 HTTP 状态设置: %v", err)
	}
	httpStatusEnvironment := app.terminalStatusEnvironment(started.ID, "terminal-http-status")
	assertStatusEnvironment(t, httpStatusEnvironment, app.statusHTTP.APIURL(), started.ID, "terminal-http-status")
}

func TestAppInjectsTaskIDIntoBackgroundCommandsAndScripts(t *testing.T) {
	item := settings.TaskMenuItem{
		ID: "background-command", Kind: settings.TaskMenuItemKindCommand, Name: "后台执行", Command: "main-command",
		BeforeScript: &settings.TaskScript{Script: "prepare"},
		AfterScript:  &settings.TaskScript{Script: "cleanup"},
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("读取菜单命令设置: %v", err)
	}
	current.HTTPServiceEnabled = true
	current.StatusManagementHTTPPort = availableLoopbackPort(t)
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("启用独立 HTTP 服务: %v", err)
	}
	if app.statusHTTP.APIURL() == "" {
		t.Fatal("独立 HTTP 服务未启动")
	}
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

func TestAppInjectsStatusAPIIntoEveryTerminalEntryWhenHTTPServiceIsListening(t *testing.T) {
	port := availableLoopbackPort(t)
	item := settings.TaskMenuItem{
		ID: "custom-status-command", Kind: settings.TaskMenuItemKindCommand, Name: "状态命令", Command: "status-command", ShowTerminal: true,
	}
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	t.Cleanup(func() { _ = app.statusHTTP.Close() })
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot:            t.TempDir(),
		TaskTreeWidth:            settings.DefaultTaskTreeWidth,
		TaskMenuItems:            []settings.TaskMenuItem{item},
		StatusManagementMode:     settings.StatusManagementModeTitleChange,
		HTTPServiceEnabled:       true,
		StatusManagementHTTPPort: port,
	}); err != nil {
		t.Fatalf("保存独立 HTTP 服务设置: %v", err)
	}
	created, err := app.CreateTask("状态环境", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)

	normal, err := app.CreateTerminal(started.ID, 100, 32)
	if err != nil {
		t.Fatalf("创建普通终端: %v", err)
	}
	directCommand, err := app.CreateCommandTerminal(started.ID, "direct-status-command", nil, 100, 32)
	if err != nil {
		t.Fatalf("创建显示终端命令: %v", err)
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
	assertStatusEnvironment(t, backend.request(directCommand.ID).Environment, app.statusHTTP.APIURL(), started.ID, directCommand.ID)
	assertStatusEnvironment(t, backend.request(command.Terminal.ID).Environment, app.statusHTTP.APIURL(), started.ID, command.Terminal.ID)
	if request := backend.request(directCommand.ID); request.Command != "direct-status-command" {
		t.Fatalf("直接显示终端命令 = %q，期望 direct-status-command", request.Command)
	}
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

func TestExecuteTaskMenuCommandSnapshotsMouseClipboardPolicy(t *testing.T) {
	item := settings.TaskMenuItem{
		ID:                          "claude-terminal",
		Kind:                        settings.TaskMenuItemKindCommand,
		Name:                        "Claude",
		Command:                     "claude",
		ShowTerminal:                true,
		DisableTaskAIMouseClipboard: true,
	}
	app, started := runningAppWithTaskMenuItem(t, item)
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)

	result, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32)
	if err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	if result.Terminal == nil {
		t.Fatal("显示终端的菜单命令未返回终端")
	}
	if !result.Terminal.DisableTaskAIMouseClipboard {
		t.Fatal("菜单命令未将鼠标剪贴板策略快照到终端")
	}
	if !backend.request(result.Terminal.ID).DisableTaskAIMouseClipboard {
		t.Fatal("菜单命令启动请求未传递鼠标剪贴板策略")
	}
}

func TestAppRegistersRealtimeTerminalBeforeStartingProcess(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	created, err := app.CreateTask("启动注册", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTask("打开目录", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)

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

func TestOpenTaskFolderUsesHomeForRunningTaskWithoutLifecycleChains(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("打开 Home 目录", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	if _, err := os.Stat(started.WorkspacePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("空链任务工作目录状态 = %v，期望不存在", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	var openedPath string
	app.directoryOpener = func(path string) error {
		openedPath = path
		return nil
	}
	if err := app.OpenTaskFolder(started.ID); err != nil {
		t.Fatalf("打开任务目录: %v", err)
	}
	if openedPath != home {
		t.Fatalf("打开目录 = %q，期望 Home 目录 %q", openedPath, home)
	}
}

func TestCreateTerminalUsesHomeForRunningTaskWithoutLifecycleChains(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("Home 终端", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	info, err := app.CreateTerminal(started.ID, 100, 32)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	if got := backend.request(info.ID).Directory; got != home {
		t.Fatalf("终端目录 = %q，期望 Home 目录 %q", got, home)
	}
}

func TestCreateCommandTerminalUsesHomeForRunningTaskWithoutLifecycleChains(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("Home 命令终端", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	info, err := app.CreateCommandTerminal(started.ID, "codex", []string{"--help"}, 100, 32)
	if err != nil {
		t.Fatalf("创建命令终端: %v", err)
	}
	request := backend.request(info.ID)
	if request.Directory != home || request.Command != "codex" {
		t.Fatalf("命令终端请求 = %#v，期望目录 %q 和命令 codex", request, home)
	}
}

func TestOpenTaskFolderRejectsUnavailableHomeForRunningTaskWithoutLifecycleChains(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	app.homeDirectory = func() (string, error) {
		return "", errors.New("Home 目录不可用")
	}
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("不可用 Home", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	app.directoryOpener = func(string) error {
		t.Fatal("Home 目录不可用时不应打开目录")
		return nil
	}

	err = app.OpenTaskFolder(started.ID)
	if err == nil || !strings.Contains(err.Error(), "获取用户 Home 目录失败") {
		t.Fatalf("打开目录错误 = %v，期望 Home 目录错误", err)
	}
}

func TestCreateTerminalRejectsUnavailableHomeForRunningTaskWithoutLifecycleChains(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	backend := &capturingTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	app.homeDirectory = func() (string, error) {
		return "", errors.New("Home 目录不可用")
	}
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("不可用 Home 终端", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)

	if _, err := app.CreateTerminal(started.ID, 100, 32); err == nil || !strings.Contains(err.Error(), "获取用户 Home 目录失败") {
		t.Fatalf("创建终端错误 = %v，期望 Home 目录错误", err)
	}
	if len(backend.requests) != 0 {
		t.Fatalf("Home 目录不可用时启动了终端: %#v", backend.requests)
	}
}

func TestOpenTaskFolderRejectsNonDirectoryHomeForRunningTaskWithoutLifecycleChains(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	homeFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(homeFile, nil, 0o600); err != nil {
		t.Fatalf("创建 Home 文件: %v", err)
	}
	app.homeDirectory = func() (string, error) {
		return homeFile, nil
	}
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("文件 Home", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	app.directoryOpener = func(string) error {
		t.Fatal("Home 路径不是目录时不应打开目录")
		return nil
	}

	err = app.OpenTaskFolder(started.ID)
	if err == nil || !strings.Contains(err.Error(), "用户 Home 目录不是目录") {
		t.Fatalf("打开目录错误 = %v，期望 Home 路径不是目录错误", err)
	}
}

func TestOpenTaskFolderKeepsWorkspaceForRunningTaskWithLifecycleChain(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("已选链目录", "", task.DefaultColor, nil, map[task.LifecycleHook]string{
		task.LifecycleHookPostEnd: settings.LifecycleChainDeleteWorkspaceID,
	})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	homeQueried := false
	app.homeDirectory = func() (string, error) {
		homeQueried = true
		return t.TempDir(), nil
	}

	err = app.OpenTaskFolder(started.ID)
	if err == nil || !strings.Contains(err.Error(), "检查任务工作目录") {
		t.Fatalf("打开目录错误 = %v，期望任务工作目录错误", err)
	}
	if homeQueried {
		t.Fatal("已选链任务不应查询 Home 目录")
	}
}

func TestRunTaskCommandUsesRunningTaskWorkspace(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTask("运行命令", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)

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

func TestRunTaskCommandUsesHomeForRunningTaskWithoutLifecycleChains(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("Home 后台命令", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	var directory string
	app.commandRunner = func(nextDirectory, _ string, _ string, _ []string, _ []string) error {
		directory = nextDirectory
		return nil
	}
	if err := app.RunTaskCommand(started.ID, "code", []string{"."}); err != nil {
		t.Fatalf("运行任务命令: %v", err)
	}
	if directory != home {
		t.Fatalf("任务命令目录 = %q，期望 Home 目录 %q", directory, home)
	}
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

func TestExecuteTaskMenuCommandUsesHomeForRunningTaskWithoutLifecycleChains(t *testing.T) {
	item := settings.TaskMenuItem{
		ID:           "home-command",
		Kind:         settings.TaskMenuItemKindCommand,
		Name:         "Home 命令",
		Command:      "main-command",
		BeforeScript: &settings.TaskScript{Script: "prepare"},
		AfterScript:  &settings.TaskScript{Script: "cleanup"},
	}
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: t.TempDir(), TaskTreeWidth: settings.DefaultTaskTreeWidth, TaskMenuItems: []settings.TaskMenuItem{item},
	}); err != nil {
		t.Fatalf("保存菜单设置: %v", err)
	}
	created, err := app.CreateTaskWithExtraInfoAndLifecycleChains("Home 菜单命令", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	started := startTaskAndWait(t, app, created.ID)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	var beforeDirectory, commandDirectory string
	afterDirectories := make(chan string, 1)
	waiter := &controlledCommandWaiter{done: make(chan error, 1)}
	app.scriptRunner = func(directory, _ string, script string, _ []string, _ []byte, _ []string) error {
		if script != "prepare" {
			t.Fatalf("前置脚本 = %q，期望 prepare", script)
		}
		beforeDirectory = directory
		return nil
	}
	app.commandStarter = func(directory, _ string, command string, _ []string, _ []string) (commandWaiter, error) {
		if command != "main-command" {
			t.Fatalf("菜单命令 = %q，期望 main-command", command)
		}
		commandDirectory = directory
		return waiter, nil
	}
	app.scriptStarter = func(directory, _ string, script string, _ []string, _ []byte, _ []string) (commandWaiter, error) {
		if script != "cleanup" {
			t.Fatalf("后置脚本 = %q，期望 cleanup", script)
		}
		afterDirectories <- directory
		return commandWaiterFunc(func() error { return nil }), nil
	}

	if _, err := app.ExecuteTaskMenuCommand(started.ID, item.ID, 100, 32); err != nil {
		t.Fatalf("执行菜单命令: %v", err)
	}
	if beforeDirectory != home || commandDirectory != home {
		t.Fatalf("前置脚本目录 = %q，菜单命令目录 = %q，期望 Home 目录 %q", beforeDirectory, commandDirectory, home)
	}
	waiter.done <- nil
	select {
	case afterDirectory := <-afterDirectories:
		if afterDirectory != home {
			t.Fatalf("后置脚本目录 = %q，期望 Home 目录 %q", afterDirectory, home)
		}
	case <-time.After(time.Second):
		t.Fatal("后置脚本未启动")
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
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	if _, err := app.SaveSettings(settings.Settings{
		WorkspaceRoot: t.TempDir(), TaskTreeWidth: settings.DefaultTaskTreeWidth, TaskMenuItems: []settings.TaskMenuItem{item},
	}); err != nil {
		t.Fatalf("保存菜单设置: %v", err)
	}
	created, err := app.CreateTask("运行命令", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("创建任务: %v", err)
	}
	return app, startTaskAndWait(t, app, created.ID)
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
func (capturingTerminalSession) Wait() (terminal.ExitResult, error) {
	exitCode := 0
	return terminal.ExitResult{ExitCode: &exitCode}, nil
}

type activeTerminalBackend struct{}

type exitedTerminalBackend struct {
	exitResult terminal.ExitResult
}

func (backend exitedTerminalBackend) Start(request terminal.StartRequest) (terminal.Session, error) {
	return &exitedTerminalSession{id: request.ID, exitResult: backend.exitResult}, nil
}

type exitedTerminalSession struct {
	id         string
	exitResult terminal.ExitResult
}

func (session *exitedTerminalSession) ID() string            { return session.id }
func (exitedTerminalSession) Read([]byte) (int, error)       { return 0, io.EOF }
func (exitedTerminalSession) Write(data []byte) (int, error) { return len(data), nil }
func (exitedTerminalSession) Resize(uint16, uint16) error    { return nil }
func (session *exitedTerminalSession) Wait() (terminal.ExitResult, error) {
	return session.exitResult, nil
}
func (exitedTerminalSession) Close() error { return nil }

func intPointer(value int) *int { return &value }

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
func (session *activeTerminalSession) Wait() (terminal.ExitResult, error) {
	exitCode := 0
	return terminal.ExitResult{ExitCode: &exitCode}, nil
}
func (session *activeTerminalSession) Close() error {
	_ = session.writer.Close()
	return session.reader.Close()
}

type failingCloseTerminalBackend struct {
	session *failingCloseTerminalSession
}

func (backend *failingCloseTerminalBackend) Start(request terminal.StartRequest) (terminal.Session, error) {
	reader, writer := io.Pipe()
	backend.session = &failingCloseTerminalSession{id: request.ID, reader: reader, writer: writer}
	return backend.session, nil
}

func (backend *failingCloseTerminalBackend) stop() {
	if backend.session == nil {
		return
	}
	_ = backend.session.writer.Close()
	_ = backend.session.reader.Close()
}

type failingCloseTerminalSession struct {
	id     string
	reader *io.PipeReader
	writer *io.PipeWriter
}

func (session *failingCloseTerminalSession) ID() string { return session.id }
func (session *failingCloseTerminalSession) Read(data []byte) (int, error) {
	return session.reader.Read(data)
}
func (session *failingCloseTerminalSession) Write(data []byte) (int, error) { return len(data), nil }
func (session *failingCloseTerminalSession) Resize(uint16, uint16) error    { return nil }
func (session *failingCloseTerminalSession) Wait() (terminal.ExitResult, error) {
	exitCode := 0
	return terminal.ExitResult{ExitCode: &exitCode}, nil
}
func (session *failingCloseTerminalSession) Close() error { return errors.New("关闭终端失败") }

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
