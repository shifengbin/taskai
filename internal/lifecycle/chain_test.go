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

func TestCommandChainRunnerClonesSpecifiedRepositoryDirectlyToTarget(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	var calls []GitInvocation
	runner := NewCommandChainRunner(CommandExecutorFunc(func(CommandInvocation) (CommandResult, error) {
		t.Fatal("指定仓库克隆不应调用 Shell 执行器")
		return CommandResult{}, nil
	}))
	runner.gitExecutor = GitExecutorFunc(func(invocation GitInvocation) (CommandResult, error) {
		calls = append(calls, invocation)
		return CommandResult{}, nil
	})

	input := []byte(`{"taskId":"task-1"}`)
	output, err := runner.Run(CommandChainRequest{
		Task:                     task.Task{ID: "task-1", ExtraInfo: []task.TaskExtraInfo{builtInGitTaskInfo("ignored", "https://example.com/ignored.git", "ignored")}},
		WorkspacePath:            workspacePath,
		GitCloneRepositoryBranch: "release/1.2",
		Input:                    input,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandGitCloneRepositoryID, Kind: settings.LifecycleCommandKindGitCloneRepository, Name: "克隆指定 Git 仓库", Arguments: []string{"repository=https://example.com/template.git", "dir=template"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(output) != string(input) {
		t.Fatalf("指定仓库克隆未原样传递标准输入: %q", output)
	}
	want := []GitInvocation{
		{Directory: workspacePath, Arguments: []string{"ls-remote", "--exit-code", "--heads", "--", "https://example.com/template.git", "refs/heads/release/1.2"}},
		{Directory: workspacePath, Arguments: []string{"clone", "--branch", "release/1.2", "--", "https://example.com/template.git", filepath.Join(workspacePath, "template")}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("Git 调用 = %#v，期望 %#v", calls, want)
	}
	if info, err := os.Stat(filepath.Join(workspacePath, "template")); err != nil || !info.IsDir() {
		t.Fatalf("安全目标目录未创建: info=%#v, err=%v", info, err)
	}
}

func TestCommandChainRunnerClonesSpecifiedRepositoryWithRemoteDefaultBranch(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	var call GitInvocation
	runner := NewCommandChainRunner(nil)
	runner.gitExecutor = GitExecutorFunc(func(invocation GitInvocation) (CommandResult, error) {
		call = invocation
		if invocation.Arguments[0] == "clone" {
			if err := os.Mkdir(filepath.Join(workspacePath, ".git"), 0o700); err != nil {
				t.Fatalf("模拟 clone 创建 .git: %v", err)
			}
		}
		return CommandResult{}, nil
	})

	_, err := runner.Run(CommandChainRequest{
		WorkspacePath: workspacePath,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandGitCloneRepositoryID, Kind: settings.LifecycleCommandKindGitCloneRepository, Name: "克隆指定 Git 仓库", Arguments: []string{"repository=https://example.com/template.git"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := []string{"clone", "--", "https://example.com/template.git", workspacePath}
	if !reflect.DeepEqual(call.Arguments, want) {
		t.Fatalf("远程默认分支 Git 调用 = %#v，期望 %#v", call.Arguments, want)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, ".git")); err != nil {
		t.Fatalf("成功克隆后未保留 .git: %v", err)
	}
}

func TestCommandChainRunnerRejectsUnsafeSpecifiedRepositoryTargetBeforeGit(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(workspacePath string) error
	}{
		{name: "工作目录不存在", setup: func(string) error { return nil }},
		{name: "非空目标", setup: func(workspacePath string) error {
			target := filepath.Join(workspacePath, "template")
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(target, "existing.txt"), []byte("keep"), 0o600)
		}},
		{name: "目标是普通文件", setup: func(workspacePath string) error {
			return os.WriteFile(filepath.Join(workspacePath, "template"), []byte("keep"), 0o600)
		}},
		{name: "符号链接目标", setup: func(workspacePath string) error {
			outside := filepath.Join(t.TempDir(), "outside")
			if err := os.MkdirAll(outside, 0o700); err != nil {
				return err
			}
			return os.Symlink(outside, filepath.Join(workspacePath, "template"))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspacePath := filepath.Join(t.TempDir(), "task-1")
			if test.name != "工作目录不存在" {
				if err := os.MkdirAll(workspacePath, 0o700); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			}
			if err := test.setup(workspacePath); err != nil {
				t.Fatalf("setup() error = %v", err)
			}
			runner := NewCommandChainRunner(nil)
			runner.gitExecutor = GitExecutorFunc(func(GitInvocation) (CommandResult, error) {
				t.Fatal("目标不安全时不应调用 Git")
				return CommandResult{}, nil
			})

			_, err := runner.Run(CommandChainRequest{
				WorkspacePath: workspacePath,
				Commands: []settings.LifecycleCommand{{
					ID: settings.LifecycleCommandGitCloneRepositoryID, Kind: settings.LifecycleCommandKindGitCloneRepository, Name: "克隆指定 Git 仓库", Arguments: []string{"repository=https://example.com/template.git", "dir=template"},
				}},
			})
			if err == nil {
				t.Fatal("Run() error = nil，期望拒绝不安全克隆目标")
			}
		})
	}
}

func TestCommandChainRunnerComposesSpecifiedRepositoryAndGitInfoCloneInOrder(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	var calls []GitInvocation
	runner := NewCommandChainRunner(nil)
	runner.gitExecutor = GitExecutorFunc(func(invocation GitInvocation) (CommandResult, error) {
		calls = append(calls, invocation)
		if invocation.Arguments[0] == "clone" {
			if err := os.MkdirAll(invocation.Arguments[len(invocation.Arguments)-1], 0o700); err != nil {
				t.Fatalf("模拟 clone 创建目录: %v", err)
			}
		}
		return CommandResult{}, nil
	})

	_, err := runner.Run(CommandChainRequest{
		Task:          task.Task{ID: "task-1", ExtraInfo: []task.TaskExtraInfo{builtInGitTaskInfo("api", "https://example.com/api.git", "")}},
		WorkspacePath: workspacePath,
		Commands: []settings.LifecycleCommand{
			{ID: settings.LifecycleCommandGitCloneRepositoryID, Kind: settings.LifecycleCommandKindGitCloneRepository, Name: "克隆指定 Git 仓库", Arguments: []string{"repository=https://example.com/template.git", "dir=template"}},
			{ID: settings.LifecycleCommandGitCloneID, Kind: settings.LifecycleCommandKindGitClone, Name: "Git 仓库克隆", Arguments: []string{"dir=repositories"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := []GitInvocation{
		{Directory: workspacePath, Arguments: []string{"clone", "--", "https://example.com/template.git", filepath.Join(workspacePath, "template")}},
		{Directory: filepath.Join(workspacePath, "repositories"), Arguments: []string{"clone", "--", "https://example.com/api.git", filepath.Join(workspacePath, "repositories", "api")}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("组合 Git 调用 = %#v，期望 %#v", calls, want)
	}
}

func TestCommandChainRunnerSkipsGitCloneWhenTaskHasNoBuiltInGitInfo(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	input := []byte(`{"id":"task-1"}`)
	runner := NewCommandChainRunner(nil)

	output, err := runner.Run(CommandChainRequest{
		Task:          task.Task{ID: "task-1", ExtraInfo: []task.TaskExtraInfo{}},
		Directory:     workspacePath,
		WorkspacePath: workspacePath,
		Input:         input,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandGitCloneID, Kind: settings.LifecycleCommandKindGitClone, Name: "Git 仓库克隆", Arguments: []string{"dir=repositories"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(output) != string(input) {
		t.Fatalf("Git 无仓库时输出 = %q，期望原样透传 %q", output, input)
	}
}

func TestCommandChainRunnerClonesGitRepositoriesToTaskWorkspaceWhenDirectoryOmitted(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	var cloneTarget string
	runner := NewCommandChainRunner(nil)
	runner.gitExecutor = GitExecutorFunc(func(invocation GitInvocation) (CommandResult, error) {
		if invocation.Arguments[0] != "clone" {
			return CommandResult{}, nil
		}
		cloneTarget = invocation.Arguments[len(invocation.Arguments)-1]
		return CommandResult{}, nil
	})

	_, err := runner.Run(CommandChainRequest{
		Task:          task.Task{ID: "task-1", ExtraInfo: []task.TaskExtraInfo{builtInGitTaskInfo("api", "https://example.com/api.git", "")}},
		Directory:     workspacePath,
		WorkspacePath: workspacePath,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandGitCloneID, Kind: settings.LifecycleCommandKindGitClone, Name: "Git 仓库克隆",
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if want := filepath.Join(workspacePath, "api"); cloneTarget != want {
		t.Fatalf("Git 克隆目标 = %q，期望 %q", cloneTarget, want)
	}
}

func TestCommandChainRunnerClonesGitRepositoriesForRemoteAndLocalBranches(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	var calls []GitInvocation
	runner := NewCommandChainRunner(nil)
	runner.gitExecutor = GitExecutorFunc(func(invocation GitInvocation) (CommandResult, error) {
		calls = append(calls, invocation)
		switch invocation.Arguments[0] {
		case "ls-remote":
			if invocation.Arguments[len(invocation.Arguments)-1] == "refs/heads/feature/local" {
				return CommandResult{}, &GitCommandExitError{Code: 2, Cause: errors.New("exit status 2")}
			}
			return CommandResult{}, nil
		case "clone":
			target := invocation.Arguments[len(invocation.Arguments)-1]
			if err := os.MkdirAll(target, 0o700); err != nil {
				t.Fatalf("模拟 clone 创建目录: %v", err)
			}
		}
		return CommandResult{}, nil
	})
	input := []byte(`{"id":"task-1"}`)

	output, err := runner.Run(CommandChainRequest{
		Task: task.Task{ID: "task-1", ExtraInfo: []task.TaskExtraInfo{
			builtInGitTaskInfo("api", "https://example.com/api.git", "release"),
			builtInGitTaskInfo("web", "https://example.com/web.git", "feature/local"),
		}},
		Directory:     workspacePath,
		WorkspacePath: workspacePath,
		Input:         input,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandGitCloneID, Kind: settings.LifecycleCommandKindGitClone, Name: "Git 仓库克隆", Arguments: []string{"dir=repositories"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(output) != string(input) {
		t.Fatalf("Git 命令输出 = %q，期望原样透传", output)
	}
	want := []GitInvocation{
		{Directory: filepath.Join(workspacePath, "repositories"), Arguments: []string{"ls-remote", "--exit-code", "--heads", "--", "https://example.com/api.git", "refs/heads/release"}},
		{Directory: filepath.Join(workspacePath, "repositories"), Arguments: []string{"clone", "--branch", "release", "--", "https://example.com/api.git", filepath.Join(workspacePath, "repositories", "api")}},
		{Directory: filepath.Join(workspacePath, "repositories"), Arguments: []string{"ls-remote", "--exit-code", "--heads", "--", "https://example.com/web.git", "refs/heads/feature/local"}},
		{Directory: filepath.Join(workspacePath, "repositories"), Arguments: []string{"clone", "--", "https://example.com/web.git", filepath.Join(workspacePath, "repositories", "web")}},
		{Directory: filepath.Join(workspacePath, "repositories", "web"), Arguments: []string{"switch", "--create", "feature/local"}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("Git 调用 = %#v，期望 %#v", calls, want)
	}
}

func TestCommandChainRunnerSkipsExistingGitProjectDirectory(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	target := filepath.Join(workspacePath, "repositories", "api")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	runner := NewCommandChainRunner(nil)
	runner.gitExecutor = GitExecutorFunc(func(GitInvocation) (CommandResult, error) {
		t.Fatal("目标目录已存在时不应调用 Git")
		return CommandResult{}, nil
	})

	_, err := runner.Run(CommandChainRequest{
		Task:          task.Task{ID: "task-1", ExtraInfo: []task.TaskExtraInfo{builtInGitTaskInfo("api", "https://example.com/api.git", "main")}},
		Directory:     workspacePath,
		WorkspacePath: workspacePath,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandGitCloneID, Kind: settings.LifecycleCommandKindGitClone, Name: "Git 仓库克隆", Arguments: []string{"dir=repositories"},
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestCommandChainRunnerRetriesGitCloneAfterSkippingSuccessfulProjects(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	failWeb := true
	var cloneTargets []string
	runner := NewCommandChainRunner(nil)
	runner.gitExecutor = GitExecutorFunc(func(invocation GitInvocation) (CommandResult, error) {
		if invocation.Arguments[0] != "clone" {
			return CommandResult{}, nil
		}
		target := invocation.Arguments[len(invocation.Arguments)-1]
		cloneTargets = append(cloneTargets, target)
		if strings.HasSuffix(target, string(filepath.Separator)+"web") && failWeb {
			return CommandResult{StandardError: []byte("网络不可用")}, errors.New("exit status 1")
		}
		if err := os.MkdirAll(target, 0o700); err != nil {
			t.Fatalf("模拟 clone 创建目录: %v", err)
		}
		return CommandResult{}, nil
	})
	request := CommandChainRequest{
		Task: task.Task{ID: "task-1", ExtraInfo: []task.TaskExtraInfo{
			builtInGitTaskInfo("api", "https://example.com/api.git", ""),
			builtInGitTaskInfo("web", "https://example.com/web.git", ""),
			builtInGitTaskInfo("worker", "https://example.com/worker.git", ""),
		}},
		Directory:     workspacePath,
		WorkspacePath: workspacePath,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandGitCloneID, Kind: settings.LifecycleCommandKindGitClone, Name: "Git 仓库克隆", Arguments: []string{"dir=repositories"},
		}},
	}
	if _, err := runner.Run(request); err == nil || !strings.Contains(err.Error(), "网络不可用") {
		t.Fatalf("首次 Run() error = %v，期望包含 Git 失败信息", err)
	}
	if got, want := cloneTargets, []string{filepath.Join(workspacePath, "repositories", "api"), filepath.Join(workspacePath, "repositories", "web")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("首次克隆目标 = %#v，期望 %#v", got, want)
	}

	failWeb = false
	if _, err := runner.Run(request); err != nil {
		t.Fatalf("重试 Run() error = %v", err)
	}
	if got, want := cloneTargets, []string{
		filepath.Join(workspacePath, "repositories", "api"),
		filepath.Join(workspacePath, "repositories", "web"),
		filepath.Join(workspacePath, "repositories", "web"),
		filepath.Join(workspacePath, "repositories", "worker"),
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("重试克隆目标 = %#v，期望 %#v", got, want)
	}
}

func builtInGitTaskInfo(name, repository, branch string) task.TaskExtraInfo {
	return task.TaskExtraInfo{
		TemplateID: task.BuiltInGitTemplate().ID,
		Fields: []task.ExtraInfoField{
			{Key: "name", Value: name},
			{Key: "repository", Value: repository},
		},
		Parameters: []task.ExtraInfoParameter{{Key: "branch", Value: branch}},
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
