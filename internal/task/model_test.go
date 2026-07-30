package task

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewTaskStartsPendingWithConfiguredColor(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 0, 0, 0, time.UTC)

	task, err := NewTask("  编写登录页  ", "实现邮箱登录表单", "#A1B2C3", now)

	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if task.ID == "" {
		t.Fatal("NewTask() ID is empty")
	}
	if task.Title != "编写登录页" {
		t.Errorf("NewTask() Title = %q, want %q", task.Title, "编写登录页")
	}
	if task.Description != "实现邮箱登录表单" {
		t.Errorf("NewTask() Description = %q, want %q", task.Description, "实现邮箱登录表单")
	}
	if task.Status != StatusPending {
		t.Errorf("NewTask() Status = %q, want %q", task.Status, StatusPending)
	}
	if task.Color != "#a1b2c3" {
		t.Errorf("NewTask() Color = %q, want %q", task.Color, "#a1b2c3")
	}
	if !task.CreatedAt.Equal(now) {
		t.Errorf("NewTask() CreatedAt = %v, want %v", task.CreatedAt, now)
	}
	if task.WorkspacePath != "" {
		t.Errorf("NewTask() WorkspacePath = %q, want empty", task.WorkspacePath)
	}
	if task.LifecycleChains == nil {
		t.Fatal("NewTask() LifecycleChains = nil, want empty map")
	}
}

func TestTaskPersistsShelvedFlagAndDefaultsLegacyData(t *testing.T) {
	var legacy Task
	if err := json.Unmarshal([]byte(`{"id":"task-1","status":"running"}`), &legacy); err != nil {
		t.Fatalf("Unmarshal legacy task error = %v", err)
	}
	shelved := reflect.ValueOf(legacy).FieldByName("Shelved")
	if !shelved.IsValid() {
		t.Fatal("Task 缺少 Shelved 字段")
	}
	if shelved.Bool() {
		t.Fatal("旧任务 Shelved = true，期望 false")
	}

	encoded, err := json.Marshal(Task{ID: "task-2", Status: StatusRunning})
	if err != nil {
		t.Fatalf("Marshal task error = %v", err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("Unmarshal payload error = %v", err)
	}
	if string(payload["shelved"]) != "false" {
		t.Fatalf("任务 JSON shelved = %s，期望 false", payload["shelved"])
	}
}

func TestTaskLifecycleChainsAndExecutionNormalizeForPersistence(t *testing.T) {
	chains, err := NormalizeLifecycleChains(map[LifecycleHook]string{
		LifecycleHookBeforeStart: " prepare ",
		LifecycleHookPostEnd:     "",
	})
	if err != nil {
		t.Fatalf("NormalizeLifecycleChains() error = %v", err)
	}
	if got := chains[LifecycleHookBeforeStart]; got != "prepare" {
		t.Fatalf("beforeStart 链 = %q，期望 prepare", got)
	}
	if _, found := chains[LifecycleHookPostEnd]; found {
		t.Fatalf("空链选择未被移除: %#v", chains)
	}

	execution, err := NormalizeLifecycleExecution(&LifecycleExecution{
		Hook:               LifecycleHookPostStart,
		ChainID:            " chain-1 ",
		RunID:              " run-7 ",
		Revision:           2,
		CurrentCommandID:   " command-2 ",
		CurrentCommandName: " 初始化仓库 ",
		CurrentIndex:       2,
		CommandCount:       3,
		State:              LifecycleExecutionFailed,
		Error:              " 命令退出码为 1 ",
	})
	if err != nil {
		t.Fatalf("NormalizeLifecycleExecution() error = %v", err)
	}
	if execution.ChainID != "chain-1" || execution.RunID != "run-7" || execution.Revision != 2 || execution.CurrentCommandID != "command-2" || execution.CurrentCommandName != "初始化仓库" || execution.Error != "命令退出码为 1" {
		t.Fatalf("执行记录未规范化: %#v", execution)
	}
	if execution.CurrentIndex != 2 || execution.CommandCount != 3 || !(Task{LifecycleExecution: execution}).IsLifecycleLocked() {
		t.Fatalf("执行记录或锁定状态错误: %#v", execution)
	}

	encoded, err := json.Marshal(Task{
		ID:                 "task-1",
		LifecycleChains:    chains,
		LifecycleExecution: execution,
	})
	if err != nil {
		t.Fatalf("Marshal task error = %v", err)
	}
	if !strings.Contains(string(encoded), `"lifecycleChains":{"beforeStart":"prepare"}`) || !strings.Contains(string(encoded), `"runId":"run-7"`) || !strings.Contains(string(encoded), `"revision":2`) {
		t.Fatalf("任务生命周期字段未持久化: %s", encoded)
	}

	legacy, err := NormalizeLifecycleExecution(&LifecycleExecution{
		Hook:               LifecycleHookPostStart,
		ChainID:            "chain-1",
		CurrentCommandID:   "command-1",
		CurrentCommandName: "旧命令",
		CurrentIndex:       1,
		CommandCount:       1,
		State:              LifecycleExecutionFailed,
	})
	if err != nil || legacy.RunID != "" || legacy.Revision != 0 {
		t.Fatalf("旧版执行记录兼容失败: execution=%#v, err=%v", legacy, err)
	}
}

func TestTaskLifecycleValidationRejectsUnknownHookAndInvalidProgress(t *testing.T) {
	if _, err := NormalizeLifecycleChains(map[LifecycleHook]string{"unknown": "chain-1"}); err == nil {
		t.Fatal("NormalizeLifecycleChains() error = nil，期望拒绝未知钩子")
	}
	if _, err := NormalizeLifecycleExecution(&LifecycleExecution{
		Hook:         LifecycleHookBeforeEnd,
		ChainID:      "chain-1",
		CurrentIndex: 2,
		CommandCount: 1,
		State:        LifecycleExecutionRunning,
	}); err == nil {
		t.Fatal("NormalizeLifecycleExecution() error = nil，期望拒绝超出范围的执行进度")
	}
}

func TestNewTaskRejectsBlankTitle(t *testing.T) {
	_, err := NewTask(" \t\n ", "", "#4f46e5", time.Now())

	if err == nil {
		t.Fatal("NewTask() error = nil, want title validation error")
	}
}

func TestNewTaskRejectsInvalidColor(t *testing.T) {
	_, err := NewTask("编写登录页", "", "red", time.Now())

	if err == nil {
		t.Fatal("NewTask() error = nil, want invalid color error")
	}
}

func TestNewExtraInfoTemplateValidatesFieldsAndBuildsTaskSnapshot(t *testing.T) {
	template, err := NewExtraInfoTemplate(" git ", " API 仓库 ", []ExtraInfoField{{Key: " repository ", DisplayName: " 仓库地址 ", Value: " git@example.com:team/api.git "}}, []ExtraInfoParameterDefinition{
		{Key: "branch", DisplayName: "分支", Required: true},
		{Key: "environment", DisplayName: "环境", Required: false},
	})
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}
	if template.ID == "" || template.Catalogue != "git" || template.DisplayName != "API 仓库" || len(template.Fields) != 2 || template.Fields[0].Key != "name" || template.Fields[1].Key != "repository" || template.Fields[1].DefaultValue != "git@example.com:team/api.git" {
		t.Fatalf("模板规范化结果 = %#v", template)
	}

}

func TestExtraInfoTemplateRejectsDuplicateOrConflictingParameterKeys(t *testing.T) {
	_, err := NewExtraInfoTemplate("git", "API 仓库", []ExtraInfoField{{Key: "repository", DisplayName: "仓库", Value: "git@example.com:team/api.git"}}, []ExtraInfoParameterDefinition{
		{Key: "repository", DisplayName: "重复", Required: false},
	})
	if err == nil {
		t.Fatal("NewExtraInfoTemplate() error = nil, want parameter key conflict")
	}

	first, err := NewExtraInfoTemplate("git", "API 仓库", []ExtraInfoField{{Key: "repository", DisplayName: "仓库", Value: "git@example.com:team/api.git"}}, nil)
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() first error = %v", err)
	}
	second, err := NewExtraInfoTemplate("git", "API 仓库", []ExtraInfoField{{Key: "repository", DisplayName: "仓库", Value: "git@example.com:team/web.git"}}, nil)
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() second error = %v", err)
	}
	if _, err := ValidateExtraInfoTemplates([]ExtraInfoTemplate{first, second}); err == nil {
		t.Fatal("ValidateExtraInfoTemplates() error = nil, want duplicate display name error")
	}
}

func TestExtraInfoTemplateSupportsMultipleFieldsAndMigratesLegacyField(t *testing.T) {
	template, err := NewExtraInfoTemplate("git", "API 仓库", []ExtraInfoField{
		{Key: "repository", DisplayName: "仓库地址", Value: "git@example.com:team/api.git"},
		{Key: "remote", DisplayName: "远程名称", Value: "origin"},
	}, []ExtraInfoParameterDefinition{{Key: "branch", DisplayName: "分支", Required: true}})
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}
	if len(template.Fields) != 3 || template.Fields[2].Key != "remote" {
		t.Fatalf("模板固定字段 = %#v，期望三个字段", template.Fields)
	}

	legacy, err := NormalizeExtraInfoTemplate(ExtraInfoTemplate{
		ID: "legacy-template", Catalogue: "git", DisplayName: "旧仓库",
		Key: "repository", KeyDisplayName: "仓库地址", Value: "git@example.com:team/legacy.git",
	})
	if err != nil {
		t.Fatalf("NormalizeExtraInfoTemplate() legacy error = %v", err)
	}
	if len(legacy.Fields) != 2 || legacy.Fields[1].Key != "repository" || legacy.Fields[1].DefaultValue != "git@example.com:team/legacy.git" {
		t.Fatalf("旧模板迁移结果 = %#v", legacy)
	}
}

func TestNewExtraInfoTemplateAddsNameFieldAndAllowsBlankDefaults(t *testing.T) {
	template, err := NewExtraInfoTemplate("issue", "缺陷", []ExtraInfoField{{
		Key:         "url",
		DisplayName: "缺陷地址",
	}}, nil)
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}

	if len(template.Fields) != 2 {
		t.Fatalf("模板固定字段数量 = %d，期望 2", len(template.Fields))
	}
	if template.Fields[0].Key != "name" || template.Fields[0].DisplayName != "名称" || template.Fields[0].Value != "" {
		t.Fatalf("默认名称字段 = %#v", template.Fields[0])
	}
	if template.Fields[1].Key != "url" || template.Fields[1].Value != "" {
		t.Fatalf("默认字段 = %#v", template.Fields[1])
	}
}

func TestNewExtraInfoBuildsReusableInformationFromFixedValues(t *testing.T) {
	template, err := NewExtraInfoTemplate("git", "Git", []ExtraInfoField{
		{Key: "name", DisplayName: "项目名称"},
		{Key: "repository", DisplayName: "仓库地址", Value: "git@example.com:team/default.git"},
	}, nil)
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}

	information, err := NewExtraInfo(template, map[string]string{
		"name": " API 服务 ",
	})
	if err != nil {
		t.Fatalf("NewExtraInfo() error = %v", err)
	}

	if information.ID == template.ID {
		t.Fatal("NewExtraInfo() reused template ID, want an independent information ID")
	}
	if got := extraInfoFieldValue(information.Fields, "name"); got != "API 服务" {
		t.Fatalf("信息名称 = %q，期望 %q", got, "API 服务")
	}
	if got := extraInfoFieldValue(information.Fields, "repository"); got != "git@example.com:team/default.git" {
		t.Fatalf("信息仓库地址 = %q，期望模板默认值", got)
	}
}

func TestNewTaskExtraInfoCopiesFixedValuesAndAllowsTaskParameters(t *testing.T) {
	template, err := NewExtraInfoTemplate("git", "Git", []ExtraInfoField{
		{Key: "name", DisplayName: "项目名称"},
		{Key: "repository", DisplayName: "仓库地址"},
	}, []ExtraInfoParameterDefinition{{Key: "branch", DisplayName: "仓库分支", Required: true}})
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}
	information, err := NewExtraInfo(template, map[string]string{
		"name":       "API 服务",
		"repository": "git@example.com:team/api.git",
	})
	if err != nil {
		t.Fatalf("NewExtraInfo() error = %v", err)
	}

	snapshot, err := NewTaskExtraInfo(information, template, map[string]string{"branch": " main "}, []ExtraInfoParameter{{
		Key:         "environment",
		DisplayName: "环境",
		Value:       "production",
	}})
	if err != nil {
		t.Fatalf("NewTaskExtraInfo() error = %v", err)
	}
	if snapshot.InformationID != information.ID || snapshot.TemplateID != template.ID {
		t.Fatalf("任务快照来源 = %#v", snapshot)
	}
	if got := extraInfoFieldValue(snapshot.Fields, "repository"); got != "git@example.com:team/api.git" {
		t.Fatalf("任务固定字段 = %q，期望信息固定值", got)
	}
	if len(snapshot.Parameters) != 2 || snapshot.Parameters[0].Value != "main" || snapshot.Parameters[1].Key != "environment" {
		t.Fatalf("任务动态参数 = %#v", snapshot.Parameters)
	}

	_, err = NewTaskExtraInfo(information, template, map[string]string{"branch": "main"}, []ExtraInfoParameter{{
		Key:         "repository",
		DisplayName: "重复键",
		Value:       "ignored",
	}})
	if err == nil {
		t.Fatal("NewTaskExtraInfo() error = nil，期望任务级参数键冲突错误")
	}
}

func TestInformationParametersProvideTaskDefaultsAndRejectTemplateKeyConflicts(t *testing.T) {
	template, err := NewExtraInfoTemplate("git", "Git", []ExtraInfoField{
		{Key: "name", DisplayName: "项目名称"},
		{Key: "repository", DisplayName: "仓库地址"},
	}, []ExtraInfoParameterDefinition{{Key: "branch", DisplayName: "仓库分支", Required: true}})
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}

	information, err := NewExtraInfoWithParameters(template, map[string]string{
		"name":       "API 服务",
		"repository": "git@example.com:team/api.git",
	}, []ExtraInfoParameter{{Key: "environment", DisplayName: "环境", Required: true, Value: " production "}})
	if err != nil {
		t.Fatalf("NewExtraInfoWithParameters() error = %v", err)
	}
	if len(information.Parameters) != 1 || information.Parameters[0].Value != "production" {
		t.Fatalf("信息动态参数 = %#v", information.Parameters)
	}

	snapshot, err := NewTaskExtraInfo(information, template, map[string]string{"branch": "main"}, nil)
	if err != nil {
		t.Fatalf("NewTaskExtraInfo() error = %v", err)
	}
	if len(snapshot.Parameters) != 2 || snapshot.Parameters[1].Key != "environment" || snapshot.Parameters[1].Value != "production" {
		t.Fatalf("任务快照未带入信息级参数默认值: %#v", snapshot.Parameters)
	}

	_, err = NewExtraInfoWithParameters(template, map[string]string{
		"name":       "冲突服务",
		"repository": "git@example.com:team/conflict.git",
	}, []ExtraInfoParameter{{Key: "branch", DisplayName: "环境", Value: "production"}})
	if err == nil {
		t.Fatal("NewExtraInfoWithParameters() error = nil，期望信息参数与模板参数冲突失败")
	}
}

func TestCheckboxParametersNormalizeToBooleanNonRequiredValues(t *testing.T) {
	var template ExtraInfoTemplate
	if err := json.Unmarshal([]byte(`{
		"id":"deploy-template",
		"catalogue":"deploy",
		"fields":[{"key":"name","displayName":"名称"}],
		"parameters":[{"key":"deploy","displayName":"允许部署","required":true,"inputType":"checkbox"}]
	}`), &template); err != nil {
		t.Fatalf("Unmarshal template error = %v", err)
	}

	normalizedTemplate, err := NormalizeExtraInfoTemplate(template)
	if err != nil {
		t.Fatalf("NormalizeExtraInfoTemplate() error = %v", err)
	}
	if normalizedTemplate.Parameters[0].Required {
		t.Fatal("复选框参数仍标记为必填")
	}
	encodedTemplate, err := json.Marshal(normalizedTemplate)
	if err != nil {
		t.Fatalf("Marshal template error = %v", err)
	}
	if !strings.Contains(string(encodedTemplate), `"inputType":"checkbox"`) {
		t.Fatalf("模板未保存复选框类型: %s", encodedTemplate)
	}

	information, err := NewExtraInfo(normalizedTemplate, map[string]string{"name": "部署服务"})
	if err != nil {
		t.Fatalf("NewExtraInfo() error = %v", err)
	}
	snapshot, err := NewTaskExtraInfo(information, normalizedTemplate, nil, nil)
	if err != nil {
		t.Fatalf("NewTaskExtraInfo() error = %v", err)
	}
	if len(snapshot.Parameters) != 1 || snapshot.Parameters[0].Value != "false" || snapshot.Parameters[0].Required {
		t.Fatalf("任务复选框默认值 = %#v，期望 false 且非必填", snapshot.Parameters)
	}
	encodedSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal snapshot error = %v", err)
	}
	if !strings.Contains(string(encodedSnapshot), `"inputType":"checkbox"`) {
		t.Fatalf("任务快照未保存复选框类型: %s", encodedSnapshot)
	}

	var invalid TaskExtraInfo
	if err := json.Unmarshal([]byte(`{
		"id":"deploy-snapshot",
		"catalogue":"deploy",
		"fields":[{"key":"name","displayName":"名称","value":"部署服务"}],
		"parameters":[{"key":"deploy","displayName":"允许部署","inputType":"checkbox","value":"yes"}]
	}`), &invalid); err != nil {
		t.Fatalf("Unmarshal snapshot error = %v", err)
	}
	if _, err := NormalizeTaskExtraInfo(invalid); err == nil {
		t.Fatal("NormalizeTaskExtraInfo() error = nil，期望拒绝非法复选框值")
	}
}

func TestBuiltInGitTemplateProtectsBuiltInKeysAndLabels(t *testing.T) {
	gitTemplate := ExtraInfoTemplate{
		ID:        "builtin-extra-info-template-git",
		Catalogue: "git",
		BuiltIn:   true,
		Fields: []ExtraInfoField{
			{Key: "name", DisplayName: "项目名称", DefaultValue: "默认项目"},
			{Key: "repository", DisplayName: "仓库地址", DefaultValue: "git@example.com:team/default.git"},
			{Key: "remote", DisplayName: "远程名称"},
		},
		Parameters: []ExtraInfoParameterDefinition{
			{Key: "branch", DisplayName: "仓库分支", Required: true},
			{Key: "environment", DisplayName: "环境"},
		},
	}
	if _, err := NormalizeExtraInfoTemplate(gitTemplate); err != nil {
		t.Fatalf("NormalizeExtraInfoTemplate() error = %v", err)
	}

	gitTemplate.Fields[1].DisplayName = "地址"
	if _, err := NormalizeExtraInfoTemplate(gitTemplate); err == nil {
		t.Fatal("NormalizeExtraInfoTemplate() error = nil，期望 Git 内置字段修改被拒绝")
	}

	gitTemplate.Fields[1].DisplayName = "仓库地址"
	gitTemplate.Parameters = gitTemplate.Parameters[1:]
	if _, err := NormalizeExtraInfoTemplate(gitTemplate); err == nil {
		t.Fatal("NormalizeExtraInfoTemplate() error = nil，期望 Git 内置参数删除被拒绝")
	}
}

func extraInfoFieldValue(fields []ExtraInfoField, key string) string {
	for _, field := range fields {
		if field.Key == key {
			return field.Value
		}
	}
	return ""
}
