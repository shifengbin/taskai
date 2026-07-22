package settings

import (
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
