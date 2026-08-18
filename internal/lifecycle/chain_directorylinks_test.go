//go:build !windows

package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taskai/internal/settings"
	"taskai/internal/task"
	"taskai/internal/workspace"
)

func TestCommandChainRunnerSyncsDirectoryLinksAndPreservesInput(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token := strings.Repeat("b", 64)
	created, err := workspace.CreateOwned(root, "task-1", token)
	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	source := filepath.Join(t.TempDir(), "source")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	template := task.TaskTemplate{ID: "directories", Name: "目录", Fields: []task.TaskTemplateField{{
		Key: "sources", DisplayName: "来源目录", InputType: task.TaskTemplateFieldInputDirectories, Multiple: true,
	}}}
	input := []byte(`{"id":"task-1"}`)
	output, err := NewCommandChainRunner(nil).Run(CommandChainRequest{
		Task:           task.Task{ID: "task-1", TemplateFields: map[string]any{"sources": []string{source}}},
		TaskTemplate:   &template,
		TemplateFields: map[string]any{"sources": []string{source}},
		WorkspaceRoot:  root,
		WorkspacePath:  created.Path,
		WorkspaceToken: token,
		Input:          input,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandSyncDirectoryLinksID, Kind: settings.LifecycleCommandKindSyncDirectoryLinks, Name: "同步任务目录链接",
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(output) != string(input) {
		t.Fatalf("同步命令输出 = %q，期望原样透传 %q", output, input)
	}
	target, err := os.Readlink(filepath.Join(created.Path, "source"))
	if err != nil || target != source {
		t.Fatalf("同步后的目录链接 = %q err=%v，期望 %q", target, err, source)
	}
}

func TestCommandChainRunnerSyncDirectoryLinksUsesCurrentTemplateAndReportsSource(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token := strings.Repeat("c", 64)
	created, err := workspace.CreateOwned(root, "task-1", token)
	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	template := task.TaskTemplate{ID: "directories", Name: "目录", Fields: []task.TaskTemplateField{{
		Key: "sources", DisplayName: "来源目录", InputType: task.TaskTemplateFieldInputDirectories, Multiple: true,
	}}}
	_, err = NewCommandChainRunner(nil).Run(CommandChainRequest{
		Task:           task.Task{ID: "task-1", TemplateFields: map[string]any{"sources": []string{missing}}},
		TaskTemplate:   &template,
		TemplateFields: map[string]any{"sources": []string{missing}},
		WorkspaceRoot:  root,
		WorkspacePath:  created.Path,
		WorkspaceToken: token,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandSyncDirectoryLinksID, Kind: settings.LifecycleCommandKindSyncDirectoryLinks, Name: "同步任务目录链接",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "来源目录") || !strings.Contains(err.Error(), missing) {
		t.Fatalf("Run() error = %v，期望包含字段与来源路径", err)
	}
}
