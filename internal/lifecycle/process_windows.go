//go:build windows

package lifecycle

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func ConfigureBackgroundProcess(process *exec.Cmd) {
	process.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
