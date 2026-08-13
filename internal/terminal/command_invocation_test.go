package terminal

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCommandInvocationForWindowsShells(t *testing.T) {
	tests := []struct {
		name        string
		shellPath   string
		command     string
		arguments   []string
		wantCommand string
		wantArgs    []string
		wantEnv     map[string]string
	}{
		{
			name: "cmd 包装 cmd 命令", shellPath: `C:\Windows\System32\cmd.exe`, command: `C:\tools\codex.cmd`,
			arguments: []string{"--yolo"}, wantCommand: `C:\Windows\System32\cmd.exe`,
			wantArgs: []string{"/C", `C:\tools\codex.cmd`, "--yolo"},
		},
		{
			name: "PowerShell 包装 cmd 命令", shellPath: `C:\Program Files\PowerShell\7\pwsh.exe`, command: `C:\tools\claude.cmd`,
			arguments: []string{"--dangerously-skip-permissions"}, wantCommand: `C:\Program Files\PowerShell\7\pwsh.exe`,
			wantArgs: []string{"-NoLogo", "-Command", `$taskaiArguments = ConvertFrom-Json -InputObject $env:TASKAI_EXEC_ARGUMENTS; & $env:TASKAI_EXEC_COMMAND @($taskaiArguments)`},
			wantEnv: map[string]string{
				"TASKAI_EXEC_COMMAND":   `C:\tools\claude.cmd`,
				"TASKAI_EXEC_ARGUMENTS": mustJSONArguments(t, []string{"--dangerously-skip-permissions"}),
			},
		},
		{
			name: "没有配置 Shell 时直接启动 exe", command: `C:\tools\agent.exe`, arguments: []string{"--safe"},
			wantCommand: `C:\tools\agent.exe`, wantArgs: []string{"--safe"},
		},
	}

	for _, current := range tests {
		t.Run(current.name, func(t *testing.T) {
			got := CommandInvocationForPlatform("windows", current.shellPath, current.command, current.arguments)
			if got.Command != current.wantCommand {
				t.Fatalf("Command = %q，期望 %q", got.Command, current.wantCommand)
			}
			if !reflect.DeepEqual(got.Arguments, current.wantArgs) {
				t.Fatalf("Arguments = %#v，期望 %#v", got.Arguments, current.wantArgs)
			}
			if !reflect.DeepEqual(got.Environment, current.wantEnv) {
				t.Fatalf("Environment = %#v，期望 %#v", got.Environment, current.wantEnv)
			}
		})
	}
}

func mustJSONArguments(t *testing.T, arguments []string) string {
	t.Helper()
	encoded, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return string(encoded)
}
