package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
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
