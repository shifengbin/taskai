package main

import (
	"path/filepath"
	"testing"
)

func TestApplicationOptionsKeepWebViewDropEnabledForNativeFileDrop(t *testing.T) {
	options := applicationOptions(&App{}, "/tmp/taskai-test")
	if options.DragAndDrop == nil {
		t.Fatal("应用选项未配置文件拖放")
	}
	if !options.DragAndDrop.EnableFileDrop {
		t.Fatal("应用选项未启用原生文件拖放")
	}
	if options.DragAndDrop.DisableWebViewDrop {
		t.Fatal("启用原生文件拖放时不能禁用 WebView 拖放")
	}
}

func TestApplicationOptionsPreventConcurrentTaskAIInstances(t *testing.T) {
	dataDirectory := "/tmp/taskai-test"
	options := applicationOptions(&App{}, dataDirectory)
	if options.SingleInstanceLock == nil {
		t.Fatal("应用选项未启用单实例锁")
	}
	if options.SingleInstanceLock.UniqueId != applicationSingleInstanceID(dataDirectory) {
		t.Fatalf("单实例标识 = %q", options.SingleInstanceLock.UniqueId)
	}
}

func TestApplicationOptionsIsolateDifferentDataDirectories(t *testing.T) {
	first := applicationOptions(&App{}, "/tmp/taskai-first")
	second := applicationOptions(&App{}, "/tmp/taskai-second")
	if first.SingleInstanceLock.UniqueId == second.SingleInstanceLock.UniqueId {
		t.Fatal("不同数据目录使用了相同窗口单实例标识")
	}
}

func TestApplicationInstanceLockRejectsSecondProcessBeforeRepositoryStartup(t *testing.T) {
	dataDirectory := filepath.Join(t.TempDir(), "taskai")
	first, err := acquireApplicationInstanceLock(dataDirectory)
	if err != nil {
		t.Fatalf("first acquireApplicationInstanceLock() error = %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })

	if second, err := acquireApplicationInstanceLock(dataDirectory); err == nil {
		_ = second.Close()
		t.Fatal("second acquireApplicationInstanceLock() error = nil")
	}

	if err := first.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	third, err := acquireApplicationInstanceLock(dataDirectory)
	if err != nil {
		t.Fatalf("third acquireApplicationInstanceLock() error = %v", err)
	}
	if err := third.Close(); err != nil {
		t.Fatalf("third Close() error = %v", err)
	}
}
