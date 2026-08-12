//go:build windows

package backgroundprocess

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func Configure(process *exec.Cmd) {
	process.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
