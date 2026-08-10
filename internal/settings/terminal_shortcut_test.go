package settings

import (
	"reflect"
	"testing"
)

func TestNormalizeTerminalShortcutsCanonicalizesAndGeneratesIDs(t *testing.T) {
	got, err := NormalizeTerminalShortcuts([]TerminalShortcut{{
		Shortcut: " enter + shift ",
		Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepText, Text: "\\"}, {
			Kind:      TerminalShortcutStepKey,
			Key:       "enter",
			Modifiers: []string{"shift", "ctrl"},
		}},
	}})
	if err != nil {
		t.Fatalf("NormalizeTerminalShortcuts() error = %v", err)
	}
	if len(got) != 1 || got[0].ID == "" || got[0].Shortcut != "Shift+Enter" {
		t.Fatalf("NormalizeTerminalShortcuts() = %#v", got)
	}
	wantSteps := []TerminalShortcutStep{{Kind: TerminalShortcutStepText, Text: "\\"}, {
		Kind:      TerminalShortcutStepKey,
		Key:       "Enter",
		Modifiers: []string{"Ctrl", "Shift"},
	}}
	if !reflect.DeepEqual(got[0].Steps, wantSteps) {
		t.Fatalf("NormalizeTerminalShortcuts() steps = %#v, want %#v", got[0].Steps, wantSteps)
	}
}

func TestNormalizeTerminalShortcutsMigratesLegacyEnterStep(t *testing.T) {
	got, err := NormalizeTerminalShortcuts([]TerminalShortcut{{
		Shortcut: "Shift+Enter",
		Steps:    []TerminalShortcutStep{{Kind: TerminalShortcutStepEnter}},
	}})
	if err != nil {
		t.Fatalf("NormalizeTerminalShortcuts() error = %v", err)
	}
	want := []TerminalShortcutStep{{Kind: TerminalShortcutStepKey, Key: "Enter"}}
	if !reflect.DeepEqual(got[0].Steps, want) {
		t.Fatalf("NormalizeTerminalShortcuts() steps = %#v, want %#v", got[0].Steps, want)
	}
}

func TestNormalizeTerminalShortcutKeyAcceptsCommonTerminalKeys(t *testing.T) {
	key, modifiers, err := NormalizeTerminalShortcutKey("page down", []string{"alt", "shift"})
	if err != nil {
		t.Fatalf("NormalizeTerminalShortcutKey() error = %v", err)
	}
	if key != "PageDown" || !reflect.DeepEqual(modifiers, []string{"Alt", "Shift"}) {
		t.Fatalf("NormalizeTerminalShortcutKey() = %q, %#v", key, modifiers)
	}
}

func TestNormalizeTerminalShortcutsRejectsInvalidAndConflictingBindings(t *testing.T) {
	tests := []struct {
		name      string
		shortcuts []TerminalShortcut
	}{
		{name: "empty shortcut", shortcuts: []TerminalShortcut{{Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepEnter}}}}},
		{name: "empty action", shortcuts: []TerminalShortcut{{Shortcut: "Shift+Enter"}}},
		{name: "empty text", shortcuts: []TerminalShortcut{{Shortcut: "Shift+Enter", Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepText}}}}},
		{name: "unknown step", shortcuts: []TerminalShortcut{{Shortcut: "Shift+Enter", Steps: []TerminalShortcutStep{{Kind: "unknown"}}}}},
		{name: "unknown key", shortcuts: []TerminalShortcut{{Shortcut: "Shift+Enter", Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepKey, Key: "unknown"}}}}},
		{name: "unsupported function key", shortcuts: []TerminalShortcut{{Shortcut: "Shift+Enter", Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepKey, Key: "F13"}}}}},
		{name: "duplicate modifier", shortcuts: []TerminalShortcut{{Shortcut: "Shift+Enter", Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepKey, Key: "Enter", Modifiers: []string{"Shift", "shift"}}}}}},
		{name: "duplicate shortcut", shortcuts: []TerminalShortcut{
			{Shortcut: "Shift+Enter", Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepEnter}}},
			{Shortcut: "shift + enter", Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepEnter}}},
		}},
		{name: "reserved quick input shortcut", shortcuts: []TerminalShortcut{{Shortcut: "Ctrl+Shift+P", Steps: []TerminalShortcutStep{{Kind: TerminalShortcutStepEnter}}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NormalizeTerminalShortcuts(test.shortcuts); err == nil {
				t.Fatal("NormalizeTerminalShortcuts() error = nil")
			}
		})
	}
}

func TestDefaultLeavesTerminalShortcutsAndWindowMaximizedDisabled(t *testing.T) {
	current := Default(t.TempDir())
	if current.TerminalShortcuts == nil {
		t.Fatal("Default() TerminalShortcuts = nil")
	}
	if len(current.TerminalShortcuts) != 0 {
		t.Fatalf("Default() TerminalShortcuts = %#v, want empty", current.TerminalShortcuts)
	}
	if current.WindowMaximized {
		t.Fatal("Default() WindowMaximized = true, want false")
	}
}
