package repositorygit

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Action string

const (
	ActionNone    Action = "none"
	ActionCommit  Action = "commit"
	ActionPublish Action = "publish"
	ActionSync    Action = "sync"
)

type Repository struct {
	Path               string `json:"path"`
	Branch             string `json:"branch,omitempty"`
	Remote             string `json:"remote,omitempty"`
	Notice             string `json:"notice,omitempty"`
	Dirty              bool   `json:"dirty"`
	HasUpstream        bool   `json:"hasUpstream"`
	RemoteBranchExists bool   `json:"remoteBranchExists"`
	Synchronized       bool   `json:"synchronized"`
	Action             Action `json:"action"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (service *Service) List(workspacePath string, maximumDepth int) ([]Repository, error) {
	if maximumDepth < 1 {
		return nil, fmt.Errorf("Git 最大扫描深度至少为 1")
	}
	workspace, err := resolveWorkspace(workspacePath)
	if err != nil {
		return nil, err
	}
	directories, err := findRepositoryDirectories(workspace, maximumDepth)
	if err != nil {
		return nil, err
	}
	repositories := make([]Repository, 0, len(directories))
	for _, directory := range directories {
		repository, err := service.status(workspace, directory)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, repository)
	}
	sort.Slice(repositories, func(left, right int) bool { return repositories[left].Path < repositories[right].Path })
	return repositories, nil
}

func (service *Service) Commit(workspacePath, repositoryPath, message string) (Repository, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return Repository{}, fmt.Errorf("必须填写提交信息")
	}
	workspace, directory, err := resolveRepository(workspacePath, repositoryPath)
	if err != nil {
		return Repository{}, err
	}
	current, err := service.status(workspace, directory)
	if err != nil {
		return Repository{}, err
	}
	if !current.Dirty {
		return current, fmt.Errorf("仓库没有待提交的改动")
	}
	if _, err := runGit(directory, "add", "-A"); err != nil {
		return current, gitError("暂存全部改动失败", err)
	}
	if _, err := runGit(directory, "commit", "-m", message); err != nil {
		return current, gitError("提交改动失败", err)
	}
	return service.status(workspace, directory)
}

func (service *Service) Publish(workspacePath, repositoryPath string) (Repository, error) {
	workspace, directory, err := resolveRepository(workspacePath, repositoryPath)
	if err != nil {
		return Repository{}, err
	}
	current, err := service.status(workspace, directory)
	if err != nil {
		return Repository{}, err
	}
	if current.Dirty {
		return current, fmt.Errorf("请先提交")
	}
	if current.Branch == "" {
		return current, fmt.Errorf("当前仓库未处于本地分支，无法发布分支")
	}
	if current.Remote == "" {
		return current, remoteConfigurationError(directory)
	}
	if current.RemoteBranchExists {
		return current, fmt.Errorf("远程已存在当前分支，请同步远程")
	}
	if _, err := runGit(directory, "push", "-u", current.Remote, current.Branch); err != nil {
		return current, gitError("发布分支失败", err)
	}
	return service.status(workspace, directory)
}

func (service *Service) Sync(workspacePath, repositoryPath string) (Repository, error) {
	workspace, directory, err := resolveRepository(workspacePath, repositoryPath)
	if err != nil {
		return Repository{}, err
	}
	current, err := service.status(workspace, directory)
	if err != nil {
		return Repository{}, err
	}
	if current.Dirty {
		return current, fmt.Errorf("请先提交")
	}
	if current.Branch == "" {
		return current, fmt.Errorf("当前仓库未处于本地分支，无法同步远程")
	}
	if current.Remote == "" {
		return current, remoteConfigurationError(directory)
	}
	if current.Notice != "" {
		return current, fmt.Errorf("%s", current.Notice)
	}
	if !current.RemoteBranchExists {
		return current, fmt.Errorf("远程不存在当前分支，请先发布分支")
	}
	if !current.HasUpstream {
		if _, err := runGit(directory, "fetch", current.Remote, current.Branch); err != nil {
			return current, gitError("读取远程分支失败", err)
		}
		if _, err := runGit(directory, "branch", "--set-upstream-to="+current.Remote+"/"+current.Branch); err != nil {
			return current, gitError("设置上游分支失败", err)
		}
	}
	if _, err := runGit(directory, "pull", current.Remote, current.Branch); err != nil {
		return current, gitError("拉取远程分支失败", err)
	}
	if _, err := runGit(directory, "push", current.Remote, current.Branch); err != nil {
		return current, gitError("推送远程分支失败", err)
	}
	return service.status(workspace, directory)
}

func (service *Service) status(workspace, directory string) (Repository, error) {
	relativePath, err := safeRelativePath(workspace, directory)
	if err != nil {
		return Repository{}, err
	}
	dirtyOutput, err := runGit(directory, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return Repository{}, gitError("读取仓库状态失败", err)
	}
	branch, _ := runGit(directory, "symbolic-ref", "--quiet", "--short", "HEAD")
	remotesOutput, err := runGit(directory, "remote")
	if err != nil {
		return Repository{}, gitError("读取远程仓库失败", err)
	}
	remote, notice := selectRemote(strings.Fields(string(remotesOutput)))
	repository := Repository{
		Path:   relativePath,
		Branch: strings.TrimSpace(string(branch)),
		Remote: remote,
		Notice: notice,
		Dirty:  len(bytes.TrimSpace(dirtyOutput)) > 0,
		Action: ActionNone,
	}
	if repository.Branch == "" {
		repository.Notice = "当前仓库未处于本地分支，无法提交、发布或同步"
		return repository, nil
	}
	if repository.Dirty {
		repository.Action = ActionCommit
		return repository, nil
	}
	if remote == "" {
		repository.Action = ActionSync
		return repository, nil
	}
	remoteHead, err := remoteBranchHead(directory, remote, repository.Branch)
	if err != nil {
		repository.Notice = err.Error()
		repository.Action = ActionSync
		return repository, nil
	}
	repository.RemoteBranchExists = remoteHead != ""
	repository.HasUpstream = hasUpstream(directory)
	if repository.RemoteBranchExists {
		repository.Synchronized = isSynchronized(directory, remote, repository.Branch, remoteHead)
		repository.Action = ActionSync
		return repository, nil
	}
	repository.Action = ActionPublish
	return repository, nil
}

func findRepositoryDirectories(workspace string, maximumDepth int) ([]string, error) {
	directories := map[string]struct{}{}
	err := filepath.WalkDir(workspace, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == workspace {
				return fmt.Errorf("读取任务工作目录失败: %w", walkErr)
			}
			return filepath.SkipDir
		}
		if path != workspace && entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if entry.Name() == ".git" {
			return filepath.SkipDir
		}
		depth, err := directoryDepth(workspace, path)
		if err != nil {
			return filepath.SkipDir
		}
		if depth > maximumDepth {
			return filepath.SkipDir
		}
		topLevel, err := gitTopLevel(path)
		if err != nil {
			if depth == maximumDepth {
				return filepath.SkipDir
			}
			return nil
		}
		canonicalTopLevel, err := filepath.EvalSymlinks(topLevel)
		if err != nil {
			return nil
		}
		if _, err := safeRelativePath(workspace, canonicalTopLevel); err != nil {
			return nil
		}
		directories[canonicalTopLevel] = struct{}{}
		if depth == maximumDepth {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	results := make([]string, 0, len(directories))
	for directory := range directories {
		results = append(results, directory)
	}
	sort.Strings(results)
	return results, nil
}

func directoryDepth(workspace, directory string) (int, error) {
	relativePath, err := safeRelativePath(workspace, directory)
	if err != nil {
		return 0, err
	}
	if relativePath == "." {
		return 1, nil
	}
	return len(strings.Split(relativePath, string(filepath.Separator))) + 1, nil
}

func resolveRepository(workspacePath, repositoryPath string) (string, string, error) {
	workspace, err := resolveWorkspace(workspacePath)
	if err != nil {
		return "", "", err
	}
	repositoryPath, err = cleanRepositoryPath(repositoryPath)
	if err != nil {
		return "", "", err
	}
	directory := filepath.Join(workspace, repositoryPath)
	canonicalDirectory, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", "", fmt.Errorf("仓库目录不可用: %w", err)
	}
	if _, err := safeRelativePath(workspace, canonicalDirectory); err != nil {
		return "", "", err
	}
	topLevel, err := gitTopLevel(canonicalDirectory)
	if err != nil {
		return "", "", fmt.Errorf("指定目录不是 Git 仓库")
	}
	canonicalTopLevel, err := filepath.EvalSymlinks(topLevel)
	if err != nil {
		return "", "", fmt.Errorf("解析 Git 仓库目录失败: %w", err)
	}
	relativePath, err := safeRelativePath(workspace, canonicalTopLevel)
	if err != nil || relativePath != repositoryPath {
		return "", "", fmt.Errorf("仓库路径无效")
	}
	return workspace, canonicalTopLevel, nil
}

func resolveWorkspace(workspacePath string) (string, error) {
	if strings.TrimSpace(workspacePath) == "" {
		return "", fmt.Errorf("任务缺少工作目录")
	}
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", fmt.Errorf("解析任务工作目录失败: %w", err)
	}
	canonicalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("任务工作目录不可用: %w", err)
	}
	info, err := os.Stat(canonicalPath)
	if err != nil {
		return "", fmt.Errorf("检查任务工作目录失败: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("任务工作目录不是目录")
	}
	return filepath.Clean(canonicalPath), nil
}

func cleanRepositoryPath(repositoryPath string) (string, error) {
	if strings.TrimSpace(repositoryPath) == "" {
		return "", fmt.Errorf("仓库路径不能为空")
	}
	cleanedPath := filepath.Clean(repositoryPath)
	if filepath.IsAbs(cleanedPath) || cleanedPath == ".." || strings.HasPrefix(cleanedPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("仓库路径无效")
	}
	return cleanedPath, nil
}

func safeRelativePath(root, candidate string) (string, error) {
	relativePath, err := filepath.Rel(root, candidate)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("仓库目录不在任务工作目录内")
	}
	if relativePath == "." {
		return ".", nil
	}
	return filepath.Clean(relativePath), nil
}

func gitTopLevel(directory string) (string, error) {
	output, err := runGit(directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func selectRemote(remotes []string) (string, string) {
	if len(remotes) == 0 {
		return "", "请先配置远程仓库"
	}
	for _, remote := range remotes {
		if remote == "origin" {
			return remote, ""
		}
	}
	if len(remotes) == 1 {
		return remotes[0], ""
	}
	return "", "存在多个远程仓库但没有 origin，请配置要使用的远程仓库"
}

func remoteBranchHead(directory, remote, branch string) (string, error) {
	output, err := runGit(directory, "ls-remote", "--exit-code", "--heads", remote, "refs/heads/"+branch)
	if err == nil {
		fields := strings.Fields(string(output))
		if len(fields) > 0 {
			return fields[0], nil
		}
		return "", nil
	}
	if exitCode(err) == 2 {
		return "", nil
	}
	return "", gitError("读取远程分支失败", err)
}

func hasUpstream(directory string) bool {
	_, err := runGit(directory, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	return err == nil
}

func isSynchronized(directory, remote, branch, remoteHead string) bool {
	upstream, err := runGit(directory, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil || strings.TrimSpace(string(upstream)) != remote+"/"+branch {
		return false
	}
	trackedHead, err := runGit(directory, "rev-parse", remote+"/"+branch)
	if err != nil || strings.TrimSpace(string(trackedHead)) != remoteHead {
		return false
	}
	differences, err := runGit(directory, "rev-list", "--left-right", "--count", remote+"/"+branch+"...HEAD")
	if err != nil {
		return false
	}
	counts := strings.Fields(string(differences))
	return len(counts) == 2 && counts[0] == "0" && counts[1] == "0"
}

func remoteConfigurationError(directory string) error {
	remotesOutput, err := runGit(directory, "remote")
	if err != nil {
		return gitError("读取远程仓库失败", err)
	}
	if len(strings.Fields(string(remotesOutput))) == 0 {
		return fmt.Errorf("请先配置远程仓库")
	}
	return fmt.Errorf("存在多个远程仓库但没有 origin，请配置要使用的远程仓库")
}

type commandError struct {
	output string
	err    error
}

func (err *commandError) Error() string {
	return err.err.Error()
}

func (err *commandError) Unwrap() error {
	return err.err
}

func (err *commandError) ExitCode() int {
	var exitError *exec.ExitError
	if errors.As(err.err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

func runGit(directory string, arguments ...string) ([]byte, error) {
	command := exec.Command("git", arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var output bytes.Buffer
	var standardError bytes.Buffer
	command.Stdout = &output
	command.Stderr = &standardError
	if err := command.Run(); err != nil {
		return output.Bytes(), &commandError{output: strings.TrimSpace(standardError.String()), err: err}
	}
	return output.Bytes(), nil
}

func exitCode(err error) int {
	if commandErr, ok := err.(*commandError); ok {
		return commandErr.ExitCode()
	}
	return -1
}

func gitError(action string, err error) error {
	if commandErr, ok := err.(*commandError); ok && commandErr.output != "" {
		return fmt.Errorf("%s: %s", action, commandErr.output)
	}
	return fmt.Errorf("%s: %w", action, err)
}
