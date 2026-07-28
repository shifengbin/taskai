package task

import (
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
	if template.ID == "" || template.Catalogue != "git" || template.DisplayName != "API 仓库" || len(template.Fields) != 1 || template.Fields[0].Key != "repository" || template.Fields[0].Value != "git@example.com:team/api.git" {
		t.Fatalf("模板规范化结果 = %#v", template)
	}

	snapshot, err := NewExtraInfo(template, map[string]string{"branch": " main ", "environment": " "})
	if err != nil {
		t.Fatalf("NewExtraInfo() error = %v", err)
	}
	if snapshot.ID != template.ID || len(snapshot.Parameters) != 2 || snapshot.Parameters[0].Value != "main" || snapshot.Parameters[1].Value != "" {
		t.Fatalf("任务附加信息快照 = %#v", snapshot)
	}

	if _, err := NewExtraInfo(template, map[string]string{}); err == nil {
		t.Fatal("NewExtraInfo() error = nil, want required parameter validation error")
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
	if len(template.Fields) != 2 || template.Fields[1].Key != "remote" {
		t.Fatalf("模板固定字段 = %#v，期望两个字段", template.Fields)
	}

	legacy, err := NormalizeExtraInfoTemplate(ExtraInfoTemplate{
		ID: "legacy-template", Catalogue: "git", DisplayName: "旧仓库",
		Key: "repository", KeyDisplayName: "仓库地址", Value: "git@example.com:team/legacy.git",
	})
	if err != nil {
		t.Fatalf("NormalizeExtraInfoTemplate() legacy error = %v", err)
	}
	if len(legacy.Fields) != 1 || legacy.Fields[0].Key != "repository" || legacy.Fields[0].Value != "git@example.com:team/legacy.git" {
		t.Fatalf("旧模板迁移结果 = %#v", legacy)
	}
}
