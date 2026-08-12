package repositorygit

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestServiceListsRepositoriesAndChoosesPrimaryAction(t *testing.T) {
	workspacePath := t.TempDir()
	remotePath := filepath.Join(t.TempDir(), "remote.git")
	executeGit(t, workspacePath, "init", "--initial-branch=main")
	executeGit(t, workspacePath, "config", "user.name", "测试用户")
	executeGit(t, workspacePath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(workspacePath, "README.md"), "initial\n")
	executeGit(t, workspacePath, "add", "README.md")
	executeGit(t, workspacePath, "commit", "-m", "initial")
	executeGit(t, filepath.Dir(remotePath), "init", "--bare", remotePath)
	executeGit(t, workspacePath, "remote", "add", "origin", remotePath)
	executeGit(t, workspacePath, "push", "-u", "origin", "main")

	service := NewService()
	repositories, err := service.List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(repositories) != 1 {
		t.Fatalf("List() repositories = %#v", repositories)
	}
	if repository := repositories[0]; repository.Path != "." || repository.Action != ActionSync || repository.Remote != "origin" || !repository.RemoteBranchExists {
		t.Fatalf("同步仓库状态 = %#v", repository)
	}

	writeFile(t, filepath.Join(workspacePath, "README.md"), "changed\n")
	repositories, err = service.List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() dirty error = %v", err)
	}
	if repository := repositories[0]; !repository.Dirty || repository.Action != ActionCommit {
		t.Fatalf("有改动的仓库状态 = %#v", repository)
	}

	executeGit(t, workspacePath, "add", "README.md")
	executeGit(t, workspacePath, "commit", "-m", "changed")
	executeGit(t, workspacePath, "switch", "-c", "feature/task-git")
	repositories, err = service.List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() unpublished branch error = %v", err)
	}
	if repository := repositories[0]; repository.Action != ActionPublish || repository.RemoteBranchExists {
		t.Fatalf("未发布分支状态 = %#v", repository)
	}
}

func TestServiceReportsSynchronizedRepository(t *testing.T) {
	workspacePath := t.TempDir()
	remotePath := filepath.Join(t.TempDir(), "remote.git")
	setupRepositoryWithRemote(t, workspacePath, remotePath)
	service := NewService()
	if _, err := service.Publish(workspacePath, "."); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	repositories, err := service.List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(repositories) != 1 || !repositories[0].Synchronized {
		t.Fatalf("完全同步的仓库状态 = %#v", repositories)
	}
}

func TestServiceDoesNotReportSynchronizedWhenLocalOrRemoteDiffer(t *testing.T) {
	workspacePath := t.TempDir()
	remotePath := filepath.Join(t.TempDir(), "remote.git")
	setupRepositoryWithRemote(t, workspacePath, remotePath)
	service := NewService()
	if _, err := service.Publish(workspacePath, "."); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	writeFile(t, filepath.Join(workspacePath, "local.txt"), "local\n")
	executeGit(t, workspacePath, "add", "local.txt")
	executeGit(t, workspacePath, "commit", "-m", "local update")
	repositories, err := service.List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() after local update error = %v", err)
	}
	if len(repositories) != 1 || repositories[0].Synchronized {
		t.Fatalf("本地领先的仓库状态 = %#v", repositories)
	}
	if _, err := service.Sync(workspacePath, "."); err != nil {
		t.Fatalf("Sync() local update error = %v", err)
	}

	otherPath := filepath.Join(t.TempDir(), "other")
	executeGit(t, filepath.Dir(otherPath), "clone", "--branch", "main", remotePath, otherPath)
	executeGit(t, otherPath, "config", "user.name", "测试用户")
	executeGit(t, otherPath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(otherPath, "remote.txt"), "remote\n")
	executeGit(t, otherPath, "add", "remote.txt")
	executeGit(t, otherPath, "commit", "-m", "remote update")
	executeGit(t, otherPath, "push", "origin", "HEAD")

	repositories, err = service.List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() after remote update error = %v", err)
	}
	if len(repositories) != 1 || repositories[0].Synchronized {
		t.Fatalf("远程领先的仓库状态 = %#v", repositories)
	}
}

func TestServiceFindsNestedRepositoryAndGitWorktree(t *testing.T) {
	workspacePath := t.TempDir()
	mainPath := filepath.Join(workspacePath, "main")
	nestedPath := filepath.Join(mainPath, "nested")
	worktreePath := filepath.Join(workspacePath, "worktree")
	executeGit(t, mainPath, "init", "--initial-branch=main")
	executeGit(t, mainPath, "config", "user.name", "测试用户")
	executeGit(t, mainPath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(mainPath, "README.md"), "main\n")
	executeGit(t, mainPath, "add", "README.md")
	executeGit(t, mainPath, "commit", "-m", "initial")
	executeGit(t, mainPath, "worktree", "add", "-b", "feature/worktree", worktreePath)
	executeGit(t, nestedPath, "init", "--initial-branch=main")

	repositories, err := NewService().List(workspacePath, 3)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	paths := make(map[string]bool, len(repositories))
	for _, repository := range repositories {
		paths[repository.Path] = true
	}
	for _, path := range []string{"main", filepath.Join("main", "nested"), "worktree"} {
		if !paths[path] {
			t.Fatalf("List() paths = %#v, missing %q", paths, path)
		}
	}
}

func TestServiceLimitsRepositoryScanDepth(t *testing.T) {
	workspacePath := t.TempDir()
	childPath := filepath.Join(workspacePath, "child")
	grandchildPath := filepath.Join(childPath, "grandchild")
	for _, path := range []string{workspacePath, childPath, grandchildPath} {
		executeGit(t, path, "init", "--initial-branch=main")
	}

	shallow, err := NewService().List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() depth 2 error = %v", err)
	}
	if got, want := repositoryPaths(shallow), []string{".", "child"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List() depth 2 paths = %#v，期望 %#v", got, want)
	}

	deep, err := NewService().List(workspacePath, 3)
	if err != nil {
		t.Fatalf("List() depth 3 error = %v", err)
	}
	if got, want := repositoryPaths(deep), []string{".", "child", filepath.Join("child", "grandchild")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List() depth 3 paths = %#v，期望 %#v", got, want)
	}
}

func TestServiceCommitsAllChanges(t *testing.T) {
	workspacePath := t.TempDir()
	executeGit(t, workspacePath, "init", "--initial-branch=main")
	executeGit(t, workspacePath, "config", "user.name", "测试用户")
	executeGit(t, workspacePath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(workspacePath, "tracked.txt"), "before\n")
	executeGit(t, workspacePath, "add", "tracked.txt")
	executeGit(t, workspacePath, "commit", "-m", "initial")
	writeFile(t, filepath.Join(workspacePath, "tracked.txt"), "after\n")
	writeFile(t, filepath.Join(workspacePath, "untracked.txt"), "new\n")

	repository, err := NewService().Commit(workspacePath, ".", "保存全部改动")
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if repository.Dirty || repository.Action != ActionSync {
		t.Fatalf("Commit() repository = %#v", repository)
	}
	status, err := runGit(workspacePath, "status", "--porcelain")
	if err != nil {
		t.Fatalf("git status error = %v", err)
	}
	if got := string(status); got != "" {
		t.Fatalf("提交后状态 = %q，期望干净", got)
	}
	files, err := runGit(workspacePath, "show", "--format=", "--name-only", "HEAD")
	if err != nil {
		t.Fatalf("git show error = %v", err)
	}
	if got := strings.Fields(string(files)); !reflect.DeepEqual(got, []string{"tracked.txt", "untracked.txt"}) {
		t.Fatalf("提交文件 = %#v", got)
	}
}

func TestServiceRejectsEmptyCommitMessage(t *testing.T) {
	workspacePath := t.TempDir()
	executeGit(t, workspacePath, "init", "--initial-branch=main")
	writeFile(t, filepath.Join(workspacePath, "new.txt"), "new\n")

	_, err := NewService().Commit(workspacePath, ".", "   ")
	if err == nil || !strings.Contains(err.Error(), "必须填写提交信息") {
		t.Fatalf("Commit() error = %v", err)
	}
}

func TestServicePublishesAndSynchronizesBranch(t *testing.T) {
	workspacePath := t.TempDir()
	remotePath := filepath.Join(t.TempDir(), "remote.git")
	setupRepositoryWithRemote(t, workspacePath, remotePath)

	service := NewService()
	beforePublish, err := service.List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() before publish error = %v", err)
	}
	if beforePublish[0].Action != ActionPublish {
		t.Fatalf("发布前状态 = %#v", beforePublish[0])
	}
	published, err := service.Publish(workspacePath, ".")
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if published.Action != ActionSync || !published.HasUpstream || !published.RemoteBranchExists {
		t.Fatalf("发布后状态 = %#v", published)
	}

	otherPath := filepath.Join(t.TempDir(), "other")
	executeGit(t, filepath.Dir(otherPath), "clone", "--branch", "main", remotePath, otherPath)
	executeGit(t, otherPath, "config", "user.name", "测试用户")
	executeGit(t, otherPath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(otherPath, "from-remote.txt"), "remote\n")
	executeGit(t, otherPath, "add", "from-remote.txt")
	executeGit(t, otherPath, "commit", "-m", "remote update")
	executeGit(t, otherPath, "push", "origin", "HEAD")

	synced, err := service.Sync(workspacePath, ".")
	if err != nil {
		t.Fatalf("Sync() pull error = %v", err)
	}
	if synced.Action != ActionSync {
		t.Fatalf("同步拉取后状态 = %#v", synced)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "from-remote.txt")); err != nil {
		t.Fatalf("同步后缺少远程文件: %v", err)
	}

	writeFile(t, filepath.Join(workspacePath, "from-local.txt"), "local\n")
	executeGit(t, workspacePath, "add", "from-local.txt")
	executeGit(t, workspacePath, "commit", "-m", "local update")
	if _, err := service.Sync(workspacePath, "."); err != nil {
		t.Fatalf("Sync() push error = %v", err)
	}
	executeGit(t, otherPath, "pull", "origin", "main")
	if _, err := os.Stat(filepath.Join(otherPath, "from-local.txt")); err != nil {
		t.Fatalf("同步推送后远程克隆缺少本地文件: %v", err)
	}
}

func TestServiceSynchronizesExistingRemoteBranchWithoutLocalUpstream(t *testing.T) {
	remotePath := filepath.Join(t.TempDir(), "remote.git")
	sourcePath := filepath.Join(t.TempDir(), "source")
	setupRepositoryWithRemote(t, sourcePath, remotePath)
	if _, err := NewService().Publish(sourcePath, "."); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	workspacePath := filepath.Join(t.TempDir(), "workspace")
	executeGit(t, filepath.Dir(workspacePath), "clone", "--branch", "main", remotePath, workspacePath)
	executeGit(t, workspacePath, "branch", "--unset-upstream")
	executeGit(t, workspacePath, "update-ref", "-d", "refs/remotes/origin/main")
	repositories, err := NewService().List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(repositories) != 1 || repositories[0].Action != ActionSync || repositories[0].HasUpstream || repositories[0].Synchronized {
		t.Fatalf("同步前状态 = %#v", repositories)
	}

	synced, err := NewService().Sync(workspacePath, ".")
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if !synced.HasUpstream || !synced.Synchronized || synced.Action != ActionSync {
		t.Fatalf("同步后状态 = %#v", synced)
	}
}

func TestServiceKeepsOtherRepositoriesVisibleWhenRemoteIsUnavailable(t *testing.T) {
	workspacePath := t.TempDir()
	unavailablePath := filepath.Join(workspacePath, "unavailable")
	availablePath := filepath.Join(workspacePath, "available")
	unavailableRemotePath := filepath.Join(t.TempDir(), "missing.git")
	setupRepositoryWithRemote(t, unavailablePath, unavailableRemotePath)
	setupRepositoryWithRemote(t, availablePath, filepath.Join(t.TempDir(), "available.git"))
	if err := os.RemoveAll(unavailableRemotePath); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	repositories, err := NewService().List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(repositories) != 2 {
		t.Fatalf("List() = %#v", repositories)
	}
	if repositories[0].Path != "available" || repositories[0].Notice != "" {
		t.Fatalf("可用仓库状态 = %#v", repositories[0])
	}
	if repositories[1].Path != "unavailable" || !strings.Contains(repositories[1].Notice, "读取远程分支失败") {
		t.Fatalf("不可用仓库状态 = %#v", repositories[1])
	}
	if _, err := NewService().Sync(workspacePath, "unavailable"); err == nil || !strings.Contains(err.Error(), "读取远程分支失败") {
		t.Fatalf("Sync() error = %v", err)
	}
}

func TestServiceDoesNotPushWhenPullFails(t *testing.T) {
	workspacePath := t.TempDir()
	remotePath := filepath.Join(t.TempDir(), "remote.git")
	setupRepositoryWithRemote(t, workspacePath, remotePath)
	service := NewService()
	if _, err := service.Publish(workspacePath, "."); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	otherPath := filepath.Join(t.TempDir(), "other")
	executeGit(t, filepath.Dir(otherPath), "clone", "--branch", "main", remotePath, otherPath)
	executeGit(t, otherPath, "config", "user.name", "测试用户")
	executeGit(t, otherPath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(otherPath, "README.md"), "remote\n")
	executeGit(t, otherPath, "add", "README.md")
	executeGit(t, otherPath, "commit", "-m", "remote conflict")
	executeGit(t, otherPath, "push", "origin", "HEAD")
	remoteHead, err := runGit(otherPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("remote head error = %v", err)
	}

	writeFile(t, filepath.Join(workspacePath, "README.md"), "local\n")
	executeGit(t, workspacePath, "add", "README.md")
	executeGit(t, workspacePath, "commit", "-m", "local conflict")
	if _, err := service.Sync(workspacePath, "."); err == nil || !strings.Contains(err.Error(), "拉取远程分支失败") {
		t.Fatalf("Sync() error = %v", err)
	}
	currentRemoteHead, err := runGit(otherPath, "rev-parse", "origin/main")
	if err != nil {
		t.Fatalf("read remote head error = %v", err)
	}
	if string(currentRemoteHead) != string(remoteHead) {
		t.Fatalf("拉取失败后远程 HEAD = %q，期望 %q", currentRemoteHead, remoteHead)
	}
}

func TestServiceListsRepositoriesWithRemoteConfigurationNotice(t *testing.T) {
	workspacePath := t.TempDir()
	executeGit(t, workspacePath, "init", "--initial-branch=main")
	executeGit(t, workspacePath, "config", "user.name", "测试用户")
	executeGit(t, workspacePath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(workspacePath, "README.md"), "initial\n")
	executeGit(t, workspacePath, "add", "README.md")
	executeGit(t, workspacePath, "commit", "-m", "initial")
	executeGit(t, workspacePath, "remote", "add", "fork", filepath.Join(t.TempDir(), "fork.git"))
	executeGit(t, workspacePath, "remote", "add", "upstream", filepath.Join(t.TempDir(), "upstream.git"))

	repositories, err := NewService().List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(repositories) != 1 || repositories[0].Action != ActionSync || !strings.Contains(repositories[0].Notice, "多个远程仓库") {
		t.Fatalf("List() = %#v", repositories)
	}
}

func TestServiceExplainsDetachedHeadWithoutOfferingOperation(t *testing.T) {
	workspacePath := t.TempDir()
	executeGit(t, workspacePath, "init", "--initial-branch=main")
	executeGit(t, workspacePath, "config", "user.name", "测试用户")
	executeGit(t, workspacePath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(workspacePath, "README.md"), "initial\n")
	executeGit(t, workspacePath, "add", "README.md")
	executeGit(t, workspacePath, "commit", "-m", "initial")
	executeGit(t, workspacePath, "switch", "--detach")

	repositories, err := NewService().List(workspacePath, 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(repositories) != 1 || repositories[0].Action != ActionNone || !strings.Contains(repositories[0].Notice, "本地分支") {
		t.Fatalf("List() = %#v", repositories)
	}
}

func setupRepositoryWithRemote(t *testing.T, workspacePath, remotePath string) {
	t.Helper()
	executeGit(t, workspacePath, "init", "--initial-branch=main")
	executeGit(t, workspacePath, "config", "user.name", "测试用户")
	executeGit(t, workspacePath, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(workspacePath, "README.md"), "initial\n")
	executeGit(t, workspacePath, "add", "README.md")
	executeGit(t, workspacePath, "commit", "-m", "initial")
	executeGit(t, filepath.Dir(remotePath), "init", "--bare", remotePath)
	executeGit(t, workspacePath, "remote", "add", "origin", remotePath)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func executeGit(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", directory, err)
	}
	command := exec.Command("git", arguments...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v error = %v, output = %s", arguments, err, output)
	}
}

func repositoryPaths(repositories []Repository) []string {
	paths := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		paths = append(paths, repository.Path)
	}
	return paths
}
