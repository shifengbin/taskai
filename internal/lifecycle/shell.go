package lifecycle

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"taskai/internal/backgroundprocess"
)

type ShellCommandExecutor struct {
	platform string
}

func NewShellCommandExecutor() *ShellCommandExecutor {
	return &ShellCommandExecutor{platform: runtime.GOOS}
}

func (executor *ShellCommandExecutor) Run(invocation CommandInvocation) (CommandResult, error) {
	process := shellCommandProcess(executor.platform, invocation.ShellPath, invocation.Command, invocation.Arguments)
	backgroundprocess.Configure(process)
	process.Dir = invocation.Directory
	process.Stdin = bytes.NewReader(invocation.Input)
	if process.Env == nil {
		process.Env = os.Environ()
	}
	process.Env = append(process.Env, invocation.Environment...)
	var output bytes.Buffer
	var standardError bytes.Buffer
	process.Stdout = &output
	process.Stderr = &standardError
	err := process.Run()
	return CommandResult{Output: output.Bytes(), StandardError: standardError.Bytes()}, err
}

func shellCommandProcess(platform, shellPath, command string, arguments []string) *exec.Cmd {
	if shellPath == "" {
		return exec.Command(command, arguments...)
	}

	switch {
	case platform == "windows" && isPowerShellShell(shellPath):
		encodedArguments, _ := json.Marshal(append([]string(nil), arguments...))
		process := exec.Command(shellPath, "-NoLogo", "-Command", `$taskaiArguments = ConvertFrom-Json -InputObject $env:TASKAI_EXEC_ARGUMENTS; & $env:TASKAI_EXEC_COMMAND @($taskaiArguments)`)
		process.Env = append(os.Environ(), "TASKAI_EXEC_COMMAND="+command, "TASKAI_EXEC_ARGUMENTS="+string(encodedArguments))
		return process
	case platform == "windows" && shellExecutableName(shellPath) == "cmd":
		shellArguments := append([]string{"/C", command}, arguments...)
		return exec.Command(shellPath, shellArguments...)
	case platform != "windows" && shellExecutableName(shellPath) == "fish":
		shellArguments := append([]string{"-ic", "exec $argv[2..-1]", shellPath, command}, arguments...)
		return exec.Command(shellPath, shellArguments...)
	default:
		shellArguments := append([]string{"-ic", `exec "$@"`, shellPath, command}, arguments...)
		return exec.Command(shellPath, shellArguments...)
	}
}

func isPowerShellShell(shellPath string) bool {
	name := shellExecutableName(shellPath)
	return name == "powershell" || name == "pwsh"
}

func shellExecutableName(shellPath string) string {
	if index := strings.LastIndexAny(shellPath, `/\\`); index >= 0 {
		shellPath = shellPath[index+1:]
	}
	return strings.TrimSuffix(strings.ToLower(shellPath), ".exe")
}
