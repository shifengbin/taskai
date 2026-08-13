package settings

import (
	"errors"
	"reflect"
	"testing"
)

func TestDetectAgentCommandsChecksCandidatesIndependently(t *testing.T) {
	tests := []struct {
		name     string
		resolved map[string]string
		failed   map[string]error
		want     DetectedAgentCommands
	}{
		{
			name:     "两个命令都存在",
			resolved: map[string]string{"codex": "/tools/codex", "claude": "/tools/claude"},
			want:     DetectedAgentCommands{Codex: true, Claude: true},
		},
		{
			name:     "仅 Codex 存在",
			resolved: map[string]string{"codex": "/tools/codex"},
			want:     DetectedAgentCommands{Codex: true},
		},
		{
			name:     "仅 Claude 存在",
			resolved: map[string]string{"claude": "/tools/claude"},
			want:     DetectedAgentCommands{Claude: true},
		},
		{
			name: "两个命令都不存在",
			want: DetectedAgentCommands{},
		},
		{
			name:     "单个检测错误不影响另一个命令",
			resolved: map[string]string{"claude": `C:\tools\claude.cmd`},
			failed:   map[string]error{"codex": errors.New("PATH 不可读")},
			want:     DetectedAgentCommands{Claude: true},
		},
	}

	for _, current := range tests {
		t.Run(current.name, func(t *testing.T) {
			calls := make([]string, 0, 2)
			lookup := func(command string) (string, error) {
				calls = append(calls, command)
				if err := current.failed[command]; err != nil {
					return "", err
				}
				if path := current.resolved[command]; path != "" {
					return path, nil
				}
				return "", errors.New("命令不存在")
			}

			if got := DetectAgentCommands(lookup); got != current.want {
				t.Fatalf("DetectAgentCommands() = %#v，期望 %#v", got, current.want)
			}
			if want := []string{"codex", "claude"}; !reflect.DeepEqual(calls, want) {
				t.Fatalf("命令检测顺序 = %#v，期望 %#v", calls, want)
			}
		})
	}
}

func TestMergeDetectedAgentTaskMenuItemsAppendsExactDefaults(t *testing.T) {
	existing := append(DefaultTaskMenuItems(), TaskMenuItem{
		ID: "custom.deploy", Kind: TaskMenuItemKindCommand, Name: "部署", Command: "deploy", ShowTerminal: false,
	})

	merged, changed := MergeDetectedAgentTaskMenuItems(existing, DetectedAgentCommands{Codex: true, Claude: true})

	if !changed {
		t.Fatal("MergeDetectedAgentTaskMenuItems() changed = false，期望 true")
	}
	if got := merged[:len(existing)]; !reflect.DeepEqual(got, existing) {
		t.Fatalf("已有菜单项被修改 = %#v，期望 %#v", got, existing)
	}
	want := []TaskMenuItem{
		{
			ID: TaskMenuItemDetectedCodexID, Kind: TaskMenuItemKindCommand, Name: "codex", Command: "codex",
			Arguments: []string{"--yolo"}, ShowTerminal: true,
		},
		{
			ID: TaskMenuItemDetectedClaudeID, Kind: TaskMenuItemKindCommand, Name: "claude", Command: "claude",
			Arguments: []string{"--dangerously-skip-permissions"}, ShowTerminal: true,
		},
	}
	if got := merged[len(existing):]; !reflect.DeepEqual(got, want) {
		t.Fatalf("追加的代理菜单项 = %#v，期望 %#v", got, want)
	}
}

func TestMergeDetectedAgentTaskMenuItemsIsIdempotentByStableID(t *testing.T) {
	modifiedCodex := TaskMenuItem{
		ID: TaskMenuItemDetectedCodexID, Kind: TaskMenuItemKindCommand, Name: "我的 Codex", Command: "codex-wrapper",
		Arguments: []string{"--custom"}, ShowTerminal: false,
	}
	existing := []TaskMenuItem{
		fixedTaskMenuItem(TaskMenuItemEditTaskID),
		modifiedCodex,
		fixedTaskMenuItem(TaskMenuItemCreateTerminalID),
	}

	merged, changed := MergeDetectedAgentTaskMenuItems(existing, DetectedAgentCommands{Codex: true, Claude: true})

	if !changed {
		t.Fatal("缺少 Claude 自动项时 changed = false，期望 true")
	}
	if !reflect.DeepEqual(merged[:len(existing)], existing) {
		t.Fatalf("已有 Codex 自动项或位置被改变 = %#v", merged)
	}
	if got := merged[len(merged)-1]; got.ID != TaskMenuItemDetectedClaudeID {
		t.Fatalf("最后追加项 ID = %q，期望 %q", got.ID, TaskMenuItemDetectedClaudeID)
	}

	second, changed := MergeDetectedAgentTaskMenuItems(merged, DetectedAgentCommands{Codex: true, Claude: true})
	if changed {
		t.Fatal("重复合并 changed = true，期望 false")
	}
	if !reflect.DeepEqual(second, merged) {
		t.Fatalf("重复合并结果 = %#v，期望 %#v", second, merged)
	}
}

func TestMergeDetectedAgentTaskMenuItemsPreservesSimilarCustomItemsAndUnavailableEntries(t *testing.T) {
	existing := []TaskMenuItem{
		{ID: "custom.codex", Kind: TaskMenuItemKindCommand, Name: "codex", Command: "codex-wrapper", Arguments: []string{"--yolo"}, ShowTerminal: true},
		{ID: TaskMenuItemDetectedClaudeID, Kind: TaskMenuItemKindCommand, Name: "保留的 claude", Command: "claude-old", ShowTerminal: true},
	}

	merged, changed := MergeDetectedAgentTaskMenuItems(existing, DetectedAgentCommands{Codex: true})

	if !changed {
		t.Fatal("Codex 自动项缺失时 changed = false，期望 true")
	}
	if !reflect.DeepEqual(merged[:len(existing)], existing) {
		t.Fatalf("相似自定义项或不可用的已有 Claude 项被改变 = %#v", merged)
	}
	if got := merged[len(merged)-1]; got.ID != TaskMenuItemDetectedCodexID {
		t.Fatalf("追加项 ID = %q，期望 %q", got.ID, TaskMenuItemDetectedCodexID)
	}

	unchanged, changed := MergeDetectedAgentTaskMenuItems(existing, DetectedAgentCommands{})
	if changed || !reflect.DeepEqual(unchanged, existing) {
		t.Fatalf("无可用命令时结果 = (%#v, %t)，期望保留已有设置", unchanged, changed)
	}
}

func TestMergeDetectedAgentTaskMenuItemsSkipsExactExistingCommands(t *testing.T) {
	existing := []TaskMenuItem{
		{ID: "custom.codex", Kind: TaskMenuItemKindCommand, Name: "我的代理", Command: "codex", Arguments: []string{"--custom"}},
		{ID: "custom.claude", Kind: TaskMenuItemKindCommand, Name: "另一个代理", Command: "claude"},
	}

	merged, changed := MergeDetectedAgentTaskMenuItems(existing, DetectedAgentCommands{Codex: true, Claude: true}, nil)

	if changed || !reflect.DeepEqual(merged, existing) {
		t.Fatalf("已有精确命令时合并结果 = (%#v, %t)，期望保持不变", merged, changed)
	}
}

func TestMergeDetectedAgentTaskMenuItemsDoesNotTreatNameOrNonCommandKindAsConfigured(t *testing.T) {
	existing := []TaskMenuItem{
		{ID: "custom.named-codex", Kind: TaskMenuItemKindCommand, Name: "codex", Command: "codex-wrapper"},
		{ID: "custom.kind-claude", Kind: TaskMenuItemKindOpenFolder, Name: "claude", Command: "claude"},
	}

	merged, changed := MergeDetectedAgentTaskMenuItems(existing, DetectedAgentCommands{Codex: true, Claude: true}, nil)

	if !changed {
		t.Fatal("名称相同或非命令类型时 changed = false，期望追加自动项")
	}
	if got := []string{merged[len(merged)-2].ID, merged[len(merged)-1].ID}; !reflect.DeepEqual(got, []string{TaskMenuItemDetectedCodexID, TaskMenuItemDetectedClaudeID}) {
		t.Fatalf("追加自动项 ID = %#v", got)
	}
}

func TestMergeDetectedAgentTaskMenuItemsSkipsDismissedAutomaticItems(t *testing.T) {
	merged, changed := MergeDetectedAgentTaskMenuItems(nil, DetectedAgentCommands{Codex: true, Claude: true}, []string{TaskMenuItemDetectedCodexID})

	if !changed || len(merged) != 1 || merged[0].ID != TaskMenuItemDetectedClaudeID {
		t.Fatalf("删除记录抑制后的合并结果 = (%#v, %t)，期望只追加 Claude", merged, changed)
	}
}
