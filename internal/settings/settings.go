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

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
)

const DefaultActiveTaskStatus = TaskStatusPending

type TaskMenuItemKind string

const (
	TaskMenuItemKindEditTask       TaskMenuItemKind = "edit-task"
	TaskMenuItemKindCreateTerminal TaskMenuItemKind = "create-terminal"
	TaskMenuItemKindOpenFolder     TaskMenuItemKind = "open-folder"
	TaskMenuItemKindCommand        TaskMenuItemKind = "command"

	TaskMenuItemEditTaskID       = "system.edit-task"
	TaskMenuItemCreateTerminalID = "system.create-terminal"
	TaskMenuItemOpenFolderID     = "system.open-folder"
)

type TaskMenuItem struct {
	ID           string           `json:"id"`
	Kind         TaskMenuItemKind `json:"kind"`
	Name         string           `json:"name"`
	Command      string           `json:"command,omitempty"`
	Arguments    []string         `json:"arguments,omitempty"`
	ShowTerminal bool             `json:"showTerminal"`
}

type Settings struct {
	WorkspaceRoot    string         `json:"workspaceRoot"`
	TaskTreeWidth    int            `json:"taskTreeWidth"`
	ColorScheme      ColorScheme    `json:"colorScheme"`
	ShellPath        string         `json:"shellPath"`
	TaskMenuItems    []TaskMenuItem `json:"taskMenuItems"`
	ActiveTaskStatus TaskStatus     `json:"activeTaskStatus"`
}

func Default(applicationDataDirectory string) Settings {
	return Settings{
		WorkspaceRoot:    filepath.Join(applicationDataDirectory, "workspaces"),
		TaskTreeWidth:    DefaultTaskTreeWidth,
		ColorScheme:      DefaultColorScheme,
		ShellPath:        DefaultShellPath(),
		TaskMenuItems:    DefaultTaskMenuItems(),
		ActiveTaskStatus: DefaultActiveTaskStatus,
	}
}

func DefaultTaskMenuItems() []TaskMenuItem {
	return []TaskMenuItem{
		fixedTaskMenuItem(TaskMenuItemEditTaskID),
		fixedTaskMenuItem(TaskMenuItemCreateTerminalID),
		fixedTaskMenuItem(TaskMenuItemOpenFolderID),
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
	if next.ActiveTaskStatus == "" {
		next.ActiveTaskStatus = DefaultActiveTaskStatus
	}
	if next.ActiveTaskStatus != TaskStatusPending && next.ActiveTaskStatus != TaskStatusRunning && next.ActiveTaskStatus != TaskStatusCompleted {
		return Settings{}, fmt.Errorf("不支持的当前任务标签: %q", next.ActiveTaskStatus)
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
	menuItems, err := normalizeTaskMenuItems(next.TaskMenuItems)
	if err != nil {
		return Settings{}, err
	}
	next.TaskMenuItems = menuItems

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

func fixedTaskMenuItem(id string) TaskMenuItem {
	switch id {
	case TaskMenuItemEditTaskID:
		return TaskMenuItem{ID: id, Kind: TaskMenuItemKindEditTask, Name: "编辑任务"}
	case TaskMenuItemCreateTerminalID:
		return TaskMenuItem{ID: id, Kind: TaskMenuItemKindCreateTerminal, Name: "新增终端"}
	case TaskMenuItemOpenFolderID:
		return TaskMenuItem{ID: id, Kind: TaskMenuItemKindOpenFolder, Name: "打开任务文件夹"}
	default:
		return TaskMenuItem{}
	}
}

func normalizeTaskMenuItems(items []TaskMenuItem) ([]TaskMenuItem, error) {
	if len(items) == 0 {
		return DefaultTaskMenuItems(), nil
	}

	normalized := make([]TaskMenuItem, 0, len(items)+3)
	seen := make(map[string]bool)
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		if item.ID == "" {
			return nil, fmt.Errorf("任务菜单项 ID 不能为空")
		}
		if seen[item.ID] {
			return nil, fmt.Errorf("任务菜单项 ID 重复: %q", item.ID)
		}
		seen[item.ID] = true

		if fixed := fixedTaskMenuItem(item.ID); fixed.ID != "" {
			normalized = append(normalized, fixed)
			continue
		}
		if item.Kind != TaskMenuItemKindCommand {
			return nil, fmt.Errorf("不支持的自定义任务菜单项类型: %q", item.Kind)
		}
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			return nil, fmt.Errorf("自定义任务菜单项名称不能为空")
		}
		item.Command = strings.TrimSpace(item.Command)
		if item.Command == "" {
			return nil, fmt.Errorf("自定义任务菜单项启动命令不能为空")
		}
		item.Arguments = normalizeArguments(item.Arguments)
		normalized = append(normalized, item)
	}

	for _, fixed := range DefaultTaskMenuItems() {
		if !seen[fixed.ID] {
			normalized = append(normalized, fixed)
		}
	}
	return normalized, nil
}

func normalizeArguments(arguments []string) []string {
	normalized := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if trimmed := strings.TrimSpace(argument); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}
