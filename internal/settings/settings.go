package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultTaskTreeWidth = 360
const MinimumTaskTreeWidth = 280

type ColorScheme string

const (
	ColorSchemeLight ColorScheme = "light"
	ColorSchemeDark  ColorScheme = "dark"
)

const DefaultColorScheme = ColorSchemeLight

type Settings struct {
	WorkspaceRoot string      `json:"workspaceRoot"`
	TaskTreeWidth int         `json:"taskTreeWidth"`
	ColorScheme   ColorScheme `json:"colorScheme"`
	ShellPath     string      `json:"shellPath"`
}

func Default(applicationDataDirectory string) Settings {
	return Settings{
		WorkspaceRoot: filepath.Join(applicationDataDirectory, "workspaces"),
		TaskTreeWidth: DefaultTaskTreeWidth,
		ColorScheme:   DefaultColorScheme,
		ShellPath:     DefaultShellPath(),
	}
}

func Validate(next Settings) (Settings, error) {
	if strings.TrimSpace(next.WorkspaceRoot) == "" {
		return Settings{}, fmt.Errorf("任务工作区根目录不能为空")
	}
	if next.ColorScheme == "" {
		next.ColorScheme = DefaultColorScheme
	}
	if next.ColorScheme != ColorSchemeLight && next.ColorScheme != ColorSchemeDark {
		return Settings{}, fmt.Errorf("不支持的颜色模式: %q", next.ColorScheme)
	}
	if strings.TrimSpace(next.ShellPath) == "" {
		next.ShellPath = DefaultShellPath()
	}
	if next.ShellPath != "" {
		normalizedShellPath, err := NormalizeShellPath(next.ShellPath)
		if err != nil {
			return Settings{}, err
		}
		next.ShellPath = normalizedShellPath
	}

	absoluteRoot, err := filepath.Abs(next.WorkspaceRoot)
	if err != nil {
		return Settings{}, fmt.Errorf("解析任务工作区根目录失败: %w", err)
	}
	next.WorkspaceRoot = filepath.Clean(absoluteRoot)
	if err := os.MkdirAll(next.WorkspaceRoot, 0o700); err != nil {
		return Settings{}, fmt.Errorf("创建任务工作区根目录失败: %w", err)
	}

	probe, err := os.CreateTemp(next.WorkspaceRoot, ".taskai-write-check-*")
	if err != nil {
		return Settings{}, fmt.Errorf("任务工作区根目录不可写: %w", err)
	}
	probePath := probe.Name()
	if err := probe.Close(); err != nil {
		os.Remove(probePath)
		return Settings{}, fmt.Errorf("验证任务工作区根目录失败: %w", err)
	}
	if err := os.Remove(probePath); err != nil {
		return Settings{}, fmt.Errorf("清理任务工作区验证文件失败: %w", err)
	}
	if next.TaskTreeWidth < MinimumTaskTreeWidth {
		next.TaskTreeWidth = MinimumTaskTreeWidth
	}

	return next, nil
}
