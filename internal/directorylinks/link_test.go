//go:build !windows

package directorylinks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeDirectoryLinkCreatesReadsAndRemovesOnlyLink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	contents := filepath.Join(source, "keep.txt")
	if err := os.WriteFile(contents, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(root, "linked")
	links := NativeDirectoryLinkFS()
	if err := links.Create(linkPath, source); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("链接模式 = %v，期望 symlink", info.Mode())
	}
	target, exists, err := links.Read(linkPath)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !exists || target != source {
		t.Fatalf("Read() = (%q, %t)，期望 (%q, true)", target, exists, source)
	}
	if err := links.Remove(linkPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("Remove() 后链接仍存在: %v", err)
	}
	if got, err := os.ReadFile(contents); err != nil || string(got) != "keep" {
		t.Fatalf("Remove() 修改了来源内容: contents=%q err=%v", got, err)
	}
}

func TestNativeDirectoryLinkRejectsOrdinaryDirectoryRemoval(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "ordinary")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := NativeDirectoryLinkFS().Remove(directory); err == nil {
		t.Fatal("Remove() error = nil，期望拒绝普通目录")
	}
	if info, err := os.Stat(directory); err != nil || !info.IsDir() {
		t.Fatalf("普通目录被修改: info=%v err=%v", info, err)
	}
}
