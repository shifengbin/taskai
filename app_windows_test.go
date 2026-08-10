//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

// TestConfigureCommandProcessHidesWindowsConsole 校验任务菜单后台命令的共享配置入口
// 在 Windows 上为经 cmd、PowerShell 包裹或直接执行的三种形态都设置无控制台窗口属性，
// 对应 spec「后台启动的任务菜单命令不显示窗口」。
func TestConfigureCommandProcessHidesWindowsConsole(t *testing.T) {
	cases := []struct {
		name      string
		shellPath string
		command   string
		arguments []string
	}{
		{"cmd", "cmd.exe", "echo", []string{"hi"}},
		{"powershell", "powershell.exe", "codex", nil},
		{"direct", "", "notepad", nil},
	}

	for _, current := range cases {
		t.Run(current.name, func(t *testing.T) {
			process := commandProcess(current.shellPath, current.command, current.arguments)
			configureCommandProcess(process, `C:\`, nil)

			if process.SysProcAttr == nil {
				t.Fatalf("%s：后台命令进程未配置 Windows 进程属性", current.name)
			}
			if !process.SysProcAttr.HideWindow {
				t.Fatalf("%s：后台命令进程未隐藏启动窗口", current.name)
			}
			if process.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
				t.Fatalf("%s：后台命令进程创建标志 = %#x，未设置 CREATE_NO_WINDOW", current.name, process.SysProcAttr.CreationFlags)
			}
		})
	}
}
