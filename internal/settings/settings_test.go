package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
