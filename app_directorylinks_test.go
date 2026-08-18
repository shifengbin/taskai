//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"taskai/internal/task"
)

func TestAppDirectoryLinksPreserveLifecycleCommitAndRetrySemantics(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.TaskTemplates = []task.TaskTemplate{{
		ID: "directories", Name: "目录", Fields: []task.TaskTemplateField{{
			Key: "sources", DisplayName: "来源目录", InputType: task.TaskTemplateFieldInputDirectories, Multiple: true, Required: true,
		}},
	}}
	current.ActiveTaskTemplateID = "directories"
	if _, err := app.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	sourceA := filepath.Join(t.TempDir(), "source-a")
	if err := os.Mkdir(sourceA, 0o755); err != nil {
		t.Fatal(err)
	}
	created, err := app.CreateTaskWithExtraInfoAndTemplateFields("目录任务", "", task.DefaultColor, nil, map[string]any{"sources": []string{sourceA}})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	switched, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	switched.TaskTemplates = append(switched.TaskTemplates, task.TaskTemplate{
		ID: "release", Name: "发布", Fields: []task.TaskTemplateField{{
			Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, DefaultValue: "production",
		}},
	})
	switched.ActiveTaskTemplateID = "release"
	if _, err := app.SaveSettings(switched); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if err := os.Remove(sourceA); err != nil {
		t.Fatal(err)
	}
	if _, err := app.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	failedStart := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusPending && lifecycleHookFailed(current, task.LifecycleHookBeforeStart)
	})
	if failedStart.LifecycleExecution == nil || failedStart.LifecycleExecution.WorkspaceOwnership != task.LifecycleWorkspaceCreated {
		t.Fatalf("开始失败后的工作目录归属 = %#v", failedStart.LifecycleExecution)
	}
	if err := os.Mkdir(sourceA, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := app.RetryTaskLifecycleCommandChain(created.ID); err != nil {
		t.Fatalf("RetryTaskLifecycleCommandChain(start) error = %v", err)
	}
	started := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusRunning && current.LifecycleExecution == nil
	})
	assertAppDirectoryLink(t, filepath.Join(started.WorkspacePath, "source-a"), sourceA)

	sourceB := filepath.Join(t.TempDir(), "source-b")
	if err := os.Mkdir(sourceB, 0o755); err != nil {
		t.Fatal(err)
	}
	conflict := filepath.Join(started.WorkspacePath, "source-b")
	if err := os.Mkdir(conflict, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateTaskWithExtraInfoAndTemplateFields(created.ID, "已保存的新目录", "", task.DefaultColor, nil, map[string]any{"sources": []string{sourceB}}); err != nil {
		t.Fatalf("UpdateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	failedUpdate := waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusRunning && current.Title == "已保存的新目录" && lifecycleHookFailed(current, task.LifecycleHookUpdateTask)
	})
	if want := []string{sourceB}; !reflect.DeepEqual(failedUpdate.TemplateFields["sources"], want) {
		t.Fatalf("更新同步失败后目录值 = %#v，期望保留 %#v", failedUpdate.TemplateFields["sources"], want)
	}
	if err := os.Remove(conflict); err != nil {
		t.Fatal(err)
	}
	if _, err := app.RetryTaskLifecycleCommandChain(created.ID); err != nil {
		t.Fatalf("RetryTaskLifecycleCommandChain(update) error = %v", err)
	}
	waitForTask(t, app, created.ID, func(current task.Task) bool {
		return current.Status == task.StatusRunning && current.LifecycleExecution == nil
	})
	assertAppDirectoryLink(t, filepath.Join(started.WorkspacePath, "source-b"), sourceB)
	if _, err := os.Lstat(filepath.Join(started.WorkspacePath, "source-a")); !os.IsNotExist(err) {
		t.Fatalf("重试后旧目录链接仍存在: %v", err)
	}
}

func assertAppDirectoryLink(t *testing.T, path, wantTarget string) {
	t.Helper()
	target, err := os.Readlink(path)
	if err != nil || target != wantTarget {
		t.Fatalf("目录链接 %q = %q err=%v，期望 %q", path, target, err, wantTarget)
	}
}
