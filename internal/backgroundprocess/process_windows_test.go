//go:build windows

package backgroundprocess

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

func TestConfigureHidesWindowsConsole(t *testing.T) {
	process := exec.Command("example")

	Configure(process)

	if process.SysProcAttr == nil {
		t.Fatal("后台进程未配置 Windows 进程属性")
	}
	if !process.SysProcAttr.HideWindow {
		t.Fatal("后台进程未隐藏启动窗口")
	}
	if process.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("后台进程创建标志 = %#x，未设置 CREATE_NO_WINDOW", process.SysProcAttr.CreationFlags)
	}
}
