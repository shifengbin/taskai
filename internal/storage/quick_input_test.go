package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taskai/internal/quickinput"
	"taskai/internal/settings"
)

func TestRepositoryLoadsMissingQuickInputsAsEmptySlice(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(dataPath, []byte(`{"tasks":[],"settings":{}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.QuickInputs == nil {
		t.Fatal("Load() QuickInputs = nil, want empty slice")
	}
	if len(loaded.QuickInputs) != 0 {
		t.Fatalf("Load() QuickInputs = %#v, want empty", loaded.QuickInputs)
	}
}

func TestRepositoryManagesQuickInputsByStableIDAndOrder(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	content := "第一行\n" + strings.Repeat("内容", 1_000)
	first, err := repository.SaveQuickInput(quickinput.QuickInput{Name: "部署", Content: content})
	if err != nil {
		t.Fatalf("SaveQuickInput() first error = %v", err)
	}
	second, err := repository.SaveQuickInput(quickinput.QuickInput{Name: "部署", Content: "go test ./..."})
	if err != nil {
		t.Fatalf("SaveQuickInput() second error = %v", err)
	}
	if first.ID == "" || second.ID == "" || first.ID == second.ID {
		t.Fatalf("保存同名输入后的 ID = %q, %q", first.ID, second.ID)
	}

	first.Name = "  生产部署  "
	updated, err := repository.SaveQuickInput(first)
	if err != nil {
		t.Fatalf("SaveQuickInput() update error = %v", err)
	}
	if updated.Name != "生产部署" || updated.Content != content {
		t.Fatalf("更新结果 = %#v", updated)
	}

	reordered, err := repository.ReorderQuickInputs([]string{second.ID, first.ID})
	if err != nil {
		t.Fatalf("ReorderQuickInputs() error = %v", err)
	}
	if len(reordered) != 2 || reordered[0].ID != second.ID || reordered[1].ID != first.ID {
		t.Fatalf("重排结果 = %#v", reordered)
	}
	if _, err := repository.ReorderQuickInputs([]string{second.ID, second.ID}); err == nil {
		t.Fatal("ReorderQuickInputs() error = nil, want duplicate ID rejection")
	}
	if _, err := repository.ReorderQuickInputs([]string{second.ID, "missing"}); err == nil {
		t.Fatal("ReorderQuickInputs() error = nil, want unknown ID rejection")
	}
	if _, err := repository.SaveQuickInput(quickinput.QuickInput{ID: "forged", Name: "部署", Content: "rm -rf ./build"}); err == nil {
		t.Fatal("SaveQuickInput() error = nil, want unknown ID rejection")
	}

	if err := repository.DeleteQuickInput(first.ID); err != nil {
		t.Fatalf("DeleteQuickInput() error = %v", err)
	}
	listed, err := repository.ListQuickInputs()
	if err != nil {
		t.Fatalf("ListQuickInputs() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != second.ID {
		t.Fatalf("删除后的快捷输入 = %#v", listed)
	}
}
