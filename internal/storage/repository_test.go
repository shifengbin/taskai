package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"taskai/internal/settings"
	"taskai/internal/task"
)

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

func TestRepositoryListsEmptyExtraInfoTemplatesAsJSONArray(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))

	templates, err := repository.ListExtraInfoTemplates()
	if err != nil {
		t.Fatalf("ListExtraInfoTemplates() error = %v", err)
	}
	encoded, err := json.Marshal(templates)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got, want := string(encoded), "[]"; got != want {
		t.Fatalf("空额外信息模板 JSON = %s，期望 %s", got, want)
	}
}

func TestRepositoryManagesExtraInfoCataloguesAndRequiresEmptyBeforeDelete(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	if _, err := repository.SaveExtraInfoCatalogue(" git "); err != nil {
		t.Fatalf("SaveExtraInfoCatalogue() error = %v", err)
	}
	if got, err := repository.ListExtraInfoCatalogues(); err != nil || !reflect.DeepEqual(got, []string{"git"}) {
		t.Fatalf("ListExtraInfoCatalogues() = %#v, %v，期望 [git], nil", got, err)
	}
	template, err := task.NewExtraInfoTemplate("git", "API 仓库", []task.ExtraInfoField{{Key: "repository", DisplayName: "仓库", Value: "git@example.com:team/api.git"}}, nil)
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}
	if _, err := repository.SaveExtraInfoTemplate(template); err != nil {
		t.Fatalf("SaveExtraInfoTemplate() error = %v", err)
	}
	if err := repository.DeleteExtraInfoCatalogue("git"); err == nil {
		t.Fatal("DeleteExtraInfoCatalogue() error = nil，期望分类仍有信息时失败")
	}
	if err := repository.DeleteExtraInfoTemplate(template.ID); err != nil {
		t.Fatalf("DeleteExtraInfoTemplate() error = %v", err)
	}
	if err := repository.DeleteExtraInfoCatalogue("git"); err != nil {
		t.Fatalf("DeleteExtraInfoCatalogue() error = %v", err)
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
	if data.ExtraInfoTemplates == nil || len(data.ExtraInfoTemplates) != 0 {
		t.Errorf("Load() ExtraInfoTemplates = %#v, want empty slice", data.ExtraInfoTemplates)
	}
}

func TestRepositoryPersistsAndDeletesExtraInfoTemplateWithoutTouchingTasks(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	if _, err := repository.SaveExtraInfoCatalogue("git"); err != nil {
		t.Fatalf("SaveExtraInfoCatalogue() error = %v", err)
	}
	template, err := task.NewExtraInfoTemplate("git", "API 仓库", []task.ExtraInfoField{{Key: "repository", DisplayName: "仓库", Value: "git@example.com:team/api.git"}}, []task.ExtraInfoParameterDefinition{{Key: "branch", DisplayName: "分支", Required: true}})
	if err != nil {
		t.Fatalf("NewExtraInfoTemplate() error = %v", err)
	}
	snapshot, err := task.NewExtraInfo(template, map[string]string{"branch": "main"})
	if err != nil {
		t.Fatalf("NewExtraInfo() error = %v", err)
	}
	data := Data{Tasks: []task.Task{{ID: "task-1", Title: "测试", Color: task.DefaultColor, ExtraInfo: []task.ExtraInfo{snapshot}}}, ExtraInfoCatalogues: []string{"git"}, Settings: settings.Default(t.TempDir())}
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
	if len(loaded.ExtraInfoTemplates) != 0 {
		t.Errorf("Load() ExtraInfoTemplates = %#v, want empty slice", loaded.ExtraInfoTemplates)
	}
	if got := loaded.Tasks[0].ExtraInfo; !reflect.DeepEqual(got, []task.ExtraInfo{snapshot}) {
		t.Errorf("Load() task extra info = %#v, want %#v", got, []task.ExtraInfo{snapshot})
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
			ExtraInfo:   []task.ExtraInfo{},
		}},
		Settings: settings.Settings{
			WorkspaceRoot: filepath.Join(t.TempDir(), "workspaces"),
			TaskTreeWidth: 420,
		},
	}

	if err := repository.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Tasks) != 1 || !reflect.DeepEqual(got.Tasks[0], want.Tasks[0]) {
		t.Errorf("Load() Tasks = %#v, want %#v", got.Tasks, want.Tasks)
	}
	if !reflect.DeepEqual(got.Settings, want.Settings) {
		t.Errorf("Load() Settings = %#v, want %#v", got.Settings, want.Settings)
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
	repository := New(dataPath, settings.Default(t.TempDir()))
	originalTask := task.Task{
		ID:            "task-1",
		Title:         "编写登录页",
		Status:        task.StatusRunning,
		WorkspaceRoot: filepath.Join(t.TempDir(), "original-root"),
		WorkspacePath: filepath.Join(t.TempDir(), "original-root", "task-1"),
	}
	if err := repository.Save(Data{
		Tasks:    []task.Task{originalTask},
		Settings: settings.Default(t.TempDir()),
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	nextSettings := settings.Settings{
		WorkspaceRoot: filepath.Join(t.TempDir(), "next-root"),
		TaskTreeWidth: 440,
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
	if !reflect.DeepEqual(got.Settings, nextSettings) {
		t.Errorf("SaveSettings() Settings = %#v, want %#v", got.Settings, nextSettings)
	}
}
