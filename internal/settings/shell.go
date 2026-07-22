package settings

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func DetectShells() []string {
	seen := make(map[string]bool)
	shells := make([]string, 0)
	for _, candidate := range shellCandidates() {
		resolved, err := NormalizeShellPath(candidate)
		if err != nil || seen[resolved] {
			continue
		}
		seen[resolved] = true
		shells = append(shells, resolved)
	}
	return shells
}

func DefaultShellPath() string {
	shells := DetectShells()
	if len(shells) == 0 {
		return ""
	}
	return shells[0]
}

func NormalizeShellPath(shellPath string) (string, error) {
	shellPath = strings.TrimSpace(shellPath)
	if shellPath == "" {
		return "", fmt.Errorf("Shell 路径不能为空")
	}
	resolved, err := exec.LookPath(shellPath)
	if err != nil {
		return "", fmt.Errorf("找不到 Shell: %w", err)
	}
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("解析 Shell 路径失败: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("读取 Shell 路径失败: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("Shell 路径不能是文件夹")
	}
	return filepath.Clean(absPath), nil
}

func shellCandidates() []string {
	if runtime.GOOS == "windows" {
		candidates := []string{
			os.Getenv("ComSpec"),
			os.Getenv("COMSPEC"),
			"pwsh.exe",
			"powershell.exe",
			"cmd.exe",
		}
		if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
			candidates = append(candidates, filepath.Join(programFiles, "PowerShell", "7", "pwsh.exe"))
		}
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = os.Getenv("WINDIR")
		}
		if systemRoot != "" {
			candidates = append(candidates,
				filepath.Join(systemRoot, "System32", "cmd.exe"),
				filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
			)
		}
		return candidates
	}

	return []string{
		os.Getenv("SHELL"),
		"zsh",
		"bash",
		"fish",
		"sh",
		"dash",
		"ksh",
		"/bin/zsh",
		"/bin/bash",
		"/bin/fish",
		"/bin/sh",
		"/bin/dash",
		"/bin/ksh",
		"/usr/bin/zsh",
		"/usr/bin/bash",
		"/usr/bin/fish",
	}
}
