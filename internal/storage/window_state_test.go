package storage

import (
	"os"
	"path/filepath"
	"testing"

	"taskai/internal/settings"
)

func TestRepositoryLoadsMissingWindowAndShortcutSettings(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(dataPath, []byte(`{"tasks":[],"settings":{}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := New(dataPath, settings.Default(t.TempDir())).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Settings.TerminalShortcuts == nil {
		t.Fatal("Load() TerminalShortcuts = nil")
	}
	if loaded.Settings.WindowMaximized {
		t.Fatal("Load() WindowMaximized = true, want false")
	}
}

func TestRepositorySavesWindowMaximizedWithoutReplacingOtherSettings(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	current, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	current.Settings.ShellPath = "custom-shell"
	if err := repository.Save(current); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := repository.SaveWindowMaximized(true); err != nil {
		t.Fatalf("SaveWindowMaximized() error = %v", err)
	}
	loaded, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() after SaveWindowMaximized() error = %v", err)
	}
	if !loaded.Settings.WindowMaximized || loaded.Settings.ShellPath != "custom-shell" {
		t.Fatalf("settings after SaveWindowMaximized() = %#v", loaded.Settings)
	}
}
