package settings

import "os/exec"

const (
	TaskMenuItemDetectedCodexID  = "auto.agent.codex"
	TaskMenuItemDetectedClaudeID = "auto.agent.claude"
)

type CommandLookup func(string) (string, error)

type DetectedAgentCommands struct {
	Codex  bool
	Claude bool
}

func DetectInstalledAgentCommands() DetectedAgentCommands {
	return DetectAgentCommands(exec.LookPath)
}

func DetectAgentCommands(lookup CommandLookup) DetectedAgentCommands {
	_, codexError := lookup("codex")
	_, claudeError := lookup("claude")
	return DetectedAgentCommands{
		Codex:  codexError == nil,
		Claude: claudeError == nil,
	}
}

func MergeDetectedAgentTaskMenuItems(items []TaskMenuItem, detected DetectedAgentCommands, dismissedLists ...[]string) ([]TaskMenuItem, bool) {
	merged := append([]TaskMenuItem(nil), items...)
	seen := make(map[string]bool, len(items))
	configuredCommands := make(map[string]bool, len(items))
	for _, item := range items {
		seen[item.ID] = true
		if item.Kind == TaskMenuItemKindCommand {
			configuredCommands[item.Command] = true
		}
	}
	dismissed := make(map[string]bool, 2)
	for _, dismissedList := range dismissedLists {
		for _, id := range dismissedList {
			dismissed[id] = true
		}
	}

	changed := false
	for _, item := range detectedAgentTaskMenuItems(detected) {
		if seen[item.ID] || dismissed[item.ID] || configuredCommands[item.Command] {
			continue
		}
		merged = append(merged, item)
		changed = true
	}
	return merged, changed
}

func detectedAgentTaskMenuItems(detected DetectedAgentCommands) []TaskMenuItem {
	items := make([]TaskMenuItem, 0, 2)
	if detected.Codex {
		items = append(items, TaskMenuItem{
			ID:           TaskMenuItemDetectedCodexID,
			Kind:         TaskMenuItemKindCommand,
			Name:         "codex",
			Command:      "codex",
			Arguments:    []string{"--yolo"},
			ShowTerminal: true,
		})
	}
	if detected.Claude {
		items = append(items, TaskMenuItem{
			ID:           TaskMenuItemDetectedClaudeID,
			Kind:         TaskMenuItemKindCommand,
			Name:         "claude",
			Command:      "claude",
			Arguments:    []string{"--dangerously-skip-permissions"},
			ShowTerminal: true,
		})
	}
	return items
}
