package main

import (
	"os"
	"path/filepath"
	"testing"

	"taskai/internal/settings"
	"taskai/internal/task"
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
	var arguments []string
	app.commandRunner = func(nextDirectory, nextShellPath, nextCommand string, nextArguments []string) error {
		directory = nextDirectory
		shellPath = nextShellPath
		command = nextCommand
		arguments = nextArguments
		return nil
	}
	if err := app.RunTaskCommand(started.ID, "code", []string{"."}); err != nil {
		t.Fatalf("运行任务命令: %v", err)
	}
	if directory != started.WorkspacePath || shellPath == "" || command != "code" || len(arguments) != 1 || arguments[0] != "." {
		t.Fatalf("运行任务命令参数 = directory:%q shell:%q command:%q arguments:%#v", directory, shellPath, command, arguments)
	}
}
