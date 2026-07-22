package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"taskai/internal/settings"
	"taskai/internal/storage"
	"taskai/internal/task"
)

type closerStub struct {
	closedTaskIDs []string
	err           error
}

func (closer *closerStub) CloseTask(taskID string) error {
	closer.closedTaskIDs = append(closer.closedTaskIDs, taskID)
	return closer.err
}

func TestServiceCreatesPendingTask(t *testing.T) {
	service, repository, _ := newService(t)

	created, err := service.CreateTask("编写登录页", "实现邮箱登录表单", "#ef4444")

	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if created.Status != task.StatusPending {
		t.Errorf("CreateTask() Status = %q, want %q", created.Status, task.StatusPending)
	}
	if created.Color != "#ef4444" {
		t.Errorf("CreateTask() Color = %q, want %q", created.Color, "#ef4444")
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.Tasks) != 1 || data.Tasks[0] != created {
		t.Errorf("CreateTask() persisted Tasks = %#v, want %#v", data.Tasks, created)
	}
}

func TestServiceStartsPendingTaskWithWorkspaceSnapshot(t *testing.T) {
	service, _, root := newService(t)
	created, err := service.CreateTask("编写登录页", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	started, err := service.StartTask(created.ID)

	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if started.Status != task.StatusRunning {
		t.Errorf("StartTask() Status = %q, want %q", started.Status, task.StatusRunning)
	}
	if started.WorkspaceRoot != root {
		t.Errorf("StartTask() WorkspaceRoot = %q, want %q", started.WorkspaceRoot, root)
	}
	if started.WorkspacePath != filepath.Join(root, created.ID) {
		t.Errorf("StartTask() WorkspacePath = %q, want %q", started.WorkspacePath, filepath.Join(root, created.ID))
	}
	if info, err := os.Stat(started.WorkspacePath); err != nil || !info.IsDir() {
		t.Errorf("StartTask() workspace missing or invalid: %v", err)
	}
}

func TestServiceKeepsPendingTaskWhenWorkspaceCreationFails(t *testing.T) {
	rootParent := t.TempDir()
	root := filepath.Join(rootParent, "not-a-directory")
	if err := os.WriteFile(root, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	service, repository := newServiceWithRoot(t, root)
	created, err := service.CreateTask("编写登录页", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	_, err = service.StartTask(created.ID)

	if err == nil {
		t.Fatal("StartTask() error = nil, want workspace creation error")
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Tasks[0].Status != task.StatusPending || data.Tasks[0].WorkspacePath != "" {
		t.Errorf("StartTask() changed task after workspace error: %#v", data.Tasks[0])
	}
}

func TestServiceFinishesTaskAfterClosingTerminalsAndDeletingWorkspace(t *testing.T) {
	service, _, _ := newService(t)
	created, err := service.CreateTask("编写登录页", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	started, err := service.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(started.WorkspacePath, "temporary.txt"), []byte("temporary"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	completed, err := service.FinishTask(created.ID)

	if err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	if completed.Status != task.StatusCompleted || completed.CompletedAt == nil {
		t.Errorf("FinishTask() task = %#v, want completed task", completed)
	}
	if _, err := os.Stat(started.WorkspacePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("FinishTask() workspace still exists, Stat() error = %v", err)
	}
	if len(service.closer.(*closerStub).closedTaskIDs) != 1 || service.closer.(*closerStub).closedTaskIDs[0] != created.ID {
		t.Errorf("FinishTask() closed task IDs = %#v, want %#v", service.closer.(*closerStub).closedTaskIDs, []string{created.ID})
	}
}

func TestServiceKeepsRunningTaskWhenTerminalCleanupFails(t *testing.T) {
	service, repository, _ := newService(t)
	service.closer.(*closerStub).err = errors.New("close failed")
	created, err := service.CreateTask("编写登录页", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	started, err := service.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}

	_, err = service.FinishTask(created.ID)

	if err == nil {
		t.Fatal("FinishTask() error = nil, want terminal cleanup error")
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Tasks[0].Status != task.StatusRunning {
		t.Errorf("FinishTask() status = %q, want %q", data.Tasks[0].Status, task.StatusRunning)
	}
	if _, err := os.Stat(started.WorkspacePath); err != nil {
		t.Errorf("FinishTask() removed workspace after close failure: %v", err)
	}
}

func TestServiceKeepsRunningTaskWhenWorkspaceCleanupFails(t *testing.T) {
	service, repository, _ := newService(t)
	created, err := service.CreateTask("编写登录页", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := service.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Tasks[0].WorkspacePath = filepath.Join(t.TempDir(), "outside", created.ID)
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err = service.FinishTask(created.ID)

	if err == nil {
		t.Fatal("FinishTask() error = nil, want workspace cleanup error")
	}
	data, err = repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Tasks[0].Status != task.StatusRunning {
		t.Errorf("FinishTask() status = %q, want %q", data.Tasks[0].Status, task.StatusRunning)
	}
}

func TestServiceDoesNotRestartCompletedTask(t *testing.T) {
	service, _, _ := newService(t)
	created, err := service.CreateTask("编写登录页", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := service.StartTask(created.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if _, err := service.FinishTask(created.ID); err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}

	_, err = service.StartTask(created.ID)

	if err == nil {
		t.Fatal("StartTask() error = nil, want completed task rejection")
	}
}

func newService(t *testing.T) (*Service, storage.Repository, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspaces")
	service, repository := newServiceWithRoot(t, root)
	return service, repository, root
}

func newServiceWithRoot(t *testing.T, root string) (*Service, storage.Repository) {
	t.Helper()
	repository := storage.New(filepath.Join(t.TempDir(), "state.json"), settings.Settings{
		WorkspaceRoot: root,
		TaskTreeWidth: settings.DefaultTaskTreeWidth,
	})
	closer := &closerStub{}
	return New(repository, closer, func() time.Time {
		return time.Date(2026, time.July, 22, 10, 0, 0, 0, time.UTC)
	}), repository
}
