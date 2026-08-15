//go:build windows

package updater

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

var shellExecute = windows.ShellExecute

// startInstallerDetached 通过 ShellExecute 的 open 动词启动安装程序。
// NSIS 安装程序的 UAC 清单要求管理员权限，CreateProcess 无法触发 UAC
// 提升，只会立即返回 ERROR_ELEVATION_REQUIRED，必须经由 ShellExecute
// 启动；ShellExecute 也不会为 GUI 安装向导额外创建控制台窗口。
func startInstallerDetached(invocation Invocation) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(invocation.Command)
	if err != nil {
		return err
	}
	directory, err := windows.UTF16PtrFromString(filepath.Dir(invocation.Command))
	if err != nil {
		return err
	}
	return shellExecute(0, verb, file, nil, directory, windows.SW_SHOWNORMAL)
}
