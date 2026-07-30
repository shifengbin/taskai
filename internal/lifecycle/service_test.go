package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func taskIDs(tasks []task.Task) []string {
	ids := make([]string, 0, len(tasks))
	for _, current := range tasks {
		ids = append(ids, current.ID)
	}
	return ids
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
	if got := created.LifecycleChains[task.LifecycleHookBeforeStart]; got != settings.LifecycleChainCreateWorkspaceID {
		t.Errorf("CreateTask() beforeStart 默认链 = %q", got)
	}
	if got := created.LifecycleChains[task.LifecycleHookPostEnd]; got != settings.LifecycleChainDeleteWorkspaceID {
		t.Errorf("CreateTask() postEnd 默认链 = %q", got)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.Tasks) != 1 || !reflect.DeepEqual(data.Tasks[0], created) {
		t.Errorf("CreateTask() persisted Tasks = %#v, want %#v", data.Tasks, created)
	}
}

func TestServiceStartingTaskClearsShelvedFlag(t *testing.T) {
	service, repository, _ := newService(t)
	created, err := service.CreateTask("待开始任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Tasks[0].Shelved = true
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	started, err := service.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if started.Shelved {
		t.Fatalf("StartTask() Shelved = true，期望 false")
	}
}

func TestServiceFinishingTaskClearsShelvedFlag(t *testing.T) {
	service, repository, _ := newService(t)
	created, err := service.CreateTask("待结束任务", "", task.DefaultColor)
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
	data.Tasks[0].Shelved = true
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	completed, err := service.FinishTask(created.ID)
	if err != nil {
		t.Fatalf("FinishTask() error = %v", err)
	}
	if completed.Shelved {
		t.Fatalf("FinishTask() Shelved = true，期望 false")
	}
}

func TestServiceCreatesTaskLifecycleChainSelectionsForApplicableHooks(t *testing.T) {
	service, _, _ := newService(t)
	selected := map[task.LifecycleHook]string{
		task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
		task.LifecycleHookPostEnd:     settings.LifecycleChainDeleteWorkspaceID,
	}

	created, err := service.CreateTaskWithExtraInfoAndLifecycleChains("编写登录页", "", task.DefaultColor, nil, selected)
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndLifecycleChains() error = %v", err)
	}
	if !reflect.DeepEqual(created.LifecycleChains, selected) {
		t.Fatalf("创建任务的命令链选择 = %#v，期望 %#v", created.LifecycleChains, selected)
	}

	if _, err := service.CreateTaskWithExtraInfoAndLifecycleChains("无效选择", "", task.DefaultColor, nil, map[task.LifecycleHook]string{
		task.LifecycleHookPostEnd: "missing-chain",
	}); err == nil {
		t.Fatal("选择不存在命令链 error = nil")
	}
	if _, err := service.CreateTaskWithExtraInfoAndLifecycleChains("范围不匹配", "", task.DefaultColor, nil, map[task.LifecycleHook]string{
		task.LifecycleHookPostStart: settings.LifecycleChainCreateWorkspaceID,
	}); err == nil {
		t.Fatal("选择不适用于钩子的命令链 error = nil")
	}
}

func TestServiceUpdatesTaskDetailsWithoutChangingLifecycle(t *testing.T) {
	service, repository, _ := newService(t)
	created, err := service.CreateTask("编写登录页", "旧描述", "#ef4444")
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	started, err := service.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}

	updated, err := service.UpdateTask(started.ID, "  更新登录页  ", "新描述", "#A1B2C3")

	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Title != "更新登录页" || updated.Description != "新描述" || updated.Color != "#a1b2c3" {
		t.Errorf("UpdateTask() details = %#v", updated)
	}
	if updated.Status != started.Status || updated.CreatedAt != started.CreatedAt || updated.WorkspaceRoot != started.WorkspaceRoot || updated.WorkspacePath != started.WorkspacePath {
		t.Errorf("UpdateTask() changed lifecycle fields: %#v", updated)
	}
	if !reflect.DeepEqual(updated.LifecycleChains, started.LifecycleChains) {
		t.Errorf("UpdateTask() changed lifecycle chain selections: %#v", updated.LifecycleChains)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.Tasks) != 1 || !reflect.DeepEqual(data.Tasks[0], updated) {
		t.Errorf("UpdateTask() persisted Tasks = %#v, want %#v", data.Tasks, updated)
	}
}

func TestServiceUpdatesLifecycleChainsOnlyForPendingTask(t *testing.T) {
	service, repository, _ := newService(t)
	pending, err := service.CreateTask("待编辑任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	selected := map[task.LifecycleHook]string{
		task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
	}

	updated, err := service.UpdateTaskWithExtraInfoAndLifecycleChains(pending.ID, "已更新待编辑任务", "", task.DefaultColor, nil, selected)
	if err != nil {
		t.Fatalf("未执行任务更新命令链 error = %v", err)
	}
	if !reflect.DeepEqual(updated.LifecycleChains, selected) {
		t.Fatalf("未执行任务命令链 = %#v，期望 %#v", updated.LifecycleChains, selected)
	}
	if updated.LifecycleExecution != nil {
		t.Fatalf("未执行任务修改不应产生生命周期执行记录 = %#v", updated.LifecycleExecution)
	}
	if _, err := service.UpdateTaskWithExtraInfoAndLifecycleChains(pending.ID, "范围不匹配", "", task.DefaultColor, nil, map[task.LifecycleHook]string{
		task.LifecycleHookPostStart: settings.LifecycleChainCreateWorkspaceID,
	}); err == nil {
		t.Fatal("未执行任务选择范围不匹配的命令链 error = nil")
	}

	for _, status := range []task.Status{task.StatusRunning, task.StatusCompleted} {
		other, err := service.CreateTask(string(status)+" 任务", "", task.DefaultColor)
		if err != nil {
			t.Fatalf("CreateTask(%s) error = %v", status, err)
		}
		data, err := repository.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		index, err := taskIndex(data.Tasks, other.ID)
		if err != nil {
			t.Fatalf("taskIndex() error = %v", err)
		}
		data.Tasks[index].Status = status
		if err := repository.Save(data); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		if _, err := service.UpdateTaskWithExtraInfoAndLifecycleChains(other.ID, "不应更新", "", task.DefaultColor, nil, selected); err == nil {
			t.Fatalf("%s 任务更新命令链 error = nil", status)
		}
	}
}

func TestServicePersistsTaskLifecycleExecution(t *testing.T) {
	service, repository, _ := newService(t)
	created, err := service.CreateTask("编写登录页", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	execution := &task.LifecycleExecution{
		Hook:               task.LifecycleHookBeforeStart,
		ChainID:            "chain-1",
		CurrentCommandID:   "command-1",
		CurrentCommandName: "创建目录",
		CurrentIndex:       1,
		CommandCount:       2,
		State:              task.LifecycleExecutionRunning,
	}

	updated, err := service.UpdateLifecycleExecution(created.ID, execution)
	if err != nil {
		t.Fatalf("UpdateLifecycleExecution() error = %v", err)
	}
	if !updated.IsLifecycleLocked() || updated.LifecycleExecution == nil || updated.LifecycleExecution.CurrentCommandName != "创建目录" {
		t.Fatalf("更新后的执行记录 = %#v", updated.LifecycleExecution)
	}
	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(persisted.Tasks[0].LifecycleExecution, updated.LifecycleExecution) {
		t.Fatalf("持久化执行记录 = %#v，期望 %#v", persisted.Tasks[0].LifecycleExecution, updated.LifecycleExecution)
	}

	cleared, err := service.UpdateLifecycleExecution(created.ID, nil)
	if err != nil {
		t.Fatalf("clear UpdateLifecycleExecution() error = %v", err)
	}
	if cleared.LifecycleExecution != nil || cleared.IsLifecycleLocked() {
		t.Fatalf("清除后的执行记录 = %#v", cleared.LifecycleExecution)
	}
}

func TestServiceConditionallyUpdatesLifecycleExecutionByRunAndRevision(t *testing.T) {
	service, _, _ := newService(t)
	created, err := service.CreateTask("执行命令链", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	initial := &task.LifecycleExecution{
		RunID:              "run-current",
		Revision:           1,
		Hook:               task.LifecycleHookPostStart,
		ChainID:            "chain-1",
		CurrentCommandID:   "command-1",
		CurrentCommandName: "初始化",
		CurrentIndex:       1,
		CommandCount:       2,
		State:              task.LifecycleExecutionRunning,
	}
	if _, err := service.UpdateLifecycleExecution(created.ID, initial); err != nil {
		t.Fatalf("UpdateLifecycleExecution() error = %v", err)
	}

	progressed := *initial
	progressed.Revision = 2
	progressed.CurrentCommandID = "command-2"
	progressed.CurrentCommandName = "安装依赖"
	progressed.CurrentIndex = 2
	updated, applied, err := service.UpdateLifecycleExecutionIfNewer(created.ID, &progressed)
	if err != nil || !applied || updated.LifecycleExecution == nil || updated.LifecycleExecution.Revision != 2 {
		t.Fatalf("UpdateLifecycleExecutionIfNewer() = (%#v, %t, %v)", updated.LifecycleExecution, applied, err)
	}

	stale := *initial
	stale.CurrentCommandName = "过期进度"
	updated, applied, err = service.UpdateLifecycleExecutionIfNewer(created.ID, &stale)
	if err != nil || applied || updated.LifecycleExecution == nil || updated.LifecycleExecution.CurrentCommandName != "安装依赖" {
		t.Fatalf("低版本更新不应覆盖当前记录: (%#v, %t, %v)", updated.LifecycleExecution, applied, err)
	}

	otherRun := progressed
	otherRun.RunID = "run-retry"
	otherRun.Revision = 3
	updated, applied, err = service.UpdateLifecycleExecutionIfNewer(created.ID, &otherRun)
	if err != nil || applied || updated.LifecycleExecution == nil || updated.LifecycleExecution.RunID != "run-current" {
		t.Fatalf("旧运行不应覆盖新运行: (%#v, %t, %v)", updated.LifecycleExecution, applied, err)
	}

	updated, applied, err = service.ClearLifecycleExecutionIfCurrent(created.ID, "run-current", 1)
	if err != nil || applied || updated.LifecycleExecution == nil {
		t.Fatalf("旧版本清除不应生效: (%#v, %t, %v)", updated.LifecycleExecution, applied, err)
	}
	updated, applied, err = service.ClearLifecycleExecutionIfCurrent(created.ID, "run-current", 2)
	if err != nil || !applied || updated.LifecycleExecution != nil {
		t.Fatalf("当前版本清除失败: (%#v, %t, %v)", updated.LifecycleExecution, applied, err)
	}
}

func TestServiceReordersTasksWithinStatusAndPersistsOrder(t *testing.T) {
	service, repository, _ := newService(t)
	first, err := service.CreateTask("第一个待办", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	running, err := service.CreateTask("执行中的任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	second, err := service.CreateTask("第二个待办", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Tasks[1].Status = task.StatusRunning
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reordered, err := service.ReorderTasks(task.StatusPending, []string{second.ID, first.ID})

	if err != nil {
		t.Fatalf("ReorderTasks() error = %v", err)
	}
	if got, want := []string{reordered[0].ID, reordered[1].ID, reordered[2].ID}, []string{second.ID, running.ID, first.ID}; !sameTaskIDs(got, want) {
		t.Errorf("ReorderTasks() IDs = %#v, want %#v", got, want)
	}
	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := []string{persisted.Tasks[0].ID, persisted.Tasks[1].ID, persisted.Tasks[2].ID}, []string{second.ID, running.ID, first.ID}; !sameTaskIDs(got, want) {
		t.Errorf("persisted task IDs = %#v, want %#v", got, want)
	}
}

func TestServiceSetsTaskShelvedAndMaintainsRunningGroups(t *testing.T) {
	service, repository, _ := newService(t)
	first, err := service.CreateTask("正常任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	target, err := service.CreateTask("待搁置任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	existing, err := service.CreateTask("已有搁置任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	for _, current := range []task.Task{first, target, existing} {
		if _, err := service.StartTask(current.ID); err != nil {
			t.Fatalf("StartTask(%q) error = %v", current.ID, err)
		}
	}

	shelver, ok := any(service).(interface {
		SetTaskShelved(taskID string, shelved bool) ([]task.Task, error)
	})
	if !ok {
		t.Fatal("Service 缺少 SetTaskShelved()")
	}
	if _, err := shelver.SetTaskShelved(existing.ID, true); err != nil {
		t.Fatalf("SetTaskShelved(existing) error = %v", err)
	}
	shelved, err := shelver.SetTaskShelved(target.ID, true)
	if err != nil {
		t.Fatalf("SetTaskShelved(target) error = %v", err)
	}
	if got, want := taskIDs(shelved), []string{first.ID, existing.ID, target.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("搁置后的任务顺序 = %#v，期望 %#v", got, want)
	}
	if !shelved[1].Shelved || !shelved[2].Shelved {
		t.Fatalf("搁置后的任务标记 = %#v", shelved)
	}

	restored, err := shelver.SetTaskShelved(existing.ID, false)
	if err != nil {
		t.Fatalf("SetTaskShelved(existing, false) error = %v", err)
	}
	if got, want := taskIDs(restored), []string{first.ID, existing.ID, target.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("取消搁置后的任务顺序 = %#v，期望 %#v", got, want)
	}
	if restored[1].Shelved || !restored[2].Shelved {
		t.Fatalf("取消搁置后的任务标记 = %#v", restored)
	}
	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := taskIDs(persisted.Tasks), taskIDs(restored); !reflect.DeepEqual(got, want) {
		t.Fatalf("持久化任务顺序 = %#v，期望 %#v", got, want)
	}
}

func TestServiceRejectsShelvingNonRunningOrLockedTasks(t *testing.T) {
	service, repository, _ := newService(t)
	pending, err := service.CreateTask("未执行任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	before, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, err := service.SetTaskShelved(pending.ID, true); err == nil {
		t.Fatal("SetTaskShelved() error = nil，期望拒绝未执行任务")
	}
	after, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(after.Tasks, before.Tasks) {
		t.Fatalf("未执行任务切换搁置状态后数据被修改: %#v", after.Tasks)
	}

	running, err := service.CreateTask("锁定任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, err := service.StartTask(running.ID); err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if _, err := service.UpdateLifecycleExecution(running.ID, &task.LifecycleExecution{
		Hook:         task.LifecycleHookPostStart,
		ChainID:      "chain-1",
		CurrentIndex: 1,
		CommandCount: 1,
		State:        task.LifecycleExecutionRunning,
	}); err != nil {
		t.Fatalf("UpdateLifecycleExecution() error = %v", err)
	}
	if _, err := service.SetTaskShelved(running.ID, true); err == nil {
		t.Fatal("SetTaskShelved() error = nil，期望拒绝锁定任务")
	}
}

func TestServiceRejectsInvalidTaskOrder(t *testing.T) {
	service, repository, _ := newService(t)
	first, err := service.CreateTask("第一个待办", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	second, err := service.CreateTask("第二个待办", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	_, err = service.ReorderTasks(task.StatusPending, []string{first.ID, first.ID})
	if err == nil {
		t.Fatal("ReorderTasks() error = nil, want duplicate task ID error")
	}
	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := []string{persisted.Tasks[0].ID, persisted.Tasks[1].ID}, []string{first.ID, second.ID}; !sameTaskIDs(got, want) {
		t.Errorf("invalid order changed persisted IDs = %#v, want %#v", got, want)
	}
}

func TestServiceStartsPendingTaskWithWorkspaceSnapshotWithoutCreatingDirectory(t *testing.T) {
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
	if _, err := os.Stat(started.WorkspacePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("StartTask() 不应隐式创建工作目录: %v", err)
	}
}

func TestServiceStartsTaskWithoutRequiringWorkspaceDirectoryCreation(t *testing.T) {
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

	started, err := service.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Tasks[0].Status != task.StatusRunning || started.WorkspacePath == "" {
		t.Errorf("StartTask() 未保存工作目录快照: %#v", data.Tasks[0])
	}
}

func TestServiceFinishesTaskAfterClosingTerminalsWithoutDeletingWorkspace(t *testing.T) {
	service, _, _ := newService(t)
	created, err := service.CreateTask("编写登录页", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	started, err := service.StartTask(created.ID)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if err := os.MkdirAll(started.WorkspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
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
	if _, err := os.Stat(started.WorkspacePath); err != nil {
		t.Errorf("FinishTask() 不应隐式删除工作目录: %v", err)
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
	if err := os.MkdirAll(started.WorkspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
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

func newService(t *testing.T) (*Service, *storage.Repository, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspaces")
	service, repository := newServiceWithRoot(t, root)
	return service, repository, root
}

func newServiceWithRoot(t *testing.T, root string) (*Service, *storage.Repository) {
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

func sameTaskIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
