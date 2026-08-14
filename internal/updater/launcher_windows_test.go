//go:build windows

package updater

import (
	"testing"

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
