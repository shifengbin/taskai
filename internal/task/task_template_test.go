package task

import (
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
