//go:build !windows

package directorylinks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taskai/internal/workspace"
)

func TestSynchronizerCreatesUpdatesAndRemovesManagedLinksIdempotently(t *testing.T) {
	root, workspacePath, taskID, token := createOwnedWorkspace(t)
	projectA := makeDirectory(t, t.TempDir(), "project-a", "src")
	projectB := makeDirectory(t, t.TempDir(), "project-b", "src")
	synchronizer := NewSynchronizer(NativeDirectoryLinkFS())

	initial := []Link{{Name: "src", SourcePath: projectA, CanonicalPath: projectA, FieldKey: "sources", FieldName: "来源目录"}}
	if err := synchronizer.Sync(root, workspacePath, taskID, token, initial); err != nil {
		t.Fatalf("Sync(initial) error = %v", err)
	}
	assertDirectoryLink(t, filepath.Join(workspacePath, "src"), projectA)
	if err := synchronizer.Sync(root, workspacePath, taskID, token, initial); err != nil {
		t.Fatalf("Sync(idempotent) error = %v", err)
	}

	updated := []Link{
		{Name: "project-a-src", SourcePath: projectA, CanonicalPath: projectA, FieldKey: "sources", FieldName: "来源目录"},
		{Name: "project-b-src", SourcePath: projectB, CanonicalPath: projectB, FieldKey: "sources", FieldName: "来源目录"},
	}
	if err := synchronizer.Sync(root, workspacePath, taskID, token, updated); err != nil {
		t.Fatalf("Sync(updated) error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(workspacePath, "src")); !os.IsNotExist(err) {
		t.Fatalf("旧链接仍存在: %v", err)
	}
	assertDirectoryLink(t, filepath.Join(workspacePath, "project-a-src"), projectA)
	assertDirectoryLink(t, filepath.Join(workspacePath, "project-b-src"), projectB)

	ordinary := filepath.Join(workspacePath, "keep")
	if err := os.Mkdir(ordinary, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := synchronizer.Sync(root, workspacePath, taskID, token, nil); err != nil {
		t.Fatalf("Sync(empty) error = %v", err)
	}
	if info, err := os.Stat(ordinary); err != nil || !info.IsDir() {
		t.Fatalf("空期望修改了普通目录: info=%v err=%v", info, err)
	}
	assertSourceExists(t, projectA)
	assertSourceExists(t, projectB)

	manifest, found, err := readManifest(manifestPath(root, token))
	if err != nil || !found || manifest.State != manifestStateStable || len(manifest.Links) != 0 {
		t.Fatalf("稳定清单 = %#v found=%t err=%v", manifest, found, err)
	}
}

func TestSynchronizerRecoversPendingManifestAfterPartialCreation(t *testing.T) {
	root, workspacePath, taskID, token := createOwnedWorkspace(t)
	api := makeDirectory(t, t.TempDir(), "api")
	web := makeDirectory(t, t.TempDir(), "web")
	desired := []Link{
		{Name: "api", SourcePath: api, CanonicalPath: api, FieldKey: "sources", FieldName: "来源目录"},
		{Name: "web", SourcePath: web, CanonicalPath: web, FieldKey: "sources", FieldName: "来源目录"},
	}
	failing := &failCreateDirectoryLinkFS{DirectoryLinkFS: NativeDirectoryLinkFS(), failOnCall: 2}
	if err := NewSynchronizer(failing).Sync(root, workspacePath, taskID, token, desired); err == nil {
		t.Fatal("Sync() error = nil，期望模拟第二次创建失败")
	}
	manifest, found, err := readManifest(manifestPath(root, token))
	if err != nil || !found || manifest.State != manifestStatePending {
		t.Fatalf("中断后的清单 = %#v found=%t err=%v", manifest, found, err)
	}
	if err := NewSynchronizer(NativeDirectoryLinkFS()).Sync(root, workspacePath, taskID, token, desired); err != nil {
		t.Fatalf("Sync(retry) error = %v", err)
	}
	assertDirectoryLink(t, filepath.Join(workspacePath, "api"), api)
	assertDirectoryLink(t, filepath.Join(workspacePath, "web"), web)
	manifest, found, err = readManifest(manifestPath(root, token))
	if err != nil || !found || manifest.State != manifestStateStable || len(manifest.Links) != 2 {
		t.Fatalf("恢复后的清单 = %#v found=%t err=%v", manifest, found, err)
	}
}

func TestSynchronizerRejectsUnknownConflictsBeforeChangingOtherEntries(t *testing.T) {
	for _, conflictType := range []string{"file", "directory", "link"} {
		t.Run(conflictType, func(t *testing.T) {
			root, workspacePath, taskID, token := createOwnedWorkspace(t)
			source := makeDirectory(t, t.TempDir(), "conflict")
			other := makeDirectory(t, t.TempDir(), "other")
			conflictPath := filepath.Join(workspacePath, "conflict")
			switch conflictType {
			case "file":
				if err := os.WriteFile(conflictPath, []byte("keep"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(conflictPath, 0o755); err != nil {
					t.Fatal(err)
				}
			case "link":
				foreign := makeDirectory(t, t.TempDir(), "foreign")
				if err := os.Symlink(foreign, conflictPath); err != nil {
					t.Fatal(err)
				}
			}
			desired := []Link{
				{Name: "conflict", SourcePath: source, CanonicalPath: source, FieldKey: "sources", FieldName: "来源目录"},
				{Name: "other", SourcePath: other, CanonicalPath: other, FieldKey: "sources", FieldName: "来源目录"},
			}
			if err := NewSynchronizer(NativeDirectoryLinkFS()).Sync(root, workspacePath, taskID, token, desired); err == nil {
				t.Fatal("Sync() error = nil，期望拒绝未知占用")
			}
			if _, err := os.Lstat(filepath.Join(workspacePath, "other")); !os.IsNotExist(err) {
				t.Fatalf("预检失败后创建了其他链接: %v", err)
			}
			if _, err := os.Lstat(conflictPath); err != nil {
				t.Fatalf("预检失败后冲突条目丢失: %v", err)
			}
		})
	}
}

func TestSynchronizerRejectsReplacedManagedLinkAndCorruptManifest(t *testing.T) {
	t.Run("replaced link", func(t *testing.T) {
		root, workspacePath, taskID, token := createOwnedWorkspace(t)
		source := makeDirectory(t, t.TempDir(), "source")
		desired := []Link{{Name: "source", SourcePath: source, CanonicalPath: source, FieldKey: "sources", FieldName: "来源目录"}}
		synchronizer := NewSynchronizer(NativeDirectoryLinkFS())
		if err := synchronizer.Sync(root, workspacePath, taskID, token, desired); err != nil {
			t.Fatal(err)
		}
		linkPath := filepath.Join(workspacePath, "source")
		if err := os.Remove(linkPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(linkPath, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := synchronizer.Sync(root, workspacePath, taskID, token, nil); err == nil {
			t.Fatal("Sync() error = nil，期望拒绝删除被替换的登记链接")
		}
		if info, err := os.Stat(linkPath); err != nil || !info.IsDir() {
			t.Fatalf("被替换的普通目录被修改: info=%v err=%v", info, err)
		}
		assertSourceExists(t, source)
	})

	t.Run("corrupt manifest", func(t *testing.T) {
		root, workspacePath, taskID, token := createOwnedWorkspace(t)
		path := manifestPath(root, token)
		if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
		source := makeDirectory(t, t.TempDir(), "source")
		err := NewSynchronizer(NativeDirectoryLinkFS()).Sync(root, workspacePath, taskID, token, []Link{{Name: "source", SourcePath: source}})
		if err == nil {
			t.Fatal("Sync() error = nil，期望拒绝损坏清单")
		}
		if _, statErr := os.Lstat(filepath.Join(workspacePath, "source")); !os.IsNotExist(statErr) {
			t.Fatalf("损坏清单后修改了工作目录: %v", statErr)
		}
	})
}

func TestRemoveOwnedWorkspaceCleansManifestWithoutFollowingManagedLinks(t *testing.T) {
	root, workspacePath, taskID, token := createOwnedWorkspace(t)
	source := makeDirectory(t, t.TempDir(), "source")
	contents := filepath.Join(source, "keep.txt")
	if err := os.WriteFile(contents, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	desired := []Link{{Name: "source", SourcePath: source, CanonicalPath: source, FieldKey: "sources", FieldName: "来源目录"}}
	if err := NewSynchronizer(NativeDirectoryLinkFS()).Sync(root, workspacePath, taskID, token, desired); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	removed, err := workspace.RemoveOwned(root, workspacePath, taskID, token)
	if err != nil || !removed {
		t.Fatalf("RemoveOwned() = (%t, %v)，期望成功", removed, err)
	}
	if _, err := os.Lstat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("任务工作目录仍存在: %v", err)
	}
	if _, err := os.Lstat(manifestPath(root, token)); !os.IsNotExist(err) {
		t.Fatalf("目录链接清单仍存在: %v", err)
	}
	if got, err := os.ReadFile(contents); err != nil || string(got) != "keep" {
		t.Fatalf("来源内容被删除: contents=%q err=%v", got, err)
	}
}

type failCreateDirectoryLinkFS struct {
	DirectoryLinkFS
	createCalls int
	failOnCall  int
}

func (links *failCreateDirectoryLinkFS) Create(linkPath, targetPath string) error {
	links.createCalls++
	if links.createCalls == links.failOnCall {
		return errors.New("模拟创建失败")
	}
	return links.DirectoryLinkFS.Create(linkPath, targetPath)
}

func createOwnedWorkspace(t *testing.T) (string, string, string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspaces")
	taskID := "task-directory-links"
	token := strings.Repeat("a", 64)
	created, err := workspace.CreateOwned(root, taskID, token)
	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	return root, created.Path, taskID, token
}

func assertDirectoryLink(t *testing.T, linkPath, wantTarget string) {
	t.Helper()
	target, exists, err := NativeDirectoryLinkFS().Read(linkPath)
	if err != nil || !exists || target != wantTarget {
		t.Fatalf("目录链接 %q = (%q, %t)，期望 %q，err=%v", linkPath, target, exists, wantTarget, err)
	}
}

func assertSourceExists(t *testing.T, source string) {
	t.Helper()
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		t.Fatalf("来源目录被修改: source=%q info=%v err=%v", source, info, err)
	}
}
