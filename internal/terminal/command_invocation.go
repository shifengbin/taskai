package terminal

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

const powerShellCommandInvocation = `$taskaiArguments = ConvertFrom-Json -InputObject $env:TASKAI_EXEC_ARGUMENTS; & $env:TASKAI_EXEC_COMMAND @($taskaiArguments)`

type CommandInvocation struct {
	Command     string
	Arguments   []string
	Environment map[string]string
}

func CommandInvocationForPlatform(platform, shellPath, command string, arguments []string) CommandInvocation {
	arguments = append([]string(nil), arguments...)
	if shellPath == "" {
		return CommandInvocation{Command: command, Arguments: arguments}
	}

	switch {
	case platform == "windows" && isPowerShell(shellPath):
		encodedArguments, _ := json.Marshal(arguments)
		return CommandInvocation{
			Command:   shellPath,
			Arguments: []string{"-NoLogo", "-Command", powerShellCommandInvocation},
			Environment: map[string]string{
				"TASKAI_EXEC_COMMAND":   command,
				"TASKAI_EXEC_ARGUMENTS": string(encodedArguments),
			},
		}
	case platform == "windows" && commandName(shellPath) == "cmd":
		return CommandInvocation{Command: shellPath, Arguments: append([]string{"/C", command}, arguments...)}
	case platform != "windows" && commandName(shellPath) == "fish":
		return CommandInvocation{Command: shellPath, Arguments: append([]string{"-ic", "exec $argv[2..-1]", shellPath, command}, arguments...)}
	default:
		return CommandInvocation{Command: shellPath, Arguments: append([]string{"-ic", `exec "$@"`, shellPath, command}, arguments...)}
	}
}

func (invocation CommandInvocation) EnvironmentEntries() []string {
	entries := make([]string, 0, len(invocation.Environment))
	for _, key := range []string{"TASKAI_EXEC_COMMAND", "TASKAI_EXEC_ARGUMENTS"} {
		if value, found := invocation.Environment[key]; found {
			entries = append(entries, key+"="+value)
		}
	}
	return entries
}

func isPowerShell(path string) bool {
	name := commandName(path)
	return name == "powershell" || name == "pwsh"
}

func commandName(path string) string {
	name := filepath.Base(strings.ReplaceAll(path, `\`, "/"))
	name = strings.TrimSuffix(strings.ToLower(name), ".exe")
	return strings.TrimSuffix(name, ".com")
}
