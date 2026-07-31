package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
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
	Task                     task.Task
	Directory                string
	WorkspaceRoot            string
	WorkspacePath            string
	ShellPath                string
	Environment              []string
	Input                    []byte
	Commands                 []settings.LifecycleCommand
	GitCloneRepositoryBranch string
	OnProgress               func(index, count int, command settings.LifecycleCommand)
}

type CommandChainRunner struct {
	executor        CommandExecutor
	gitExecutor     GitExecutor
	createWorkspace func(root, taskID string) (string, error)
	removeWorkspace func(root, path, taskID string) error
}

func NewCommandChainRunner(executor CommandExecutor) *CommandChainRunner {
	return &CommandChainRunner{
		executor:        executor,
		gitExecutor:     NewDirectGitExecutor(),
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
		case settings.LifecycleCommandKindGitClone:
			if err := runner.cloneGitRepositories(request, command.Arguments); err != nil {
				return nil, commandError(command.Name, nil, err)
			}
		case settings.LifecycleCommandKindGitCloneRepository:
			if err := runner.cloneGitRepositoryToTarget(request, command.Arguments); err != nil {
				return nil, commandError(command.Name, nil, err)
			}
		case settings.LifecycleCommandKindManifestFile:
			if err := runner.writeManifestFile(request, command.Arguments); err != nil {
				return nil, commandError(command.Name, nil, err)
			}
		default:
			return nil, fmt.Errorf("不支持的生命周期命令类型: %q", command.Kind)
		}
	}
	return output, nil
}

func (runner *CommandChainRunner) cloneGitRepositoryToTarget(request CommandChainRequest, arguments []string) error {
	parameters, err := settings.GitCloneRepositoryArguments(arguments)
	if err != nil {
		return err
	}
	if runner.gitExecutor == nil {
		return fmt.Errorf("Git 命令执行器不可用")
	}
	target, err := prepareGitCloneRepositoryTarget(request.WorkspacePath, parameters.Directory)
	if err != nil {
		return err
	}
	if err := runner.cloneGitRepository(parameters.Repository, target, strings.TrimSpace(request.GitCloneRepositoryBranch)); err != nil {
		return fmt.Errorf("克隆指定 Git 仓库失败: %w", err)
	}
	return nil
}

func prepareGitCloneRepositoryTarget(workspacePath, directory string) (string, error) {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return "", fmt.Errorf("任务工作目录不可用")
	}
	workspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", fmt.Errorf("解析任务工作目录失败: %w", err)
	}
	workspacePath = filepath.Clean(workspacePath)
	workspaceInfo, err := os.Lstat(workspacePath)
	if err != nil || !workspaceInfo.IsDir() {
		if err != nil {
			return "", fmt.Errorf("任务工作目录不可用: %w", err)
		}
		return "", fmt.Errorf("任务工作目录不可用")
	}
	if workspaceInfo.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("Git 克隆目标不安全: 任务工作目录不能是符号链接")
	}

	target := filepath.Clean(filepath.Join(workspacePath, directory))
	if !isWithinWorkspace(workspacePath, target) {
		return "", fmt.Errorf("Git 克隆目标不安全")
	}
	relative, err := filepath.Rel(workspacePath, target)
	if err != nil {
		return "", fmt.Errorf("解析 Git 克隆目标失败: %w", err)
	}
	current := workspacePath
	if relative != "." {
		for _, component := range strings.Split(relative, string(filepath.Separator)) {
			current = filepath.Join(current, component)
			info, err := os.Lstat(current)
			if os.IsNotExist(err) {
				if err := os.Mkdir(current, 0o700); err != nil {
					return "", fmt.Errorf("创建 Git 克隆目标失败: %w", err)
				}
				continue
			}
			if err != nil {
				return "", fmt.Errorf("检查 Git 克隆目标失败: %w", err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("Git 克隆目标不安全: 不能使用符号链接")
			}
			if !info.IsDir() {
				return "", fmt.Errorf("Git 克隆目标不可用: 目标不是目录")
			}
		}
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return "", fmt.Errorf("检查 Git 克隆目标失败: %w", err)
	}
	if len(entries) != 0 {
		return "", fmt.Errorf("Git 克隆目标不可用: 目标目录非空")
	}
	return target, nil
}

func (runner *CommandChainRunner) cloneGitRepositories(request CommandChainRequest, arguments []string) error {
	directory, err := settings.GitCloneDirectory(arguments)
	if err != nil {
		return err
	}
	gitInfos := builtInGitInfos(request.Task.ExtraInfo)
	if len(gitInfos) == 0 {
		return nil
	}
	if runner.gitExecutor == nil {
		return fmt.Errorf("Git 命令执行器不可用")
	}
	if info, err := os.Stat(request.WorkspacePath); err != nil || !info.IsDir() {
		if err != nil {
			return fmt.Errorf("任务工作目录不可用: %w", err)
		}
		return fmt.Errorf("任务工作目录不可用")
	}
	cloneRoot := filepath.Join(request.WorkspacePath, directory)
	if !isWithinWorkspace(request.WorkspacePath, cloneRoot) {
		return fmt.Errorf("Git 仓库克隆目录不安全")
	}
	if err := os.MkdirAll(cloneRoot, 0o700); err != nil {
		return fmt.Errorf("创建 Git 仓库克隆目录失败: %w", err)
	}

	for _, information := range gitInfos {
		name, repository, branch, err := gitInformationValues(information)
		if err != nil {
			return err
		}
		target := filepath.Join(cloneRoot, name)
		if !isWithinWorkspace(request.WorkspacePath, target) {
			return fmt.Errorf("Git 项目目录不安全: %q", name)
		}
		if _, err := os.Lstat(target); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("检查 Git 项目目录失败: %w", err)
		}
		if err := runner.cloneGitRepository(repository, target, branch); err != nil {
			return fmt.Errorf("克隆 Git 项目 %q 失败: %w", name, err)
		}
	}
	return nil
}

func builtInGitInfos(extraInfo []task.TaskExtraInfo) []task.TaskExtraInfo {
	templateID := task.BuiltInGitTemplate().ID
	infos := make([]task.TaskExtraInfo, 0, len(extraInfo))
	for _, information := range extraInfo {
		if information.TemplateID == templateID {
			infos = append(infos, information)
		}
	}
	return infos
}

func gitInformationValues(information task.TaskExtraInfo) (string, string, string, error) {
	var name string
	var repository string
	for _, field := range information.Fields {
		switch field.Key {
		case "name":
			name = strings.TrimSpace(field.Value)
		case "repository":
			repository = strings.TrimSpace(field.Value)
		}
	}
	branch := ""
	for _, parameter := range information.Parameters {
		if parameter.Key == "branch" {
			branch = strings.TrimSpace(parameter.Value)
			break
		}
	}
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return "", "", "", fmt.Errorf("Git 项目名称无效: %q", name)
	}
	if repository == "" {
		return "", "", "", fmt.Errorf("Git 项目 %q 缺少仓库地址", name)
	}
	return name, repository, branch, nil
}

func isWithinWorkspace(workspacePath, candidate string) bool {
	relativePath, err := filepath.Rel(workspacePath, candidate)
	if err != nil {
		return false
	}
	return relativePath == "." || (relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator)))
}

func (runner *CommandChainRunner) cloneGitRepository(repository, target, branch string) error {
	directory := filepath.Dir(target)
	if branch == "" {
		return runner.runGit("克隆仓库", directory, []string{"clone", "--", repository, target})
	}
	result, err := runner.gitExecutor.Run(GitInvocation{Directory: directory, Arguments: []string{"ls-remote", "--exit-code", "--heads", "--", repository, "refs/heads/" + branch}})
	if err == nil {
		return runner.runGit("克隆远程分支", directory, []string{"clone", "--branch", branch, "--", repository, target})
	}
	if !isGitRemoteBranchMissing(err) {
		return gitCommandError("检查远程分支", result.StandardError, err)
	}
	if err := runner.runGit("克隆默认分支", directory, []string{"clone", "--", repository, target}); err != nil {
		return err
	}
	return runner.runGit("创建本地分支", target, []string{"switch", "--create", branch})
}

func (runner *CommandChainRunner) runGit(action, directory string, arguments []string) error {
	result, err := runner.gitExecutor.Run(GitInvocation{Directory: directory, Arguments: arguments})
	if err != nil {
		return gitCommandError(action, result.StandardError, err)
	}
	return nil
}

func gitCommandError(action string, standardError []byte, cause error) error {
	if message := strings.TrimSpace(string(standardError)); message != "" {
		return fmt.Errorf("Git %s失败: %v: %s", action, cause, message)
	}
	return fmt.Errorf("Git %s失败: %w", action, cause)
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
