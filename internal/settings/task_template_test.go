package settings

import (
	"testing"

	"taskai/internal/task"
)

func TestValidateNormalizesTaskTemplatesAndActiveSelection(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot:        t.TempDir(),
		TaskTreeWidth:        DefaultTaskTreeWidth,
		ActiveTaskTemplateID: " template-release ",
		TaskTemplates: []task.TaskTemplate{{
			ID: " template-release ", Name: " 发布任务 ", Fields: []task.TaskTemplateField{{
				Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, DefaultValue: "production",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated.ActiveTaskTemplateID != "template-release" || len(validated.TaskTemplates) != 1 || validated.TaskTemplates[0].Name != "发布任务" {
		t.Fatalf("任务模板设置规范化结果 = %#v", validated)
	}
}

func TestValidateRejectsUnknownActiveTaskTemplateAndDuplicateNames(t *testing.T) {
	base := Settings{
		WorkspaceRoot: t.TempDir(), TaskTreeWidth: DefaultTaskTreeWidth,
		TaskTemplates: []task.TaskTemplate{{ID: "release", Name: "发布", Fields: []task.TaskTemplateField{{
			Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, DefaultValue: "production",
		}}}},
	}

	unknown := base
	unknown.ActiveTaskTemplateID = "missing"
	if _, err := Validate(unknown); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝不存在的当前任务模板")
	}

	duplicate := base
	duplicate.TaskTemplates = append(duplicate.TaskTemplates, task.TaskTemplate{ID: "release-copy", Name: "发布", Fields: []task.TaskTemplateField{{
		Key: "deploy", DisplayName: "允许部署", InputType: task.TaskTemplateFieldInputBool, DefaultValue: false,
	}}})
	if _, err := Validate(duplicate); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝重复模板名称")
	}
}
