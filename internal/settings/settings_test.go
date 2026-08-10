package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"taskai/internal/task"
)

func TestDefaultUsesApplicationDataWorkspaces(t *testing.T) {
	applicationDataDirectory := filepath.Join("testdata", "taskai")

	settings := Default(applicationDataDirectory)

	if settings.WorkspaceRoot != filepath.Join(applicationDataDirectory, "workspaces") {
		t.Errorf("Default() WorkspaceRoot = %q", settings.WorkspaceRoot)
	}
	if settings.TaskTreeWidth != DefaultTaskTreeWidth {
		t.Errorf("Default() TaskTreeWidth = %d, want %d", settings.TaskTreeWidth, DefaultTaskTreeWidth)
	}
	if settings.StatusManagementMode != StatusManagementModeTitleChange {
		t.Errorf("Default() StatusManagementMode = %q，期望 %q", settings.StatusManagementMode, StatusManagementModeTitleChange)
	}
	if settings.StatusManagementHTTPPort != 0 {
		t.Errorf("Default() StatusManagementHTTPPort = %d，期望 0", settings.StatusManagementHTTPPort)
	}
	if settings.HTTPServiceEnabled {
		t.Error("Default() HTTPServiceEnabled = true，期望 false")
	}
}

func TestValidateNormalizesTerminalFontFamilyWithoutRequiringInstalledFont(t *testing.T) {
	current := Default(t.TempDir())
	current.TerminalFontFamily = "  已移除的终端字体  "

	validated, err := Validate(current)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got, want := validated.TerminalFontFamily, "已移除的终端字体"; got != want {
		t.Fatalf("Validate() TerminalFontFamily = %q, want %q", got, want)
	}
}

func TestDefaultUsesTerminalFontSize(t *testing.T) {
	settings := Default(t.TempDir())

	if got, want := settings.TerminalFontSize, DefaultTerminalFontSize; got != want {
		t.Fatalf("Default() TerminalFontSize = %d, want %d", got, want)
	}
}

func TestDefaultUsesCurrentDarkTerminalTheme(t *testing.T) {
	current := Default(t.TempDir())

	if got, want := current.TerminalTheme, DefaultTerminalTheme(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Default() TerminalTheme = %#v, want %#v", got, want)
	}
}

func TestNormalizeTerminalThemePreservesValidColorsAndRestoresInvalidValues(t *testing.T) {
	current := TerminalTheme{
		Background:          "#102030",
		Foreground:          "#invalid",
		Cursor:              "#abcdef",
		SelectionBackground: "#1234567f",
	}

	normalized := NormalizeTerminalTheme(current)
	defaults := DefaultTerminalTheme()
	if got, want := normalized.Background, "#102030"; got != want {
		t.Fatalf("NormalizeTerminalTheme() Background = %q, want %q", got, want)
	}
	if got, want := normalized.Cursor, "#ABCDEF"; got != want {
		t.Fatalf("NormalizeTerminalTheme() Cursor = %q, want %q", got, want)
	}
	if got, want := normalized.SelectionBackground, "#1234567F"; got != want {
		t.Fatalf("NormalizeTerminalTheme() SelectionBackground = %q, want %q", got, want)
	}
	if got, want := normalized.Foreground, defaults.Foreground; got != want {
		t.Fatalf("NormalizeTerminalTheme() Foreground = %q, want %q", got, want)
	}
	if got, want := normalized.BrightCyan, defaults.BrightCyan; got != want {
		t.Fatalf("NormalizeTerminalTheme() BrightCyan = %q, want %q", got, want)
	}
}

func TestValidateNormalizesTerminalFontSize(t *testing.T) {
	tests := []struct {
		name string
		size int
		want int
	}{
		{name: "旧设置缺少字段", size: 0, want: DefaultTerminalFontSize},
		{name: "小于最小值", size: MinimumTerminalFontSize - 1, want: MinimumTerminalFontSize},
		{name: "大于最大值", size: MaximumTerminalFontSize + 1, want: MaximumTerminalFontSize},
		{name: "范围内字号", size: 16, want: 16},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := Default(t.TempDir())
			current.TerminalFontSize = test.size

			validated, err := Validate(current)
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if got := validated.TerminalFontSize; got != test.want {
				t.Fatalf("Validate() TerminalFontSize = %d, want %d", got, test.want)
			}
		})
	}
}

func TestDefaultIncludesFixedTaskMenuItems(t *testing.T) {
	settings := Default(t.TempDir())

	if len(settings.TaskMenuItems) != 3 {
		t.Fatalf("默认任务菜单项数量 = %d，期望 3", len(settings.TaskMenuItems))
	}
	if settings.TaskMenuItems[0].ID != TaskMenuItemEditTaskID || settings.TaskMenuItems[1].ID != TaskMenuItemCreateTerminalID || settings.TaskMenuItems[2].ID != TaskMenuItemOpenFolderID {
		t.Errorf("默认任务菜单项 = %#v", settings.TaskMenuItems)
	}
}

func TestDefaultIncludesLifecycleDirectoryCommandsAndChains(t *testing.T) {
	current := Default(t.TempDir())

	if len(current.LifecycleCommands) != 6 {
		t.Fatalf("默认生命周期命令数量 = %d，期望 6", len(current.LifecycleCommands))
	}
	if current.LifecycleCommands[0].ID != LifecycleCommandCreateWorkspaceID || current.LifecycleCommands[1].ID != LifecycleCommandDeleteWorkspaceID || current.LifecycleCommands[2].ID != LifecycleCommandGitCloneID || current.LifecycleCommands[3].ID != LifecycleCommandGitCloneRepositoryID || current.LifecycleCommands[4].ID != "system.lifecycle.manifest-file" || current.LifecycleCommands[5].ID != "system.lifecycle.update-default-branch" {
		t.Fatalf("默认生命周期命令 = %#v", current.LifecycleCommands)
	}
	if !reflect.DeepEqual(current.LifecycleCommands[0].ApplicableHooks, []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart}) || !reflect.DeepEqual(current.LifecycleCommands[1].ApplicableHooks, []LifecycleHook{LifecycleHookPostEnd}) {
		t.Fatalf("内置命令适用范围 = %#v", current.LifecycleCommands)
	}
	if !reflect.DeepEqual(current.LifecycleChains[0].ApplicableHooks, []LifecycleHook{LifecycleHookBeforeStart}) || !reflect.DeepEqual(current.LifecycleChains[1].ApplicableHooks, []LifecycleHook{LifecycleHookPostEnd}) {
		t.Fatalf("内置命令链适用范围 = %#v", current.LifecycleChains)
	}
	defaultChains := current.DefaultLifecyclePresetChains()
	if got := defaultChains[LifecycleHookBeforeStart]; got != LifecycleChainCreateWorkspaceID {
		t.Fatalf("beforeStart 默认链 = %q，期望 %q", got, LifecycleChainCreateWorkspaceID)
	}
	if got := defaultChains[LifecycleHookPostEnd]; got != LifecycleChainDeleteWorkspaceID {
		t.Fatalf("postEnd 默认链 = %q，期望 %q", got, LifecycleChainDeleteWorkspaceID)
	}
	if got := lifecycleCommandByID(current.LifecycleCommands, LifecycleCommandCreateWorkspaceID).ChainArgumentMode; got != LifecycleCommandChainArgumentModeDisabled {
		t.Fatalf("创建工作目录命令的链级参数模式 = %q，期望禁止", got)
	}
	if got := lifecycleCommandByID(current.LifecycleCommands, LifecycleCommandDeleteWorkspaceID).ChainArgumentMode; got != LifecycleCommandChainArgumentModeDisabled {
		t.Fatalf("删除工作目录命令的链级参数模式 = %q，期望禁止", got)
	}
	if got := lifecycleCommandByID(current.LifecycleCommands, LifecycleCommandGitCloneID).ChainArgumentMode; got != LifecycleCommandChainArgumentModeEnabled {
		t.Fatalf("Git 仓库克隆命令的链级参数模式 = %q，期望允许", got)
	}
}

func TestDefaultIncludesLifecyclePreset(t *testing.T) {
	current := Default(t.TempDir())
	want := LifecyclePreset{
		ID:   DefaultLifecyclePresetID,
		Name: "默认预设",
		Chains: map[task.LifecycleHook]string{
			LifecycleHookBeforeStart: LifecycleChainCreateWorkspaceID,
			LifecycleHookPostEnd:     LifecycleChainDeleteWorkspaceID,
		},
	}

	if current.DefaultLifecyclePresetID != DefaultLifecyclePresetID {
		t.Fatalf("默认生命周期预设 ID = %q，期望 %q", current.DefaultLifecyclePresetID, DefaultLifecyclePresetID)
	}
	if !reflect.DeepEqual(current.LifecyclePresets, []LifecyclePreset{want}) {
		t.Fatalf("默认生命周期预设 = %#v，期望 %#v", current.LifecyclePresets, []LifecyclePreset{want})
	}
	if got := current.DefaultLifecyclePresetChains(); !reflect.DeepEqual(got, want.Chains) {
		t.Fatalf("默认生命周期预设映射 = %#v，期望 %#v", got, want.Chains)
	}
	got := current.DefaultLifecyclePresetChains()
	got[LifecycleHookBeforeStart] = "changed"
	if current.LifecyclePresets[0].Chains[LifecycleHookBeforeStart] != LifecycleChainCreateWorkspaceID {
		t.Fatalf("默认生命周期预设映射必须返回副本: %#v", current.LifecyclePresets[0].Chains)
	}
}

func TestNormalizeLifecyclePresets(t *testing.T) {
	base := Default(t.TempDir())
	base.LifecycleChains = []LifecycleCommandChain{{
		ID:              "chain-prepare",
		Name:            "准备",
		Commands:        []LifecycleCommandReference{{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}}},
		ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart},
	}}

	t.Run("规范化名称并复制映射", func(t *testing.T) {
		chains := map[LifecycleHook]string{LifecycleHookBeforeStart: " chain-prepare "}
		current := base
		current.LifecyclePresets = []LifecyclePreset{{ID: " preset-prepare ", Name: " 准备预设 ", Chains: chains}, {ID: "empty", Name: "空预设", Chains: map[LifecycleHook]string{}}}
		current.DefaultLifecyclePresetID = " preset-prepare "

		normalized, err := NormalizeLifecycle(current)
		if err != nil {
			t.Fatalf("NormalizeLifecycle() error = %v", err)
		}
		if normalized.LifecyclePresets[0].ID != "preset-prepare" || normalized.LifecyclePresets[0].Name != "准备预设" {
			t.Fatalf("规范化预设 = %#v", normalized.LifecyclePresets[0])
		}
		if normalized.DefaultLifecyclePresetID != "preset-prepare" {
			t.Fatalf("规范化默认预设 ID = %q", normalized.DefaultLifecyclePresetID)
		}
		chains[LifecycleHookBeforeStart] = "changed"
		if got := normalized.LifecyclePresets[0].Chains[LifecycleHookBeforeStart]; got != "chain-prepare" {
			t.Fatalf("预设链映射没有深拷贝: %q", got)
		}
	})

	for _, scenario := range []struct {
		name    string
		mutate  func(*Settings)
		wantErr bool
	}{
		{
			name: "空名称",
			mutate: func(current *Settings) {
				current.LifecyclePresets = []LifecyclePreset{{ID: "preset", Name: " ", Chains: map[LifecycleHook]string{}}}
			},
			wantErr: true,
		},
		{
			name: "名称大小写重复",
			mutate: func(current *Settings) {
				current.LifecyclePresets = []LifecyclePreset{{ID: "first", Name: "Preset", Chains: map[LifecycleHook]string{}}, {ID: "second", Name: "preset", Chains: map[LifecycleHook]string{}}}
			},
			wantErr: true,
		},
		{
			name: "引用不存在链",
			mutate: func(current *Settings) {
				current.LifecyclePresets = []LifecyclePreset{{ID: "preset", Name: "预设", Chains: map[LifecycleHook]string{LifecycleHookBeforeStart: "missing"}}}
			},
			wantErr: true,
		},
		{
			name: "引用不适用链",
			mutate: func(current *Settings) {
				current.LifecyclePresets = []LifecyclePreset{{ID: "preset", Name: "预设", Chains: map[LifecycleHook]string{LifecycleHookPostStart: "chain-prepare"}}}
			},
			wantErr: true,
		},
		{
			name: "默认预设不存在",
			mutate: func(current *Settings) {
				current.LifecyclePresets = []LifecyclePreset{{ID: "preset", Name: "预设", Chains: map[LifecycleHook]string{}}}
				current.DefaultLifecyclePresetID = "missing"
			},
			wantErr: true,
		},
		{
			name: "空默认预设有效",
			mutate: func(current *Settings) {
				current.LifecyclePresets = []LifecyclePreset{{ID: "preset", Name: "预设", Chains: map[LifecycleHook]string{}}}
				current.DefaultLifecyclePresetID = ""
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			current := base
			scenario.mutate(&current)
			_, err := NormalizeLifecycle(current)
			if scenario.wantErr && err == nil {
				t.Fatal("NormalizeLifecycle() error = nil，期望拒绝")
			}
			if !scenario.wantErr && err != nil {
				t.Fatalf("NormalizeLifecycle() error = %v", err)
			}
		})
	}
}

func TestDefaultLifecycleConfigurationValidates(t *testing.T) {
	if _, err := Validate(Default(t.TempDir())); err != nil {
		t.Fatalf("Validate(Default()) error = %v", err)
	}
}

func TestDefaultSeedsDefaultBranchTemplateAndRepositoryPresetChains(t *testing.T) {
	current := Default(t.TempDir())

	if current.ActiveTaskTemplateID != "preset.task-template.default-branch" || len(current.TaskTemplates) != 1 {
		t.Fatalf("默认任务模板 = %#v，当前模板 = %q", current.TaskTemplates, current.ActiveTaskTemplateID)
	}
	template := current.TaskTemplates[0]
	if template.ID != "preset.task-template.default-branch" || template.Name != "默认分支" || len(template.Fields) != 1 {
		t.Fatalf("默认分支模板 = %#v", template)
	}
	field := template.Fields[0]
	if field.Key != "branch" || field.DisplayName != "默认分支" || field.InputType != task.TaskTemplateFieldInputString || !field.Required || field.DefaultValue != "" || field.InjectEnvironment {
		t.Fatalf("默认分支字段 = %#v", field)
	}

	chains := map[string]LifecycleCommandChain{}
	for _, chain := range current.LifecycleChains {
		chains[chain.ID] = chain
	}
	iterations, found := chains["preset.lifecycle-chain.iterations-ai"]
	if !found {
		t.Fatalf("缺少 iterations-ai 预置链: %#v", current.LifecycleChains)
	}
	if iterations.Name != "iterations-ai" || !reflect.DeepEqual(iterations.ApplicableHooks, []LifecycleHook{LifecycleHookBeforeStart}) {
		t.Fatalf("iterations-ai 链范围 = %#v", iterations)
	}
	if want := []LifecycleCommandReference{
		{CommandID: "system.lifecycle.update-default-branch", Arguments: []string{}},
		{CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
		{CommandID: LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=git@gitlab.jiandan100.cn:webdev/iterations-ai.git"}},
		{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
		{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
	}; !reflect.DeepEqual(iterations.Commands, want) {
		t.Fatalf("iterations-ai 命令 = %#v，期望 %#v", iterations.Commands, want)
	}

	updateRepositories, found := chains["preset.lifecycle-chain.update-repositories"]
	if !found {
		t.Fatalf("缺少更新仓库预置链: %#v", current.LifecycleChains)
	}
	if updateRepositories.Name != "更新仓库" || !reflect.DeepEqual(updateRepositories.ApplicableHooks, []LifecycleHook{LifecycleHookUpdateTask}) {
		t.Fatalf("更新仓库链范围 = %#v", updateRepositories)
	}
	if want := []LifecycleCommandReference{
		{CommandID: "system.lifecycle.update-default-branch", Arguments: []string{}},
		{CommandID: LifecycleCommandManifestFileID, Arguments: []string{}},
		{CommandID: LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
	}; !reflect.DeepEqual(updateRepositories.Commands, want) {
		t.Fatalf("更新仓库命令 = %#v，期望 %#v", updateRepositories.Commands, want)
	}
	defaultChains := current.DefaultLifecyclePresetChains()
	if defaultChains[LifecycleHookPostStart] != "" || defaultChains[LifecycleHookUpdateTask] != "" {
		t.Fatalf("预置链不应默认选中: %#v", defaultChains)
	}
	createWorkspace := lifecycleCommandByID(current.LifecycleCommands, LifecycleCommandCreateWorkspaceID)
	if createWorkspace == nil || !reflect.DeepEqual(createWorkspace.ApplicableHooks, []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart}) {
		t.Fatalf("创建任务工作目录命令范围 = %#v", createWorkspace)
	}
}

func TestDefaultIncludesUpdateDefaultBranchLifecycleCommand(t *testing.T) {
	current := Default(t.TempDir())
	command := lifecycleCommandByID(current.LifecycleCommands, "system.lifecycle.update-default-branch")
	if command == nil {
		t.Fatal("Default() 未提供更新默认分支系统命令")
	}
	if command.Kind != LifecycleCommandKind("update-default-branch") || command.Name != "更新默认分支" {
		t.Fatalf("更新默认分支系统命令 = %#v", command)
	}
	if command.ChainArgumentMode != LifecycleCommandChainArgumentModeEnabled {
		t.Fatalf("更新默认分支命令的链级参数模式 = %q，期望允许", command.ChainArgumentMode)
	}
	if want := []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart, LifecycleHookUpdateTask}; !reflect.DeepEqual(command.ApplicableHooks, want) {
		t.Fatalf("更新默认分支命令适用范围 = %#v，期望 %#v", command.ApplicableHooks, want)
	}
	if !strings.Contains(command.Documentation, "templateField=<字段键>") || !strings.Contains(command.Documentation, "branch") {
		t.Fatalf("更新默认分支系统命令文档 = %q", command.Documentation)
	}
}

func TestValidateNormalizesUpdateDefaultBranchArguments(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		want      []string
		valid     bool
	}{
		{name: "默认字段", arguments: nil, want: []string{}, valid: true},
		{name: "指定字段", arguments: []string{" templateField=release_branch "}, want: []string{"templateField=release_branch"}, valid: true},
		{name: "空字段", arguments: []string{"templateField= "}},
		{name: "重复字段", arguments: []string{"templateField=branch", "templateField=release"}},
		{name: "未知参数", arguments: []string{"branch=main"}},
		{name: "没有等号", arguments: []string{"branch"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			validated, err := Validate(Settings{
				WorkspaceRoot: t.TempDir(),
				TaskTreeWidth: DefaultTaskTreeWidth,
				LifecycleChains: []LifecycleCommandChain{{
					ID: "default-branch", Name: "默认分支", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart},
					Commands: []LifecycleCommandReference{{CommandID: "system.lifecycle.update-default-branch", Arguments: test.arguments}},
				}},
			})
			if !test.valid {
				if err == nil {
					t.Fatalf("Validate() 对参数 %#v 的错误 = nil，期望拒绝", test.arguments)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if got := validated.LifecycleChains[0].Commands[0].Arguments; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("更新默认分支参数 = %#v，期望 %#v", got, test.want)
			}
		})
	}
}

func TestApplyPresetMigrationAddsDefaultBranchCommandOnlyToUnmodifiedVersionThreePresetChains(t *testing.T) {
	legacy := Default(t.TempDir())
	legacy.PresetVersion = 3
	legacy.LifecycleChains = defaultLifecycleChainsVersionThree()

	modified := legacy
	modified.LifecycleChains = defaultLifecycleChainsVersionThree()
	chain := lifecycleChainByID(modified.LifecycleChains, LifecycleChainIterationsAIID)
	chain.Name = "已修改 iterations-ai"

	migrated, changed := ApplyPresetMigration(legacy)
	if !changed || migrated.PresetVersion != CurrentPresetVersion {
		t.Fatalf("预置版本迁移 = (%d, %t)，期望 (%d, true)", migrated.PresetVersion, changed, CurrentPresetVersion)
	}
	for _, chainID := range []string{LifecycleChainIterationsAIID, LifecycleChainUpdateRepositoriesID} {
		chain := lifecycleChainByID(migrated.LifecycleChains, chainID)
		if chain == nil || len(chain.Commands) == 0 || chain.Commands[0].CommandID != "system.lifecycle.update-default-branch" {
			t.Fatalf("%s 预置链未前置更新默认分支命令: %#v", chainID, chain)
		}
	}
	for _, chainID := range []string{LifecycleChainCreateWorkspaceID, LifecycleChainDeleteWorkspaceID} {
		chain := lifecycleChainByID(migrated.LifecycleChains, chainID)
		if chain == nil || len(chain.Commands) != 1 || chain.Commands[0].CommandID == "system.lifecycle.update-default-branch" {
			t.Fatalf("%s 不应被默认分支迁移改写: %#v", chainID, chain)
		}
	}

	migrated, changed = ApplyPresetMigration(modified)
	if !changed {
		t.Fatal("已修改预置链的版本迁移未触发")
	}
	if got := lifecycleChainByID(migrated.LifecycleChains, LifecycleChainIterationsAIID); got.Name != "已修改 iterations-ai" || got.Commands[0].CommandID == "system.lifecycle.update-default-branch" {
		t.Fatalf("已修改预置链不应被改写: %#v", got)
	}
}

func TestApplyPresetMigrationMovesUnmodifiedIterationsAIToBeforeStart(t *testing.T) {
	legacy := Default(t.TempDir())
	legacy.PresetVersion = 1
	legacy.LifecycleChains = defaultLifecycleChainsVersionOne()

	migrated, changed := ApplyPresetMigration(legacy)
	if !changed {
		t.Fatal("预置链范围修复未触发迁移")
	}
	if migrated.PresetVersion != CurrentPresetVersion {
		t.Fatalf("迁移版本 = %d，期望 %d", migrated.PresetVersion, CurrentPresetVersion)
	}
	for _, chain := range migrated.LifecycleChains {
		if chain.ID == LifecycleChainIterationsAIID {
			if !reflect.DeepEqual(chain.ApplicableHooks, []LifecycleHook{LifecycleHookBeforeStart}) {
				t.Fatalf("iterations-ai 迁移范围 = %#v", chain.ApplicableHooks)
			}
			return
		}
	}
	t.Fatalf("迁移后缺少 iterations-ai: %#v", migrated.LifecycleChains)
}

func TestApplyPresetMigrationRestoresMissingDefaultBranchTemplateFromVersionTwo(t *testing.T) {
	legacy := Default(t.TempDir())
	legacy.PresetVersion = 2
	legacy.TaskTemplates = []task.TaskTemplate{}
	legacy.ActiveTaskTemplateID = ""

	migrated, changed := ApplyPresetMigration(legacy)
	if !changed {
		t.Fatal("缺失默认分支模板时应触发迁移")
	}
	if migrated.PresetVersion != CurrentPresetVersion {
		t.Fatalf("迁移版本 = %d，期望 %d", migrated.PresetVersion, CurrentPresetVersion)
	}
	if len(migrated.TaskTemplates) != 1 || migrated.TaskTemplates[0].ID != DefaultBranchTaskTemplateID {
		t.Fatalf("恢复后的模板 = %#v", migrated.TaskTemplates)
	}
	if migrated.ActiveTaskTemplateID != DefaultBranchTaskTemplateID {
		t.Fatalf("恢复后的当前模板 = %q，期望 %q", migrated.ActiveTaskTemplateID, DefaultBranchTaskTemplateID)
	}
}

func TestDefaultIncludesDocumentedGitCloneLifecycleCommand(t *testing.T) {
	current := Default(t.TempDir())
	encoded, err := json.Marshal(current.LifecycleCommands)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var commands []map[string]any
	if err := json.Unmarshal(encoded, &commands); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, command := range commands {
		if command["id"] != "system.lifecycle.git-clone" {
			continue
		}
		if command["kind"] != "git-clone" {
			t.Fatalf("Git 系统命令类型 = %#v", command)
		}
		if command["documentation"] == "" {
			t.Fatalf("Git 系统命令缺少使用文档: %#v", command)
		}
		if documentation, _ := command["documentation"].(string); !strings.Contains(documentation, "可留空") {
			t.Fatalf("Git 系统命令文档未说明参数可留空: %#v", command["documentation"])
		}
		if !reflect.DeepEqual(command["applicableHooks"], []any{"beforeStart", "postStart", "beforeEnd", "updateTask"}) {
			t.Fatalf("Git 系统命令适用范围 = %#v", command["applicableHooks"])
		}
		return
	}
	t.Fatal("Default() 未提供 Git 仓库克隆系统命令")
}

func TestDefaultIncludesDocumentedGitCloneRepositoryLifecycleCommand(t *testing.T) {
	current := Default(t.TempDir())
	command := lifecycleCommandByID(current.LifecycleCommands, LifecycleCommandGitCloneRepositoryID)
	if command == nil {
		t.Fatal("Default() 未提供克隆指定 Git 仓库系统命令")
	}
	if command.Kind != LifecycleCommandKindGitCloneRepository || command.Name != "克隆指定 Git 仓库" {
		t.Fatalf("指定仓库克隆系统命令 = %#v", command)
	}
	if command.ChainArgumentMode != LifecycleCommandChainArgumentModeEnabled {
		t.Fatalf("指定仓库克隆命令的链级参数模式 = %q，期望允许", command.ChainArgumentMode)
	}
	if want := []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart}; !reflect.DeepEqual(command.ApplicableHooks, want) {
		t.Fatalf("指定仓库克隆命令适用范围 = %#v，期望 %#v", command.ApplicableHooks, want)
	}
	if !strings.Contains(command.Documentation, "repository=<仓库地址>") || !strings.Contains(command.Documentation, "dir=<相对目录>") {
		t.Fatalf("指定仓库克隆系统命令文档 = %q", command.Documentation)
	}
}

func TestDefaultIncludesManifestFileLifecycleCommand(t *testing.T) {
	current := Default(t.TempDir())
	command := lifecycleCommandByID(current.LifecycleCommands, "system.lifecycle.manifest-file")
	if command == nil {
		t.Fatal("Default() 未提供生成清单文件系统命令")
	}
	if command.Kind != LifecycleCommandKind("manifest-file") || command.Name != "生成清单文件" {
		t.Fatalf("清单文件系统命令 = %#v", command)
	}
	if command.ChainArgumentMode != LifecycleCommandChainArgumentModeEnabled {
		t.Fatalf("清单文件命令的链级参数模式 = %q，期望允许", command.ChainArgumentMode)
	}
	if want := []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart, LifecycleHookUpdateTask}; !reflect.DeepEqual(command.ApplicableHooks, want) {
		t.Fatalf("清单文件命令适用范围 = %#v，期望 %#v", command.ApplicableHooks, want)
	}
	if !strings.Contains(command.Documentation, "dir=<相对目录>") || !strings.Contains(command.Documentation, "name=<文件名>") {
		t.Fatalf("清单文件系统命令文档 = %q", command.Documentation)
	}
}

func TestValidateNormalizesManifestFileArguments(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		want      []string
		valid     bool
	}{
		{name: "默认参数", arguments: nil, want: []string{}, valid: true},
		{name: "指定目录与文件名", arguments: []string{" name=iteration.yaml ", " dir=configs/task "}, want: []string{"dir=configs/task", "name=iteration.yaml"}, valid: true},
		{name: "重复目录", arguments: []string{"dir=one", "dir=two"}},
		{name: "重复文件名", arguments: []string{"name=one.yaml", "name=two.yaml"}},
		{name: "目录越界", arguments: []string{"dir=../outside"}},
		{name: "文件名包含目录", arguments: []string{"name=config/manifest.yaml"}},
		{name: "未知参数", arguments: []string{"format=json"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			validated, err := Validate(Settings{
				WorkspaceRoot: t.TempDir(),
				TaskTreeWidth: DefaultTaskTreeWidth,
				LifecycleChains: []LifecycleCommandChain{{
					ID: "manifest", Name: "生成清单", ApplicableHooks: []LifecycleHook{LifecycleHookPostStart},
					Commands: []LifecycleCommandReference{{CommandID: "system.lifecycle.manifest-file", Arguments: test.arguments}},
				}},
			})
			if !test.valid {
				if err == nil {
					t.Fatalf("Validate() 对参数 %#v 的错误 = nil，期望拒绝", test.arguments)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if got := validated.LifecycleChains[0].Commands[0].Arguments; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("清单文件参数 = %#v，期望 %#v", got, test.want)
			}
		})
	}
}

func TestGitCloneRepositoryArgumentsRequireOneRepositoryAndOptionalSafeDirectory(t *testing.T) {
	for _, test := range []struct {
		name       string
		arguments  []string
		repository string
		directory  string
		valid      bool
	}{
		{name: "默认任务目录", arguments: []string{"repository=https://example.com/template.git"}, repository: "https://example.com/template.git", directory: ".", valid: true},
		{name: "指定子目录且顺序无关", arguments: []string{"dir=template", "repository=https://example.com/template.git?ref=a=b"}, repository: "https://example.com/template.git?ref=a=b", directory: "template", valid: true},
		{name: "缺少仓库", arguments: []string{"dir=template"}},
		{name: "空仓库", arguments: []string{"repository= "}},
		{name: "重复仓库", arguments: []string{"repository=one", "repository=two"}},
		{name: "重复目录", arguments: []string{"repository=one", "dir=one", "dir=two"}},
		{name: "未知参数", arguments: []string{"repository=one", "branch=main"}},
		{name: "空目录", arguments: []string{"repository=one", "dir= "}},
		{name: "绝对目录", arguments: []string{"repository=one", "dir=" + filepath.Join(string(filepath.Separator), "tmp", "template")}},
		{name: "越界目录", arguments: []string{"repository=one", "dir=../template"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := GitCloneRepositoryArguments(test.arguments)
			if !test.valid {
				if err == nil {
					t.Fatalf("GitCloneRepositoryArguments(%#v) error = nil，期望拒绝", test.arguments)
				}
				return
			}
			if err != nil {
				t.Fatalf("GitCloneRepositoryArguments(%#v) error = %v", test.arguments, err)
			}
			if parsed.Repository != test.repository || parsed.Directory != test.directory {
				t.Fatalf("解析的指定仓库克隆参数 = %#v", parsed)
			}
		})
	}
}

func TestValidateRejectsInvalidGitCloneRepositoryReference(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		LifecycleChains: []LifecycleCommandChain{{
			ID: "clone-template", Name: "初始化模板", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart},
			Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=https://example.com/template.git", "dir=template"}}},
		}},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := validated.LifecycleChains[0].Commands[0].Arguments; !reflect.DeepEqual(got, []string{"repository=https://example.com/template.git", "dir=template"}) {
		t.Fatalf("指定仓库克隆参数被意外改写: %#v", got)
	}

	_, err = Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		LifecycleChains: []LifecycleCommandChain{{
			ID: "invalid-template", Name: "错误模板", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart},
			Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandGitCloneRepositoryID, Arguments: []string{"dir=template"}}},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "repository") {
		t.Fatalf("Validate() error = %v，期望拒绝缺少 repository", err)
	}
}

func TestValidateNormalizesLifecycleCommandReferences(t *testing.T) {
	contents, err := json.Marshal(map[string]any{
		"workspaceRoot": filepath.Join(t.TempDir(), "workspaces"),
		"taskTreeWidth": DefaultTaskTreeWidth,
		"lifecycleCommands": []map[string]any{{
			"id": "prepare", "kind": "custom", "name": "准备", "command": "prepare", "arguments": []string{"--verbose"}, "applicableHooks": []LifecycleHook{LifecycleHookBeforeStart},
		}},
		"lifecycleChains": []map[string]any{{
			"id": "chain", "name": "准备链", "commands": []map[string]any{{"commandId": "prepare", "arguments": []string{" --profile ", "dev"}}}, "applicableHooks": []LifecycleHook{LifecycleHookBeforeStart},
		}},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var next Settings
	if err := json.Unmarshal(contents, &next); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	validated, err := Validate(next)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	encoded, err := json.Marshal(validated.LifecycleChains[0])
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var chain map[string]any
	if err := json.Unmarshal(encoded, &chain); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, found := chain["commandIds"]; found {
		t.Fatalf("规范化后的命令链仍保存 commandIds: %#v", chain)
	}
	if !reflect.DeepEqual(chain["commands"], []any{map[string]any{"commandId": "prepare", "arguments": []any{"--profile", "dev"}}}) {
		t.Fatalf("规范化后的命令引用 = %#v", chain["commands"])
	}
}

func TestValidateNormalizesLifecycleCommandChainArgumentModes(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		LifecycleCommands: []LifecycleCommand{
			{ID: "legacy", Kind: LifecycleCommandKindCustom, Name: "旧命令", Command: "legacy", Arguments: []string{"--fixed"}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
			{ID: "disabled", Kind: LifecycleCommandKindCustom, Name: "固定命令", Command: "fixed", Arguments: []string{"--always"}, ChainArgumentMode: LifecycleCommandChainArgumentModeDisabled, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
		},
		LifecycleChains: []LifecycleCommandChain{{
			ID: "chain", Name: "链", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart},
			Commands: []LifecycleCommandReference{
				{CommandID: "legacy", Arguments: []string{"--legacy-extra"}},
				{CommandID: "disabled", Arguments: []string{"--saved-extra"}},
			},
		}},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := lifecycleCommandByID(validated.LifecycleCommands, "legacy").ChainArgumentMode; got != LifecycleCommandChainArgumentModeEnabled {
		t.Fatalf("旧命令的链级参数模式 = %q，期望迁移为允许", got)
	}
	if got := lifecycleCommandByID(validated.LifecycleCommands, "disabled").ChainArgumentMode; got != LifecycleCommandChainArgumentModeDisabled {
		t.Fatalf("显式禁止的链级参数模式 = %q", got)
	}
	if got := validated.LifecycleChains[0].Commands; !reflect.DeepEqual(got, []LifecycleCommandReference{
		{CommandID: "legacy", Arguments: []string{"--legacy-extra"}},
		{CommandID: "disabled", Arguments: []string{"--saved-extra"}},
	}) {
		t.Fatalf("链级追加参数被改写: %#v", got)
	}

	invalid := Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		LifecycleCommands: []LifecycleCommand{{
			ID: "invalid", Kind: LifecycleCommandKindCustom, Name: "无效模式", Command: "echo", ChainArgumentMode: LifecycleCommandChainArgumentMode("unexpected"), ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart},
		}},
	}
	if _, err := Validate(invalid); err == nil || !strings.Contains(err.Error(), "参数模式") {
		t.Fatalf("Validate() error = %v，期望拒绝未知链级参数模式", err)
	}
}

func TestValidateRejectsGitCloneReferenceWithoutSingleValidDir(t *testing.T) {
	base := Settings{WorkspaceRoot: t.TempDir(), TaskTreeWidth: DefaultTaskTreeWidth}
	contents, err := json.Marshal(map[string]any{
		"workspaceRoot": base.WorkspaceRoot,
		"taskTreeWidth": base.TaskTreeWidth,
		"lifecycleChains": []map[string]any{{
			"id": "clone", "name": "克隆", "commands": []map[string]any{{"commandId": "system.lifecycle.git-clone", "arguments": []string{"dir=../outside"}}}, "applicableHooks": []LifecycleHook{LifecycleHookBeforeStart},
		}},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var next Settings
	if err := json.Unmarshal(contents, &next); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	_, err = Validate(next)
	if err == nil || !strings.Contains(err.Error(), "dir") {
		t.Fatalf("Validate() error = %v，期望拒绝无效 dir", err)
	}
}

func TestValidateAllowsGitCloneReferenceWithoutDirectory(t *testing.T) {
	for _, arguments := range [][]string{nil, {"   "}} {
		validated, err := Validate(Settings{
			WorkspaceRoot: t.TempDir(),
			TaskTreeWidth: DefaultTaskTreeWidth,
			LifecycleChains: []LifecycleCommandChain{{
				ID: "clone", Name: "克隆", Commands: []LifecycleCommandReference{{CommandID: LifecycleCommandGitCloneID, Arguments: arguments}}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart},
			}},
		})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		normalizedArguments := validated.LifecycleChains[0].Commands[0].Arguments
		if !reflect.DeepEqual(normalizedArguments, []string{}) {
			t.Fatalf("Git 克隆默认参数 = %#v，期望保留为空", normalizedArguments)
		}
		directory, err := GitCloneDirectory(normalizedArguments)
		if err != nil {
			t.Fatalf("GitCloneDirectory() error = %v", err)
		}
		if directory != "." {
			t.Fatalf("GitCloneDirectory() = %q，期望 .", directory)
		}
	}
}

func TestGitCloneDirectoryAcceptsOnlyOneSafeExplicitDirectory(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		directory string
		valid     bool
	}{
		{name: "当前任务目录", arguments: []string{"dir=."}, directory: ".", valid: true},
		{name: "子目录", arguments: []string{" dir=repositories "}, directory: "repositories", valid: true},
		{name: "多个参数", arguments: []string{"dir=repositories", "dir=other"}},
		{name: "未知参数", arguments: []string{"target=repositories"}},
		{name: "空目录", arguments: []string{"dir="}},
		{name: "绝对路径", arguments: []string{"dir=" + filepath.Join(string(filepath.Separator), "tmp", "repositories")}},
		{name: "父级目录", arguments: []string{"dir=../repositories"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory, err := GitCloneDirectory(test.arguments)
			if test.valid {
				if err != nil {
					t.Fatalf("GitCloneDirectory() error = %v", err)
				}
				if directory != test.directory {
					t.Fatalf("GitCloneDirectory() = %q，期望 %q", directory, test.directory)
				}
				return
			}
			if err == nil {
				t.Fatalf("GitCloneDirectory(%#v) error = nil，期望拒绝", test.arguments)
			}
		})
	}
}

func TestValidateNormalizesLifecycleCommandsChainsAndPresets(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		LifecycleCommands: []LifecycleCommand{
			{ID: " custom-prepare ", Kind: LifecycleCommandKindCustom, Name: " 准备仓库 ", Command: " prepare ", Arguments: []string{" --fast ", ""}, ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}},
			{ID: LifecycleCommandCreateWorkspaceID, Kind: LifecycleCommandKindCustom, Name: " 被篡改 ", Command: " rm "},
		},
		LifecycleChains: []LifecycleCommandChain{{
			ID:              " chain-prepare ",
			Name:            " 开始准备 ",
			CommandIDs:      []string{" custom-prepare ", LifecycleCommandCreateWorkspaceID},
			ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart},
		}},
		LifecyclePresets: []LifecyclePreset{{
			ID: " preset-prepare ", Name: " 开始预设 ", Chains: map[LifecycleHook]string{LifecycleHookBeforeStart: " chain-prepare "},
		}},
		DefaultLifecyclePresetID: " preset-prepare ",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	custom := lifecycleCommandByID(validated.LifecycleCommands, "custom-prepare")
	if custom == nil || custom.Name != "准备仓库" || custom.Command != "prepare" || !reflect.DeepEqual(custom.Arguments, []string{"--fast"}) {
		t.Fatalf("自定义生命周期命令 = %#v", custom)
	}
	create := lifecycleCommandByID(validated.LifecycleCommands, LifecycleCommandCreateWorkspaceID)
	if create == nil || create.Kind != LifecycleCommandKindCreateWorkspace || create.Name == "被篡改" || create.Command != "" {
		t.Fatalf("创建目录内置命令 = %#v", create)
	}
	if len(validated.LifecycleChains) != 1 || validated.LifecycleChains[0].ID != "chain-prepare" || !reflect.DeepEqual(validated.LifecycleChains[0].Commands, []LifecycleCommandReference{{CommandID: "custom-prepare", Arguments: []string{}}, {CommandID: LifecycleCommandCreateWorkspaceID, Arguments: []string{}}}) {
		t.Fatalf("生命周期链 = %#v", validated.LifecycleChains)
	}
	if got := validated.DefaultLifecyclePresetChains()[LifecycleHookBeforeStart]; got != "chain-prepare" {
		t.Fatalf("beforeStart 默认链 = %q", got)
	}
}

func TestValidateLifecycleApplicableHooks(t *testing.T) {
	allHooks := []LifecycleHook{LifecycleHookBeforeStart, LifecycleHookPostStart, LifecycleHookBeforeEnd, LifecycleHookPostEnd, LifecycleHookUpdateTask}
	legacy, err := Validate(Settings{
		WorkspaceRoot:     t.TempDir(),
		TaskTreeWidth:     DefaultTaskTreeWidth,
		LifecycleCommands: []LifecycleCommand{{ID: "legacy-command", Kind: LifecycleCommandKindCustom, Name: "旧命令", Command: "echo"}},
		LifecycleChains:   []LifecycleCommandChain{{ID: "legacy-chain", Name: "旧链", CommandIDs: []string{"legacy-command"}}},
	})
	if err != nil {
		t.Fatalf("Validate() 迁移旧范围 error = %v", err)
	}
	if !reflect.DeepEqual(legacy.LifecycleCommands[0].ApplicableHooks, allHooks) || !reflect.DeepEqual(legacy.LifecycleChains[0].ApplicableHooks, allHooks) {
		t.Fatalf("旧配置范围迁移 = %#v", legacy)
	}

	invalidCommand := Settings{
		WorkspaceRoot:     t.TempDir(),
		TaskTreeWidth:     DefaultTaskTreeWidth,
		LifecycleCommands: []LifecycleCommand{{ID: "command", Kind: LifecycleCommandKindCustom, Name: "命令", Command: "echo", ApplicableHooks: []LifecycleHook{}}},
	}
	if _, err := Validate(invalidCommand); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝没有适用范围的自定义命令")
	}

	invalidChain := Settings{
		WorkspaceRoot:     t.TempDir(),
		TaskTreeWidth:     DefaultTaskTreeWidth,
		LifecycleCommands: []LifecycleCommand{{ID: "command", Kind: LifecycleCommandKindCustom, Name: "命令", Command: "echo", ApplicableHooks: []LifecycleHook{LifecycleHookBeforeStart}}},
		LifecycleChains:   []LifecycleCommandChain{{ID: "chain", Name: "链", CommandIDs: []string{"command"}, ApplicableHooks: []LifecycleHook{LifecycleHookPostStart}}},
	}
	if _, err := Validate(invalidChain); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝范围不匹配的命令链")
	}
}

func TestValidateRejectsInvalidLifecycleCommandChainAndDefault(t *testing.T) {
	base := Settings{WorkspaceRoot: t.TempDir(), TaskTreeWidth: DefaultTaskTreeWidth}
	base.LifecycleCommands = append(DefaultLifecycleCommands(), LifecycleCommand{ID: "custom", Kind: LifecycleCommandKindCustom, Name: "命令", Command: "echo"})

	invalidCommand := base
	invalidCommand.LifecycleCommands[len(invalidCommand.LifecycleCommands)-1].Command = ""
	if _, err := Validate(invalidCommand); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝无可执行命令的自定义生命周期命令")
	}

	invalidChain := base
	invalidChain.LifecycleChains = []LifecycleCommandChain{{ID: "chain", Name: "链", CommandIDs: []string{"missing"}}}
	if _, err := Validate(invalidChain); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝引用不存在命令的链")
	}

	invalidDefault := base
	invalidDefault.LifecycleChains = []LifecycleCommandChain{{ID: "chain", Name: "链", CommandIDs: []string{"custom"}}}
	invalidDefault.LifecyclePresets = []LifecyclePreset{{ID: "preset", Name: "预设", Chains: map[LifecycleHook]string{LifecycleHookPostStart: "missing"}}}
	invalidDefault.DefaultLifecyclePresetID = "preset"
	if _, err := Validate(invalidDefault); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝引用不存在链的默认值")
	}
}

func TestValidateDefaultsOnlyAvailableLifecycleChains(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		LifecycleChains: []LifecycleCommandChain{{
			ID:         LifecycleChainCreateWorkspaceID,
			Name:       "创建任务工作目录",
			CommandIDs: []string{LifecycleCommandCreateWorkspaceID},
		}},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(validated.LifecyclePresets) != 0 || len(validated.DefaultLifecyclePresetChains()) != 0 {
		t.Fatalf("未提供预设的设置不应隐式添加默认值: %#v", validated)
	}
}

func lifecycleCommandByID(commands []LifecycleCommand, id string) *LifecycleCommand {
	for index := range commands {
		if commands[index].ID == id {
			return &commands[index]
		}
	}
	return nil
}

func lifecycleChainByID(chains []LifecycleCommandChain, id string) *LifecycleCommandChain {
	for index := range chains {
		if chains[index].ID == id {
			return &chains[index]
		}
	}
	return nil
}

func TestValidateDefaultsAndValidatesActiveTaskStatus(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated.ActiveTaskStatus != TaskStatusPending {
		t.Errorf("Validate() ActiveTaskStatus = %q, want %q", validated.ActiveTaskStatus, TaskStatusPending)
	}

	_, err = Validate(Settings{
		WorkspaceRoot:    t.TempDir(),
		TaskTreeWidth:    DefaultTaskTreeWidth,
		ActiveTaskStatus: "archived",
	})
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid active task status error")
	}
}

func TestValidateDefaultsStatusManagementModeForExistingSettings(t *testing.T) {
	contents, err := json.Marshal(map[string]any{
		"workspaceRoot": filepath.Join(t.TempDir(), "workspaces"),
		"taskTreeWidth": DefaultTaskTreeWidth,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var next Settings
	if err := json.Unmarshal(contents, &next); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	validated, err := Validate(next)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated.StatusManagementMode != StatusManagementModeTitleChange {
		t.Errorf("Validate() StatusManagementMode = %q，期望 %q", validated.StatusManagementMode, StatusManagementModeTitleChange)
	}
	if validated.HTTPServiceEnabled {
		t.Error("Validate() HTTPServiceEnabled = true，期望旧设置默认关闭")
	}
}

func TestValidateAcceptsOutputChangeStatusManagementMode(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot:        t.TempDir(),
		TaskTreeWidth:        DefaultTaskTreeWidth,
		StatusManagementMode: StatusManagementModeOutputChange,
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated.StatusManagementMode != StatusManagementModeOutputChange {
		t.Errorf("Validate() StatusManagementMode = %q，期望 %q", validated.StatusManagementMode, StatusManagementModeOutputChange)
	}
}

func TestValidateRequiresHTTPPortWhenIndependentHTTPServiceEnabled(t *testing.T) {
	base := Settings{WorkspaceRoot: t.TempDir(), TaskTreeWidth: DefaultTaskTreeWidth, StatusManagementMode: StatusManagementModeTitleChange}

	if _, err := Validate(Settings{
		WorkspaceRoot: base.WorkspaceRoot, TaskTreeWidth: base.TaskTreeWidth,
		StatusManagementMode: StatusManagementModeTitleChange, HTTPServiceEnabled: true,
	}); err == nil {
		t.Fatal("Validate() 独立 HTTP 服务未设置端口 error = nil，期望错误")
	}

	validated, err := Validate(Settings{
		WorkspaceRoot: base.WorkspaceRoot, TaskTreeWidth: base.TaskTreeWidth,
		StatusManagementMode: StatusManagementModeTitleChange, HTTPServiceEnabled: true, StatusManagementHTTPPort: 18765,
	})
	if err != nil {
		t.Fatalf("Validate() 独立 HTTP 服务 error = %v", err)
	}
	if !validated.HTTPServiceEnabled || validated.StatusManagementHTTPPort != 18765 {
		t.Fatalf("Validate() 独立 HTTP 服务设置 = %#v", validated)
	}

	if _, err := Validate(base); err != nil {
		t.Fatalf("Validate() 未开启独立 HTTP 服务 error = %v", err)
	}
}

func TestValidateRequiresHTTPPortOnlyForHTTPStatusManagement(t *testing.T) {
	base := Settings{WorkspaceRoot: t.TempDir(), TaskTreeWidth: DefaultTaskTreeWidth}

	validated, err := Validate(Settings{
		WorkspaceRoot:            base.WorkspaceRoot,
		TaskTreeWidth:            base.TaskTreeWidth,
		StatusManagementMode:     StatusManagementModeHTTP,
		StatusManagementHTTPPort: 18765,
	})
	if err != nil {
		t.Fatalf("Validate() HTTP 设置 error = %v", err)
	}
	if validated.StatusManagementHTTPPort != 18765 {
		t.Errorf("Validate() StatusManagementHTTPPort = %d，期望 18765", validated.StatusManagementHTTPPort)
	}

	for _, port := range []int{0, -1, 65536} {
		_, err := Validate(Settings{
			WorkspaceRoot:            base.WorkspaceRoot,
			TaskTreeWidth:            base.TaskTreeWidth,
			StatusManagementMode:     StatusManagementModeHTTP,
			StatusManagementHTTPPort: port,
		})
		if err == nil {
			t.Errorf("Validate() HTTP 端口 %d error = nil，期望错误", port)
		}
	}

	validated, err = Validate(base)
	if err != nil {
		t.Fatalf("Validate() 标题变化默认设置 error = %v", err)
	}
	if validated.StatusManagementMode != StatusManagementModeTitleChange {
		t.Errorf("Validate() 标题变化默认模式 = %q，期望 %q", validated.StatusManagementMode, StatusManagementModeTitleChange)
	}
}

func TestValidateNormalizesFixedTaskMenuItemsAndKeepsOrder(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		TaskMenuItems: []TaskMenuItem{
			{ID: "custom-codex", Kind: TaskMenuItemKindCommand, Name: "Codex", Command: "codex", Arguments: []string{"--full-auto"}, ShowTerminal: true, BeforeScript: &TaskScript{Script: " prepare-codex ", Arguments: []string{" --task ", "", "  "}}},
			{ID: TaskMenuItemOpenFolderID, Kind: TaskMenuItemKindCommand, Name: "被篡改", Command: "rm", Arguments: []string{"-rf"}, ShowTerminal: true, BeforeScript: &TaskScript{Script: "不得保存"}},
			{ID: TaskMenuItemEditTaskID, Kind: TaskMenuItemKindCommand, Name: "被篡改"},
		},
	})

	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(validated.TaskMenuItems) != 4 {
		t.Fatalf("任务菜单项数量 = %d，期望 4", len(validated.TaskMenuItems))
	}
	if validated.TaskMenuItems[0].ID != "custom-codex" || !reflect.DeepEqual(validated.TaskMenuItems[1], fixedTaskMenuItem(TaskMenuItemOpenFolderID)) || !reflect.DeepEqual(validated.TaskMenuItems[2], fixedTaskMenuItem(TaskMenuItemEditTaskID)) || !reflect.DeepEqual(validated.TaskMenuItems[3], fixedTaskMenuItem(TaskMenuItemCreateTerminalID)) {
		t.Errorf("规范化任务菜单项 = %#v", validated.TaskMenuItems)
	}
	if got, want := validated.TaskMenuItems[0].BeforeScript, (&TaskScript{Script: "prepare-codex", Arguments: []string{"--task"}}); !reflect.DeepEqual(got, want) {
		t.Errorf("前置脚本 = %#v，期望 %#v", got, want)
	}
	if validated.TaskMenuItems[0].AfterScript != nil {
		t.Errorf("空后置脚本 = %#v，期望 nil", validated.TaskMenuItems[0].AfterScript)
	}
	contents, err := json.Marshal(validated.TaskMenuItems[0])
	if err != nil {
		t.Fatalf("序列化自定义菜单项: %v", err)
	}
	var persisted map[string]any
	if err := json.Unmarshal(contents, &persisted); err != nil {
		t.Fatalf("解析自定义菜单项 JSON: %v", err)
	}
	if _, ok := persisted["beforeHook"]; ok {
		t.Fatalf("不应持久化旧钩子字段: %#v", persisted)
	}
	beforeScript, ok := persisted["beforeScript"].(map[string]any)
	if !ok || beforeScript["script"] != "prepare-codex" {
		t.Fatalf("持久化前置脚本 = %#v", persisted["beforeScript"])
	}
}

func TestValidateNormalizesTaskMenuMouseClipboardPolicy(t *testing.T) {
	validated, err := Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		TaskMenuItems: []TaskMenuItem{
			{
				ID:                          "custom-terminal",
				Kind:                        TaskMenuItemKindCommand,
				Name:                        "Claude",
				Command:                     "claude",
				ShowTerminal:                true,
				DisableTaskAIMouseClipboard: true,
			},
			{
				ID:                          "custom-background",
				Kind:                        TaskMenuItemKindCommand,
				Name:                        "后台命令",
				Command:                     "echo",
				DisableTaskAIMouseClipboard: true,
			},
			{
				ID:                          TaskMenuItemEditTaskID,
				DisableTaskAIMouseClipboard: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !validated.TaskMenuItems[0].DisableTaskAIMouseClipboard {
		t.Fatalf("终端命令策略 = false，期望 true")
	}
	if validated.TaskMenuItems[1].DisableTaskAIMouseClipboard {
		t.Fatalf("非终端命令策略 = true，期望 false")
	}
	if validated.TaskMenuItems[2].DisableTaskAIMouseClipboard {
		t.Fatalf("固定菜单项策略 = true，期望 false")
	}

	contents, err := json.Marshal(validated.TaskMenuItems[0])
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var persisted map[string]any
	if err := json.Unmarshal(contents, &persisted); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if persisted["disableTaskAIMouseClipboard"] != true {
		t.Fatalf("持久化鼠标剪贴板策略 = %#v，期望 true", persisted["disableTaskAIMouseClipboard"])
	}

	var legacy TaskMenuItem
	if err := json.Unmarshal([]byte(`{"id":"custom-legacy","kind":"command","name":"Legacy","command":"legacy","showTerminal":true}`), &legacy); err != nil {
		t.Fatalf("Unmarshal() 旧菜单项 error = %v", err)
	}
	if legacy.DisableTaskAIMouseClipboard {
		t.Fatal("旧菜单项缺失配置时不应禁用 TaskAI 鼠标剪贴板")
	}
}

func TestValidateRejectsInvalidCustomTaskMenuItem(t *testing.T) {
	_, err := Validate(Settings{
		WorkspaceRoot: t.TempDir(),
		TaskTreeWidth: DefaultTaskTreeWidth,
		TaskMenuItems: []TaskMenuItem{{
			ID:   "custom-invalid",
			Kind: TaskMenuItemKindCommand,
			Name: "无命令",
		}},
	})

	if err == nil {
		t.Fatal("Validate() error = nil，期望拒绝无命令的自定义菜单项")
	}
}

func TestDefaultPersistsLightColorScheme(t *testing.T) {
	contents, err := json.Marshal(Default(t.TempDir()))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var persisted map[string]any
	if err := json.Unmarshal(contents, &persisted); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := persisted["colorScheme"], "light"; got != want {
		t.Errorf("持久化颜色模式 = %#v，期望 %q", got, want)
	}
}

func TestValidateRejectsUnsupportedColorScheme(t *testing.T) {
	contents, err := json.Marshal(map[string]any{
		"workspaceRoot": t.TempDir(),
		"taskTreeWidth": DefaultTaskTreeWidth,
		"colorScheme":   "system",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var next Settings
	if err := json.Unmarshal(contents, &next); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, err := Validate(next); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝不支持的颜色模式")
	}
}

func TestValidateNormalizesConfiguredShellPath(t *testing.T) {
	shellPath, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	absShellPath, err := filepath.Abs(shellPath)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	contents, err := json.Marshal(map[string]any{
		"workspaceRoot": t.TempDir(),
		"taskTreeWidth": DefaultTaskTreeWidth,
		"colorScheme":   "light",
		"shellPath":     shellPath,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var next Settings
	if err := json.Unmarshal(contents, &next); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	validated, err := Validate(next)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	persisted, err := json.Marshal(validated)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var settingsData map[string]any
	if err := json.Unmarshal(persisted, &settingsData); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := settingsData["shellPath"], filepath.Clean(absShellPath); got != want {
		t.Errorf("持久化 Shell 路径 = %#v，期望 %q", got, want)
	}
}

func TestValidateRejectsUnavailableShellPath(t *testing.T) {
	contents, err := json.Marshal(map[string]any{
		"workspaceRoot": t.TempDir(),
		"taskTreeWidth": DefaultTaskTreeWidth,
		"colorScheme":   "light",
		"shellPath":     filepath.Join(t.TempDir(), "missing-shell"),
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var next Settings
	if err := json.Unmarshal(contents, &next); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, err := Validate(next); err == nil {
		t.Fatal("Validate() error = nil，期望拒绝不可用的 Shell 路径")
	}
}

func TestValidateCreatesUsableAbsoluteWorkspaceRoot(t *testing.T) {
	requestedRoot := filepath.Join(t.TempDir(), "nested", "workspaces")

	validated, err := Validate(Settings{
		WorkspaceRoot: requestedRoot,
		TaskTreeWidth: 120,
	})

	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !filepath.IsAbs(validated.WorkspaceRoot) {
		t.Errorf("Validate() WorkspaceRoot = %q, want absolute path", validated.WorkspaceRoot)
	}
	if _, err := filepath.EvalSymlinks(validated.WorkspaceRoot); err != nil {
		t.Errorf("Validate() did not create workspace root: %v", err)
	}
	if validated.TaskTreeWidth != MinimumTaskTreeWidth {
		t.Errorf("Validate() TaskTreeWidth = %d, want %d", validated.TaskTreeWidth, MinimumTaskTreeWidth)
	}
}

func TestValidateRejectsWorkspaceRootThatIsAFile(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(workspaceRoot, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Validate(Settings{WorkspaceRoot: workspaceRoot, TaskTreeWidth: DefaultTaskTreeWidth})

	if err == nil {
		t.Fatal("Validate() error = nil, want invalid workspace root error")
	}
}
