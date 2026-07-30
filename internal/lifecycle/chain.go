package lifecycle

import (
	"fmt"
	"strings"

	"taskai/internal/settings"
	"taskai/internal/task"
	"taskai/internal/workspace"
)

type CommandInvocation struct {
	Directory   string
	ShellPath   string
	Command     string
	Arguments   []string
	Input       []byte
	Environment []string
}

type CommandResult struct {
	Output        []byte
	StandardError []byte
}

type CommandExecutor interface {
	Run(CommandInvocation) (CommandResult, error)
}

type CommandExecutorFunc func(CommandInvocation) (CommandResult, error)

func (function CommandExecutorFunc) Run(invocation CommandInvocation) (CommandResult, error) {
	return function(invocation)
}

type CommandChainRequest struct {
	Task          task.Task
	Directory     string
	WorkspaceRoot string
	WorkspacePath string
	ShellPath     string
	Environment   []string
	Input         []byte
	Commands      []settings.LifecycleCommand
	OnProgress    func(index, count int, command settings.LifecycleCommand)
}

type CommandChainRunner struct {
	executor        CommandExecutor
	createWorkspace func(root, taskID string) (string, error)
	removeWorkspace func(root, path, taskID string) error
}

func NewCommandChainRunner(executor CommandExecutor) *CommandChainRunner {
	return &CommandChainRunner{
		executor:        executor,
		createWorkspace: workspace.Create,
		removeWorkspace: workspace.Remove,
	}
}

func (runner *CommandChainRunner) Run(request CommandChainRequest) ([]byte, error) {
	output := append([]byte(nil), request.Input...)
	for index, command := range request.Commands {
		if request.OnProgress != nil {
			request.OnProgress(index+1, len(request.Commands), command)
		}
		switch command.Kind {
		case settings.LifecycleCommandKindCustom:
			if runner.executor == nil {
				return nil, fmt.Errorf("生命周期命令执行器不可用")
			}
			result, err := runner.executor.Run(CommandInvocation{
				Directory:   request.Directory,
				ShellPath:   request.ShellPath,
				Command:     command.Command,
				Arguments:   append([]string(nil), command.Arguments...),
				Input:       output,
				Environment: append([]string(nil), request.Environment...),
			})
			if err != nil {
				return nil, commandError(command.Name, result.StandardError, err)
			}
			output = result.Output
		case settings.LifecycleCommandKindCreateWorkspace:
			if _, err := runner.createWorkspace(request.WorkspaceRoot, request.Task.ID); err != nil {
				return nil, commandError(command.Name, nil, err)
			}
		case settings.LifecycleCommandKindDeleteWorkspace:
			if err := runner.removeWorkspace(request.WorkspaceRoot, request.WorkspacePath, request.Task.ID); err != nil {
				return nil, commandError(command.Name, nil, err)
			}
		default:
			return nil, fmt.Errorf("不支持的生命周期命令类型: %q", command.Kind)
		}
	}
	return output, nil
}

func commandError(name string, standardError []byte, cause error) error {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "生命周期命令"
	}
	if message := strings.TrimSpace(string(standardError)); message != "" {
		return fmt.Errorf("执行%s失败: %v: %s", name, cause, message)
	}
	return fmt.Errorf("执行%s失败: %w", name, cause)
}
