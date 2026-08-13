package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"taskai/internal/settings"
	"taskai/internal/task"
)

func TestRepositorySeedsCompanyFrameworkForNewInstallation(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	repository := New(dataPath, settings.Default(t.TempDir()))

	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	assertCompanyFrameworkDefaults(t, data.Settings)

	reloaded, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("第二次 Load() error = %v", err)
	}
	assertCompanyFrameworkDefaults(t, reloaded.Settings)
}

func TestRepositoryPreservesExistingLifecycleDefaults(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	current := settings.Default(t.TempDir())
	current.LifecycleChains = settings.DefaultLifecycleChains()
	updateIndex := lifecycleCommandChainIndex(current.LifecycleChains, settings.LifecycleChainUpdateRepositoriesID)
	current.LifecycleChains[updateIndex].Name = "自定义仓库同步"
	current.LifecyclePresets = []settings.LifecyclePreset{{
		ID:     "user.lifecycle-preset",
		Name:   "用户预设",
		Chains: map[task.LifecycleHook]string{task.LifecycleHookPostEnd: settings.LifecycleChainDeleteWorkspaceID},
	}}
	current.DefaultLifecyclePresetID = "user.lifecycle-preset"
	contents, err := json.Marshal(Data{Tasks: []task.Task{}, Settings: current})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Settings.DefaultLifecyclePresetID != "user.lifecycle-preset" || !reflect.DeepEqual(data.Settings.LifecyclePresets, current.LifecyclePresets) {
		t.Fatalf("已有预设被改写: %#v，默认 = %q", data.Settings.LifecyclePresets, data.Settings.DefaultLifecyclePresetID)
	}
	updateIndex = lifecycleCommandChainIndex(data.Settings.LifecycleChains, settings.LifecycleChainUpdateRepositoriesID)
	if updateIndex < 0 || data.Settings.LifecycleChains[updateIndex].Name != "自定义仓库同步" {
		t.Fatalf("已有仓库更新链被改写: %#v", data.Settings.LifecycleChains)
	}
	if lifecyclePresetIndex(data.Settings.LifecyclePresets, settings.CompanyFrameworkLifecyclePresetID) >= 0 {
		t.Fatalf("已有设置被补入公司框架: %#v", data.Settings.LifecyclePresets)
	}
}

func assertCompanyFrameworkDefaults(t *testing.T, current settings.Settings) {
	t.Helper()
	if current.DefaultLifecyclePresetID != settings.CompanyFrameworkLifecyclePresetID || len(current.LifecyclePresets) != 2 {
		t.Fatalf("新安装预设 = %#v，默认 = %q", current.LifecyclePresets, current.DefaultLifecyclePresetID)
	}
	companyIndex := lifecyclePresetIndex(current.LifecyclePresets, settings.CompanyFrameworkLifecyclePresetID)
	if companyIndex < 0 {
		t.Fatalf("新安装缺少公司框架: %#v", current.LifecyclePresets)
	}
	wantChains := map[task.LifecycleHook]string{
		task.LifecycleHookBeforeStart: settings.LifecycleChainIterationsAIID,
		task.LifecycleHookPostEnd:     settings.LifecycleChainDeleteWorkspaceID,
		task.LifecycleHookUpdateTask:  settings.LifecycleChainUpdateRepositoriesID,
	}
	if !reflect.DeepEqual(current.LifecyclePresets[companyIndex].Chains, wantChains) {
		t.Fatalf("公司框架映射 = %#v，期望 %#v", current.LifecyclePresets[companyIndex].Chains, wantChains)
	}
	updateIndex := lifecycleCommandChainIndex(current.LifecycleChains, settings.LifecycleChainUpdateRepositoriesID)
	if updateIndex < 0 || current.LifecycleChains[updateIndex].Name != "更新框架仓库" {
		t.Fatalf("新安装仓库更新链 = %#v", current.LifecycleChains)
	}
}

func TestRepositoryLoadsPersistedColorScheme(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents, err := json.Marshal(map[string]any{
		"tasks": []task.Task{},
		"settings": map[string]any{
			"workspaceRoot": filepath.Join(t.TempDir(), "workspaces"),
			"taskTreeWidth": settings.DefaultTaskTreeWidth,
			"colorScheme":   "dark",
		},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	persisted, err := json.Marshal(data.Settings)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var settingsData map[string]any
	if err := json.Unmarshal(persisted, &settingsData); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := settingsData["colorScheme"], "dark"; got != want {
		t.Errorf("加载的颜色模式 = %#v，期望 %q", got, want)
	}
}

func TestRepositoryMergeDetectedAgentTaskMenusUsesLatestSettingsAtomically(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	current, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	current.Settings.ColorScheme = "dark"
	current.Settings.TaskMenuItems = append(current.Settings.TaskMenuItems, settings.TaskMenuItem{
		ID: "custom.keep", Kind: settings.TaskMenuItemKindCommand, Name: "保留用户菜单", Command: "printf",
	})
	if _, err := repository.SaveSettings(current.Settings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	merged, changed, err := repository.MergeDetectedAgentTaskMenuItems(settings.DetectedAgentCommands{Codex: true, Claude: true})
	if err != nil {
		t.Fatalf("MergeDetectedAgentTaskMenuItems() error = %v", err)
	}
	if !changed {
		t.Fatal("MergeDetectedAgentTaskMenuItems() changed = false，期望 true")
	}
	if merged.ColorScheme != "dark" {
		t.Fatalf("原子合并覆盖了最新颜色设置 = %q", merged.ColorScheme)
	}
	if got := merged.TaskMenuItems[len(merged.TaskMenuItems)-3].ID; got != "custom.keep" {
		t.Fatalf("原子合并覆盖了用户菜单项，倒数第三项 ID = %q", got)
	}

	second, changed, err := repository.MergeDetectedAgentTaskMenuItems(settings.DetectedAgentCommands{Codex: true, Claude: true})
	if err != nil {
		t.Fatalf("第二次 MergeDetectedAgentTaskMenuItems() error = %v", err)
	}
	if changed || !reflect.DeepEqual(second, merged) {
		t.Fatalf("第二次原子合并 = (%#v, %t)，期望幂等", second, changed)
	}
}

func TestRepositoryRemembersDeletedAutomaticAgentMenuAcrossReload(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	repository := New(dataPath, settings.Default(t.TempDir()))
	merged, changed, err := repository.MergeDetectedAgentTaskMenuItems(settings.DetectedAgentCommands{Codex: true, Claude: true})
	if err != nil || !changed {
		t.Fatalf("首次 MergeDetectedAgentTaskMenuItems() = (%#v, %t, %v)", merged, changed, err)
	}
	merged.TaskMenuItems = removeTaskMenuItemByID(merged.TaskMenuItems, settings.TaskMenuItemDetectedCodexID)
	if _, err := repository.SaveSettings(merged); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	reloaded := New(dataPath, settings.Default(t.TempDir()))
	afterRestart, changed, err := reloaded.MergeDetectedAgentTaskMenuItems(settings.DetectedAgentCommands{Codex: true, Claude: true})
	if err != nil {
		t.Fatalf("重载 MergeDetectedAgentTaskMenuItems() error = %v", err)
	}
	if changed {
		t.Fatal("删除 Codex 自动项后重载 changed = true，期望不恢复")
	}
	if containsTaskMenuItemID(afterRestart.TaskMenuItems, settings.TaskMenuItemDetectedCodexID) {
		t.Fatalf("删除后的 Codex 自动项被恢复 = %#v", afterRestart.TaskMenuItems)
	}
	if !containsTaskMenuItemID(afterRestart.TaskMenuItems, settings.TaskMenuItemDetectedClaudeID) {
		t.Fatalf("Claude 自动项未保留 = %#v", afterRestart.TaskMenuItems)
	}
	data, err := reloaded.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(data.DismissedAgentTaskMenuItemIDs, []string{settings.TaskMenuItemDetectedCodexID}) {
		t.Fatalf("删除记录 = %#v", data.DismissedAgentTaskMenuItemIDs)
	}
}

func TestRepositoryDoesNotRecordDeletedCustomAgentMenu(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	repository := New(dataPath, settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskMenuItems = append(data.Settings.TaskMenuItems, settings.TaskMenuItem{
		ID: "custom.codex", Kind: settings.TaskMenuItemKindCommand, Name: "我的 Codex", Command: "codex",
	})
	saved, err := repository.SaveSettings(data.Settings)
	if err != nil {
		t.Fatalf("保存自定义菜单: %v", err)
	}
	saved.TaskMenuItems = removeTaskMenuItemByID(saved.TaskMenuItems, "custom.codex")
	if _, err := repository.SaveSettings(saved); err != nil {
		t.Fatalf("删除自定义菜单: %v", err)
	}

	persisted, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(persisted.DismissedAgentTaskMenuItemIDs) != 0 {
		t.Fatalf("删除自定义菜单产生了自动项删除记录 = %#v", persisted.DismissedAgentTaskMenuItemIDs)
	}
}

func TestRepositoryFailedSettingsSaveDoesNotRecordAutomaticAgentDeletion(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	repository := New(dataPath, settings.Default(t.TempDir()))
	merged, _, err := repository.MergeDetectedAgentTaskMenuItems(settings.DetectedAgentCommands{Codex: true})
	if err != nil {
		t.Fatalf("MergeDetectedAgentTaskMenuItems() error = %v", err)
	}
	merged.TaskMenuItems = append(
		removeTaskMenuItemByID(merged.TaskMenuItems, settings.TaskMenuItemDetectedCodexID),
		settings.TaskMenuItem{ID: "invalid", Kind: settings.TaskMenuItemKindCommand, Name: ""},
	)
	if _, err := repository.SaveSettings(merged); err == nil {
		t.Fatal("SaveSettings() error = nil，期望无效菜单导致保存失败")
	}

	reloaded := New(dataPath, settings.Default(t.TempDir()))
	persisted, err := reloaded.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !containsTaskMenuItemID(persisted.Settings.TaskMenuItems, settings.TaskMenuItemDetectedCodexID) {
		t.Fatalf("失败保存改变了自动菜单 = %#v", persisted.Settings.TaskMenuItems)
	}
	if len(persisted.DismissedAgentTaskMenuItemIDs) != 0 {
		t.Fatalf("失败保存留下删除记录 = %#v", persisted.DismissedAgentTaskMenuItemIDs)
	}
}

func TestNormalizeDismissedAgentTaskMenuItemIDsKeepsKnownUniqueOrder(t *testing.T) {
	if got := normalizeDismissedAgentTaskMenuItemIDs(nil); got != nil {
		t.Fatalf("空删除记录 = %#v，期望 nil", got)
	}
	want := []string{settings.TaskMenuItemDetectedCodexID, settings.TaskMenuItemDetectedClaudeID}
	got := normalizeDismissedAgentTaskMenuItemIDs([]string{
		"unknown", settings.TaskMenuItemDetectedClaudeID, settings.TaskMenuItemDetectedCodexID, settings.TaskMenuItemDetectedClaudeID,
	})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("规范化删除记录 = %#v，期望 %#v", got, want)
	}
}

func removeTaskMenuItemByID(items []settings.TaskMenuItem, id string) []settings.TaskMenuItem {
	filtered := make([]settings.TaskMenuItem, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func containsTaskMenuItemID(items []settings.TaskMenuItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func TestRepositoryAddsMissingShelvingMenuItemToPersistedSettings(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [],
  "settings": {
    "workspaceRoot": "` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `",
    "taskTreeWidth": 360,
    "taskMenuItems": [{"id":"system.edit-task","kind":"edit-task","name":"编辑任务"}]
  }
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	repository := New(dataPath, settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.Settings.TaskMenuItems) != 4 {
		t.Fatalf("规范化后的任务菜单项数量 = %d，期望 4", len(data.Settings.TaskMenuItems))
	}
	last := data.Settings.TaskMenuItems[len(data.Settings.TaskMenuItems)-1]
	if last.ID != settings.TaskMenuItemToggleShelvedID || last.Name != "搁置任务" || last.UnshelveName == nil || *last.UnshelveName != "取消搁置" {
		t.Fatalf("补齐的搁置菜单项 = %#v", last)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(persisted), settings.TaskMenuItemToggleShelvedID) {
		t.Fatal("补齐的搁置菜单项未持久化")
	}
}

func TestRepositoryLoadsMissingTerminalFontSizeAsDefault(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents, err := json.Marshal(map[string]any{
		"tasks": []task.Task{},
		"settings": map[string]any{
			"workspaceRoot": filepath.Join(t.TempDir(), "workspaces"),
			"taskTreeWidth": settings.DefaultTaskTreeWidth,
		},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := data.Settings.TerminalFontSize, settings.DefaultTerminalFontSize; got != want {
		t.Fatalf("Load() TerminalFontSize = %d, want %d", got, want)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var stored struct {
		Settings settings.Settings `json:"settings"`
	}
	if err := json.Unmarshal(persisted, &stored); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := stored.Settings.TerminalFontSize, settings.DefaultTerminalFontSize; got != want {
		t.Fatalf("持久化 TerminalFontSize = %d, want %d", got, want)
	}
}

func TestRepositoryLoadsMissingGitScanDepthAsDefault(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents, err := json.Marshal(map[string]any{
		"tasks": []task.Task{},
		"settings": map[string]any{
			"workspaceRoot": filepath.Join(t.TempDir(), "workspaces"),
			"taskTreeWidth": settings.DefaultTaskTreeWidth,
		},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := data.Settings.GitScanDepth, settings.DefaultGitScanDepth; got != want {
		t.Fatalf("Load() GitScanDepth = %d, want %d", got, want)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var stored struct {
		Settings settings.Settings `json:"settings"`
	}
	if err := json.Unmarshal(persisted, &stored); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := stored.Settings.GitScanDepth, settings.DefaultGitScanDepth; got != want {
		t.Fatalf("持久化 GitScanDepth = %d, want %d", got, want)
	}
}

func TestRepositoryMigratesLegacyDefaultGitScanDepthAndKeepsLaterOverride(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	legacy := settings.Default(t.TempDir())
	legacy.PresetVersion = 5
	legacy.GitScanDepth = 2
	contents, err := json.Marshal(Data{Tasks: []task.Task{}, Settings: legacy})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	repository := New(dataPath, settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Settings.PresetVersion != settings.CurrentPresetVersion || data.Settings.GitScanDepth != 3 {
		t.Fatalf("迁移后的设置 = version %d, depth %d", data.Settings.PresetVersion, data.Settings.GitScanDepth)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var stored Data
	if err := json.Unmarshal(persisted, &stored); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if stored.Settings.GitScanDepth != 3 {
		t.Fatalf("持久化 GitScanDepth = %d，期望 3", stored.Settings.GitScanDepth)
	}

	override := data.Settings
	override.GitScanDepth = 2
	if _, err := repository.SaveSettings(override); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	reloaded, err := repository.Load()
	if err != nil {
		t.Fatalf("再次 Load() error = %v", err)
	}
	if reloaded.Settings.GitScanDepth != 2 {
		t.Fatalf("显式保存后的 GitScanDepth = %d，期望 2", reloaded.Settings.GitScanDepth)
	}
}

func TestRepositoryLoadsMissingTerminalThemeAsDefault(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents, err := json.Marshal(map[string]any{
		"tasks": []task.Task{},
		"settings": map[string]any{
			"workspaceRoot": filepath.Join(t.TempDir(), "workspaces"),
			"taskTreeWidth": settings.DefaultTaskTreeWidth,
		},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := data.Settings.TerminalTheme, settings.DefaultTerminalTheme(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() TerminalTheme = %#v, want %#v", got, want)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var stored struct {
		Settings settings.Settings `json:"settings"`
	}
	if err := json.Unmarshal(persisted, &stored); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := stored.Settings.TerminalTheme, settings.DefaultTerminalTheme(); !reflect.DeepEqual(got, want) {
		t.Fatalf("持久化 TerminalTheme = %#v, want %#v", got, want)
	}
}

func TestRepositoryLoadsMissingTerminalNoteTemplateAsDefault(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents, err := json.Marshal(map[string]any{
		"tasks": []task.Task{},
		"settings": map[string]any{
			"workspaceRoot": filepath.Join(t.TempDir(), "workspaces"),
			"taskTreeWidth": settings.DefaultTaskTreeWidth,
		},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := data.Settings.TerminalNoteTemplate, settings.DefaultTerminalNoteTemplate(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() TerminalNoteTemplate = %#v, want %#v", got, want)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var stored struct {
		Settings settings.Settings `json:"settings"`
	}
	if err := json.Unmarshal(persisted, &stored); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := stored.Settings.TerminalNoteTemplate, settings.DefaultTerminalNoteTemplate(); !reflect.DeepEqual(got, want) {
		t.Fatalf("持久化 TerminalNoteTemplate = %#v, want %#v", got, want)
	}
}

func TestRepositoryPreservesExplicitEmptyTerminalNoteTemplate(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents, err := json.Marshal(map[string]any{
		"tasks": []task.Task{},
		"settings": map[string]any{
			"workspaceRoot": filepath.Join(t.TempDir(), "workspaces"),
			"taskTreeWidth": settings.DefaultTaskTreeWidth,
			"terminalNoteTemplate": map[string]string{
				"originalPrefix": "",
				"notePrefix":     "",
				"listSuffix":     "",
			},
		},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := data.Settings.TerminalNoteTemplate, (settings.TerminalNoteTemplate{}); !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() TerminalNoteTemplate = %#v, want %#v", got, want)
	}
}

func TestRepositoryReturnsDefaultsForMissingDataFile(t *testing.T) {
	defaults := settings.Default(t.TempDir())
	repository := New(filepath.Join(t.TempDir(), "state.json"), defaults)

	data, err := repository.Load()

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.Tasks) != 0 {
		t.Errorf("Load() Tasks = %d, want 0", len(data.Tasks))
	}
	if !reflect.DeepEqual(data.Settings, defaults) {
		t.Errorf("Load() Settings = %#v, want %#v", data.Settings, defaults)
	}
	if data.ExtraInfoTemplates == nil {
		t.Error("Load() ExtraInfoTemplates = nil, want empty slice")
	}
}

func TestRepositoryNormalizesMissingTaskTemplateData(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [{"id":"legacy","title":"旧任务","color":"#4f46e5","extraInfo":[]}],
  "settings": {"workspaceRoot":"` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `","taskTreeWidth":360}
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.Settings.TaskTemplates) != 1 || data.Settings.ActiveTaskTemplateID != "preset.task-template.default-branch" {
		t.Fatalf("旧设置的任务模板迁移 = %#v", data.Settings)
	}
	template := data.Settings.TaskTemplates[0]
	if template.ID != "preset.task-template.default-branch" || template.Name != "默认分支" || len(template.Fields) != 1 || template.Fields[0].Key != "branch" || !template.Fields[0].Required {
		t.Fatalf("迁移后的默认分支模板 = %#v", template)
	}
	if data.Tasks[0].TemplateFields == nil || len(data.Tasks[0].TemplateFields) != 0 {
		t.Fatalf("旧任务的模板字段迁移 = %#v", data.Tasks[0].TemplateFields)
	}
}

func TestRepositoryRepairsVersionTwoMissingDefaultBranchTemplateOnce(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [],
  "extraInfoTemplates": [],
  "extraInfos": [],
  "settings": {
    "workspaceRoot": "` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `",
    "taskTreeWidth": 360,
    "taskTemplates": [],
    "activeTaskTemplateId": "",
    "presetVersion": 2
  }
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	repository := New(dataPath, settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Settings.PresetVersion != settings.CurrentPresetVersion || len(data.Settings.TaskTemplates) != 1 || data.Settings.TaskTemplates[0].ID != settings.DefaultBranchTaskTemplateID || data.Settings.ActiveTaskTemplateID != settings.DefaultBranchTaskTemplateID {
		t.Fatalf("修复后的预置设置 = %#v", data.Settings)
	}

	deleted := data.Settings
	deleted.TaskTemplates = []task.TaskTemplate{}
	deleted.ActiveTaskTemplateID = ""
	if _, err := repository.SaveSettings(deleted); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	reloaded, err := repository.Load()
	if err != nil {
		t.Fatalf("第二次 Load() error = %v", err)
	}
	if len(reloaded.Settings.TaskTemplates) != 0 || reloaded.Settings.ActiveTaskTemplateID != "" {
		t.Fatalf("版本更新后被删除的模板不应重建: %#v", reloaded.Settings)
	}
}

func TestRepositorySeedsRepositoryPresetChainsOnlyOnce(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [],
  "extraInfoTemplates": [],
  "extraInfos": [],
  "settings": {
    "workspaceRoot": "` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `",
    "taskTreeWidth": 360,
    "taskTemplates": [],
    "lifecycleChains": [],
    "lifecycleDefaultChains": {}
  }
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	repository := New(dataPath, settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if lifecycleCommandChainIndex(data.Settings.LifecycleChains, "preset.lifecycle-chain.iterations-ai") < 0 || lifecycleCommandChainIndex(data.Settings.LifecycleChains, "preset.lifecycle-chain.update-repositories") < 0 {
		t.Fatalf("迁移后的预置链 = %#v", data.Settings.LifecycleChains)
	}
	updateIndex := lifecycleCommandChainIndex(data.Settings.LifecycleChains, settings.LifecycleChainUpdateRepositoriesID)
	if data.Settings.LifecycleChains[updateIndex].Name != "更新仓库" || data.Settings.DefaultLifecyclePresetID != settings.DefaultLifecyclePresetID || len(data.Settings.LifecyclePresets) != 1 {
		t.Fatalf("历史数据迁移语义被改写: 链 = %#v，预设 = %#v，默认 = %q", data.Settings.LifecycleChains[updateIndex], data.Settings.LifecyclePresets, data.Settings.DefaultLifecyclePresetID)
	}
	if err := repository.DeleteLifecycleCommandChain("preset.lifecycle-chain.iterations-ai"); err != nil {
		t.Fatalf("DeleteLifecycleCommandChain() error = %v", err)
	}

	reloaded, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("第二次 Load() error = %v", err)
	}
	if lifecycleCommandChainIndex(reloaded.Settings.LifecycleChains, "preset.lifecycle-chain.iterations-ai") >= 0 {
		t.Fatalf("删除后的预置链被重新创建: %#v", reloaded.Settings.LifecycleChains)
	}
	if lifecycleCommandChainIndex(reloaded.Settings.LifecycleChains, "preset.lifecycle-chain.update-repositories") < 0 {
		t.Fatalf("未删除的预置链丢失: %#v", reloaded.Settings.LifecycleChains)
	}
}

func TestRepositoryMigratesVersionThreeDefaultBranchPresetChains(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*settings.Settings)
		verify func(*testing.T, settings.Settings)
	}{
		{
			name: "精确匹配的预置链前置默认分支命令",
			verify: func(t *testing.T, current settings.Settings) {
				t.Helper()
				for _, chainID := range []string{settings.LifecycleChainIterationsAIID, settings.LifecycleChainUpdateRepositoriesID} {
					index := lifecycleCommandChainIndex(current.LifecycleChains, chainID)
					if index < 0 || len(current.LifecycleChains[index].Commands) == 0 || current.LifecycleChains[index].Commands[0].CommandID != settings.LifecycleCommandUpdateDefaultBranchID {
						t.Fatalf("%s 迁移后的命令链 = %#v", chainID, current.LifecycleChains)
					}
				}
			},
		},
		{
			name: "已修改的预置链保持不变",
			mutate: func(current *settings.Settings) {
				index := lifecycleCommandChainIndex(current.LifecycleChains, settings.LifecycleChainIterationsAIID)
				current.LifecycleChains[index].Name = "自定义 iterations-ai"
			},
			verify: func(t *testing.T, current settings.Settings) {
				t.Helper()
				index := lifecycleCommandChainIndex(current.LifecycleChains, settings.LifecycleChainIterationsAIID)
				chain := current.LifecycleChains[index]
				if chain.Name != "自定义 iterations-ai" || chain.Commands[0].CommandID == settings.LifecycleCommandUpdateDefaultBranchID {
					t.Fatalf("已修改的预置链被改写: %#v", chain)
				}
			},
		},
		{
			name: "已删除的预置链不重建",
			mutate: func(current *settings.Settings) {
				index := lifecycleCommandChainIndex(current.LifecycleChains, settings.LifecycleChainUpdateRepositoriesID)
				current.LifecycleChains = append(current.LifecycleChains[:index], current.LifecycleChains[index+1:]...)
			},
			verify: func(t *testing.T, current settings.Settings) {
				t.Helper()
				if lifecycleCommandChainIndex(current.LifecycleChains, settings.LifecycleChainUpdateRepositoriesID) >= 0 {
					t.Fatalf("已删除的预置链被重建: %#v", current.LifecycleChains)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dataPath := filepath.Join(t.TempDir(), "state.json")
			legacySettings := versionThreeRepositoryPresetSettings(t.TempDir())
			if test.mutate != nil {
				test.mutate(&legacySettings)
			}
			contents, err := json.Marshal(Data{
				Tasks:               []task.Task{},
				ExtraInfoTemplates:  []task.ExtraInfoTemplate{},
				ExtraInfos:          []task.ExtraInfo{},
				ExtraInfoCatalogues: []string{},
				Settings:            legacySettings,
			})
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			data, err := New(dataPath, settings.Default(t.TempDir())).Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if data.Settings.PresetVersion != settings.CurrentPresetVersion {
				t.Fatalf("迁移后的预置版本 = %d，期望 %d", data.Settings.PresetVersion, settings.CurrentPresetVersion)
			}
			test.verify(t, data.Settings)

			reloaded, err := New(dataPath, settings.Default(t.TempDir())).Load()
			if err != nil {
				t.Fatalf("第二次 Load() error = %v", err)
			}
			test.verify(t, reloaded.Settings)
		})
	}
}

func versionThreeRepositoryPresetSettings(workspaceRoot string) settings.Settings {
	current := settings.Default(workspaceRoot)
	current.LifecycleChains = settings.DefaultLifecycleChains()
	current.LifecyclePresets = settings.DefaultLifecyclePresets()
	current.DefaultLifecyclePresetID = settings.DefaultLifecyclePresetID
	current.PresetVersion = 3
	for index := range current.LifecycleChains {
		switch current.LifecycleChains[index].ID {
		case settings.LifecycleChainIterationsAIID:
			current.LifecycleChains[index].Commands = []settings.LifecycleCommandReference{
				{CommandID: settings.LifecycleCommandCreateWorkspaceID, Arguments: []string{}},
				{CommandID: settings.LifecycleCommandGitCloneRepositoryID, Arguments: []string{"repository=" + settings.IterationsAIRepository}},
				{CommandID: settings.LifecycleCommandManifestFileID, Arguments: []string{}},
				{CommandID: settings.LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
			}
		case settings.LifecycleChainUpdateRepositoriesID:
			current.LifecycleChains[index].Commands = []settings.LifecycleCommandReference{
				{CommandID: settings.LifecycleCommandManifestFileID, Arguments: []string{}},
				{CommandID: settings.LifecycleCommandGitCloneID, Arguments: []string{"dir=workspaces"}},
			}
		}
	}
	return current
}

func TestRepositoryPresetMigrationKeepsExistingActiveTaskTemplate(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [],
  "extraInfoTemplates": [],
  "extraInfos": [],
  "settings": {
    "workspaceRoot": "` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `",
    "taskTreeWidth": 360,
    "taskTemplates": [{
      "id": "release",
      "name": "发布任务",
      "fields": [{"key":"environment","displayName":"环境","inputType":"string","required":true,"defaultValue":"production"}]
    }],
    "activeTaskTemplateId": "release"
  }
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Settings.ActiveTaskTemplateID != "release" {
		t.Fatalf("迁移后当前模板 = %q，期望保留 release", data.Settings.ActiveTaskTemplateID)
	}
	if len(data.Settings.TaskTemplates) != 2 || data.Settings.TaskTemplates[0].ID != "release" || data.Settings.TaskTemplates[1].ID != settings.DefaultBranchTaskTemplateID {
		t.Fatalf("迁移后的任务模板 = %#v", data.Settings.TaskTemplates)
	}
}

func TestRepositoryRejectsChangingTypeOfUsedTaskTemplateField(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskTemplates = []task.TaskTemplate{{
		ID: "release", Name: "发布", Fields: []task.TaskTemplateField{{
			Key: "deploy", DisplayName: "允许部署", InputType: task.TaskTemplateFieldInputBool, DefaultValue: false,
		}},
	}}
	data.Settings.ActiveTaskTemplateID = "release"
	data.Tasks = append(data.Tasks, task.Task{
		ID: "task-1", Title: "部署任务", Color: task.DefaultColor, Status: task.StatusPending,
		ExtraInfo: []task.TaskExtraInfo{}, TemplateFields: map[string]any{"deploy": false}, LifecycleChains: map[task.LifecycleHook]string{},
	})
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	next := data.Settings
	next.TaskTemplates[0].Fields[0].InputType = task.TaskTemplateFieldInputString
	next.TaskTemplates[0].Fields[0].DefaultValue = "false"
	if _, err := repository.SaveSettings(next); err == nil {
		t.Fatal("SaveSettings() error = nil，期望拒绝改变已使用字段类型")
	}
}

func TestRepositoryRejectsRemovingTaskTemplateUsedByActiveTask(t *testing.T) {
	for _, status := range []task.Status{task.StatusPending, task.StatusRunning} {
		t.Run(string(status), func(t *testing.T) {
			repository, initialSettings := repositoryWithTaskTemplateReference(t, status, "release")
			next := initialSettings
			next.TaskTemplates = []task.TaskTemplate{}
			next.ActiveTaskTemplateID = ""

			if _, err := repository.SaveSettings(next); err == nil {
				t.Fatal("SaveSettings() error = nil，期望拒绝删除被活动任务引用的模板")
			}

			persisted, err := repository.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if len(persisted.Settings.TaskTemplates) != 1 || persisted.Settings.TaskTemplates[0].ID != "release" {
				t.Fatalf("删除失败后模板设置 = %#v，期望保留 release", persisted.Settings.TaskTemplates)
			}
		})
	}
}

func TestRepositoryRejectsRemovingTaskTemplateWithLegacyActiveTemplateFields(t *testing.T) {
	for _, status := range []task.Status{task.StatusPending, task.StatusRunning} {
		t.Run(string(status), func(t *testing.T) {
			repository, initialSettings := repositoryWithTaskTemplateReference(t, status, "")
			next := initialSettings
			next.TaskTemplates = []task.TaskTemplate{}
			next.ActiveTaskTemplateID = ""

			if _, err := repository.SaveSettings(next); err == nil {
				t.Fatal("SaveSettings() error = nil，期望拒绝删除包含历史活动模板字段时的模板")
			}

			persisted, err := repository.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if len(persisted.Settings.TaskTemplates) != 1 || persisted.Settings.TaskTemplates[0].ID != "release" {
				t.Fatalf("删除失败后模板设置 = %#v，期望保留 release", persisted.Settings.TaskTemplates)
			}
		})
	}
}

func TestRepositoryAllowsRemovingTaskTemplateUsedOnlyByCompletedTask(t *testing.T) {
	for _, scenario := range []struct {
		name           string
		taskTemplateID string
	}{
		{name: "已关联模板", taskTemplateID: "release"},
		{name: "历史模板字段"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository, initialSettings := repositoryWithTaskTemplateReference(t, task.StatusCompleted, scenario.taskTemplateID)
			next := initialSettings
			next.TaskTemplates = []task.TaskTemplate{}
			next.ActiveTaskTemplateID = ""

			if _, err := repository.SaveSettings(next); err != nil {
				t.Fatalf("SaveSettings() error = %v", err)
			}

			persisted, err := repository.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if len(persisted.Settings.TaskTemplates) != 0 || persisted.Settings.ActiveTaskTemplateID != "" {
				t.Fatalf("已完成任务不应阻止模板删除: %#v", persisted.Settings)
			}
		})
	}
}

func TestRepositorySaveSettingsWithoutTaskTemplatesKeepsActiveTaskTemplate(t *testing.T) {
	repository, initialSettings := repositoryWithTaskTemplateReference(t, task.StatusRunning, "release")
	next := initialSettings
	next.TaskTemplates = nil
	next.ActiveTaskTemplateID = ""
	next.TaskTreeWidth = settings.DefaultTaskTreeWidth + 20

	saved, err := repository.SaveSettings(next)
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if len(saved.TaskTemplates) != 1 || saved.TaskTemplates[0].ID != "release" || saved.ActiveTaskTemplateID != "release" {
		t.Fatalf("未提供模板快照的设置保存改写了模板: %#v", saved)
	}
}

func TestRepositorySaveSettingsDistinguishesMissingAndExplicitEmptyTaskTemplates(t *testing.T) {
	t.Run("缺失模板集合保留已有定义", func(t *testing.T) {
		repository, initialSettings := repositoryWithTaskTemplateReference(t, task.StatusRunning, "release")
		next := initialSettings
		next.TaskTemplates = nil
		next.ActiveTaskTemplateID = ""

		if _, err := repository.SaveSettings(next); err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}
		loaded, err := repository.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if len(loaded.Settings.TaskTemplates) != 1 || loaded.Settings.TaskTemplates[0].ID != "release" || loaded.Settings.ActiveTaskTemplateID != "release" {
			t.Fatalf("缺失模板快照后的设置 = %#v", loaded.Settings)
		}
	})

	t.Run("显式空模板集合继续删除", func(t *testing.T) {
		repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
		current, err := repository.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		next := current.Settings
		next.TaskTemplates = []task.TaskTemplate{}
		next.ActiveTaskTemplateID = ""

		if _, err := repository.SaveSettings(next); err != nil {
			t.Fatalf("SaveSettings() error = %v", err)
		}
		loaded, err := repository.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if len(loaded.Settings.TaskTemplates) != 0 || loaded.Settings.ActiveTaskTemplateID != "" {
			t.Fatalf("显式清空模板后的设置 = %#v", loaded.Settings)
		}
	})
}

func repositoryWithTaskTemplateReference(t *testing.T, status task.Status, taskTemplateID string) (*Repository, settings.Settings) {
	t.Helper()
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskTemplates = []task.TaskTemplate{{
		ID: "release", Name: "发布", Fields: []task.TaskTemplateField{{
			Key: "environment", DisplayName: "环境", InputType: task.TaskTemplateFieldInputString, DefaultValue: "production",
		}},
	}}
	data.Settings.ActiveTaskTemplateID = "release"
	data.Tasks = append(data.Tasks, task.Task{
		ID: "task-1", Title: "发布任务", Color: task.DefaultColor, Status: status,
		ExtraInfo: []task.TaskExtraInfo{}, TaskTemplateID: taskTemplateID,
		TemplateFields: map[string]any{"environment": "production"}, LifecycleChains: map[task.LifecycleHook]string{},
	})
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	return repository, data.Settings
}

func TestRepositoryListsBuiltInGitTemplateAsJSONArray(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))

	templates, err := repository.ListExtraInfoTemplates()
	if err != nil {
		t.Fatalf("ListExtraInfoTemplates() error = %v", err)
	}
	encoded, err := json.Marshal(templates)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(encoded); got == "[]" || len(templates) != 1 || templates[0].Catalogue != "git" {
		t.Fatalf("内置额外信息模板 JSON = %s，期望包含 Git 模板", got)
	}
}

func TestRepositorySeedsBuiltInGitTemplate(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))

	templates, err := repository.ListExtraInfoTemplates()
	if err != nil {
		t.Fatalf("ListExtraInfoTemplates() error = %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("内置模板数量 = %d，期望 1", len(templates))
	}
	gitTemplate := templates[0]
	if !gitTemplate.BuiltIn || gitTemplate.Catalogue != "git" {
		t.Fatalf("Git 模板 = %#v", gitTemplate)
	}
	if len(gitTemplate.Fields) != 2 || gitTemplate.Fields[0].Key != "name" || gitTemplate.Fields[0].DisplayName != "项目名称" || gitTemplate.Fields[1].Key != "repository" || gitTemplate.Fields[1].DisplayName != "仓库地址" {
		t.Fatalf("Git 固定字段 = %#v", gitTemplate.Fields)
	}
	if len(gitTemplate.Parameters) != 1 || gitTemplate.Parameters[0].Key != "branch" || gitTemplate.Parameters[0].DisplayName != "仓库分支" {
		t.Fatalf("Git 动态参数 = %#v", gitTemplate.Parameters)
	}
}

func TestRepositoryMigratesCurrentTemplateIntoTemplateAndInformation(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [{
    "id": "task-1",
    "title": "部署 API",
    "color": "#4f46e5",
    "extraInfo": [{
      "id": "legacy-api",
      "catalogue": "git",
      "displayName": "API 服务",
      "fields": [{"key": "repository", "displayName": "仓库地址", "value": "git@example.com:team/api.git"}],
      "parameters": [{"key": "branch", "displayName": "分支", "required": true, "value": "main"}]
    }]
  }],
  "extraInfoCatalogues": ["git"],
  "extraInfoTemplates": [{
    "id": "legacy-api",
    "catalogue": "git",
    "displayName": "API 服务",
    "fields": [{"key": "repository", "displayName": "仓库地址", "value": "git@example.com:team/api.git"}],
    "parameters": [{"key": "branch", "displayName": "分支", "required": true}]
  }],
  "settings": {}
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.ExtraInfos) != 1 {
		t.Fatalf("迁移后的额外信息数量 = %d，期望 1", len(data.ExtraInfos))
	}
	if got := task.ExtraInfoName(data.ExtraInfos[0]); got != "API 服务" {
		t.Fatalf("迁移信息名称 = %q，期望 %q", got, "API 服务")
	}
	if got := extraInfoValue(data.ExtraInfos[0], "repository"); got != "git@example.com:team/api.git" {
		t.Fatalf("迁移信息仓库地址 = %q", got)
	}
	if len(data.ExtraInfoTemplates) != 2 {
		t.Fatalf("迁移模板数量 = %d，期望 Git 内置模板和一个旧模板", len(data.ExtraInfoTemplates))
	}
	if got := data.Tasks[0].ExtraInfo[0].Fields[0].Key; got != "repository" {
		t.Fatalf("历史任务快照字段 = %q，期望 repository", got)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !jsonContainsKey(persisted, "extraInfos") {
		t.Fatalf("迁移后的数据未持久化 extraInfos: %s", persisted)
	}
	loadedAgain, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("第二次 Load() error = %v", err)
	}
	if !reflect.DeepEqual(loadedAgain.ExtraInfos, data.ExtraInfos) {
		t.Fatalf("迁移不是幂等的: %#v != %#v", loadedAgain.ExtraInfos, data.ExtraInfos)
	}
}

func TestRepositoryMigratesLegacyTemplatesWithDifferentSchemasIntoSeparateCatalogues(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [],
  "extraInfoTemplates": [
    {"id":"api","catalogue":"git","displayName":"API","fields":[{"key":"repository","displayName":"仓库地址","value":"git@example.com:team/api.git"}]},
    {"id":"web","catalogue":"git","displayName":"Web","fields":[{"key":"repository","displayName":"仓库地址","value":"git@example.com:team/web.git"},{"key":"remote","displayName":"远程名称","value":"origin"}]},
    {"id":"legacy-key","catalogue":"issue","displayName":"缺陷","key":"url","keyDisplayName":"链接","value":"https://example.com/issues/1"}
  ],
  "settings": {}
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.ExtraInfoTemplates) != 4 || len(data.ExtraInfos) != 3 {
		t.Fatalf("迁移结果模板/信息数量 = %d/%d，期望 4/3", len(data.ExtraInfoTemplates), len(data.ExtraInfos))
	}
	catalogues := make(map[string]bool, len(data.ExtraInfoTemplates))
	for _, template := range data.ExtraInfoTemplates {
		catalogues[template.Catalogue] = true
	}
	if !catalogues["git"] || !catalogues["git-legacy"] || !catalogues["git-legacy-2"] || !catalogues["issue"] {
		t.Fatalf("迁移分类 = %#v，期望 Git 内置和不同结构的稳定后缀分类", catalogues)
	}
	if got := extraInfoValue(data.ExtraInfos[2], "url"); got != "https://example.com/issues/1" {
		t.Fatalf("旧单键格式迁移值 = %q", got)
	}
}

func extraInfoValue(info task.ExtraInfo, key string) string {
	for _, field := range info.Fields {
		if field.Key == key {
			return field.Value
		}
	}
	return ""
}

func jsonContainsKey(contents []byte, key string) bool {
	var decoded map[string]json.RawMessage
	return json.Unmarshal(contents, &decoded) == nil && decoded[key] != nil
}

func TestRepositoryRequiresDeletingInformationBeforeTemplate(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	template, err := task.NewExtraInfoTemplate("deployment", "部署", []task.ExtraInfoField{{Key: "environment", DisplayName: "环境", DefaultValue: "test"}}, nil)
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}
	if _, err := repository.SaveExtraInfoTemplate(template); err != nil {
		t.Fatalf("SaveExtraInfoTemplate() error = %v", err)
	}
	info, err := task.NewExtraInfo(template, map[string]string{"name": "测试环境"})
	if err != nil {
		t.Fatalf("NewExtraInfo() error = %v", err)
	}
	if _, err := repository.SaveExtraInfo(info); err != nil {
		t.Fatalf("SaveExtraInfo() error = %v", err)
	}
	if err := repository.DeleteExtraInfoTemplate(template.ID); err == nil {
		t.Fatal("DeleteExtraInfoTemplate() error = nil，期望信息仍引用模板时失败")
	}
	renamed := template
	renamed.Catalogue = "release"
	if _, err := repository.SaveExtraInfoTemplate(renamed); err == nil {
		t.Fatal("SaveExtraInfoTemplate() error = nil，期望引用信息存在时不能修改分类")
	}
	if err := repository.DeleteExtraInfo(info.ID); err != nil {
		t.Fatalf("DeleteExtraInfo() error = %v", err)
	}
	if err := repository.DeleteExtraInfoTemplate(template.ID); err != nil {
		t.Fatalf("DeleteExtraInfoTemplate() error = %v", err)
	}
}

func TestRepositoryPreservesInformationParametersAndLoadsLegacyInformation(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	repository := New(dataPath, settings.Default(t.TempDir()))
	gitTemplate := task.BuiltInGitTemplate()
	info, err := repository.SaveExtraInfo(task.ExtraInfo{
		TemplateID: gitTemplate.ID,
		Catalogue:  gitTemplate.Catalogue,
		Fields: []task.ExtraInfoField{
			{Key: "name", Value: "API 服务"},
			{Key: "repository", Value: "git@example.com:team/api.git"},
		},
		Parameters: []task.ExtraInfoParameter{{Key: "environment", DisplayName: "环境", Required: true, Value: "production"}},
	})
	if err != nil {
		t.Fatalf("SaveExtraInfo() error = %v", err)
	}
	if len(info.Parameters) != 1 || info.Parameters[0].Value != "production" {
		t.Fatalf("SaveExtraInfo() 参数 = %#v", info.Parameters)
	}
	loaded, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.ExtraInfos) != 1 || len(loaded.ExtraInfos[0].Parameters) != 1 || loaded.ExtraInfos[0].Parameters[0].Key != "environment" {
		t.Fatalf("加载后的信息参数 = %#v", loaded.ExtraInfos)
	}

	legacyPath := filepath.Join(t.TempDir(), "legacy.json")
	legacy := []byte(`{"tasks":[],"extraInfoTemplates":[{"id":"builtin-extra-info-template-git","catalogue":"git","fields":[{"key":"name","displayName":"项目名称"},{"key":"repository","displayName":"仓库地址"}],"parameters":[{"key":"branch","displayName":"仓库分支","required":false}],"builtIn":true}],"extraInfos":[{"id":"legacy-info","templateId":"builtin-extra-info-template-git","catalogue":"git","fields":[{"key":"name","displayName":"项目名称","value":"旧服务"},{"key":"repository","displayName":"仓库地址","value":"git@example.com:team/legacy.git"}]}],"settings":{}}`)
	if err := os.WriteFile(legacyPath, legacy, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	legacyData, err := New(legacyPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() legacy error = %v", err)
	}
	if len(legacyData.ExtraInfos) != 1 || legacyData.ExtraInfos[0].Parameters == nil || len(legacyData.ExtraInfos[0].Parameters) != 0 {
		t.Fatalf("旧信息参数兼容结果 = %#v", legacyData.ExtraInfos)
	}
}

func TestRepositoryLoadsOldDataWithEmptyExtraInfoTemplates(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{"tasks":[],"settings":{}}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.ExtraInfoTemplates == nil || len(data.ExtraInfoTemplates) != 1 || data.ExtraInfoTemplates[0].Catalogue != "git" {
		t.Errorf("Load() ExtraInfoTemplates = %#v, want built-in Git template", data.ExtraInfoTemplates)
	}
}

func TestRepositoryMigratesLifecycleDefaultsForExistingTasks(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [
    {"id":"pending","title":"待开始","color":"#4f46e5","status":"pending","extraInfo":[]},
    {"id":"running","title":"执行中","color":"#4f46e5","status":"running","extraInfo":[]},
    {"id":"completed","title":"已完成","color":"#4f46e5","status":"completed","extraInfo":[]}
  ],
  "settings": {"workspaceRoot":"` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `","taskTreeWidth":360}
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data.Settings.LifecycleCommands) != 6 || len(data.Settings.LifecycleChains) != 4 {
		t.Fatalf("生命周期设置迁移 = %#v", data.Settings)
	}
	if got := data.Settings.LifecycleCommands[3].ID; got != settings.LifecycleCommandGitCloneRepositoryID {
		t.Fatalf("迁移后的新增固定命令 ID = %q，期望 %q", got, settings.LifecycleCommandGitCloneRepositoryID)
	}
	if got := data.Settings.LifecycleCommands[4].ID; got != settings.LifecycleCommandManifestFileID {
		t.Fatalf("迁移后的清单文件命令 ID = %q，期望 %q", got, settings.LifecycleCommandManifestFileID)
	}
	if got := data.Settings.LifecycleCommands[5].ID; got != settings.LifecycleCommandUpdateDefaultBranchID {
		t.Fatalf("迁移后的默认分支命令 ID = %q，期望 %q", got, settings.LifecycleCommandUpdateDefaultBranchID)
	}
	if got := data.Tasks[0].LifecycleChains; !reflect.DeepEqual(got, map[task.LifecycleHook]string{
		task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
		task.LifecycleHookPostEnd:     settings.LifecycleChainDeleteWorkspaceID,
	}) {
		t.Fatalf("未执行任务链选择 = %#v", got)
	}
	if got := data.Tasks[1].LifecycleChains; !reflect.DeepEqual(got, map[task.LifecycleHook]string{
		task.LifecycleHookPostEnd: settings.LifecycleChainDeleteWorkspaceID,
	}) {
		t.Fatalf("执行中任务链选择 = %#v", got)
	}
	if got := data.Tasks[2].LifecycleChains; len(got) != 0 {
		t.Fatalf("已完成任务链选择 = %#v，期望为空", got)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var persistedData map[string]json.RawMessage
	if err := json.Unmarshal(persisted, &persistedData); err != nil {
		t.Fatalf("Unmarshal persisted data error = %v", err)
	}
	if !jsonContainsKey(persistedData["settings"], "lifecycleCommands") || !jsonContainsKey(persistedData["settings"], "lifecycleChains") {
		t.Fatalf("迁移结果未持久化生命周期设置: %s", persisted)
	}
	var persistedSettings map[string]json.RawMessage
	if err := json.Unmarshal(persistedData["settings"], &persistedSettings); err != nil {
		t.Fatalf("Unmarshal persisted settings error = %v", err)
	}
	var persistedChains []map[string]json.RawMessage
	if err := json.Unmarshal(persistedSettings["lifecycleChains"], &persistedChains); err != nil {
		t.Fatalf("Unmarshal persisted chains error = %v", err)
	}
	if len(persistedChains) != 4 || persistedChains[0]["commands"] == nil || persistedChains[0]["commandIds"] != nil {
		t.Fatalf("迁移后的命令链结构 = %#v", persistedChains)
	}
}

func TestRepositoryMigratesLifecycleDefaultChainsToPreset(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		legacy      map[string]string
		wantChains  map[task.LifecycleHook]string
		wantPending map[task.LifecycleHook]string
		wantRunning map[task.LifecycleHook]string
	}{
		{
			name: "保留旧默认映射",
			legacy: map[string]string{
				string(task.LifecycleHookBeforeStart): settings.LifecycleChainCreateWorkspaceID,
				string(task.LifecycleHookPostEnd):     settings.LifecycleChainDeleteWorkspaceID,
			},
			wantChains: map[task.LifecycleHook]string{
				task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
				task.LifecycleHookPostEnd:     settings.LifecycleChainDeleteWorkspaceID,
			},
			wantPending: map[task.LifecycleHook]string{
				task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
				task.LifecycleHookPostEnd:     settings.LifecycleChainDeleteWorkspaceID,
			},
			wantRunning: map[task.LifecycleHook]string{task.LifecycleHookPostEnd: settings.LifecycleChainDeleteWorkspaceID},
		},
		{
			name:        "保留显式空映射",
			legacy:      map[string]string{},
			wantChains:  map[task.LifecycleHook]string{},
			wantPending: map[task.LifecycleHook]string{},
			wantRunning: map[task.LifecycleHook]string{},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			dataPath := filepath.Join(t.TempDir(), "state.json")
			contents, err := json.Marshal(map[string]any{
				"tasks": []map[string]any{
					{"id": "pending", "title": "待开始", "color": task.DefaultColor, "status": task.StatusPending, "extraInfo": []any{}},
					{"id": "running", "title": "执行中", "color": task.DefaultColor, "status": task.StatusRunning, "extraInfo": []any{}},
					{"id": "completed", "title": "已完成", "color": task.DefaultColor, "status": task.StatusCompleted, "extraInfo": []any{}},
				},
				"settings": map[string]any{
					"workspaceRoot":          filepath.Join(t.TempDir(), "workspaces"),
					"taskTreeWidth":          settings.DefaultTaskTreeWidth,
					"presetVersion":          4,
					"lifecycleDefaultChains": scenario.legacy,
				},
			})
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			repository := New(dataPath, settings.Default(t.TempDir()))
			data, err := repository.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			want := settings.LifecyclePreset{ID: settings.DefaultLifecyclePresetID, Name: "默认预设", Chains: scenario.wantChains}
			if data.Settings.DefaultLifecyclePresetID != settings.DefaultLifecyclePresetID || !reflect.DeepEqual(data.Settings.LifecyclePresets, []settings.LifecyclePreset{want}) {
				t.Fatalf("迁移后的生命周期预设 = %#v，默认 = %q", data.Settings.LifecyclePresets, data.Settings.DefaultLifecyclePresetID)
			}
			if !reflect.DeepEqual(data.Tasks[0].LifecycleChains, scenario.wantPending) || !reflect.DeepEqual(data.Tasks[1].LifecycleChains, scenario.wantRunning) || len(data.Tasks[2].LifecycleChains) != 0 {
				t.Fatalf("迁移后的任务链 = %#v", data.Tasks)
			}

			persisted, err := os.ReadFile(dataPath)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			var persistedData map[string]json.RawMessage
			if err := json.Unmarshal(persisted, &persistedData); err != nil {
				t.Fatalf("Unmarshal persisted data error = %v", err)
			}
			if jsonContainsKey(persistedData["settings"], "lifecycleDefaultChains") {
				t.Fatalf("旧默认映射仍被持久化: %s", persisted)
			}

			data.Settings.LifecyclePresets = []settings.LifecyclePreset{}
			data.Settings.DefaultLifecyclePresetID = ""
			if err := repository.Save(data); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			reloaded, err := repository.Load()
			if err != nil {
				t.Fatalf("第二次 Load() error = %v", err)
			}
			if len(reloaded.Settings.LifecyclePresets) != 0 || reloaded.Settings.DefaultLifecyclePresetID != "" {
				t.Fatalf("迁移完成后被删除的预设不应重建: %#v", reloaded.Settings)
			}
		})
	}
}

func TestRepositoryMigratesLifecycleApplicableHooks(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [],
  "settings": {
    "workspaceRoot": "` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `",
    "taskTreeWidth": 360,
    "lifecycleCommands": [{"id":"legacy-command","kind":"custom","name":"旧命令","command":"echo","arguments":[]}],
    "lifecycleChains": [{"id":"legacy-chain","name":"旧链","commandIds":["legacy-command"]}],
    "lifecycleDefaultChains": {"postStart":"legacy-chain"}
  }
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []settings.LifecycleHook{
		settings.LifecycleHookBeforeStart,
		settings.LifecycleHookPostStart,
		settings.LifecycleHookBeforeEnd,
		settings.LifecycleHookPostEnd,
		settings.LifecycleHookUpdateTask,
	}
	if !reflect.DeepEqual(data.Settings.LifecycleCommands[0].ApplicableHooks, want) || !reflect.DeepEqual(data.Settings.LifecycleChains[0].ApplicableHooks, want) {
		t.Fatalf("迁移后的命令链范围 = %#v", data.Settings)
	}
}

func TestRepositoryMigratesLegacyLifecycleCommandChainArgumentMode(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [],
  "settings": {
    "workspaceRoot": "` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `",
    "taskTreeWidth": 360,
    "lifecycleCommands": [{"id":"legacy-command","kind":"custom","name":"旧命令","command":"echo","arguments":["--fixed"],"applicableHooks":["beforeStart"]}],
    "lifecycleChains": [{"id":"legacy-chain","name":"旧链","commands":[{"commandId":"legacy-command","arguments":["--saved-extra"]}],"applicableHooks":["beforeStart"]}],
    "lifecycleDefaultChains": {"beforeStart":"legacy-chain"}
  }
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := data.Settings.LifecycleCommands[0].ChainArgumentMode; got != settings.LifecycleCommandChainArgumentModeEnabled {
		t.Fatalf("旧命令链级参数模式 = %q，期望允许", got)
	}
	if got := data.Settings.LifecycleCommands[0].Arguments; !reflect.DeepEqual(got, []string{"--fixed"}) {
		t.Fatalf("旧命令固定参数 = %#v", got)
	}
	if got := data.Settings.LifecycleChains[0].Commands[0].Arguments; !reflect.DeepEqual(got, []string{"--saved-extra"}) {
		t.Fatalf("旧命令链追加参数 = %#v", got)
	}

	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var persistedData struct {
		Settings struct {
			LifecycleCommands []struct {
				ChainArgumentMode settings.LifecycleCommandChainArgumentMode `json:"chainArgumentMode"`
			} `json:"lifecycleCommands"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(persisted, &persistedData); err != nil {
		t.Fatalf("Unmarshal persisted data error = %v", err)
	}
	if got := persistedData.Settings.LifecycleCommands[0].ChainArgumentMode; got != settings.LifecycleCommandChainArgumentModeEnabled {
		t.Fatalf("持久化后的链级参数模式 = %q，期望允许", got)
	}
}

func TestRepositoryPreservesLegacyChainWithoutCommonApplicableHook(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [],
  "settings": {
    "workspaceRoot": "` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `",
    "taskTreeWidth": 360,
    "lifecycleChains": [{"id":"legacy-mixed","name":"旧混合链","commandIds":["system.lifecycle.create-workspace","system.lifecycle.delete-workspace"]}],
    "lifecycleDefaultChains": {}
  }
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	repository := New(dataPath, settings.Default(t.TempDir()))
	if _, err := repository.Load(); err != nil {
		t.Fatalf("第一次 Load() error = %v", err)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("第二次 Load() error = %v", err)
	}
	if len(data.Settings.LifecycleChains) != 3 || data.Settings.LifecycleChains[0].ID != "legacy-mixed" || len(data.Settings.LifecycleChains[0].ApplicableHooks) != 0 {
		t.Fatalf("无共同范围的旧链 = %#v", data.Settings.LifecycleChains)
	}
}

func TestRepositoryMarksInterruptedLifecycleExecutionAsFailed(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [{
    "id":"running","title":"执行中","color":"#4f46e5","status":"running","extraInfo":[],
    "lifecycleChains":{"postStart":"chain"},
    "lifecycleExecution":{"runId":"restart-run","revision":4,"hook":"beforeStart","chainId":"chain","currentCommandId":"command","currentCommandName":"初始化工作区","currentIndex":1,"commandCount":1,"state":"running","workspaceRoot":"/tmp/taskai-workspaces","workspacePath":"/tmp/taskai-workspaces/running","workspaceOwnership":"created","workspaceToken":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
  }],
  "settings": {"workspaceRoot":"` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `","taskTreeWidth":360,
    "lifecycleCommands":[{"id":"command","kind":"custom","name":"命令","command":"echo","arguments":[]}],
    "lifecycleChains":[{"id":"chain","name":"链","commandIds":["command"]}],
    "lifecycleDefaultChains":{}}
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	execution := data.Tasks[0].LifecycleExecution
	if execution == nil || execution.State != task.LifecycleExecutionFailed || execution.Error == "" || execution.RunID != "restart-run" || execution.Revision != 4 || execution.CurrentCommandName != "初始化工作区" || execution.WorkspaceRoot != "/tmp/taskai-workspaces" || execution.WorkspacePath != "/tmp/taskai-workspaces/running" || execution.WorkspaceOwnership != task.LifecycleWorkspaceCreated {
		t.Fatalf("中断执行记录 = %#v", execution)
	}
}

func TestRepositoryMigratesLegacyLifecycleWorkspaceOwnershipToUnknown(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	contents := []byte(`{
  "tasks": [{
    "id":"pending","title":"未执行","color":"#4f46e5","status":"pending","extraInfo":[],
    "lifecycleChains":{"beforeStart":"chain"},
    "lifecycleExecution":{"hook":"beforeStart","chainId":"chain","currentCommandId":"command","currentCommandName":"初始化工作区","currentIndex":1,"commandCount":1,"state":"failed"}
  }],
  "settings": {"workspaceRoot":"` + filepath.ToSlash(filepath.Join(t.TempDir(), "workspaces")) + `","taskTreeWidth":360,
    "lifecycleCommands":[{"id":"command","kind":"custom","name":"命令","command":"echo","arguments":[]}],
    "lifecycleChains":[{"id":"chain","name":"链","commandIds":["command"]}],
    "lifecycleDefaultChains":{}}
}`)
	if err := os.WriteFile(dataPath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	execution := data.Tasks[0].LifecycleExecution
	if execution == nil || execution.WorkspaceOwnership != task.LifecycleWorkspaceUnknown {
		t.Fatalf("旧执行记录目录归属 = %#v，期望 unknown", execution)
	}
	persisted, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(persisted), `"workspaceOwnership": "unknown"`) {
		t.Fatalf("旧执行记录目录归属未回写: %s", persisted)
	}
}

func TestRepositoryKeepsInProcessLifecycleExecutionRunning(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("首次 Load() error = %v", err)
	}
	created, err := task.NewTask("运行中的任务", "", task.DefaultColor, time.Now())
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	created.LifecycleExecution = &task.LifecycleExecution{
		RunID:              "in-process-run",
		Revision:           1,
		Hook:               task.LifecycleHookPostStart,
		ChainID:            "chain",
		CurrentCommandID:   "command",
		CurrentCommandName: "初始化工作区",
		CurrentIndex:       1,
		CommandCount:       1,
		State:              task.LifecycleExecutionRunning,
	}
	data.Tasks = append(data.Tasks, created)
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := repository.Load()
	if err != nil {
		t.Fatalf("再次 Load() error = %v", err)
	}
	execution := loaded.Tasks[0].LifecycleExecution
	if execution == nil || execution.State != task.LifecycleExecutionRunning || execution.RunID != "in-process-run" || execution.CurrentCommandName != "初始化工作区" {
		t.Fatalf("进程内运行记录被错误迁移: %#v", execution)
	}
}

func TestRepositoryManagesLifecycleCommandsChainsAndDeletionConstraints(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	if _, err := repository.SaveLifecycleCommand(settings.LifecycleCommand{Name: "没有范围", Command: "prepare"}); err == nil {
		t.Fatal("SaveLifecycleCommand() error = nil，期望拒绝没有适用范围的命令")
	}
	command, err := repository.SaveLifecycleCommand(settings.LifecycleCommand{
		Name: "准备仓库", Command: "prepare", Arguments: []string{"--fast"}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart, settings.LifecycleHookPostStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommand() error = %v", err)
	}
	if command.ID == "" || command.Kind != settings.LifecycleCommandKindCustom {
		t.Fatalf("保存的生命周期命令 = %#v", command)
	}
	chain, err := repository.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
		Name: "开始前准备", Commands: []settings.LifecycleCommandReference{{CommandID: command.ID, Arguments: []string{}}}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommandChain() error = %v", err)
	}
	if chain.ID == "" || !reflect.DeepEqual(chain.Commands, []settings.LifecycleCommandReference{{CommandID: command.ID, Arguments: []string{}}}) {
		t.Fatalf("保存的生命周期链 = %#v", chain)
	}
	copy, err := repository.CopyLifecycleCommandChain(chain.ID)
	if err != nil {
		t.Fatalf("CopyLifecycleCommandChain() error = %v", err)
	}
	if copy.ID == chain.ID || !reflect.DeepEqual(copy.Commands, chain.Commands) {
		t.Fatalf("复制的生命周期链 = %#v，源链 = %#v", copy, chain)
	}

	if err := repository.DeleteLifecycleCommand(command.ID); err == nil {
		t.Fatal("DeleteLifecycleCommand() error = nil，期望拒绝删除仍被链引用的命令")
	}

	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Tasks = []task.Task{{
		ID: "pending", Title: "未执行", Color: task.DefaultColor, Status: task.StatusPending, ExtraInfo: []task.TaskExtraInfo{},
		LifecycleChains: map[task.LifecycleHook]string{task.LifecycleHookBeforeStart: chain.ID},
	}}
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := repository.DeleteLifecycleCommandChain(chain.ID); err == nil {
		t.Fatal("DeleteLifecycleCommandChain() error = nil，期望拒绝删除被未执行任务引用的链")
	}
	if _, err := repository.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
		ID: chain.ID, Name: chain.Name, Commands: chain.Commands, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookPostStart},
	}); err == nil {
		t.Fatal("SaveLifecycleCommandChain() error = nil，期望拒绝缩小未执行任务引用链的适用范围")
	}

	data.Tasks[0].Status = task.StatusCompleted
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() completed task error = %v", err)
	}
	if err := repository.DeleteLifecycleCommandChain(chain.ID); err != nil {
		t.Fatalf("DeleteLifecycleCommandChain() completed chain error = %v", err)
	}
	if err := repository.DeleteLifecycleCommandChain(copy.ID); err != nil {
		t.Fatalf("DeleteLifecycleCommandChain() copied chain error = %v", err)
	}
	if err := repository.DeleteLifecycleCommand(command.ID); err != nil {
		t.Fatalf("DeleteLifecycleCommand() unreferenced command error = %v", err)
	}
}

func TestRepositorySaveSettingsPreservesLifecycleConfiguration(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		staleSettings func(settings.Settings) settings.Settings
	}{
		{
			name: "缺少生命周期字段",
			staleSettings: func(next settings.Settings) settings.Settings {
				next.LifecycleCommands = nil
				next.LifecycleChains = nil
				next.LifecyclePresets = nil
				next.DefaultLifecyclePresetID = ""
				return next
			},
		},
		{
			name: "携带过期生命周期字段",
			staleSettings: func(next settings.Settings) settings.Settings {
				return next
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
			stale, err := repository.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			command, err := repository.SaveLifecycleCommand(settings.LifecycleCommand{
				Name: "准备仓库", Command: "prepare", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
			})
			if err != nil {
				t.Fatalf("SaveLifecycleCommand() error = %v", err)
			}
			chain, err := repository.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
				Name: "开始前准备", Commands: []settings.LifecycleCommandReference{{CommandID: command.ID}}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
			})
			if err != nil {
				t.Fatalf("SaveLifecycleCommandChain() error = %v", err)
			}
			current, err := repository.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			defaultPresetIndex := lifecyclePresetIndex(current.Settings.LifecyclePresets, current.Settings.DefaultLifecyclePresetID)
			if defaultPresetIndex < 0 {
				t.Fatalf("找不到当前默认预设: %#v", current.Settings.LifecyclePresets)
			}
			current.Settings.LifecyclePresets[defaultPresetIndex].Chains[task.LifecycleHookBeforeStart] = chain.ID
			if err := repository.Save(current); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			next := scenario.staleSettings(stale.Settings)
			next.TaskTreeWidth = settings.DefaultTaskTreeWidth + 40
			saved, err := repository.SaveSettings(next)
			if err != nil {
				t.Fatalf("SaveSettings() error = %v", err)
			}
			if saved.TaskTreeWidth != next.TaskTreeWidth {
				t.Fatalf("普通设置未保存: taskTreeWidth = %d，期望 %d", saved.TaskTreeWidth, next.TaskTreeWidth)
			}
			if lifecycleCommandIndex(saved.LifecycleCommands, command.ID) < 0 {
				t.Fatalf("普通设置保存后丢失生命周期命令: %#v", saved.LifecycleCommands)
			}
			if lifecycleCommandChainIndex(saved.LifecycleChains, chain.ID) < 0 {
				t.Fatalf("普通设置保存后丢失生命周期命令链: %#v", saved.LifecycleChains)
			}
			if got := saved.DefaultLifecyclePresetChains()[task.LifecycleHookBeforeStart]; got != chain.ID {
				t.Fatalf("普通设置保存后默认链 = %q，期望 %q", got, chain.ID)
			}
		})
	}
}

func TestRepositoryManagesLifecyclePresets(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.TaskTreeWidth = settings.DefaultTaskTreeWidth + 40
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	command, err := repository.SaveLifecycleCommand(settings.LifecycleCommand{
		Name: "开始后准备", Command: "prepare", ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookPostStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommand() error = %v", err)
	}
	chain, err := repository.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
		Name: "开始后准备", Commands: []settings.LifecycleCommandReference{{CommandID: command.ID}}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookPostStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommandChain() error = %v", err)

	}
	preset, err := repository.SaveLifecyclePreset(settings.LifecyclePreset{
		Name:   " 开始后预设 ",
		Chains: map[task.LifecycleHook]string{task.LifecycleHookPostStart: " " + chain.ID + " "},
	})
	if err != nil {
		t.Fatalf("SaveLifecyclePreset() error = %v", err)
	}
	if preset.ID == "" || preset.Name != "开始后预设" || preset.Chains[task.LifecycleHookPostStart] != chain.ID {
		t.Fatalf("保存的预设 = %#v", preset)
	}
	preset.Chains[task.LifecycleHookPostStart] = "changed"
	listed, err := repository.ListLifecyclePresets()
	if err != nil {
		t.Fatalf("ListLifecyclePresets() error = %v", err)
	}
	presetIndex := lifecyclePresetIndex(listed, preset.ID)
	if len(listed) != 3 || presetIndex < 0 || listed[presetIndex].Chains[task.LifecycleHookPostStart] != chain.ID {
		t.Fatalf("列出的预设 = %#v", listed)
	}
	copy, err := repository.CopyLifecyclePreset(preset.ID)
	if err != nil {
		t.Fatalf("CopyLifecyclePreset() error = %v", err)
	}
	if copy.ID == preset.ID || !reflect.DeepEqual(copy.Chains, map[task.LifecycleHook]string{task.LifecycleHookPostStart: chain.ID}) {
		t.Fatalf("复制的预设 = %#v", copy)
	}
	data, err = repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Tasks = []task.Task{{
		ID: "pending", Title: "保留预设快照", Color: task.DefaultColor, Status: task.StatusPending, ExtraInfo: []task.TaskExtraInfo{},
		LifecycleChains: map[task.LifecycleHook]string{task.LifecycleHookPostStart: chain.ID},
	}}
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() task snapshot error = %v", err)
	}
	preset.Name = "已修改预设"
	preset.Chains = map[task.LifecycleHook]string{}
	if _, err := repository.SaveLifecyclePreset(preset); err != nil {
		t.Fatalf("SaveLifecyclePreset() update error = %v", err)
	}
	saved, err := repository.SaveDefaultLifecyclePreset(preset.ID)
	if err != nil {
		t.Fatalf("SaveDefaultLifecyclePreset() error = %v", err)
	}
	if saved.DefaultLifecyclePresetID != preset.ID || saved.TaskTreeWidth != settings.DefaultTaskTreeWidth+40 {
		t.Fatalf("保存默认预设后的设置 = %#v", saved)
	}
	if saved, err = repository.SaveDefaultLifecyclePreset(""); err != nil || saved.DefaultLifecyclePresetID != "" {
		t.Fatalf("SaveDefaultLifecyclePreset(\"\") = (%#v, %v)，期望清除默认预设", saved, err)
	}
	if _, err := repository.SaveDefaultLifecyclePreset(preset.ID); err != nil {
		t.Fatalf("SaveDefaultLifecyclePreset() restore error = %v", err)
	}
	if err := repository.DeleteLifecyclePreset(preset.ID); err != nil {
		t.Fatalf("DeleteLifecyclePreset() error = %v", err)
	}
	loaded, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Settings.DefaultLifecyclePresetID != "" || len(loaded.Settings.LifecyclePresets) != 3 {
		t.Fatalf("删除默认预设后的设置 = %#v", loaded.Settings)
	}
	if got := loaded.Tasks[0].LifecycleChains; !reflect.DeepEqual(got, map[task.LifecycleHook]string{task.LifecycleHookPostStart: chain.ID}) {
		t.Fatalf("预设改动不应改写任务命令链: %#v", got)
	}
	if _, err := repository.SaveLifecyclePreset(settings.LifecyclePreset{Name: "无效", Chains: map[task.LifecycleHook]string{task.LifecycleHookBeforeStart: "missing"}}); err == nil {
		t.Fatal("SaveLifecyclePreset() error = nil，期望拒绝不存在的命令链")
	}
	if _, err := repository.SaveLifecyclePreset(settings.LifecyclePreset{Name: "不适用", Chains: map[task.LifecycleHook]string{task.LifecycleHookBeforeStart: chain.ID}}); err == nil {
		t.Fatal("SaveLifecyclePreset() error = nil，期望拒绝不适用的命令链")
	}
}

func TestRepositorySaveTaskSnapshotPreservesLifecyclePresetAddedAfterTaskLoad(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	repository := New(dataPath, settings.Default(t.TempDir()))
	stale, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	preset, err := repository.SaveLifecyclePreset(settings.LifecyclePreset{
		Name: "发布预设",
		Chains: map[task.LifecycleHook]string{
			task.LifecycleHookBeforeStart: settings.LifecycleChainCreateWorkspaceID,
		},
	})
	if err != nil {
		t.Fatalf("SaveLifecyclePreset() error = %v", err)
	}

	stale.Tasks = append(stale.Tasks, task.Task{
		ID: "task-1", Title: "并行任务", Color: task.DefaultColor, Status: task.StatusPending, ExtraInfo: []task.TaskExtraInfo{},
	})
	if err := repository.SaveTaskSnapshot(stale.Tasks); err != nil {
		t.Fatalf("SaveTaskSnapshot() error = %v", err)
	}

	restarted, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("restart Load() error = %v", err)
	}
	if lifecyclePresetIndex(restarted.Settings.LifecyclePresets, preset.ID) < 0 {
		t.Fatalf("重启后丢失已保存的预设: %#v", restarted.Settings.LifecyclePresets)
	}
}

func TestRepositoryPreservesChainArgumentsWhenCommandDisablesChainArguments(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	command, err := repository.SaveLifecycleCommand(settings.LifecycleCommand{
		Name: "部署", Command: "deploy", Arguments: []string{"--fixed"}, ChainArgumentMode: settings.LifecycleCommandChainArgumentModeEnabled,
		ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommand() error = %v", err)
	}
	chain, err := repository.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
		Name: "部署链", Commands: []settings.LifecycleCommandReference{{CommandID: command.ID, Arguments: []string{"--saved-extra"}}},
		ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeStart},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommandChain() error = %v", err)
	}

	command.ChainArgumentMode = settings.LifecycleCommandChainArgumentModeDisabled
	if _, err := repository.SaveLifecycleCommand(command); err != nil {
		t.Fatalf("SaveLifecycleCommand() disabling chain arguments error = %v", err)
	}
	chain.Name = "禁用后仍可保存"
	saved, err := repository.SaveLifecycleCommandChain(chain)
	if err != nil {
		t.Fatalf("SaveLifecycleCommandChain() after disabling chain arguments error = %v", err)
	}
	if got := saved.Commands; !reflect.DeepEqual(got, []settings.LifecycleCommandReference{{CommandID: command.ID, Arguments: []string{"--saved-extra"}}}) {
		t.Fatalf("禁用后保存的链级参数 = %#v", got)
	}
}

func TestRepositoryProtectsLifecyclePresetChainReferences(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	chain, err := repository.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
		Name: "完成后清理", CommandIDs: []string{settings.LifecycleCommandDeleteWorkspaceID}, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookPostEnd},
	})
	if err != nil {
		t.Fatalf("SaveLifecycleCommandChain() error = %v", err)
	}
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.Settings.LifecyclePresets[0].Chains[task.LifecycleHookPostEnd] = chain.ID
	data.Tasks = []task.Task{{
		ID: "completed", Title: "已完成", Color: task.DefaultColor, Status: task.StatusCompleted, ExtraInfo: []task.TaskExtraInfo{},
		LifecycleChains: map[task.LifecycleHook]string{task.LifecycleHookPostEnd: chain.ID},
	}}
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := repository.DeleteLifecycleCommandChain(chain.ID); err == nil {
		t.Fatal("DeleteLifecycleCommandChain() error = nil，期望拒绝被预设引用的链")
	}
	if _, err := repository.SaveLifecycleCommandChain(settings.LifecycleCommandChain{
		ID: chain.ID, Name: chain.Name, CommandIDs: chain.CommandIDs, ApplicableHooks: []settings.LifecycleHook{settings.LifecycleHookBeforeEnd},
	}); err == nil {
		t.Fatal("SaveLifecycleCommandChain() error = nil，期望拒绝移除预设引用的适用范围")
	}
	loaded, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := loaded.Settings.LifecyclePresets[0].Chains[task.LifecycleHookPostEnd]; got != chain.ID {
		t.Fatalf("拒绝操作后预设引用 = %q，期望 %q", got, chain.ID)
	}
}

func TestRepositoryPersistsAndDeletesExtraInfoTemplateWithoutTouchingTasks(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	template, err := task.NewExtraInfoTemplate("deployment", "API 部署", []task.ExtraInfoField{{Key: "repository", DisplayName: "仓库", DefaultValue: "git@example.com:team/api.git"}}, []task.ExtraInfoParameterDefinition{{Key: "branch", DisplayName: "分支", Required: true}})
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}
	snapshot := task.TaskExtraInfo{ID: "snapshot-1", Catalogue: "deployment", DisplayName: "API 仓库", Fields: []task.ExtraInfoField{{Key: "repository", DisplayName: "仓库", Value: "git@example.com:team/api.git"}}, Parameters: []task.ExtraInfoParameter{{Key: "branch", DisplayName: "分支", Required: true, InputType: task.ExtraInfoParameterInputText, Value: "main"}}}
	data := Data{Tasks: []task.Task{{ID: "task-1", Title: "测试", Color: task.DefaultColor, ExtraInfo: []task.TaskExtraInfo{snapshot}}}, Settings: settings.Default(t.TempDir())}
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := repository.SaveExtraInfoTemplate(template); err != nil {
		t.Fatalf("SaveExtraInfoTemplate() error = %v", err)
	}
	if err := repository.DeleteExtraInfoTemplate(template.ID); err != nil {
		t.Fatalf("DeleteExtraInfoTemplate() error = %v", err)
	}

	loaded, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.ExtraInfoTemplates) != 1 || loaded.ExtraInfoTemplates[0].Catalogue != "git" {
		t.Errorf("Load() ExtraInfoTemplates = %#v, want only built-in Git template", loaded.ExtraInfoTemplates)
	}
	if got := loaded.Tasks[0].ExtraInfo; !reflect.DeepEqual(got, []task.TaskExtraInfo{snapshot}) {
		t.Errorf("Load() task extra info = %#v, want %#v", got, []task.TaskExtraInfo{snapshot})
	}
}

func TestRepositoryAtomicallyPersistsTasksAndSettings(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	defaults := settings.Default(t.TempDir())
	repository := New(dataPath, defaults)
	createdAt := time.Date(2026, time.July, 22, 9, 0, 0, 0, time.UTC)

	want := Data{
		Tasks: []task.Task{{
			ID:          "task-1",
			Title:       "编写登录页",
			Description: "实现邮箱登录表单",
			Status:      task.StatusRunning,
			CreatedAt:   createdAt,
			ExtraInfo:   []task.TaskExtraInfo{},
		}},
		Settings: settings.Settings{
			WorkspaceRoot:    filepath.Join(t.TempDir(), "workspaces"),
			GitScanDepth:     settings.DefaultGitScanDepth,
			TaskTreeWidth:    420,
			TerminalFontSize: settings.DefaultTerminalFontSize,
			TerminalTheme:    settings.DefaultTerminalTheme(),
		},
	}

	if err := repository.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	expectedTask := want.Tasks[0]
	expectedTask.LifecycleChains = map[task.LifecycleHook]string{task.LifecycleHookPostEnd: settings.LifecycleChainDeleteWorkspaceID}
	expectedTask.TemplateFields = map[string]any{}
	if len(got.Tasks) != 1 || !reflect.DeepEqual(got.Tasks[0], expectedTask) {
		t.Errorf("Load() Tasks = %#v, want %#v", got.Tasks, expectedTask)
	}
	expectedSettings, err := settings.NormalizeTaskTemplates(want.Settings)
	if err != nil {
		t.Fatalf("NormalizeTaskTemplates() error = %v", err)
	}
	expectedSettings.TaskMenuItems, err = settings.NormalizeTaskMenuItems(expectedSettings.TaskMenuItems)
	if err != nil {
		t.Fatalf("NormalizeTaskMenuItems() error = %v", err)
	}
	expectedSettings, err = settings.NormalizeLifecycle(expectedSettings)
	if err != nil {
		t.Fatalf("NormalizeLifecycle() error = %v", err)
	}
	expectedSettings, _ = settings.ApplyPresetMigration(expectedSettings)
	if !reflect.DeepEqual(got.Settings, expectedSettings) {
		t.Errorf("Load() Settings = %#v, want %#v", got.Settings, expectedSettings)
	}

	entries, err := os.ReadDir(filepath.Dir(dataPath))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		t.Errorf("temporary persistence files = %#v, want only state.json", entries)
	}
}

func TestRepositoryPreservesInvalidDataFile(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	invalidJSON := []byte("{not valid json")
	if err := os.WriteFile(dataPath, invalidJSON, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	repository := New(dataPath, settings.Default(t.TempDir()))

	_, err := repository.Load()

	if err == nil {
		t.Fatal("Load() error = nil, want invalid JSON error")
	}
	contents, readErr := os.ReadFile(dataPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(contents) != string(invalidJSON) {
		t.Errorf("Load() modified invalid JSON file: got %q, want %q", contents, invalidJSON)
	}
}

func TestRepositoryReturnsWriteErrorForInvalidParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parent, []byte("file"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	repository := New(filepath.Join(parent, "state.json"), settings.Default(t.TempDir()))

	err := repository.Save(Data{})

	if err == nil {
		t.Fatal("Save() error = nil, want invalid parent error")
	}
}

func TestRepositorySaveSettingsKeepsTaskWorkspaceSnapshot(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	initialSettings := settings.Default(t.TempDir())
	repository := New(dataPath, initialSettings)
	originalTask := task.Task{
		ID:            "task-1",
		Title:         "编写登录页",
		Status:        task.StatusRunning,
		WorkspaceRoot: filepath.Join(t.TempDir(), "original-root"),
		WorkspacePath: filepath.Join(t.TempDir(), "original-root", "task-1"),
	}
	if err := repository.Save(Data{
		Tasks:    []task.Task{originalTask},
		Settings: initialSettings,
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	nextSettings := settings.Settings{
		WorkspaceRoot:    filepath.Join(t.TempDir(), "next-root"),
		GitScanDepth:     settings.DefaultGitScanDepth,
		TaskTreeWidth:    440,
		TerminalFontSize: settings.DefaultTerminalFontSize,
		TerminalTheme:    settings.DefaultTerminalTheme(),
	}

	if _, err := repository.SaveSettings(nextSettings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	got, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Tasks) != 1 || got.Tasks[0].WorkspacePath != originalTask.WorkspacePath || got.Tasks[0].WorkspaceRoot != originalTask.WorkspaceRoot {
		t.Errorf("SaveSettings() changed task workspace snapshot: %#v", got.Tasks)
	}
	expectedSettings, err := settings.NormalizeTaskTemplates(nextSettings)
	if err != nil {
		t.Fatalf("NormalizeTaskTemplates() error = %v", err)
	}
	expectedSettings.TaskMenuItems, err = settings.NormalizeTaskMenuItems(expectedSettings.TaskMenuItems)
	if err != nil {
		t.Fatalf("NormalizeTaskMenuItems() error = %v", err)
	}
	expectedSettings.LifecycleCommands = initialSettings.LifecycleCommands
	expectedSettings.LifecycleChains = initialSettings.LifecycleChains
	expectedSettings.LifecyclePresets = initialSettings.LifecyclePresets
	expectedSettings.DefaultLifecyclePresetID = initialSettings.DefaultLifecyclePresetID
	expectedSettings.TaskTemplates = initialSettings.TaskTemplates
	expectedSettings.ActiveTaskTemplateID = initialSettings.ActiveTaskTemplateID
	expectedSettings.PresetVersion = initialSettings.PresetVersion
	expectedSettings, err = settings.NormalizeLifecycle(expectedSettings)
	if err != nil {
		t.Fatalf("NormalizeLifecycle() error = %v", err)
	}
	if !reflect.DeepEqual(got.Settings, expectedSettings) {
		t.Errorf("SaveSettings() Settings = %#v, want %#v", got.Settings, expectedSettings)
	}
}
