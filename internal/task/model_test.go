package task

import (
	"testing"
	"time"
)

func TestNewTaskStartsPending(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 0, 0, 0, time.UTC)

	task, err := NewTask("  编写登录页  ", "实现邮箱登录表单", now)

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
	if !task.CreatedAt.Equal(now) {
		t.Errorf("NewTask() CreatedAt = %v, want %v", task.CreatedAt, now)
	}
	if task.WorkspacePath != "" {
		t.Errorf("NewTask() WorkspacePath = %q, want empty", task.WorkspacePath)
	}
}

func TestNewTaskRejectsBlankTitle(t *testing.T) {
	_, err := NewTask(" \t\n ", "", time.Now())

	if err == nil {
		t.Fatal("NewTask() error = nil, want title validation error")
	}
}
