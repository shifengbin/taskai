package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"taskai/internal/settings"
	"taskai/internal/task"
)

func TestCommandChainRunnerPassesStandardOutputToNextCommand(t *testing.T) {
	var calls []CommandInvocation
	runner := NewCommandChainRunner(CommandExecutorFunc(func(invocation CommandInvocation) (CommandResult, error) {
		calls = append(calls, invocation)
		if invocation.Command == "prepare" {
			return CommandResult{Output: []byte("prepared")}, nil
		}
		return CommandResult{Output: []byte("finished")}, nil
	}))

	output, err := runner.Run(CommandChainRequest{
		Task:        task.Task{ID: "task-1"},
		Directory:   "/workspace/task-1",
		ShellPath:   "/bin/sh",
		Environment: []string{"TASKAI_TASK_ID=task-1"},
		Input:       []byte(`{"id":"task-1"}`),
		Commands: []settings.LifecycleCommand{
			{ID: "prepare", Kind: settings.LifecycleCommandKindCustom, Name: "准备", Command: "prepare", Arguments: []string{"--fast"}},
			{ID: "finish", Kind: settings.LifecycleCommandKindCustom, Name: "完成", Command: "finish"},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(output) != "finished" {
		t.Fatalf("Run() output = %q，期望 finished", output)
	}
	if len(calls) != 2 || string(calls[0].Input) != `{"id":"task-1"}` || string(calls[1].Input) != "prepared" {
		t.Fatalf("命令标准输入 = %#v", calls)
	}
	if calls[0].ShellPath != "/bin/sh" || calls[0].Directory != "/workspace/task-1" || !reflect.DeepEqual(calls[0].Arguments, []string{"--fast"}) || !reflect.DeepEqual(calls[0].Environment, []string{"TASKAI_TASK_ID=task-1"}) {
		t.Fatalf("首命令调用 = %#v", calls[0])
	}
}

func TestCommandChainRunnerIncludesStandardErrorInFailure(t *testing.T) {
	runner := NewCommandChainRunner(CommandExecutorFunc(func(CommandInvocation) (CommandResult, error) {
		return CommandResult{StandardError: []byte("missing token")}, errors.New("exit status 1")
	}))

	_, err := runner.Run(CommandChainRequest{
		Task:      task.Task{ID: "task-1"},
		Directory: "/workspace/task-1",
		ShellPath: "/bin/sh",
		Commands:  []settings.LifecycleCommand{{ID: "failed", Kind: settings.LifecycleCommandKindCustom, Name: "失败命令", Command: "failed"}},
	})
	if err == nil || !strings.Contains(err.Error(), "失败命令") || !strings.Contains(err.Error(), "missing token") {
		t.Fatalf("Run() error = %v，期望包含命令名称和标准错误", err)
	}
}

func TestCommandChainRunnerRunsDirectoryCommandsAndPreservesInput(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	path := filepath.Join(root, "task-1")
	var received []byte
	runner := NewCommandChainRunner(CommandExecutorFunc(func(invocation CommandInvocation) (CommandResult, error) {
		if _, err := os.Stat(invocation.Directory); err != nil {
			t.Fatalf("目录创建命令后外部命令工作目录不可用: %v", err)
		}
		received = append([]byte(nil), invocation.Input...)
		return CommandResult{Output: []byte("custom-output")}, nil
	}))
	input := []byte(`{"id":"task-1"}`)
	output, err := runner.Run(CommandChainRequest{
		Task:          task.Task{ID: "task-1"},
		Directory:     path,
		WorkspaceRoot: root,
		WorkspacePath: path,
		Input:         input,
		Commands: []settings.LifecycleCommand{
			{ID: settings.LifecycleCommandCreateWorkspaceID, Kind: settings.LifecycleCommandKindCreateWorkspace, Name: "创建任务工作目录"},
			{ID: "custom", Kind: settings.LifecycleCommandKindCustom, Name: "自定义", Command: "custom"},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(received) != string(input) || string(output) != "custom-output" {
		t.Fatalf("目录命令未透传输入: received=%q output=%q", received, output)
	}

	output, err = runner.Run(CommandChainRequest{
		Task:          task.Task{ID: "task-1"},
		Directory:     root,
		WorkspaceRoot: root,
		WorkspacePath: path,
		Input:         []byte("delete-input"),
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandDeleteWorkspaceID, Kind: settings.LifecycleCommandKindDeleteWorkspace, Name: "删除任务工作目录",
		}},
	})
	if err != nil {
		t.Fatalf("删除目录 Run() error = %v", err)
	}
	if string(output) != "delete-input" {
		t.Fatalf("删除目录输出 = %q，期望原样透传", output)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("目录未删除: %v", err)
	}
}

func TestShellCommandExecutorRunsCommandWithInputAndWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("此测试使用 POSIX Shell")
	}
	directory := t.TempDir()
	executor := NewShellCommandExecutor()
	result, err := executor.Run(CommandInvocation{
		Directory: directory,
		ShellPath: "/bin/sh",
		Command:   "sh",
		Arguments: []string{"-c", "cat; pwd"},
		Input:     []byte("context\n"),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := string(result.Output), "context\n"+directory+"\n"; got != want {
		t.Fatalf("ShellCommandExecutor output = %q，期望 %q", got, want)
	}
}
