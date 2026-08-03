//go:build windows

package lifecycle

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWriteManifestContentsAllowsReparsePointInWorkspaceAncestor(t *testing.T) {
	targetRoot := filepath.Join(t.TempDir(), "target-root")
	if err := os.Mkdir(targetRoot, 0o700); err != nil {
		t.Fatalf("创建联接点目标目录失败: %v", err)
	}

	linkedRoot := filepath.Join(t.TempDir(), "linked-root")
	createManifestWindowsTestJunction(t, linkedRoot, targetRoot)
	assertManifestWindowsTestReparsePoint(t, linkedRoot)
	workspacePath := filepath.Join(linkedRoot, "task-1")
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		t.Fatalf("创建任务工作目录失败: %v", err)
	}

	contents := []byte("iteration: task-1\nrepos: []\n")
	if err := writeManifestContents(workspacePath, ".", "manifest.yaml", contents); err != nil {
		t.Fatalf("writeManifestContents() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(targetRoot, "task-1", "manifest.yaml"))
	if err != nil {
		t.Fatalf("读取清单文件失败: %v", err)
	}
	if string(got) != string(contents) {
		t.Fatalf("清单内容 = %q，期望 %q", got, contents)
	}
}

func TestWriteManifestContentsRejectsReparsePointWorkspace(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(targetPath, 0o700); err != nil {
		t.Fatalf("创建联接点目标目录失败: %v", err)
	}

	workspacePath := filepath.Join(t.TempDir(), "workspace-link")
	createManifestWindowsTestJunction(t, workspacePath, targetPath)
	assertManifestWindowsTestReparsePoint(t, workspacePath)

	err := writeManifestContents(workspacePath, ".", "manifest.yaml", []byte("iteration: task-1\nrepos: []\n"))
	if err == nil {
		t.Fatal("writeManifestContents() error = nil，期望拒绝重解析点任务工作目录")
	}
	assertManifestWindowsReparseError(t, err)
	if _, err := os.Stat(filepath.Join(targetPath, "manifest.yaml")); !os.IsNotExist(err) {
		t.Fatalf("联接点目标中的清单文件状态 = %v，期望不存在", err)
	}
}

func TestWriteManifestContentsRejectsReparsePointOutputDirectory(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		t.Fatalf("创建任务工作目录失败: %v", err)
	}
	targetPath := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(targetPath, 0o700); err != nil {
		t.Fatalf("创建联接点目标目录失败: %v", err)
	}

	outputDirectory := filepath.Join(workspacePath, "config")
	createManifestWindowsTestJunction(t, outputDirectory, targetPath)
	assertManifestWindowsTestReparsePoint(t, outputDirectory)

	err := writeManifestContents(workspacePath, "config", "manifest.yaml", []byte("iteration: task-1\nrepos: []\n"))
	if err == nil {
		t.Fatal("writeManifestContents() error = nil，期望拒绝重解析点输出目录")
	}
	assertManifestWindowsReparseError(t, err)
	if _, err := os.Stat(filepath.Join(targetPath, "manifest.yaml")); !os.IsNotExist(err) {
		t.Fatalf("联接点目标中的清单文件状态 = %v，期望不存在", err)
	}
}

func TestWriteManifestContentsRejectsReparsePointTarget(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		t.Fatalf("创建任务工作目录失败: %v", err)
	}
	targetPath := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(targetPath, 0o700); err != nil {
		t.Fatalf("创建联接点目标目录失败: %v", err)
	}

	manifestPath := filepath.Join(workspacePath, "manifest.yaml")
	createManifestWindowsTestJunction(t, manifestPath, targetPath)
	assertManifestWindowsTestReparsePoint(t, manifestPath)

	err := writeManifestContents(workspacePath, ".", "manifest.yaml", []byte("iteration: task-1\nrepos: []\n"))
	if err == nil {
		t.Fatal("writeManifestContents() error = nil，期望拒绝重解析点清单目标")
	}
	if entries, err := os.ReadDir(targetPath); err != nil || len(entries) != 0 {
		t.Fatalf("联接点目标目录内容 = %#v, %v，期望为空", entries, err)
	}
}

func createManifestWindowsTestJunction(t *testing.T, linkPath, targetPath string) {
	t.Helper()
	command := exec.Command("cmd", "/c", "mklink", "/J", linkPath, targetPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("创建目录联接点失败: %v: %s", err, output)
	}
	t.Cleanup(func() {
		if err := os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
			t.Errorf("删除目录联接点失败: %v", err)
		}
	})
}

func assertManifestWindowsTestReparsePoint(t *testing.T, path string) {
	t.Helper()
	encodedPath, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatalf("编码目录联接点路径失败: %v", err)
	}
	attributes, err := windows.GetFileAttributes(encodedPath)
	if err != nil {
		t.Fatalf("读取目录联接点属性失败: %v", err)
	}
	if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
		t.Fatalf("目录联接点属性 = %#x，期望包含重解析点标记", attributes)
	}
}

func assertManifestWindowsReparseError(t *testing.T, err error) {
	t.Helper()
	if !strings.Contains(err.Error(), "重解析点") {
		t.Fatalf("错误 = %q，期望说明重解析点不安全", err)
	}
}
