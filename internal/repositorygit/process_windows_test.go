//go:build windows

package repositorygit

import (
	"bytes"
	"slices"
	"testing"

	"golang.org/x/sys/windows"
)

func TestGitCommandHidesWindowsConsoleAndPreservesConfiguration(t *testing.T) {
	for _, arguments := range [][]string{
		{"status", "--porcelain"},
		{"push", "origin", "main"},
	} {
		t.Run(arguments[0], func(t *testing.T) {
			var output bytes.Buffer
			var standardError bytes.Buffer
			process := gitCommand(`C:\workspace\repository`, &output, &standardError, arguments...)

			if process.Dir != `C:\workspace\repository` {
				t.Fatalf("Git 工作目录 = %q，期望 %q", process.Dir, `C:\workspace\repository`)
			}
			if !slices.Contains(process.Env, "GIT_TERMINAL_PROMPT=0") {
				t.Fatalf("Git 环境变量未禁用交互认证: %#v", process.Env)
			}
			if process.Stdout != &output || process.Stderr != &standardError {
				t.Fatal("Git 标准输出或标准错误未保留")
			}
			if process.SysProcAttr == nil {
				t.Fatal("Git 后台进程未配置 Windows 进程属性")
			}
			if !process.SysProcAttr.HideWindow {
				t.Fatal("Git 后台进程未隐藏启动窗口")
			}
			if process.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
				t.Fatalf("Git 后台进程创建标志 = %#x，未设置 CREATE_NO_WINDOW", process.SysProcAttr.CreationFlags)
			}
		})
	}
}
