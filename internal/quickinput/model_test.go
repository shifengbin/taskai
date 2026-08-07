package quickinput

import (
	"strings"
	"testing"
)

func TestNewQuickInputNormalizesNameAndPreservesContent(t *testing.T) {
	content := "第一行\n  第二行\n"
	created, err := New("  部署  ", content)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("New() ID is empty")
	}
	if created.Name != "部署" {
		t.Fatalf("New() Name = %q, want %q", created.Name, "部署")
	}
	if created.Content != content {
		t.Fatalf("New() Content = %q, want %q", created.Content, content)
	}
}

func TestNewQuickInputAllowsDuplicateNames(t *testing.T) {
	first, err := New("部署", "pnpm build")
	if err != nil {
		t.Fatalf("New() first error = %v", err)
	}
	second, err := New("部署", "go test ./...")
	if err != nil {
		t.Fatalf("New() second error = %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("同名快捷输入 ID 相同: %q", first.ID)
	}
}

func TestNewQuickInputRejectsInvalidNameOrContent(t *testing.T) {
	for _, candidate := range []struct {
		name    string
		content string
	}{
		{name: " \t\n", content: "git status"},
		{name: strings.Repeat("名", MaxNameLength+1), content: "git status"},
		{name: "部署", content: " \t\n"},
	} {
		if _, err := New(candidate.name, candidate.content); err == nil {
			t.Fatalf("New(%q, %q) error = nil", candidate.name, candidate.content)
		}
	}
}

func TestNormalizeRejectsMissingIDAndPreservesValidContent(t *testing.T) {
	if _, err := Normalize(QuickInput{Name: "部署", Content: "git status"}); err == nil {
		t.Fatal("Normalize() error = nil, want missing ID error")
	}

	content := "\n  export TOKEN=value\n"
	normalized, err := Normalize(QuickInput{ID: " quick-input-1 ", Name: " 部署 ", Content: content})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if normalized.ID != "quick-input-1" || normalized.Name != "部署" || normalized.Content != content {
		t.Fatalf("Normalize() = %#v", normalized)
	}
}
