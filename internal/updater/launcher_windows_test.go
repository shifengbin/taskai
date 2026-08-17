//go:build windows

package updater

import (
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestDetachedCommandHidesWindowsConsole(t *testing.T) {
	command := detachedCommand(Invocation{Command: "example.exe"})
	if command.SysProcAttr == nil || !command.SysProcAttr.HideWindow {
		t.Fatal("Windows updater process does not hide its console window")
	}
	if command.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("creation flags = %#x, want CREATE_NO_WINDOW", command.SysProcAttr.CreationFlags)
	}
}

func TestWindowsLaunchInstallerUsesShellExecuteOpen(t *testing.T) {
	installerPath := filepath.Join(`C:\Updates`, "taskai-amd64-installer.exe")
	calls := 0
	var gotVerb, gotFile, gotDirectory string
	var gotArgs *uint16
	var gotShowCmd int32
	original := shellExecute
	shellExecute = func(_ windows.Handle, verb, file, args, cwd *uint16, showCmd int32) error {
		calls++
		gotVerb = utf16PtrToString(verb)
		gotFile = utf16PtrToString(file)
		gotDirectory = utf16PtrToString(cwd)
		gotArgs = args
		gotShowCmd = showCmd
		return nil
	}
	t.Cleanup(func() { shellExecute = original })

	if err := DefaultSystemLauncher().LaunchInstaller(installerPath); err != nil {
		t.Fatalf("LaunchInstaller() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("ShellExecute 调用次数 = %d，期望 1 次", calls)
	}
	if gotVerb != "open" {
		t.Fatalf("ShellExecute 动词 = %q，期望 open（触发 UAC 提升）", gotVerb)
	}
	if gotFile != installerPath {
		t.Fatalf("ShellExecute 目标 = %q，期望 %q", gotFile, installerPath)
	}
	if want := filepath.Dir(installerPath); gotDirectory != want {
		t.Fatalf("ShellExecute 工作目录 = %q，期望 %q", gotDirectory, want)
	}
	if gotArgs != nil {
		t.Fatalf("ShellExecute 参数 = %q，期望无参数", utf16PtrToString(gotArgs))
	}
	if gotShowCmd != windows.SW_SHOWNORMAL {
		t.Fatalf("ShellExecute 显示方式 = %d，期望 SW_SHOWNORMAL", gotShowCmd)
	}
}

func utf16PtrToString(pointer *uint16) string {
	if pointer == nil {
		return ""
	}
	length := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(pointer), length*2)) != 0 {
		length++
	}
	return windows.UTF16ToString(unsafe.Slice(pointer, length))
}

