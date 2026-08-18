package lifecycle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"taskai/internal/settings"
	"taskai/internal/storage"
	"taskai/internal/task"
	"taskai/internal/workspace"
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

func TestServiceCreatesTaskFromDefaultLifecyclePreset(t *testing.T) {
	service, repository, _ := newService(t)
	chain, err := repository.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
		Name: "开始后准备", Commands: []settings.LifecycleCommandReference{{CommandID: settings.LifecycleCommandCreateWorkspaceID}},
		ApplicableHooks: []settings.LifecycleHook{task.LifecycleHookPostStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommandChain() error = %v", err)
	}
	preset, err := repository.SaveLifecyclePreset(settings.LifecyclePreset{
		Name: "开始后预设", Chains: map[task.LifecycleHook]string{task.LifecycleHookPostStart: chain.ID},
	})
	if err != nil {
		t.Fatalf("SaveLifecyclePreset() error = %v", err)
	}
	if _, err := repository.SaveDefaultLifecyclePreset(preset.ID); err != nil {
		t.Fatalf("SaveDefaultLifecyclePreset() error = %v", err)
	}

	created, err := service.CreateTask("使用预设", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if want := map[task.LifecycleHook]string{task.LifecycleHookPostStart: chain.ID}; !reflect.DeepEqual(created.LifecycleChains, want) {
		t.Fatalf("CreateTask() 生命周期链 = %#v，期望 %#v", created.LifecycleChains, want)
	}
}

func TestServiceCreatesTaskFromCompanyFrameworkDefaultPreset(t *testing.T) {
	dataDirectory := t.TempDir()
	repository := storage.New(filepath.Join(dataDirectory, "state.json"), settings.Default(dataDirectory))
	service := New(repository, &closerStub{}, func() time.Time {
		return time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	})

	created, err := service.CreateTask("使用公司框架", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	want := map[task.LifecycleHook]string{
		task.LifecycleHookBeforeStart: settings.LifecycleChainIterationsAIID,
		task.LifecycleHookPostEnd:     settings.LifecycleChainDeleteWorkspaceID,
		task.LifecycleHookUpdateTask:  settings.LifecycleChainUpdateRepositoriesID,
	}
	if !reflect.DeepEqual(created.LifecycleChains, want) {
		t.Fatalf("CreateTask() 生命周期链 = %#v，期望 %#v", created.LifecycleChains, want)
	}
	encoded, err := json.Marshal(created)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "lifecyclePreset") {
		t.Fatalf("任务不应保存预设关联: %s", encoded)
	}
}

func TestServicePersistsExplicitEmptyLifecycleChains(t *testing.T) {
	service, _, _ := newService(t)

	created, err := service.CreateTaskWithExtraInfoAndLifecycleChains("不使用预设", "", task.DefaultColor, nil, map[task.LifecycleHook]string{})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndLifecycleChains() error = %v", err)
	}
	if len(created.LifecycleChains) != 0 {
		t.Fatalf("显式空生命周期链 = %#v，期望为空", created.LifecycleChains)
	}
}

func TestServiceCreatesAndUpdatesCurrentTaskTemplateFields(t *testing.T) {
	service, repository, _ := newService(t)
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskTemplates = []task.TaskTemplate{{
		ID: "release", Name: "发布", Fields: []task.TaskTemplateField{
			{Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, Required: true, DefaultValue: "production"},
			{Key: "deploy", DisplayName: "允许部署", InputType: task.TaskTemplateFieldInputBool, DefaultValue: false},
		},
	}}
	data.Settings.ActiveTaskTemplateID = "release"
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	created, err := service.CreateTaskWithTemplateFields("发布 API", "", task.DefaultColor, map[string]any{"environment": "staging"})
	if err != nil {
		t.Fatalf("CreateTaskWithTemplateFields() error = %v", err)
	}
	if want := map[string]any{"environment": "staging", "deploy": false}; !reflect.DeepEqual(created.TemplateFields, want) {
		t.Fatalf("创建任务模板字段 = %#v，期望 %#v", created.TemplateFields, want)
	}
	if created.TaskTemplateID != "release" {
		t.Fatalf("创建任务模板 ID = %q，期望 release", created.TaskTemplateID)
	}

	data, err = repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.Tasks) != 1 || data.Tasks[0].TaskTemplateID != "release" {
		t.Fatalf("重载后的任务模板 ID = %#v，期望 release", data.Tasks)
	}
	data.Tasks[0].TemplateFields["retired_field"] = "keep"
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	updated, err := service.UpdateTaskWithTemplateFields(created.ID, "发布 API", "", task.DefaultColor, map[string]any{"environment": "production", "deploy": true})
	if err != nil {
		t.Fatalf("UpdateTaskWithTemplateFields() error = %v", err)
	}
	if want := map[string]any{"environment": "production", "deploy": true, "retired_field": "keep"}; !reflect.DeepEqual(updated.TemplateFields, want) {
		t.Fatalf("更新任务模板字段 = %#v，期望 %#v", updated.TemplateFields, want)
	}
	if updated.TaskTemplateID != "release" {
		t.Fatalf("更新任务模板 ID = %q，期望 release", updated.TaskTemplateID)
	}
}

func TestServiceCreatesLockedDirectoryFieldAndRejectsAtomicUpdate(t *testing.T) {
	service, repository, _ := newService(t)
	locked := false
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskTemplates = []task.TaskTemplate{{
		ID: "directories", Name: "目录", Fields: []task.TaskTemplateField{{
			Key: "source", DisplayName: "来源目录", InputType: task.TaskTemplateFieldInputDirectories, Required: true, Updatable: &locked,
		}},
	}}
	data.Settings.ActiveTaskTemplateID = "directories"
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	projectA := t.TempDir()
	projectB := t.TempDir()
	created, err := service.CreateTaskWithTemplateFields("原始标题", "", task.DefaultColor, map[string]any{"source": []string{projectA}})
	if err != nil {
		t.Fatalf("CreateTaskWithTemplateFields() error = %v", err)
	}
	if _, err := service.UpdateTaskWithTemplateFields(created.ID, "不应保存的标题", "", task.DefaultColor, map[string]any{"source": []string{projectB}}); err == nil {
		t.Fatal("UpdateTaskWithTemplateFields() error = nil，期望拒绝锁定字段变更")
	}

	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := persisted.Tasks[0].Title; got != "原始标题" {
		t.Fatalf("校验失败后任务标题 = %q，期望原始标题", got)
	}
	if want := []string{projectA}; !reflect.DeepEqual(persisted.Tasks[0].TemplateFields["source"], want) {
		t.Fatalf("校验失败后目录字段 = %#v，期望 %#v", persisted.Tasks[0].TemplateFields["source"], want)
	}
}

func TestServiceUpdatesTaskWithItsOriginalTemplateAfterActiveTemplateChanges(t *testing.T) {
	service, repository, _ := newService(t)
	locked := false
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskTemplates = []task.TaskTemplate{
		{ID: "directories", Name: "目录", Fields: []task.TaskTemplateField{{
			Key: "source", DisplayName: "来源目录", InputType: task.TaskTemplateFieldInputDirectories, Required: true, Updatable: &locked,
		}}},
		{ID: "release", Name: "发布", Fields: []task.TaskTemplateField{{
			Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, DefaultValue: "production",
		}}},
	}
	data.Settings.ActiveTaskTemplateID = "directories"
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	projectA := t.TempDir()
	projectB := t.TempDir()
	created, err := service.CreateTaskWithTemplateFields("目录任务", "", task.DefaultColor, map[string]any{"source": []string{projectA}})
	if err != nil {
		t.Fatalf("CreateTaskWithTemplateFields() error = %v", err)
	}
	data, err = repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.ActiveTaskTemplateID = "release"
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	updated, err := service.UpdateTaskWithTemplateFields(created.ID, "更新标题", "", task.DefaultColor, map[string]any{"source": []string{projectA}})
	if err != nil {
		t.Fatalf("UpdateTaskWithTemplateFields() unchanged error = %v", err)
	}
	if updated.TaskTemplateID != "directories" || !reflect.DeepEqual(updated.TemplateFields["source"], []string{projectA}) {
		t.Fatalf("更新后的模板绑定 = %q fields=%#v，期望保留 directories", updated.TaskTemplateID, updated.TemplateFields)
	}
	if _, err := service.UpdateTaskWithTemplateFields(created.ID, "不应保存", "", task.DefaultColor, map[string]any{"source": []string{projectB}}); err == nil {
		t.Fatal("UpdateTaskWithTemplateFields() changed error = nil，期望按原模板拒绝锁定目录变更")
	}
}

func TestServiceDoesNotCopyTaskTemplateBranchIntoGitExtraInfoSnapshots(t *testing.T) {
	service, repository, _ := newService(t)
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskTemplates = []task.TaskTemplate{{
		ID: "release", Name: "发布任务", Fields: []task.TaskTemplateField{{
			Key: "branch", DisplayName: "模板分支", InputType: task.TaskTemplateFieldInputString, DefaultValue: "",
		}},
	}}
	data.Settings.ActiveTaskTemplateID = "release"
	gitInfo, err := task.NewExtraInfoWithParameters(task.BuiltInGitTemplate(), map[string]string{
		"name": "API", "repository": "https://example.com/api.git",
	}, nil)
	if err != nil {
		t.Fatalf("NewExtraInfoWithParameters() error = %v", err)
	}
	data.ExtraInfos = []task.ExtraInfo{gitInfo}
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	emptyBranch := []task.TaskExtraInfo{{InformationID: gitInfo.ID, Parameters: []task.ExtraInfoParameter{{Key: "branch", Value: ""}}}}
	created, err := service.CreateTaskWithExtraInfoAndTemplateFields("发布 API", "", task.DefaultColor, emptyBranch, map[string]any{"branch": "release/1.2"})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() error = %v", err)
	}
	if got := taskExtraInfoParameterValue(created.ExtraInfo[0], "branch"); got != "" {
		t.Fatalf("创建任务不应回填 Git 分支，得到 %q", got)
	}

	explicitBranch := []task.TaskExtraInfo{{InformationID: gitInfo.ID, Parameters: []task.ExtraInfoParameter{{Key: "branch", Value: "hotfix"}}}}
	explicit, err := service.CreateTaskWithExtraInfoAndTemplateFields("热修复", "", task.DefaultColor, explicitBranch, map[string]any{"branch": "release/1.2"})
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() explicit error = %v", err)
	}
	if got := taskExtraInfoParameterValue(explicit.ExtraInfo[0], "branch"); got != "hotfix" {
		t.Fatalf("显式 Git 分支 = %q，期望 hotfix", got)
	}

	emptyTemplate, err := service.CreateTaskWithExtraInfoAndTemplateFields("默认分支", "", task.DefaultColor, emptyBranch, nil)
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoAndTemplateFields() empty template error = %v", err)
	}
	if got := taskExtraInfoParameterValue(emptyTemplate.ExtraInfo[0], "branch"); got != "" {
		t.Fatalf("空模板分支不应改写 Git 分支，得到 %q", got)
	}
	updated, err := service.UpdateTaskWithTemplateFields(emptyTemplate.ID, "默认分支", "", task.DefaultColor, map[string]any{"branch": "stable"})
	if err != nil {
		t.Fatalf("UpdateTaskWithTemplateFields() error = %v", err)
	}
	if got := taskExtraInfoParameterValue(updated.ExtraInfo[0], "branch"); got != "" {
		t.Fatalf("更新模板字段不应回填 Git 分支，得到 %q", got)
	}

	data, err = repository.Load()
	if err != nil {
		t.Fatalf("Load() after create error = %v", err)
	}
	data.Settings.TaskTemplates[0].Fields[0].DefaultValue = "main"
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() after template update error = %v", err)
	}
	persisted, err := service.GetTask(created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got := taskExtraInfoParameterValue(persisted.ExtraInfo[0], "branch"); got != "" {
		t.Fatalf("模板后续修改改写了任务 Git 快照: %q", got)
	}
}

func TestServiceAllowsNonStringTemplateBranchUntilDefaultBranchCommandRuns(t *testing.T) {
	service, repository, _ := newService(t)
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskTemplates = []task.TaskTemplate{{
		ID: "invalid", Name: "错误模板", Fields: []task.TaskTemplateField{{
			Key: "branch", DisplayName: "模板分支", InputType: task.TaskTemplateFieldInputBool, DefaultValue: false,
		}},
	}}
	data.Settings.ActiveTaskTemplateID = "invalid"
	data.Settings.LifecycleChains = append(data.Settings.LifecycleChains, settings.LifecycleCommandChain{
		ID: "clone-template", Name: "初始化模板", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
		Commands: []settings.LifecycleCommandReference{{
			CommandID: settings.LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=https://example.com/template.git"},
		}},
	})
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	created, err := service.CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(
		"错误任务", "", task.DefaultColor, nil, map[string]any{"branch": true},
		map[task.LifecycleHook]string{task.LifecycleHookBeforeStart: "clone-template"},
	)
	if err != nil {
		t.Fatalf("CreateTaskWithExtraInfoTemplateFieldsAndLifecycleChains() error = %v", err)
	}
	if got := created.TemplateFields["branch"]; got != true {
		t.Fatalf("任务模板字段 = %#v，期望保留 true", created.TemplateFields)
	}
}

func taskExtraInfoParameterValue(information task.TaskExtraInfo, key string) string {
	for _, parameter := range information.Parameters {
		if parameter.Key == key {
			return parameter.Value
		}
	}
	return ""
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

func TestServiceDeletesPendingAndCompletedTaskRecords(t *testing.T) {
	service, repository, _ := newService(t)
	first, err := service.CreateTask("第一个已完成任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask(first) error = %v", err)
	}
	first, err = service.StartTask(first.ID)
	if err != nil {
		t.Fatalf("StartTask(first) error = %v", err)
	}
	if err := os.MkdirAll(first.WorkspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll(first workspace) error = %v", err)
	}
	if _, err := service.FinishTask(first.ID); err != nil {
		t.Fatalf("FinishTask(first) error = %v", err)
	}
	second, err := service.CreateTask("第二个已完成任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask(second) error = %v", err)
	}
	second, err = service.StartTask(second.ID)
	if err != nil {
		t.Fatalf("StartTask(second) error = %v", err)
	}
	if _, err := service.FinishTask(second.ID); err != nil {
		t.Fatalf("FinishTask(second) error = %v", err)
	}
	pending, err := service.CreateTask("待删除的未执行任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask(pending) error = %v", err)
	}
	pendingWorkspacePath := filepath.Join(t.TempDir(), "retained-pending-workspace")
	if err := os.MkdirAll(pendingWorkspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll(pending workspace) error = %v", err)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for index := range data.Tasks {
		if data.Tasks[index].ID == pending.ID {
			data.Tasks[index].WorkspacePath = pendingWorkspacePath
		}
	}
	if err := repository.SaveTaskSnapshot(data.Tasks); err != nil {
		t.Fatalf("SaveTaskSnapshot() error = %v", err)
	}

	deleter, ok := any(service).(interface {
		DeleteTasks(taskIDs []string) ([]task.Task, error)
	})
	if !ok {
		t.Fatal("Service 缺少 DeleteTasks()")
	}
	remaining, err := deleter.DeleteTasks([]string{first.ID, pending.ID})
	if err != nil {
		t.Fatalf("DeleteTasks() error = %v", err)
	}
	if got, want := taskIDs(remaining), []string{second.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("DeleteTasks() tasks = %#v, want %#v", got, want)
	}
	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := taskIDs(persisted.Tasks), []string{second.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("persisted tasks = %#v, want %#v", got, want)
	}
	if _, err := os.Stat(first.WorkspacePath); err != nil {
		t.Fatalf("DeleteTasks() removed completed workspace: %v", err)
	}
	if _, err := os.Stat(pendingWorkspacePath); err != nil {
		t.Fatalf("DeleteTasks() removed pending workspace: %v", err)
	}
}

func TestServiceRejectsInvalidTaskDeletionAtomically(t *testing.T) {
	service, repository, _ := newService(t)
	completed, err := service.CreateTask("可删除任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask(completed) error = %v", err)
	}
	completed, err = service.StartTask(completed.ID)
	if err != nil {
		t.Fatalf("StartTask(completed) error = %v", err)
	}
	if _, err := service.FinishTask(completed.ID); err != nil {
		t.Fatalf("FinishTask(completed) error = %v", err)
	}
	running, err := service.CreateTask("执行中任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask(running) error = %v", err)
	}
	running, err = service.StartTask(running.ID)
	if err != nil {
		t.Fatalf("StartTask(running) error = %v", err)
	}
	locked, err := service.CreateTask("待重试清理任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask(locked) error = %v", err)
	}
	locked, err = service.StartTask(locked.ID)
	if err != nil {
		t.Fatalf("StartTask(locked) error = %v", err)
	}
	if _, err := service.FinishTask(locked.ID); err != nil {
		t.Fatalf("FinishTask(locked) error = %v", err)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for index := range data.Tasks {
		if data.Tasks[index].ID == locked.ID {
			data.Tasks[index].LifecycleExecution = &task.LifecycleExecution{
				RunID:              "failed-post-end",
				Revision:           1,
				Hook:               task.LifecycleHookPostEnd,
				ChainID:            "cleanup-chain",
				CurrentIndex:       1,
				CommandCount:       1,
				CurrentCommandID:   "cleanup-command",
				CurrentCommandName: "清理工作目录",
				State:              task.LifecycleExecutionFailed,
			}
		}
	}
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	deleter, ok := any(service).(interface {
		DeleteTasks(taskIDs []string) ([]task.Task, error)
	})
	if !ok {
		t.Fatal("Service 缺少 DeleteTasks()")
	}
	for _, testCase := range []struct {
		name    string
		taskIDs []string
	}{
		{name: "empty", taskIDs: nil},
		{name: "duplicate", taskIDs: []string{completed.ID, completed.ID}},
		{name: "running", taskIDs: []string{completed.ID, running.ID}},
		{name: "locked", taskIDs: []string{completed.ID, locked.ID}},
		{name: "missing", taskIDs: []string{completed.ID, "missing-task"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			before, err := repository.Load()
			if err != nil {
				t.Fatalf("Load(before) error = %v", err)
			}
			if _, err := deleter.DeleteTasks(testCase.taskIDs); err == nil {
				t.Fatal("DeleteTasks() error = nil")
			}
			after, err := repository.Load()
			if err != nil {
				t.Fatalf("Load(after) error = %v", err)
			}
			if !reflect.DeepEqual(after.Tasks, before.Tasks) {
				t.Fatalf("DeleteTasks() changed tasks after invalid request: %#v", after.Tasks)
			}
		})
	}
}

func TestServiceDeletesFailedBeforeStartTaskWithOwnedWorkspace(t *testing.T) {
	service, repository, root := newService(t)
	created, err := service.CreateTask("启动失败任务", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	workspacePath := filepath.Join(root, created.ID)
	setFailedBeforeStartExecution(t, repository, created.ID, root, workspacePath, task.LifecycleWorkspaceCreated)
	if err := os.WriteFile(filepath.Join(workspacePath, "generated.txt"), []byte("generated"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	remaining, err := service.DeleteTasks([]string{created.ID})
	if err != nil {
		t.Fatalf("DeleteTasks() error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("DeleteTasks() tasks = %#v，期望为空", remaining)
	}
	if _, err := os.Stat(workspacePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("TaskAI 创建的目录仍存在: %v", err)
	}
}

func TestServiceDeletesFailedBeforeStartTaskWithoutTouchingUnownedWorkspace(t *testing.T) {
	service, repository, root := newService(t)
	created, err := service.CreateTask("复用目录后失败", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	workspacePath := filepath.Join(root, created.ID)
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	markerPath := filepath.Join(workspacePath, "preexisting.txt")
	if err := os.WriteFile(markerPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	setFailedBeforeStartExecution(t, repository, created.ID, root, workspacePath, task.LifecycleWorkspaceNotCreated)

	if _, err := service.DeleteTasks([]string{created.ID}); err != nil {
		t.Fatalf("DeleteTasks() error = %v", err)
	}
	contents, err := os.ReadFile(markerPath)
	if err != nil || string(contents) != "keep" {
		t.Fatalf("未归属目录被修改: contents=%q err=%v", contents, err)
	}
}

func TestServiceDeletesWorkspaceCreatedBeforeOwnershipStateUpdate(t *testing.T) {
	service, repository, root := newService(t)
	created, err := service.CreateTask("创建后崩溃", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	token, err := workspace.NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	workspacePath := filepath.Join(root, created.ID)
	if _, err := workspace.CreateOwned(root, created.ID, token); err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	setFailedBeforeStartExecutionWithToken(t, repository, created.ID, root, workspacePath, task.LifecycleWorkspaceNotCreated, token)

	remaining, err := service.DeleteTasks([]string{created.ID})
	if err != nil {
		t.Fatalf("DeleteTasks() error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("DeleteTasks() tasks = %#v，期望为空", remaining)
	}
	if _, err := os.Stat(workspacePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("创建后崩溃留下的目录仍存在: %v", err)
	}
}

func TestServiceRejectsDeletionAfterOwnedWorkspaceIsReplaced(t *testing.T) {
	service, repository, root := newService(t)
	created, err := service.CreateTask("目录被替换", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	workspacePath := filepath.Join(root, created.ID)
	setFailedBeforeStartExecution(t, repository, created.ID, root, workspacePath, task.LifecycleWorkspaceCreated)
	if err := os.Rename(workspacePath, filepath.Join(t.TempDir(), "original")); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	markerPath := filepath.Join(workspacePath, "keep.txt")
	if err := os.WriteFile(markerPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := service.DeleteTasks([]string{created.ID}); err == nil {
		t.Fatal("DeleteTasks() error = nil")
	}
	if contents, err := os.ReadFile(markerPath); err != nil || string(contents) != "keep" {
		t.Fatalf("替换目录被修改: contents=%q err=%v", contents, err)
	}
	if _, err := service.GetTask(created.ID); err != nil {
		t.Fatalf("清理拒绝后任务未保留: %v", err)
	}
}

func TestServicePreservesConcurrentTaskCreatedDuringWorkspaceCleanup(t *testing.T) {
	service, repository, root := newService(t)
	created, err := service.CreateTask("清理较慢", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	workspacePath := filepath.Join(root, created.ID)
	setFailedBeforeStartExecution(t, repository, created.ID, root, workspacePath, task.LifecycleWorkspaceCreated)
	entered := make(chan struct{})
	release := make(chan struct{})
	service.removeOwnedWorkspace = func(root, path, taskID, token string) (bool, error) {
		close(entered)
		<-release
		return workspace.RemoveOwned(root, path, taskID, token)
	}
	deleteResult := make(chan error, 1)
	go func() {
		_, deleteErr := service.DeleteTasks([]string{created.ID})
		deleteResult <- deleteErr
	}()
	<-entered
	type createResult struct {
		task task.Task
		err  error
	}
	createdResult := make(chan createResult, 1)
	go func() {
		concurrent, createErr := service.CreateTask("并发新增", "", task.DefaultColor)
		createdResult <- createResult{task: concurrent, err: createErr}
	}()
	close(release)
	if err := <-deleteResult; err != nil {
		t.Fatalf("DeleteTasks() error = %v", err)
	}
	result := <-createdResult
	if result.err != nil {
		t.Fatalf("并发 CreateTask() error = %v", result.err)
	}
	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := taskIDs(persisted.Tasks), []string{result.task.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("并发删除后的任务 = %#v，期望 %#v", got, want)
	}
}

func TestServiceKeepsTaskRecordWhenFinalDeleteSaveFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 权限模型不使用目录写权限触发保存失败")
	}
	root := filepath.Join(t.TempDir(), "workspaces")
	repositoryDirectory := t.TempDir()
	repository := storage.New(filepath.Join(repositoryDirectory, "state.json"), settings.Settings{
		WorkspaceRoot: root,
		TaskTreeWidth: settings.DefaultTaskTreeWidth,
	})
	service := New(repository, &closerStub{}, func() time.Time {
		return time.Date(2026, time.July, 22, 10, 0, 0, 0, time.UTC)
	})
	created, err := service.CreateTask("保存失败", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	workspacePath := filepath.Join(root, created.ID)
	setFailedBeforeStartExecution(t, repository, created.ID, root, workspacePath, task.LifecycleWorkspaceCreated)
	if err := os.Chmod(repositoryDirectory, 0o500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(repositoryDirectory, 0o700) })

	if _, err := service.DeleteTasks([]string{created.ID}); err == nil {
		t.Fatal("DeleteTasks() error = nil")
	}
	if err := os.Chmod(repositoryDirectory, 0o700); err != nil {
		t.Fatalf("restore Chmod() error = %v", err)
	}
	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(persisted.Tasks) != 1 || persisted.Tasks[0].ID != created.ID {
		t.Fatalf("保存失败后任务记录变化: %#v", persisted.Tasks)
	}
}

func TestServiceRejectsFailedBeforeStartDeletionWithUnknownOrUnsafeWorkspace(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		ownership task.LifecycleWorkspaceOwnership
		unsafe    bool
	}{
		{name: "unknown", ownership: task.LifecycleWorkspaceUnknown},
		{name: "unsafe", ownership: task.LifecycleWorkspaceCreated, unsafe: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			service, repository, root := newService(t)
			created, err := service.CreateTask("不可安全删除", "", task.DefaultColor)
			if err != nil {
				t.Fatalf("CreateTask() error = %v", err)
			}
			workspacePath := filepath.Join(root, created.ID)
			if testCase.unsafe {
				workspacePath = filepath.Join(t.TempDir(), created.ID)
				if err := os.MkdirAll(workspacePath, 0o700); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			}
			setFailedBeforeStartExecution(t, repository, created.ID, root, workspacePath, testCase.ownership)

			if _, err := service.DeleteTasks([]string{created.ID}); err == nil {
				t.Fatal("DeleteTasks() error = nil")
			}
			persisted, err := repository.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if len(persisted.Tasks) != 1 || persisted.Tasks[0].ID != created.ID || persisted.Tasks[0].LifecycleExecution == nil {
				t.Fatalf("拒绝删除后任务记录变化: %#v", persisted.Tasks)
			}
			if testCase.unsafe {
				if _, err := os.Stat(workspacePath); err != nil {
					t.Fatalf("危险路径内容被删除: %v", err)
				}
			}
		})
	}
}

func TestServiceDeletesFailedBeforeStartTaskWhenOwnedWorkspaceIsMissing(t *testing.T) {
	service, repository, root := newService(t)
	created, err := service.CreateTask("目录已不存在", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	setFailedBeforeStartExecution(t, repository, created.ID, root, filepath.Join(root, created.ID), task.LifecycleWorkspaceCreated)
	if err := os.RemoveAll(filepath.Join(root, created.ID)); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	remaining, err := service.DeleteTasks([]string{created.ID})
	if err != nil {
		t.Fatalf("DeleteTasks() error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("DeleteTasks() tasks = %#v，期望为空", remaining)
	}
}

func TestServiceKeepsAllRecordsWhenLaterWorkspaceCleanupFails(t *testing.T) {
	service, repository, root := newService(t)
	first, err := service.CreateTask("先清理目录", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask(first) error = %v", err)
	}
	second, err := service.CreateTask("后清理失败", "", task.DefaultColor)
	if err != nil {
		t.Fatalf("CreateTask(second) error = %v", err)
	}
	firstPath := filepath.Join(root, first.ID)
	unsafePath := filepath.Join(t.TempDir(), second.ID)
	if err := os.MkdirAll(unsafePath, 0o700); err != nil {
		t.Fatalf("MkdirAll(second) error = %v", err)
	}
	setFailedBeforeStartExecution(t, repository, first.ID, root, firstPath, task.LifecycleWorkspaceCreated)
	setFailedBeforeStartExecution(t, repository, second.ID, root, unsafePath, task.LifecycleWorkspaceCreated)

	if _, err := service.DeleteTasks([]string{first.ID, second.ID}); err == nil {
		t.Fatal("DeleteTasks() error = nil")
	}
	if _, err := os.Stat(firstPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("首个安全目录未清理: %v", err)
	}
	if _, err := os.Stat(unsafePath); err != nil {
		t.Fatalf("危险目录被清理: %v", err)
	}
	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := taskIDs(persisted.Tasks), []string{first.ID, second.ID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("清理失败后任务记录 = %#v，期望 %#v", got, want)
	}

	data := persisted
	for index := range data.Tasks {
		if data.Tasks[index].ID == second.ID {
			data.Tasks[index].LifecycleExecution.WorkspaceOwnership = task.LifecycleWorkspaceNotCreated
		}
	}
	if err := repository.SaveTaskSnapshot(data.Tasks); err != nil {
		t.Fatalf("SaveTaskSnapshot() error = %v", err)
	}
	if _, err := service.DeleteTasks([]string{first.ID, second.ID}); err != nil {
		t.Fatalf("修复归属后的 DeleteTasks() error = %v", err)
	}
}

func setFailedBeforeStartExecution(t *testing.T, repository *storage.Repository, taskID, workspaceRoot, workspacePath string, ownership task.LifecycleWorkspaceOwnership) {
	t.Helper()
	token, err := workspace.NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	if ownership == task.LifecycleWorkspaceCreated && workspacePath == filepath.Join(workspaceRoot, taskID) {
		if _, statErr := os.Lstat(workspacePath); errors.Is(statErr, os.ErrNotExist) {
			if _, createErr := workspace.CreateOwned(workspaceRoot, taskID, token); createErr != nil {
				t.Fatalf("CreateOwned() error = %v", createErr)
			}
		}
	}
	if ownership == task.LifecycleWorkspaceUnknown {
		token = ""
	}
	setFailedBeforeStartExecutionWithToken(t, repository, taskID, workspaceRoot, workspacePath, ownership, token)
}

func setFailedBeforeStartExecutionWithToken(t *testing.T, repository *storage.Repository, taskID, workspaceRoot, workspacePath string, ownership task.LifecycleWorkspaceOwnership, token string) {
	t.Helper()
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for index := range data.Tasks {
		if data.Tasks[index].ID != taskID {
			continue
		}
		data.Tasks[index].LifecycleExecution = &task.LifecycleExecution{
			RunID:              "failed-before-start",
			Revision:           2,
			Hook:               task.LifecycleHookBeforeStart,
			ChainID:            "prepare",
			CurrentIndex:       1,
			CommandCount:       1,
			CurrentCommandID:   "prepare-command",
			CurrentCommandName: "准备任务",
			State:              task.LifecycleExecutionFailed,
			Error:              "准备失败",
			WorkspaceRoot:      workspaceRoot,
			WorkspacePath:      workspacePath,
			WorkspaceOwnership: ownership,
			WorkspaceToken:     token,
		}
	}
	if err := repository.SaveTaskSnapshot(data.Tasks); err != nil {
		t.Fatalf("SaveTaskSnapshot() error = %v", err)
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
