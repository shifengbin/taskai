package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"taskai/internal/task"
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

type StatusManagementMode string

const (
	StatusManagementModeTitleChange StatusManagementMode = "title-change"
	StatusManagementModeHTTP        StatusManagementMode = "http"
)

const DefaultStatusManagementMode = StatusManagementModeTitleChange

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

type TaskScript struct {
	Script    string   `json:"script,omitempty"`
	Arguments []string `json:"arguments,omitempty"`
}

type LifecycleHook = task.LifecycleHook

const (
	LifecycleHookBeforeStart = task.LifecycleHookBeforeStart
	LifecycleHookPostStart   = task.LifecycleHookPostStart
	LifecycleHookBeforeEnd   = task.LifecycleHookBeforeEnd
	LifecycleHookPostEnd     = task.LifecycleHookPostEnd
	LifecycleHookUpdateTask  = task.LifecycleHookUpdateTask
)

type LifecycleCommandKind string

const (
	LifecycleCommandKindCustom          LifecycleCommandKind = "custom"
	LifecycleCommandKindCreateWorkspace LifecycleCommandKind = "create-workspace"
	LifecycleCommandKindDeleteWorkspace LifecycleCommandKind = "delete-workspace"
	LifecycleCommandKindGitClone        LifecycleCommandKind = "git-clone"

	LifecycleCommandCreateWorkspaceID = "system.lifecycle.create-workspace"
	LifecycleCommandDeleteWorkspaceID = "system.lifecycle.delete-workspace"
	LifecycleCommandGitCloneID        = "system.lifecycle.git-clone"
	LifecycleChainCreateWorkspaceID   = "system.lifecycle-chain.create-workspace"
	LifecycleChainDeleteWorkspaceID   = "system.lifecycle-chain.delete-workspace"
)

type LifecycleCommand struct {
	ID              string               `json:"id"`
	Kind            LifecycleCommandKind `json:"kind"`
	Name            string               `json:"name"`
	Command         string               `json:"command,omitempty"`
	Arguments       []string             `json:"arguments"`
	Documentation   string               `json:"documentation,omitempty"`
	ApplicableHooks []LifecycleHook      `json:"applicableHooks"`
}

type LifecycleCommandReference struct {
	CommandID string   `json:"commandId"`
	Arguments []string `json:"arguments"`
}

type LifecycleCommandChain struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	Commands        []LifecycleCommandReference `json:"commands"`
	CommandIDs      []string                    `json:"commandIds,omitempty"`
	ApplicableHooks []LifecycleHook             `json:"applicableHooks"`
}

type TaskMenuItem struct {
	ID           string           `json:"id"`
	Kind         TaskMenuItemKind `json:"kind"`
	Name         string           `json:"name"`
	Command      string           `json:"command,omitempty"`
	Arguments    []string         `json:"arguments,omitempty"`
	ShowTerminal bool             `json:"showTerminal"`
	BeforeScript *TaskScript      `json:"beforeScript,omitempty"`
	AfterScript  *TaskScript      `json:"afterScript,omitempty"`
}

type Settings struct {
	WorkspaceRoot            string                   `json:"workspaceRoot"`
	TaskTreeWidth            int                      `json:"taskTreeWidth"`
	ColorScheme              ColorScheme              `json:"colorScheme"`
	ShellPath                string                   `json:"shellPath"`
	TaskMenuItems            []TaskMenuItem           `json:"taskMenuItems"`
	ActiveTaskStatus         TaskStatus               `json:"activeTaskStatus"`
	StatusManagementMode     StatusManagementMode     `json:"statusManagementMode"`
	StatusManagementHTTPPort int                      `json:"statusManagementHTTPPort"`
	HTTPServiceEnabled       bool                     `json:"httpServiceEnabled"`
	LifecycleCommands        []LifecycleCommand       `json:"lifecycleCommands"`
	LifecycleChains          []LifecycleCommandChain  `json:"lifecycleChains"`
	LifecycleDefaultChains   map[LifecycleHook]string `json:"lifecycleDefaultChains"`
}

func Default(applicationDataDirectory string) Settings {
	return Settings{
		WorkspaceRoot:          filepath.Join(applicationDataDirectory, "workspaces"),
		TaskTreeWidth:          DefaultTaskTreeWidth,
		ColorScheme:            DefaultColorScheme,
		ShellPath:              DefaultShellPath(),
		TaskMenuItems:          DefaultTaskMenuItems(),
		ActiveTaskStatus:       DefaultActiveTaskStatus,
		StatusManagementMode:   DefaultStatusManagementMode,
		LifecycleCommands:      DefaultLifecycleCommands(),
		LifecycleChains:        DefaultLifecycleChains(),
		LifecycleDefaultChains: DefaultLifecycleDefaultChains(),
	}
}

func DefaultTaskMenuItems() []TaskMenuItem {
	return []TaskMenuItem{
		fixedTaskMenuItem(TaskMenuItemEditTaskID),
		fixedTaskMenuItem(TaskMenuItemCreateTerminalID),
		fixedTaskMenuItem(TaskMenuItemOpenFolderID),
	}
}

func DefaultLifecycleCommands() []LifecycleCommand {
	return []LifecycleCommand{
		fixedLifecycleCommand(LifecycleCommandCreateWorkspaceID),
		fixedLifecycleCommand(LifecycleCommandDeleteWorkspaceID),
		fixedLifecycleCommand(LifecycleCommandGitCloneID),
	}
}

func DefaultLifecycleChains() []LifecycleCommandChain {
	return []LifecycleCommandChain{
		{ID: LifecycleChainCreateWorkspaceID, Name: "创建任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		{ID: LifecycleChainDeleteWorkspaceID, Name: "删除任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandDeleteWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookPostEnd}},
	}
}

func DefaultLifecycleDefaultChains() map[LifecycleHook]string {
	return map[LifecycleHook]string{
		LifecycleHookBeforeStart: LifecycleChainCreateWorkspaceID,
		LifecycleHookPostEnd:     LifecycleChainDeleteWorkspaceID,
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
	if next.StatusManagementMode == "" {
		next.StatusManagementMode = DefaultStatusManagementMode
	}
	if next.StatusManagementMode != StatusManagementModeTitleChange && next.StatusManagementMode != StatusManagementModeHTTP {
		return Settings{}, fmt.Errorf("不支持的状态管理方式: %q", next.StatusManagementMode)
	}
	if next.StatusManagementHTTPPort < 0 || next.StatusManagementHTTPPort > 65535 {
		return Settings{}, fmt.Errorf("状态管理 HTTP 端口必须在 0 到 65535 之间")
	}
	if (next.StatusManagementMode == StatusManagementModeHTTP || next.HTTPServiceEnabled) && next.StatusManagementHTTPPort == 0 {
		return Settings{}, fmt.Errorf("启用 HTTP 服务需要配置端口")
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
	next, err = NormalizeLifecycle(next)
	if err != nil {
		return Settings{}, err
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

func fixedLifecycleCommand(id string) LifecycleCommand {
	switch id {
	case LifecycleCommandCreateWorkspaceID:
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindCreateWorkspace, Name: "创建任务工作目录", Arguments: []string{}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}}
	case LifecycleCommandDeleteWorkspaceID:
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindDeleteWorkspace, Name: "删除任务工作目录", Arguments: []string{}, ApplicableHooks: []LifecycleHook{LifecycleHookPostEnd}}
	case LifecycleCommandGitCloneID:
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindGitClone, Name: "Git 仓库克隆", Arguments: []string{}, Documentation: "参数：dir=<相对目录>（必填）。每个内置 Git 项目将克隆到任务工作目录下的 <dir>/<项目名称>；目标已存在时跳过。指定分支存在时克隆该分支，不存在时从远程默认分支创建同名本地分支。", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookBeforeEnd, LifecycleHookUpdateTask}}
	default:
		return LifecycleCommand{}
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
		item.BeforeScript = normalizeTaskScript(item.BeforeScript)
		item.AfterScript = normalizeTaskScript(item.AfterScript)
		normalized = append(normalized, item)
	}

	for _, fixed := range DefaultTaskMenuItems() {
		if !seen[fixed.ID] {
			normalized = append(normalized, fixed)
		}
	}
	return normalized, nil
}

func normalizeTaskScript(script *TaskScript) *TaskScript {
	if script == nil {
		return nil
	}
	path := strings.TrimSpace(script.Script)
	if path == "" {
		return nil
	}
	return &TaskScript{Script: path, Arguments: normalizeArguments(script.Arguments)}
}

func NormalizeLifecycle(next Settings) (Settings, error) {
	lifecycleCommands, err := normalizeLifecycleCommands(next.LifecycleCommands)
	if err != nil {
		return Settings{}, err
	}
	next.LifecycleCommands = lifecycleCommands
	lifecycleChains, err := normalizeLifecycleChains(next.LifecycleChains, lifecycleCommands)
	if err != nil {
		return Settings{}, err
	}
	next.LifecycleChains = lifecycleChains
	lifecycleDefaults, err := normalizeLifecycleDefaultChains(next.LifecycleDefaultChains, lifecycleChains)
	if err != nil {
		return Settings{}, err
	}
	next.LifecycleDefaultChains = lifecycleDefaults
	return next, nil
}

func normalizeLifecycleCommands(commands []LifecycleCommand) ([]LifecycleCommand, error) {
	normalized := make([]LifecycleCommand, 0, len(commands)+3)
	seen := make(map[string]bool, len(commands)+3)
	for _, command := range commands {
		command.ID = strings.TrimSpace(command.ID)
		if command.ID == "" {
			return nil, fmt.Errorf("生命周期命令 ID 不能为空")
		}
		if seen[command.ID] {
			return nil, fmt.Errorf("生命周期命令 ID 重复: %q", command.ID)
		}
		seen[command.ID] = true
		if fixed := fixedLifecycleCommand(command.ID); fixed.ID != "" {
			normalized = append(normalized, fixed)
			continue
		}
		if command.Kind != LifecycleCommandKindCustom {
			return nil, fmt.Errorf("不支持的生命周期命令类型: %q", command.Kind)
		}
		command.Name = strings.TrimSpace(command.Name)
		command.Command = strings.TrimSpace(command.Command)
		if command.Name == "" || command.Command == "" {
			return nil, fmt.Errorf("自定义生命周期命令名称和可执行命令不能为空")
		}
		command.Arguments = normalizeArguments(command.Arguments)
		applicableHooks, err := normalizeLifecycleApplicableHooks(command.ApplicableHooks, allLifecycleHooks(), false)
		if err != nil {
			return nil, fmt.Errorf("自定义生命周期命令 %q 的适用范围无效: %w", command.Name, err)
		}
		command.ApplicableHooks = applicableHooks
		normalized = append(normalized, command)
	}
	for _, command := range DefaultLifecycleCommands() {
		if !seen[command.ID] {
			normalized = append(normalized, command)
		}
	}
	return normalized, nil
}

func normalizeLifecycleChains(chains []LifecycleCommandChain, commands []LifecycleCommand) ([]LifecycleCommandChain, error) {
	if chains == nil {
		return DefaultLifecycleChains(), nil
	}
	knownCommands := make(map[string]LifecycleCommand, len(commands))
	for _, command := range commands {
		knownCommands[command.ID] = command
	}
	normalized := make([]LifecycleCommandChain, 0, len(chains))
	seen := make(map[string]bool, len(chains))
	for _, chain := range chains {
		chain.ID = strings.TrimSpace(chain.ID)
		chain.Name = strings.TrimSpace(chain.Name)
		if chain.ID == "" || chain.Name == "" {
			return nil, fmt.Errorf("生命周期命令链 ID 和名称不能为空")
		}
		if seen[chain.ID] {
			return nil, fmt.Errorf("生命周期命令链 ID 重复: %q", chain.ID)
		}
		seen[chain.ID] = true
		references := chain.Commands
		if references == nil && chain.CommandIDs != nil {
			references = make([]LifecycleCommandReference, 0, len(chain.CommandIDs))
			for _, commandID := range chain.CommandIDs {
				references = append(references, LifecycleCommandReference{CommandID: commandID, Arguments: []string{}})
			}
		}
		normalizedReferences := make([]LifecycleCommandReference, 0, len(references))
		for _, reference := range references {
			normalizedReference, err := normalizeLifecycleCommandReference(reference, knownCommands)
			if err != nil {
				return nil, err
			}
			normalizedReferences = append(normalizedReferences, normalizedReference)
		}
		if len(normalizedReferences) == 0 {
			return nil, fmt.Errorf("生命周期命令链必须包含命令")
		}
		chain.Commands = normalizedReferences
		chain.CommandIDs = nil
		if chain.ApplicableHooks == nil {
			chain.ApplicableHooks = commonLifecycleHooks(normalizedReferences, knownCommands)
		} else {
			applicableHooks, err := normalizeLifecycleApplicableHooks(chain.ApplicableHooks, nil, true)
			if err != nil {
				return nil, fmt.Errorf("生命周期命令链 %q 的适用范围无效: %w", chain.Name, err)
			}
			chain.ApplicableHooks = applicableHooks
		}
		for _, reference := range normalizedReferences {
			command := knownCommands[reference.CommandID]
			if !lifecycleHooksCover(command.ApplicableHooks, chain.ApplicableHooks) {
				return nil, fmt.Errorf("生命周期命令链 %q 引用的命令 %q 不适用于全部链范围", chain.Name, command.Name)
			}
		}
		normalized = append(normalized, chain)
	}
	return normalized, nil
}

func normalizeLifecycleDefaultChains(defaults map[LifecycleHook]string, chains []LifecycleCommandChain) (map[LifecycleHook]string, error) {
	if defaults == nil {
		knownChains := make(map[string]LifecycleCommandChain, len(chains))
		for _, chain := range chains {
			knownChains[chain.ID] = chain
		}
		defaultChains := DefaultLifecycleDefaultChains()
		for hook, chainID := range defaultChains {
			chain, found := knownChains[chainID]
			if !found || !lifecycleHookIncluded(chain.ApplicableHooks, hook) {
				delete(defaultChains, hook)
			}
		}
		return defaultChains, nil
	}
	knownChains := make(map[string]LifecycleCommandChain, len(chains))
	for _, chain := range chains {
		knownChains[chain.ID] = chain
	}
	normalized := make(map[LifecycleHook]string, len(defaults))
	for hook, chainID := range defaults {
		if !task.IsLifecycleHook(hook) {
			return nil, fmt.Errorf("不支持的生命周期默认钩子: %q", hook)
		}
		chainID = strings.TrimSpace(chainID)
		if chainID == "" {
			continue
		}
		chain, found := knownChains[chainID]
		if !found {
			return nil, fmt.Errorf("生命周期默认链不存在: %q", chainID)
		}
		if !lifecycleHookIncluded(chain.ApplicableHooks, hook) {
			continue
		}
		normalized[hook] = chainID
	}
	return normalized, nil
}

func allLifecycleHooks() []LifecycleHook {
	return []LifecycleHook{
		LifecycleHookBeforeStart,
		LifecycleHookPostStart,
		LifecycleHookBeforeEnd,
		LifecycleHookPostEnd,
		LifecycleHookUpdateTask,
	}
}

func normalizeLifecycleApplicableHooks(hooks, legacyHooks []LifecycleHook, allowEmpty bool) ([]LifecycleHook, error) {
	if hooks == nil {
		hooks = legacyHooks
	}
	normalized := make([]LifecycleHook, 0, len(hooks))
	seen := make(map[LifecycleHook]bool, len(hooks))
	for _, hook := range hooks {
		if !task.IsLifecycleHook(hook) {
			return nil, fmt.Errorf("不支持的生命周期钩子: %q", hook)
		}
		if !seen[hook] {
			seen[hook] = true
			normalized = append(normalized, hook)
		}
	}
	if len(normalized) == 0 && !allowEmpty {
		return nil, fmt.Errorf("至少选择一个生命周期钩子")
	}
	return normalized, nil
}

func normalizeLifecycleCommandReference(reference LifecycleCommandReference, commands map[string]LifecycleCommand) (LifecycleCommandReference, error) {
	reference.CommandID = strings.TrimSpace(reference.CommandID)
	if reference.CommandID == "" {
		return LifecycleCommandReference{}, fmt.Errorf("生命周期命令链引用不存在的命令: %q", reference.CommandID)
	}
	command, found := commands[reference.CommandID]
	if !found {
		return LifecycleCommandReference{}, fmt.Errorf("生命周期命令链引用不存在的命令: %q", reference.CommandID)
	}
	reference.Arguments = normalizeArguments(reference.Arguments)
	if command.Kind == LifecycleCommandKindGitClone {
		arguments, err := normalizeGitCloneArguments(reference.Arguments)
		if err != nil {
			return LifecycleCommandReference{}, err
		}
		reference.Arguments = arguments
	}
	return reference, nil
}

func normalizeGitCloneArguments(arguments []string) ([]string, error) {
	if len(arguments) != 1 {
		return nil, fmt.Errorf("Git 仓库克隆命令必须配置唯一的 dir 参数")
	}
	key, directory, found := strings.Cut(arguments[0], "=")
	if !found || strings.TrimSpace(key) != "dir" {
		return nil, fmt.Errorf("Git 仓库克隆命令参数必须使用 dir=<相对目录>")
	}
	directory = strings.TrimSpace(directory)
	if directory == "" || filepath.IsAbs(directory) {
		return nil, fmt.Errorf("Git 仓库克隆命令的 dir 参数无效")
	}
	directory = filepath.Clean(directory)
	if directory == ".." || strings.HasPrefix(directory, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("Git 仓库克隆命令的 dir 参数无效")
	}
	return []string{"dir=" + directory}, nil
}

func GitCloneDirectory(arguments []string) (string, error) {
	normalized, err := normalizeGitCloneArguments(arguments)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(normalized[0], "dir="), nil
}

func commonLifecycleHooks(references []LifecycleCommandReference, commands map[string]LifecycleCommand) []LifecycleHook {
	common := allLifecycleHooks()
	for _, reference := range references {
		commandHooks := commands[reference.CommandID].ApplicableHooks
		filtered := make([]LifecycleHook, 0, len(common))
		for _, hook := range common {
			if lifecycleHookIncluded(commandHooks, hook) {
				filtered = append(filtered, hook)
			}
		}
		common = filtered
	}
	return common
}

func lifecycleHooksCover(available, required []LifecycleHook) bool {
	for _, hook := range required {
		if !lifecycleHookIncluded(available, hook) {
			return false
		}
	}
	return true
}

func lifecycleHookIncluded(hooks []LifecycleHook, expected LifecycleHook) bool {
	for _, hook := range hooks {
		if hook == expected {
			return true
		}
	}
	return false
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
