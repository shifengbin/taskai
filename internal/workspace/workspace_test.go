package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateCreatesTaskIDChildDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")

	path, err := Create(root, "task-1")

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if path != filepath.Join(root, "task-1") {
		t.Errorf("Create() path = %q, want %q", path, filepath.Join(root, "task-1"))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Create() created non-directory %q", path)
	}
}

func TestCreateReusesExistingSafeTaskWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	first, err := Create(root, "task-1")
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := Create(root, "task-1")
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if first != second {
		t.Errorf("Create() reused path = %q，期望 %q", second, first)
	}
}

func TestRemoveDeletesMatchingTaskWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	path, err := Create(root, "task-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "output.txt"), []byte("temporary"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err = Remove(root, path, "task-1")

	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Remove() workspace still exists, Stat() error = %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Errorf("Remove() removed workspace root: %v", err)
	}
}

func TestRemoveRejectsWorkspaceRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err := Remove(root, root, "task-1")

	if err == nil {
		t.Fatal("Remove() error = nil, want root deletion rejection")
	}
	if _, err := os.Stat(root); err != nil {
		t.Errorf("Remove() deleted workspace root: %v", err)
	}
}

func TestRemoveRejectsWorkspaceForDifferentTask(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	path, err := Create(root, "task-2")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err = Remove(root, path, "task-1")

	if err == nil {
		t.Fatal("Remove() error = nil, want mismatched task ID rejection")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Remove() deleted mismatched workspace: %v", err)
	}
}

func TestRemoveRejectsWorkspaceOutsideRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	outside := filepath.Join(t.TempDir(), "outside-task")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err := Remove(root, outside, "outside-task")

	if err == nil {
		t.Fatal("Remove() error = nil, want outside path rejection")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("Remove() deleted outside directory: %v", err)
	}
}

func TestRemoveRejectsSymlinkToOutsideRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside-task")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	path := filepath.Join(root, "task-1")
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("Symlink() unavailable: %v", err)
	}

	err := Remove(root, path, "task-1")

	if err == nil {
		t.Fatal("Remove() error = nil, want symlink rejection")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("Remove() deleted symlink target: %v", err)
	}
}
