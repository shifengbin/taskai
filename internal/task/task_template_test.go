package task

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormalizeTaskTemplateNormalizesTypedFields(t *testing.T) {
	template, err := NormalizeTaskTemplate(TaskTemplate{
		ID:   " template-release ",
		Name: " 发布任务 ",
		Fields: []TaskTemplateField{
			{Key: " environment ", DisplayName: " 环境 ", InputType: TaskTemplateFieldInputString, DefaultValue: "production", InjectEnvironment: true},
			{Key: " deploy ", DisplayName: " 允许部署 ", InputType: TaskTemplateFieldInputBool, DefaultValue: false, Required: true},
		},
	})
	if err != nil {
		t.Fatalf("NormalizeTaskTemplate() error = %v", err)
	}
	if template.ID != "template-release" || template.Name != "发布任务" || len(template.Fields) != 2 {
		t.Fatalf("模板规范化结果 = %#v", template)
	}
	if template.Fields[0].Key != "environment" || template.Fields[0].DefaultValue != "production" || !template.Fields[0].InjectEnvironment {
		t.Fatalf("字符串字段规范化结果 = %#v", template.Fields[0])
	}
	if template.Fields[1].InputType != TaskTemplateFieldInputBool || template.Fields[1].DefaultValue != false || !template.Fields[1].Required {
		t.Fatalf("布尔字段规范化结果 = %#v", template.Fields[1])
	}
}

func TestNormalizeTaskTemplateRejectsInvalidFieldDefinitions(t *testing.T) {
	base := TaskTemplate{ID: "template-release", Name: "发布任务", Fields: []TaskTemplateField{{
		Key: "environment", DisplayName: "环境", InputType: TaskTemplateFieldInputString, DefaultValue: "production",
	}}}
	for _, test := range []struct {
		name   string
		mutate func(*TaskTemplate)
	}{
		{name: "invalid key", mutate: func(template *TaskTemplate) { template.Fields[0].Key = "deploy-env" }},
		{name: "reserved key", mutate: func(template *TaskTemplate) { template.Fields[0].Key = "task_id" }},
		{name: "wrong default type", mutate: func(template *TaskTemplate) {
			template.Fields[0].InputType, template.Fields[0].DefaultValue = TaskTemplateFieldInputBool, "false"
		}},
		{name: "case insensitive duplicate", mutate: func(template *TaskTemplate) {
			template.Fields = append(template.Fields, TaskTemplateField{Key: "ENVIRONMENT", DisplayName: "重复", InputType: TaskTemplateFieldInputString, DefaultValue: ""})
		}},
		{name: "directory default", mutate: func(template *TaskTemplate) {
			template.Fields[0].InputType, template.Fields[0].DefaultValue = TaskTemplateFieldInputDirectories, []string{}
		}},
		{name: "directory environment", mutate: func(template *TaskTemplate) {
			template.Fields[0].InputType, template.Fields[0].DefaultValue, template.Fields[0].InjectEnvironment = TaskTemplateFieldInputDirectories, nil, true
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			template := base
			template.Fields = append([]TaskTemplateField(nil), base.Fields...)
			test.mutate(&template)
			if _, err := NormalizeTaskTemplate(template); err == nil {
				t.Fatal("NormalizeTaskTemplate() error = nil，期望拒绝无效字段定义")
			}
		})
	}
}

func TestTaskTemplateFieldsApplyDefaultsAndRetainHistoricalValues(t *testing.T) {
	template := TaskTemplate{ID: "template-release", Name: "发布任务", Fields: []TaskTemplateField{
		{Key: "environment", DisplayName: "环境", InputType: TaskTemplateFieldInputString, DefaultValue: "production", Required: true},
		{Key: "deploy", DisplayName: "允许部署", InputType: TaskTemplateFieldInputBool, DefaultValue: false, Required: true},
	}}
	existing := map[string]any{"environment": "staging", "retired_field": "preserve me"}

	visible, err := ResolveTaskTemplateFields(template, existing)
	if err != nil {
		t.Fatalf("ResolveTaskTemplateFields() error = %v", err)
	}
	if want := map[string]any{"environment": "staging", "deploy": false}; !reflect.DeepEqual(visible, want) {
		t.Fatalf("可见模板字段 = %#v，期望 %#v", visible, want)
	}

	merged, err := MergeTaskTemplateFields(template, existing, map[string]any{"environment": "production", "deploy": true})
	if err != nil {
		t.Fatalf("MergeTaskTemplateFields() error = %v", err)
	}
	if want := map[string]any{"environment": "production", "deploy": true, "retired_field": "preserve me"}; !reflect.DeepEqual(merged, want) {
		t.Fatalf("保存后的模板字段 = %#v，期望 %#v", merged, want)
	}
}

func TestMergeTaskTemplateFieldsRequiresTextAndTrueBoolean(t *testing.T) {
	template := TaskTemplate{ID: "template-release", Name: "发布任务", Fields: []TaskTemplateField{
		{Key: "environment", DisplayName: "环境", InputType: TaskTemplateFieldInputString, Required: true, DefaultValue: ""},
		{Key: "deploy", DisplayName: "允许部署", InputType: TaskTemplateFieldInputBool, Required: true, DefaultValue: false},
	}}

	if _, err := MergeTaskTemplateFields(template, nil, map[string]any{"environment": " ", "deploy": true}); err == nil {
		t.Fatal("MergeTaskTemplateFields() error = nil，期望拒绝空白必填字符串")
	}
	if _, err := MergeTaskTemplateFields(template, nil, map[string]any{"environment": "production", "deploy": false}); err == nil {
		t.Fatal("MergeTaskTemplateFields() error = nil，期望拒绝未勾选的必填布尔字段")
	}
}

func TestValidateTaskTemplateUpdateRejectsChangingUsedFieldType(t *testing.T) {
	previous := TaskTemplate{ID: "template-release", Name: "发布任务", Fields: []TaskTemplateField{{
		Key: "deploy", DisplayName: "允许部署", InputType: TaskTemplateFieldInputBool, DefaultValue: false,
	}}}
	next := TaskTemplate{ID: "template-release", Name: "发布任务", Fields: []TaskTemplateField{{
		Key: "deploy", DisplayName: "允许部署", InputType: TaskTemplateFieldInputString, DefaultValue: "false",
	}}}

	if err := ValidateTaskTemplateUpdate(previous, next, []map[string]any{{"deploy": false}}); err == nil {
		t.Fatal("ValidateTaskTemplateUpdate() error = nil，期望拒绝修改已使用字段类型")
	}
	if err := ValidateTaskTemplateUpdate(previous, next, nil); err != nil {
		t.Fatalf("ValidateTaskTemplateUpdate() 未使用字段时 error = %v", err)
	}
}

func TestTaskTemplateEnvironmentIncludesOptedInFields(t *testing.T) {
	template := TaskTemplate{ID: "template-release", Name: "发布任务", Fields: []TaskTemplateField{
		{Key: "environment", DisplayName: "环境", InputType: TaskTemplateFieldInputString, DefaultValue: "", InjectEnvironment: true},
		{Key: "deploy", DisplayName: "立即部署", InputType: TaskTemplateFieldInputBool, DefaultValue: false, InjectEnvironment: true},
		{Key: "internalNote", DisplayName: "内部备注", InputType: TaskTemplateFieldInputString, DefaultValue: "secret"},
	}}

	environment, err := TaskTemplateEnvironment(&template, map[string]any{"environment": "", "deploy": true})
	if err != nil {
		t.Fatalf("TaskTemplateEnvironment() error = %v", err)
	}
	if want := []string{"TASKAI_ENVIRONMENT=", "TASKAI_DEPLOY=true"}; !reflect.DeepEqual(environment, want) {
		t.Fatalf("模板环境变量 = %#v，期望 %#v", environment, want)
	}
}

func TestNormalizeTaskTemplateSupportsDirectoriesAndLegacyUpdatable(t *testing.T) {
	locked := false
	template, err := NormalizeTaskTemplate(TaskTemplate{
		ID:   "template-directories",
		Name: "目录模板",
		Fields: []TaskTemplateField{
			{Key: "source", DisplayName: "单目录", InputType: TaskTemplateFieldInputDirectories},
			{Key: "sources", DisplayName: "多目录", InputType: TaskTemplateFieldInputDirectories, Multiple: true, Updatable: &locked},
		},
	})
	if err != nil {
		t.Fatalf("NormalizeTaskTemplate() error = %v", err)
	}
	if template.Fields[0].InputType != TaskTemplateFieldInputDirectories || template.Fields[0].Multiple {
		t.Fatalf("单目录字段规范化结果 = %#v", template.Fields[0])
	}
	if template.Fields[0].Updatable == nil || !*template.Fields[0].Updatable {
		t.Fatalf("缺失 updatable 的旧字段应归一化为可更新: %#v", template.Fields[0])
	}
	if template.Fields[1].Updatable == nil || *template.Fields[1].Updatable {
		t.Fatalf("显式锁定字段被错误归一化: %#v", template.Fields[1])
	}
	if template.Fields[0].DefaultValue != nil || template.Fields[0].InjectEnvironment {
		t.Fatalf("目录字段不应有默认值或环境变量注入: %#v", template.Fields[0])
	}
}

func TestTaskTemplateValuesSupportDirectoryArraysAndRequiredValidation(t *testing.T) {
	projectA := t.TempDir()
	projectB := t.TempDir()
	template := TaskTemplate{ID: "template-directories", Name: "目录模板", Fields: []TaskTemplateField{
		{Key: "sources", DisplayName: "来源目录", InputType: TaskTemplateFieldInputDirectories, Multiple: true, Required: true},
	}}

	values, err := MergeTaskTemplateFields(template, nil, map[string]any{"sources": []string{projectA, projectB}})
	if err != nil {
		t.Fatalf("MergeTaskTemplateFields() error = %v", err)
	}
	if want := []string{projectA, projectB}; !reflect.DeepEqual(values["sources"], want) {
		t.Fatalf("目录数组 = %#v，期望 %#v", values["sources"], want)
	}
	if _, err := MergeTaskTemplateFields(template, nil, map[string]any{"sources": []string{}}); err == nil {
		t.Fatal("空的必填目录数组应被拒绝")
	}
}

func TestValidateTaskTemplateUpdateRejectsDirectoryShapeAndImpossibleLock(t *testing.T) {
	updatable := true
	locked := false
	previous := TaskTemplate{ID: "template-directories", Name: "目录模板", Fields: []TaskTemplateField{
		{Key: "sources", DisplayName: "来源目录", InputType: TaskTemplateFieldInputDirectories, Multiple: true, Updatable: &updatable},
	}}
	single := previous
	single.Fields = []TaskTemplateField{{Key: "sources", DisplayName: "来源目录", InputType: TaskTemplateFieldInputDirectories, Updatable: &updatable}}
	if err := ValidateTaskTemplateUpdate(previous, single, []map[string]any{{"sources": []string{"/a", "/b"}}}); err == nil {
		t.Fatal("已有多值目录改为单目录应被拒绝")
	}

	next := previous
	next.Fields = []TaskTemplateField{{Key: "sources", DisplayName: "来源目录", InputType: TaskTemplateFieldInputDirectories, Multiple: true, Required: true, Updatable: &locked}}
	if err := ValidateTaskTemplateUpdate(previous, next, []map[string]any{{}}); err == nil {
		t.Fatal("必填且锁定的空目录字段应被拒绝")
	}
}

func TestUpdateTemplateFieldsRejectsLockedDirectoryChanges(t *testing.T) {
	locked := false
	projectA := t.TempDir()
	projectB := t.TempDir()
	template := TaskTemplate{ID: "template-directories", Name: "目录模板", Fields: []TaskTemplateField{{
		Key: "source", DisplayName: "来源目录", InputType: TaskTemplateFieldInputDirectories, Updatable: &locked,
	}}}
	for _, status := range []Status{StatusPending, StatusRunning, StatusCompleted} {
		t.Run(string(status), func(t *testing.T) {
			current := Task{Status: status, TemplateFields: map[string]any{"source": []string{projectA}}}
			if _, err := current.UpdateTemplateFields(&template, map[string]any{"source": []string{projectB}}); err == nil {
				t.Fatal("不可更新目录字段被替换时应被拒绝")
			}
		})
	}
}

func TestInitializeTemplateFieldsAllowsLockedDirectoryThenLocksIt(t *testing.T) {
	locked := false
	projectA := t.TempDir()
	projectB := t.TempDir()
	template := TaskTemplate{ID: "template-directories", Name: "目录模板", Fields: []TaskTemplateField{{
		Key: "source", DisplayName: "来源目录", InputType: TaskTemplateFieldInputDirectories, Required: true, Updatable: &locked,
	}}}
	created, err := (Task{}).InitializeTemplateFields(&template, map[string]any{"source": []string{projectA}})
	if err != nil {
		t.Fatalf("InitializeTemplateFields() error = %v", err)
	}
	if _, err := created.UpdateTemplateFields(&template, map[string]any{"source": []string{projectB}}); err == nil {
		t.Fatal("创建完成后的不可更新目录字段应立即锁定")
	}
}

func TestMergeTaskTemplateFieldsRejectsMultipleValuesForSingleDirectory(t *testing.T) {
	template := TaskTemplate{ID: "template-directories", Name: "目录模板", Fields: []TaskTemplateField{{
		Key: "source", DisplayName: "来源目录", InputType: TaskTemplateFieldInputDirectories,
	}}}
	if _, err := MergeTaskTemplateFields(template, nil, map[string]any{"source": []string{t.TempDir(), t.TempDir()}}); err == nil {
		t.Fatal("单目录字段的多个目录值应被拒绝")
	}
}

func TestMergeTaskTemplateFieldsRejectsInvalidDirectorySelections(t *testing.T) {
	root := t.TempDir()
	valid := filepath.Join(root, "src")
	if err := os.Mkdir(valid, 0o755); err != nil {
		t.Fatal(err)
	}
	regularFile := filepath.Join(root, "README.md")
	if err := os.WriteFile(regularFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	template := TaskTemplate{ID: "template-directories", Name: "目录模板", Fields: []TaskTemplateField{
		{Key: "sources", DisplayName: "来源目录", InputType: TaskTemplateFieldInputDirectories, Multiple: true, Required: true},
	}}
	for name, value := range map[string]any{
		"relative":  []string{"relative/path"},
		"missing":   []string{filepath.Join(root, "missing")},
		"file":      []string{regularFile},
		"duplicate": []string{valid, filepath.Join(valid, ".")},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := MergeTaskTemplateFields(template, nil, map[string]any{"sources": value}); err == nil {
				t.Fatalf("目录选择 %q 应被拒绝", name)
			}
		})
	}
}
