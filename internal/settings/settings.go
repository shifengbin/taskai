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

const (
	CurrentPresetVersion               = 5
	DefaultBranchTaskTemplateID        = "preset.task-template.default-branch"
	DefaultLifecyclePresetID           = "preset.lifecycle-preset.default"
	LifecycleChainIterationsAIID       = "preset.lifecycle-chain.iterations-ai"
	LifecycleChainUpdateRepositoriesID = "preset.lifecycle-chain.update-repositories"
	IterationsAIRepository             = "git@gitlab.jiandan100.cn:webdev/iterations-ai.git"
)

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
	StatusManagementModeTitleChange  StatusManagementMode = "title-change"
	StatusManagementModeOutputChange StatusManagementMode = "output-change"
	StatusManagementModeHTTP         StatusManagementMode = "http"
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
	LifecycleCommandKindCustom              LifecycleCommandKind = "custom"
	LifecycleCommandKindCreateWorkspace     LifecycleCommandKind = "create-workspace"
	LifecycleCommandKindDeleteWorkspace     LifecycleCommandKind = "delete-workspace"
	LifecycleCommandKindGitClone            LifecycleCommandKind = "git-clone"
	LifecycleCommandKindGitCloneRepository  LifecycleCommandKind = "git-clone-repository"
	LifecycleCommandKindManifestFile        LifecycleCommandKind = "manifest-file"
	LifecycleCommandKindUpdateDefaultBranch LifecycleCommandKind = "update-default-branch"

	LifecycleCommandCreateWorkspaceID     = "system.lifecycle.create-workspace"
	LifecycleCommandDeleteWorkspaceID     = "system.lifecycle.delete-workspace"
	LifecycleCommandGitCloneID            = "system.lifecycle.git-clone"
	LifecycleCommandGitCloneRepositoryID  = "system.lifecycle.git-clone-repository"
	LifecycleCommandManifestFileID        = "system.lifecycle.manifest-file"
	LifecycleCommandUpdateDefaultBranchID = "system.lifecycle.update-default-branch"
	LifecycleChainCreateWorkspaceID       = "system.lifecycle-chain.create-workspace"
	LifecycleChainDeleteWorkspaceID       = "system.lifecycle-chain.delete-workspace"
)

type LifecycleCommandChainArgumentMode string

const (
	LifecycleCommandChainArgumentModeEnabled  LifecycleCommandChainArgumentMode = "enabled"
	LifecycleCommandChainArgumentModeDisabled LifecycleCommandChainArgumentMode = "disabled"
)

type LifecycleCommand struct {
	ID                string                            `json:"id"`
	Kind              LifecycleCommandKind              `json:"kind"`
	Name              string                            `json:"name"`
	Command           string                            `json:"command,omitempty"`
	Arguments         []string                          `json:"arguments"`
	ChainArgumentMode LifecycleCommandChainArgumentMode `json:"chainArgumentMode"`
	Documentation     string                            `json:"documentation,omitempty"`
	ApplicableHooks   []LifecycleHook                   `json:"applicableHooks"`
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

// LifecyclePreset is a named collection of lifecycle command-chain selections.
// Tasks store a copy of Chains instead of referencing a preset.
type LifecyclePreset struct {
	ID     string                   `json:"id"`
	Name   string                   `json:"name"`
	Chains map[LifecycleHook]string `json:"chains"`
}

type TaskMenuItem struct {
	ID                          string           `json:"id"`
	Kind                        TaskMenuItemKind `json:"kind"`
	Name                        string           `json:"name"`
	Command                     string           `json:"command,omitempty"`
	Arguments                   []string         `json:"arguments,omitempty"`
	ShowTerminal                bool             `json:"showTerminal"`
	DisableTaskAIMouseClipboard bool             `json:"disableTaskAIMouseClipboard"`
	BeforeScript                *TaskScript      `json:"beforeScript,omitempty"`
	AfterScript                 *TaskScript      `json:"afterScript,omitempty"`
}

type Settings struct {
	WorkspaceRoot                string                   `json:"workspaceRoot"`
	TaskTreeWidth                int                      `json:"taskTreeWidth"`
	ColorScheme                  ColorScheme              `json:"colorScheme"`
	ShellPath                    string                   `json:"shellPath"`
	TaskMenuItems                []TaskMenuItem           `json:"taskMenuItems"`
	ActiveTaskStatus             TaskStatus               `json:"activeTaskStatus"`
	StatusManagementMode         StatusManagementMode     `json:"statusManagementMode"`
	StatusManagementHTTPPort     int                      `json:"statusManagementHTTPPort"`
	HTTPServiceEnabled           bool                     `json:"httpServiceEnabled"`
	LifecycleCommands            []LifecycleCommand       `json:"lifecycleCommands"`
	LifecycleChains              []LifecycleCommandChain  `json:"lifecycleChains"`
	LifecyclePresets             []LifecyclePreset        `json:"lifecyclePresets"`
	DefaultLifecyclePresetID     string                   `json:"defaultLifecyclePresetId"`
	LegacyLifecycleDefaultChains map[LifecycleHook]string `json:"lifecycleDefaultChains,omitempty"`
	TaskTemplates                []task.TaskTemplate      `json:"taskTemplates"`
	ActiveTaskTemplateID         string                   `json:"activeTaskTemplateId"`
	PresetVersion                int                      `json:"presetVersion"`
}

func Default(applicationDataDirectory string) Settings {
	return Settings{
		WorkspaceRoot:            filepath.Join(applicationDataDirectory, "workspaces"),
		TaskTreeWidth:            DefaultTaskTreeWidth,
		ColorScheme:              DefaultColorScheme,
		ShellPath:                DefaultShellPath(),
		TaskMenuItems:            DefaultTaskMenuItems(),
		ActiveTaskStatus:         DefaultActiveTaskStatus,
		StatusManagementMode:     DefaultStatusManagementMode,
		LifecycleCommands:        DefaultLifecycleCommands(),
		LifecycleChains:          DefaultLifecycleChains(),
		LifecyclePresets:         DefaultLifecyclePresets(),
		DefaultLifecyclePresetID: DefaultLifecyclePresetID,
		TaskTemplates:            DefaultTaskTemplates(),
		ActiveTaskTemplateID:     DefaultBranchTaskTemplateID,
		PresetVersion:            CurrentPresetVersion,
	}
}

func (current Settings) ActiveTaskTemplate() *task.TaskTemplate {
	activeID := strings.TrimSpace(current.ActiveTaskTemplateID)
	if activeID == "" {
		return nil
	}
	for _, candidate := range current.TaskTemplates {
		if candidate.ID != activeID {
			continue
		}
		copy := candidate
		copy.Fields = append([]task.TaskTemplateField(nil), candidate.Fields...)
		return &copy
	}
	return nil
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
		fixedLifecycleCommand(LifecycleCommandGitCloneRepositoryID),
		fixedLifecycleCommand(LifecycleCommandManifestFileID),
		fixedLifecycleCommand(LifecycleCommandUpdateDefaultBranchID),
	}
}

func DefaultLifecycleChains() []LifecycleCommandChain {
	return defaultLifecycleChainsVersionFour()
}

func defaultLifecycleChainsVersionFour() []LifecycleCommandChain {
	return []LifecycleCommandChain{
		{ID: LifecycleChainCreateWorkspaceID, Name: "创建任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		{ID: LifecycleChainDeleteWorkspaceID, Name: "删除任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandDeleteWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookPostEnd}},
		{ID: LifecycleChainIterationsAIID, Name: "iterations-ai", Commands: []LifecycleCommandReference{
			{CommandID: LifecycleCommandUpdateDefaultBranchID, Arguments: []string{}},
			{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + IterationsAIRepository}},
			{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
		}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		{ID: LifecycleChainUpdateRepositoriesID, Name: "更新仓库", Commands: []LifecycleCommandReference{
			{CommandID: LifecycleCommandUpdateDefaultBranchID, Arguments: []string{}},
			{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
		}, ApplicableHooks: []LifecycleHook{LifecycleHookUpdateTask}},
	}
}

func defaultLifecycleChainsVersionThree() []LifecycleCommandChain {
	return []LifecycleCommandChain{
		{ID: LifecycleChainCreateWorkspaceID, Name: "创建任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		{ID: LifecycleChainDeleteWorkspaceID, Name: "删除任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandDeleteWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookPostEnd}},
		{ID: LifecycleChainIterationsAIID, Name: "iterations-ai", Commands: []LifecycleCommandReference{
			{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + IterationsAIRepository}},
			{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
		}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		{ID: LifecycleChainUpdateRepositoriesID, Name: "更新仓库", Commands: []LifecycleCommandReference{
			{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
		}, ApplicableHooks: []LifecycleHook{LifecycleHookUpdateTask}},
	}
}

func defaultLifecycleChainsVersionTwo() []LifecycleCommandChain {
	return []LifecycleCommandChain{
		{ID: LifecycleChainCreateWorkspaceID, Name: "创建任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		{ID: LifecycleChainDeleteWorkspaceID, Name: "删除任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandDeleteWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookPostEnd}},
		{ID: LifecycleChainIterationsAIID, Name: "iterations-ai", Commands: []LifecycleCommandReference{
			{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + IterationsAIRepository}},
			{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
		}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		{ID: LifecycleChainUpdateRepositoriesID, Name: "更新仓库", Commands: []LifecycleCommandReference{
			{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
		}, ApplicableHooks: []LifecycleHook{LifecycleHookUpdateTask}},
	}
}

func defaultLifecycleChainsVersionOne() []LifecycleCommandChain {
	return []LifecycleCommandChain{
		{ID: LifecycleChainCreateWorkspaceID, Name: "创建任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		{ID: LifecycleChainDeleteWorkspaceID, Name: "删除任务工作目录", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandDeleteWorkspaceID, Arguments: []string{}}}, ApplicableHooks: []LifecycleHook{LifecycleHookPostEnd}},
		{ID: LifecycleChainIterationsAIID, Name: "iterations-ai", Commands: []LifecycleCommandReference{
			{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + IterationsAIRepository}},
			{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
		}, ApplicableHooks: []LifecycleHook{LifecycleHookPostStart}},
		{ID: LifecycleChainUpdateRepositoriesID, Name: "更新仓库", Commands: []LifecycleCommandReference{
			{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
			{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
		}, ApplicableHooks: []LifecycleHook{LifecycleHookUpdateTask}},
	}
}

func DefaultTaskTemplates() []task.TaskTemplate {
	return []task.TaskTemplate{{
		ID:   DefaultBranchTaskTemplateID,
		Name: "默认分支",
		Fields: []task.TaskTemplateField{{
			Key:          "branch",
			DisplayName:  "默认分支",
			InputType:    task.TaskTemplateFieldInputString,
			Required:     true,
			DefaultValue: "",
		}},
	}}
}

func ApplyPresetMigration(next Settings) (Settings, bool) {
	if next.PresetVersion >= CurrentPresetVersion {
		if next.LegacyLifecycleDefaultChains == nil {
			return next, false
		}
		next.LegacyLifecycleDefaultChains = nil
		return next, true
	}

	if next.PresetVersion < 1 {
		chains := make(map[string]bool, len(next.LifecycleChains))
		for _, chain := range next.LifecycleChains {
			chains[chain.ID] = true
		}
		for _, preset := range DefaultLifecycleChains() {
			if preset.ID != LifecycleChainIterationsAIID && preset.ID != LifecycleChainUpdateRepositoriesID {
				continue
			}
			if !chains[preset.ID] {
				next.LifecycleChains = append(next.LifecycleChains, preset)
			}
		}
	}
	if next.PresetVersion < 2 {
		migrateIterationsAIHook(&next)
	}
	if next.PresetVersion < 3 {
		ensureDefaultBranchTemplate(&next)
	}
	if next.PresetVersion < 4 {
		migrateDefaultBranchLifecycleCommands(&next)
	}
	if next.PresetVersion < 5 {
		chains := next.LegacyLifecycleDefaultChains
		if chains == nil {
			chains = DefaultLifecyclePresetChains()
		}
		next.LifecyclePresets = []LifecyclePreset{{
			ID:     DefaultLifecyclePresetID,
			Name:   "默认预设",
			Chains: copyLifecyclePresetChains(chains),
		}}
		next.DefaultLifecyclePresetID = DefaultLifecyclePresetID
		next.LegacyLifecycleDefaultChains = nil
	}
	next.PresetVersion = CurrentPresetVersion
	return next, true
}

func ensureDefaultBranchTemplate(next *Settings) {
	defaultTemplateID := ""
	for _, template := range next.TaskTemplates {
		if template.ID == DefaultBranchTaskTemplateID {
			defaultTemplateID = template.ID
			break
		}
		if template.Name == "默认分支" {
			defaultTemplateID = template.ID
		}
	}
	if defaultTemplateID == "" {
		defaultTemplate := DefaultTaskTemplates()[0]
		next.TaskTemplates = append(next.TaskTemplates, defaultTemplate)
		defaultTemplateID = defaultTemplate.ID
	}
	if strings.TrimSpace(next.ActiveTaskTemplateID) == "" {
		next.ActiveTaskTemplateID = defaultTemplateID
	}
}

func migrateIterationsAIHook(next *Settings) {
	var target LifecycleCommandChain
	for _, preset := range defaultLifecycleChainsVersionTwo() {
		if preset.ID == LifecycleChainIterationsAIID {
			target = preset
			break
		}
	}
	for index := range next.LifecycleChains {
		chain := &next.LifecycleChains[index]
		if !isVersionOneIterationsAIChain(*chain) {
			continue
		}
		chain.ApplicableHooks = append([]LifecycleHook(nil), target.ApplicableHooks...)
		return
	}
}

func migrateDefaultBranchLifecycleCommands(next *Settings) {
	for index := range next.LifecycleChains {
		chain := &next.LifecycleChains[index]
		if chain.ID != LifecycleChainIterationsAIID && chain.ID != LifecycleChainUpdateRepositoriesID {
			continue
		}
		if !isVersionThreeRepositoryPresetChain(*chain) {
			continue
		}
		chain.Commands = append([]LifecycleCommandReference{{CommandID: LifecycleCommandUpdateDefaultBranchID, Arguments: []string{}}}, chain.Commands...)
	}
}

func isVersionOneIterationsAIChain(chain LifecycleCommandChain) bool {
	if chain.ID != LifecycleChainIterationsAIID || chain.Name != "iterations-ai" || !sameLifecycleHooks(chain.ApplicableHooks, []LifecycleHook{LifecycleHookPostStart}) {
		return false
	}
	for _, preset := range defaultLifecycleChainsVersionOne() {
		if preset.ID == LifecycleChainIterationsAIID {
			return sameLifecycleCommandReferences(chain.Commands, preset.Commands)
		}
	}
	return false
}

func isVersionThreeRepositoryPresetChain(chain LifecycleCommandChain) bool {
	for _, preset := range defaultLifecycleChainsVersionThree() {
		if chain.ID == preset.ID && chain.Name == preset.Name && sameLifecycleHooks(chain.ApplicableHooks, preset.ApplicableHooks) {
			return sameLifecycleCommandReferences(chain.Commands, preset.Commands)
		}
	}
	return false
}

func sameLifecycleHooks(left, right []LifecycleHook) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameLifecycleCommandReferences(left, right []LifecycleCommandReference) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].CommandID != right[index].CommandID || len(left[index].Arguments) != len(right[index].Arguments) {
			return false
		}
		for argumentIndex := range left[index].Arguments {
			if left[index].Arguments[argumentIndex] != right[index].Arguments[argumentIndex] {
				return false
			}
		}
	}
	return true
}

func DefaultLifecyclePresetChains() map[LifecycleHook]string {
	return map[LifecycleHook]string{
		LifecycleHookBeforeStart: LifecycleChainCreateWorkspaceID,
		LifecycleHookPostEnd:     LifecycleChainDeleteWorkspaceID,
	}
}

func DefaultLifecyclePresets() []LifecyclePreset {
	return []LifecyclePreset{{
		ID:     DefaultLifecyclePresetID,
		Name:   "默认预设",
		Chains: DefaultLifecyclePresetChains(),
	}}
}

func (current Settings) DefaultLifecyclePresetChains() map[LifecycleHook]string {
	defaultID := strings.TrimSpace(current.DefaultLifecyclePresetID)
	if defaultID == "" {
		return map[LifecycleHook]string{}
	}
	for _, preset := range current.LifecyclePresets {
		if preset.ID == defaultID {
			return copyLifecyclePresetChains(preset.Chains)
		}
	}
	return map[LifecycleHook]string{}
}

func copyLifecyclePresetChains(chains map[LifecycleHook]string) map[LifecycleHook]string {
	copy := make(map[LifecycleHook]string, len(chains))
	for hook, chainID := range chains {
		if chainID != "" {
			copy[hook] = chainID
		}
	}
	return copy
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
	if next.StatusManagementMode != StatusManagementModeTitleChange && next.StatusManagementMode != StatusManagementModeOutputChange && next.StatusManagementMode != StatusManagementModeHTTP {
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
	next, err = NormalizeTaskTemplates(next)
	if err != nil {
		return Settings{}, err
	}
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

func NormalizeTaskTemplates(next Settings) (Settings, error) {
	templates, err := task.ValidateTaskTemplates(next.TaskTemplates)
	if err != nil {
		return Settings{}, err
	}
	next.TaskTemplates = templates
	next.ActiveTaskTemplateID = strings.TrimSpace(next.ActiveTaskTemplateID)
	if next.ActiveTaskTemplateID == "" {
		return next, nil
	}
	for _, template := range next.TaskTemplates {
		if template.ID == next.ActiveTaskTemplateID {
			return next, nil
		}
	}
	return Settings{}, fmt.Errorf("当前任务模板不存在: %q", next.ActiveTaskTemplateID)
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
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindCreateWorkspace, Name: "创建任务工作目录", Arguments: []string{}, ChainArgumentMode: LifecycleCommandChainArgumentModeDisabled, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart}}
	case LifecycleCommandDeleteWorkspaceID:
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindDeleteWorkspace, Name: "删除任务工作目录", Arguments: []string{}, ChainArgumentMode: LifecycleCommandChainArgumentModeDisabled, ApplicableHooks: []LifecycleHook{LifecycleHookPostEnd}}
	case LifecycleCommandGitCloneID:
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindGitClone, Name: "Git 仓库克隆", Arguments: []string{}, ChainArgumentMode: LifecycleCommandChainArgumentModeEnabled, Documentation: "参数可留空；留空时每个内置 Git 项目将克隆到任务工作目录下的 <项目名称>。填写时使用 dir=<相对目录>，将克隆到任务工作目录下的 <dir>/<项目名称>；目标已存在时跳过。指定分支存在时克隆该分支，不存在时从远程默认分支创建同名本地分支。", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart, LifecycleHookBeforeEnd, LifecycleHookUpdateTask}}
	case LifecycleCommandGitCloneRepositoryID:
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindGitCloneRepository, Name: "克隆指定 Git 仓库", Arguments: []string{}, ChainArgumentMode: LifecycleCommandChainArgumentModeEnabled, Documentation: "参数：repository=<仓库地址>（必填）；dir=<相对目录>（可选）。仓库直接克隆到任务工作目录或指定子目录本身，不读取 Git 附加信息。目标必须为空目录，非空目录会失败。分支由此前的更新默认分支命令决定；未执行时使用远程默认分支。", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart}}
	case LifecycleCommandManifestFileID:
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindManifestFile, Name: "生成清单文件", Arguments: []string{}, ChainArgumentMode: LifecycleCommandChainArgumentModeEnabled, Documentation: "参数可留空；dir=<相对目录>（可选）指定任务工作目录内的输出目录，name=<文件名>（可选）指定清单文件名。默认生成 <任务工作目录>/manifest.yaml。", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart, LifecycleHookUpdateTask}}
	case LifecycleCommandUpdateDefaultBranchID:
		return LifecycleCommand{ID: id, Kind: LifecycleCommandKindUpdateDefaultBranch, Name: "更新默认分支", Arguments: []string{}, ChainArgumentMode: LifecycleCommandChainArgumentModeEnabled, Documentation: "参数可留空；templateField=<字段键>（可选）指定从任务模板读取默认分支的字段，省略时使用 branch。仅在当前命令链执行期间，将空的内置 Git 项目 branch 参数更新为该字段值。", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart, LifecycleHookUpdateTask}}
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
		if !item.ShowTerminal {
			item.DisableTaskAIMouseClipboard = false
		}
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
	legacyDefaults, err := normalizeLegacyLifecycleDefaultChains(next.LegacyLifecycleDefaultChains, lifecycleChains)
	if err != nil {
		return Settings{}, err
	}
	next.LegacyLifecycleDefaultChains = legacyDefaults
	lifecyclePresets, defaultPresetID, err := normalizeLifecyclePresets(next.LifecyclePresets, next.DefaultLifecyclePresetID, lifecycleChains)
	if err != nil {
		return Settings{}, err
	}
	next.LifecyclePresets = lifecyclePresets
	next.DefaultLifecyclePresetID = defaultPresetID
	return next, nil
}

func normalizeLifecycleCommands(commands []LifecycleCommand) ([]LifecycleCommand, error) {
	normalized := make([]LifecycleCommand, 0, len(commands)+4)
	seen := make(map[string]bool, len(commands)+4)
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
		chainArgumentMode, err := normalizeLifecycleCommandChainArgumentMode(command.ChainArgumentMode)
		if err != nil {
			return nil, fmt.Errorf("自定义生命周期命令 %q 的链级参数模式无效: %w", command.Name, err)
		}
		command.ChainArgumentMode = chainArgumentMode
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

func normalizeLifecycleCommandChainArgumentMode(mode LifecycleCommandChainArgumentMode) (LifecycleCommandChainArgumentMode, error) {
	switch mode {
	case "":
		return LifecycleCommandChainArgumentModeEnabled, nil
	case LifecycleCommandChainArgumentModeEnabled, LifecycleCommandChainArgumentModeDisabled:
		return mode, nil
	default:
		return "", fmt.Errorf("不支持的链级参数模式: %q", mode)
	}
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

func normalizeLegacyLifecycleDefaultChains(defaults map[LifecycleHook]string, chains []LifecycleCommandChain) (map[LifecycleHook]string, error) {
	if defaults == nil {
		return nil, nil
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
			return nil, fmt.Errorf("旧生命周期默认链不存在: %q", chainID)
		}
		if !lifecycleHookIncluded(chain.ApplicableHooks, hook) {
			return nil, fmt.Errorf("旧生命周期默认链 %q 不适用于 %s", chain.Name, hook)
		}
		normalized[hook] = chainID
	}
	return normalized, nil
}

func normalizeLifecyclePresets(presets []LifecyclePreset, defaultPresetID string, chains []LifecycleCommandChain) ([]LifecyclePreset, string, error) {
	knownChains := make(map[string]LifecycleCommandChain, len(chains))
	for _, chain := range chains {
		knownChains[chain.ID] = chain
	}
	normalized := make([]LifecyclePreset, 0, len(presets))
	seenIDs := make(map[string]bool, len(presets))
	seenNames := make(map[string]bool, len(presets))
	for _, preset := range presets {
		preset.ID = strings.TrimSpace(preset.ID)
		preset.Name = strings.TrimSpace(preset.Name)
		if preset.ID == "" || preset.Name == "" {
			return nil, "", fmt.Errorf("生命周期预设 ID 和名称不能为空")
		}
		if seenIDs[preset.ID] {
			return nil, "", fmt.Errorf("生命周期预设 ID 重复: %q", preset.ID)
		}
		nameKey := strings.ToLower(preset.Name)
		if seenNames[nameKey] {
			return nil, "", fmt.Errorf("生命周期预设名称重复: %q", preset.Name)
		}
		seenIDs[preset.ID] = true
		seenNames[nameKey] = true

		normalizedChains := make(map[LifecycleHook]string, len(preset.Chains))
		for hook, chainID := range preset.Chains {
			if !task.IsLifecycleHook(hook) {
				return nil, "", fmt.Errorf("生命周期预设 %q 包含不支持的钩子: %q", preset.Name, hook)
			}
			chainID = strings.TrimSpace(chainID)
			if chainID == "" {
				continue
			}
			chain, found := knownChains[chainID]
			if !found {
				return nil, "", fmt.Errorf("生命周期预设 %q 引用的命令链不存在: %q", preset.Name, chainID)
			}
			if !lifecycleHookIncluded(chain.ApplicableHooks, hook) {
				return nil, "", fmt.Errorf("生命周期预设 %q 引用的命令链 %q 不适用于 %s", preset.Name, chain.Name, hook)
			}
			normalizedChains[hook] = chainID
		}
		preset.Chains = normalizedChains
		normalized = append(normalized, preset)
	}

	defaultPresetID = strings.TrimSpace(defaultPresetID)
	if defaultPresetID == "" {
		return normalized, "", nil
	}
	if !seenIDs[defaultPresetID] {
		return nil, "", fmt.Errorf("默认生命周期预设不存在: %q", defaultPresetID)
	}
	return normalized, defaultPresetID, nil
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
	if command.Kind == LifecycleCommandKindGitCloneRepository {
		arguments, err := normalizeGitCloneRepositoryArguments(reference.Arguments)
		if err != nil {
			return LifecycleCommandReference{}, err
		}
		reference.Arguments = arguments
	}
	if command.Kind == LifecycleCommandKindManifestFile {
		arguments, err := normalizeManifestFileArguments(reference.Arguments)
		if err != nil {
			return LifecycleCommandReference{}, err
		}
		reference.Arguments = arguments
	}
	if command.Kind == LifecycleCommandKindUpdateDefaultBranch {
		arguments, err := normalizeUpdateDefaultBranchArguments(reference.Arguments)
		if err != nil {
			return LifecycleCommandReference{}, err
		}
		reference.Arguments = arguments
	}
	return reference, nil
}

func normalizeGitCloneArguments(arguments []string) ([]string, error) {
	if len(arguments) == 0 {
		return []string{}, nil
	}
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
	if len(normalized) == 0 {
		return ".", nil
	}
	return strings.TrimPrefix(normalized[0], "dir="), nil
}

type GitCloneRepositoryParameters struct {
	Repository string
	Directory  string
}

func GitCloneRepositoryArguments(arguments []string) (GitCloneRepositoryParameters, error) {
	normalized, err := normalizeGitCloneRepositoryArguments(arguments)
	if err != nil {
		return GitCloneRepositoryParameters{}, err
	}
	parameters := GitCloneRepositoryParameters{Directory: "."}
	for _, argument := range normalized {
		key, value, _ := strings.Cut(argument, "=")
		switch key {
		case "repository":
			parameters.Repository = value
		case "dir":
			parameters.Directory = value
		}
	}
	return parameters, nil
}

type ManifestFileParameters struct {
	Directory string
	Name      string
}

func ManifestFileArguments(arguments []string) (ManifestFileParameters, error) {
	normalized, err := normalizeManifestFileArguments(arguments)
	if err != nil {
		return ManifestFileParameters{}, err
	}
	parameters := ManifestFileParameters{Directory: ".", Name: "manifest.yaml"}
	for _, argument := range normalized {
		key, value, _ := strings.Cut(argument, "=")
		switch key {
		case "dir":
			parameters.Directory = value
		case "name":
			parameters.Name = value
		}
	}
	return parameters, nil
}

func UpdateDefaultBranchTemplateField(arguments []string) (string, error) {
	normalized, err := normalizeUpdateDefaultBranchArguments(arguments)
	if err != nil {
		return "", err
	}
	if len(normalized) == 0 {
		return "branch", nil
	}
	return strings.TrimPrefix(normalized[0], "templateField="), nil
}

func normalizeUpdateDefaultBranchArguments(arguments []string) ([]string, error) {
	if len(arguments) == 0 {
		return []string{}, nil
	}
	if len(arguments) != 1 {
		return nil, fmt.Errorf("更新默认分支命令必须配置唯一的 templateField 参数")
	}
	key, value, found := strings.Cut(strings.TrimSpace(arguments[0]), "=")
	if !found || strings.TrimSpace(key) != "templateField" {
		return nil, fmt.Errorf("更新默认分支命令参数必须使用 templateField=<字段键>")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("更新默认分支命令的 templateField 参数不能为空")
	}
	return []string{"templateField=" + value}, nil
}

func normalizeManifestFileArguments(arguments []string) ([]string, error) {
	var directory, name string
	seenDirectory := false
	seenName := false
	for _, argument := range normalizeArguments(arguments) {
		key, value, found := strings.Cut(argument, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !found {
			return nil, fmt.Errorf("生成清单文件命令参数必须使用 dir=<相对目录> 或 name=<文件名>")
		}
		switch key {
		case "dir":
			if seenDirectory || value == "" || filepath.IsAbs(value) {
				return nil, fmt.Errorf("生成清单文件命令的 dir 参数无效")
			}
			directory = filepath.Clean(value)
			if directory == ".." || strings.HasPrefix(directory, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("生成清单文件命令的 dir 参数无效")
			}
			seenDirectory = true
		case "name":
			if seenName || value == "" || filepath.IsAbs(value) || value == "." || value == ".." || strings.ContainsAny(value, `/\\`) || filepath.Base(value) != value {
				return nil, fmt.Errorf("生成清单文件命令的 name 参数无效")
			}
			name = value
			seenName = true
		default:
			return nil, fmt.Errorf("生成清单文件命令不支持参数: %s", key)
		}
	}
	normalized := make([]string, 0, 2)
	if seenDirectory {
		normalized = append(normalized, "dir="+directory)
	}
	if seenName {
		normalized = append(normalized, "name="+name)
	}
	return normalized, nil
}

func normalizeGitCloneRepositoryArguments(arguments []string) ([]string, error) {
	normalized := make([]string, 0, len(arguments))
	seenRepository := false
	seenDirectory := false
	for _, argument := range normalizeArguments(arguments) {
		key, value, found := strings.Cut(argument, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !found {
			return nil, fmt.Errorf("克隆指定 Git 仓库命令参数必须使用 repository=<仓库地址> 或 dir=<相对目录>")
		}
		switch key {
		case "repository":
			if seenRepository || value == "" {
				return nil, fmt.Errorf("克隆指定 Git 仓库命令必须配置唯一且非空的 repository 参数")
			}
			seenRepository = true
			normalized = append(normalized, "repository="+value)
		case "dir":
			if seenDirectory || value == "" || filepath.IsAbs(value) {
				return nil, fmt.Errorf("克隆指定 Git 仓库命令的 dir 参数无效")
			}
			value = filepath.Clean(value)
			if value == ".." || strings.HasPrefix(value, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("克隆指定 Git 仓库命令的 dir 参数无效")
			}
			seenDirectory = true
			normalized = append(normalized, "dir="+value)
		default:
			return nil, fmt.Errorf("克隆指定 Git 仓库命令不支持参数: %s", key)
		}
	}
	if !seenRepository {
		return nil, fmt.Errorf("克隆指定 Git 仓库命令必须配置 repository=<仓库地址>")
	}
	return normalized, nil
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
