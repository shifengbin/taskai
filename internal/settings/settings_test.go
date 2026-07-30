package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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

	if len(current.LifecycleCommands) != 3 {
		t.Fatalf("默认生命周期命令数量 = %d，期望 3", len(current.LifecycleCommands))
	}
	if current.LifecycleCommands[0].ID != LifecycleCommandCreateWorkspaceID || current.LifecycleCommands[1].ID != LifecycleCommandDeleteWorkspaceID || current.LifecycleCommands[2].ID != LifecycleCommandGitCloneID {
		t.Fatalf("默认生命周期命令 = %#v", current.LifecycleCommands)
	}
	if !reflect.DeepEqual(current.LifecycleCommands[0].ApplicableHooks, []LifecycleHook{LifecycleHookBeforeStart}) || !reflect.DeepEqual(current.LifecycleCommands[1].ApplicableHooks, []LifecycleHook{LifecycleHookPostEnd}) {
		t.Fatalf("内置命令适用范围 = %#v", current.LifecycleCommands)
	}
	if !reflect.DeepEqual(current.LifecycleChains[0].ApplicableHooks, []LifecycleHook{LifecycleHookBeforeStart}) || !reflect.DeepEqual(current.LifecycleChains[1].ApplicableHooks, []LifecycleHook{LifecycleHookPostEnd}) {
		t.Fatalf("内置命令链适用范围 = %#v", current.LifecycleChains)
	}
	if got := current.LifecycleDefaultChains[LifecycleHookBeforeStart]; got != LifecycleChainCreateWorkspaceID {
		t.Fatalf("beforeStart 默认链 = %q，期望 %q", got, LifecycleChainCreateWorkspaceID)
	}
	if got := current.LifecycleDefaultChains[LifecycleHookPostEnd]; got != LifecycleChainDeleteWorkspaceID {
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
		if !reflect.DeepEqual(command["applicableHooks"], []any{"beforeStart", "beforeEnd", "updateTask"}) {
			t.Fatalf("Git 系统命令适用范围 = %#v", command["applicableHooks"])
		}
		return
	}
	t.Fatal("Default() 未提供 Git 仓库克隆系统命令")
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

func TestValidateNormalizesLifecycleCommandsChainsAndDefaults(t *testing.T) {
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
		LifecycleDefaultChains: map[LifecycleHook]string{LifecycleHookBeforeStart: " chain-prepare "},
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
	if got := validated.LifecycleDefaultChains[LifecycleHookBeforeStart]; got != "chain-prepare" {
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
	invalidDefault.LifecycleDefaultChains = map[LifecycleHook]string{LifecycleHookPostStart: "missing"}
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
	if !reflect.DeepEqual(validated.LifecycleDefaultChains, map[LifecycleHook]string{LifecycleHookBeforeStart: LifecycleChainCreateWorkspaceID}) {
		t.Fatalf("可用默认链 = %#v", validated.LifecycleDefaultChains)
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
