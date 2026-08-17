package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taskai/internal/gitlab"
	"taskai/internal/settings"
	"taskai/internal/task"
)

func TestRepositorySavesGitLabImportDefaultsAsPlaintext(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	if _, err := repository.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	saved, err := repository.SaveGitLabImportDefaults(gitlab.ConnectionDefaults{
		Address:  "  https://gitlab.example.com/private/gitlab/  ",
		Username: " integration-user ",
		Token:    "integration-token",
	})
	if err != nil {
		t.Fatalf("SaveGitLabImportDefaults() error = %v", err)
	}
	want := gitlab.ConnectionDefaults{
		Address:  "https://gitlab.example.com/private/gitlab",
		Username: "integration-user",
		Token:    "integration-token",
	}
	if saved != want {
		t.Fatalf("SaveGitLabImportDefaults() = %#v, want %#v", saved, want)
	}
	loaded, err := repository.GetGitLabImportDefaults()
	if err != nil {
		t.Fatalf("GetGitLabImportDefaults() error = %v", err)
	}
	if loaded != want {
		t.Fatalf("GetGitLabImportDefaults() = %#v, want %#v", loaded, want)
	}
	contents, err := os.ReadFile(repository.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(contents), `"token": "integration-token"`) {
		t.Fatalf("数据文件未明文保存 GitLab token: %s", contents)
	}
}

func TestRepositoryPreservesGitLabImportDefaultsWhenSaveFails(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	previous := gitlab.ConnectionDefaults{Address: "https://gitlab.example.com", Username: "previous-user", Token: "previous-token"}
	if _, err := repository.SaveGitLabImportDefaults(previous); err != nil {
		t.Fatalf("SaveGitLabImportDefaults(previous) error = %v", err)
	}
	repository.persistData = func(Data) error { return errors.New("受控保存失败") }

	if _, err := repository.SaveGitLabImportDefaults(gitlab.ConnectionDefaults{Address: "https://new.example.com", Username: "new-user", Token: "new-token"}); err == nil {
		t.Fatal("SaveGitLabImportDefaults(new) error = nil")
	}
	loaded, err := repository.GetGitLabImportDefaults()
	if err != nil {
		t.Fatalf("GetGitLabImportDefaults() error = %v", err)
	}
	if loaded != previous {
		t.Fatalf("GetGitLabImportDefaults() = %#v, want previous %#v", loaded, previous)
	}
}

func TestRepositoryImportsGitLabProjectsWithCurrentTemplateDefaultsInOneSave(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data.ExtraInfoTemplates[0].Fields = append(data.ExtraInfoTemplates[0].Fields, task.ExtraInfoField{Key: "owner", DisplayName: "负责人", DefaultValue: "platform"})
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	saveCalls := 0
	originalPersist := repository.persistData
	repository.persistData = func(next Data) error {
		saveCalls++
		return originalPersist(next)
	}

	result, err := repository.ImportGitLabProjects([]gitlab.Project{
		gitLabProject(1, "api", "team/api", "git@gitlab.example.com:team/api.git", "https://gitlab.example.com/team/api.git"),
		gitLabProject(2, "web", "team/web", "git@gitlab.example.com:team/web.git", "https://gitlab.example.com/team/web.git"),
	}, gitlab.CloneURLSSH)
	if err != nil {
		t.Fatalf("ImportGitLabProjects() error = %v", err)
	}
	if result.Imported != 2 || result.Skipped != 0 || len(result.Infos) != 2 {
		t.Fatalf("ImportGitLabProjects() result = %#v", result)
	}
	if saveCalls != 1 {
		t.Fatalf("保存次数 = %d，期望 1", saveCalls)
	}
	loaded, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.ExtraInfos) != 2 {
		t.Fatalf("ExtraInfos = %#v", loaded.ExtraInfos)
	}
	if task.ExtraInfoName(loaded.ExtraInfos[0]) != "api" || extraInfoValue(loaded.ExtraInfos[0], "repository") != "git@gitlab.example.com:team/api.git" || extraInfoValue(loaded.ExtraInfos[0], "owner") != "platform" {
		t.Fatalf("导入信息 = %#v", loaded.ExtraInfos[0])
	}
}

func TestRepositoryImportsHTTPSAndSkipsLatestCrossProtocolDuplicates(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	template := data.ExtraInfoTemplates[0]
	existing, err := task.NewExtraInfo(template, map[string]string{
		"name":       "api",
		"repository": "https://gitlab-a.example.com/team/api.git",
	})
	if err != nil {
		t.Fatalf("NewExtraInfo() error = %v", err)
	}
	data.ExtraInfos = append(data.ExtraInfos, existing)
	if err := repository.Save(data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	result, err := repository.ImportGitLabProjects([]gitlab.Project{
		gitLabProject(1, "api", "team/api", "git@gitlab-a.example.com:team/api.git", "https://gitlab-a.example.com/team/api.git"),
		gitLabProject(2, "api", "team/api", "git@gitlab-b.example.com:team/api.git", "https://gitlab-b.example.com/team/api.git"),
		gitLabProject(3, "api", "team/api", "git@gitlab-b.example.com:team/api.git", "https://gitlab-b.example.com/team/api.git"),
	}, gitlab.CloneURLHTTPS)
	if err != nil {
		t.Fatalf("ImportGitLabProjects() error = %v", err)
	}
	if result.Imported != 1 || result.Skipped != 2 || len(result.Infos) != 1 {
		t.Fatalf("ImportGitLabProjects() result = %#v", result)
	}
	if repositoryURL := extraInfoValue(result.Infos[0], "repository"); repositoryURL != "https://gitlab-b.example.com/team/api.git" {
		t.Fatalf("导入地址 = %q", repositoryURL)
	}
	loaded, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.ExtraInfos) != 2 {
		t.Fatalf("ExtraInfos 数量 = %d，期望 2", len(loaded.ExtraInfos))
	}
}

func TestRepositoryImportsRelativeURLProjectWithDifferentHTTPAndSSHPorts(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	if _, err := repository.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	project := gitLabProject(
		1,
		"api",
		"team/api",
		"ssh://git@gitlab.example.com:2424/team/api.git",
		"http://gitlab.example.com:8929/private/gitlab/team/api.git",
	)
	first, err := repository.ImportGitLabProjects([]gitlab.Project{project}, gitlab.CloneURLHTTPS)
	if err != nil {
		t.Fatalf("ImportGitLabProjects() first error = %v", err)
	}
	if first.Imported != 1 || first.Skipped != 0 {
		t.Fatalf("ImportGitLabProjects() first = %#v", first)
	}
	second, err := repository.ImportGitLabProjects([]gitlab.Project{project}, gitlab.CloneURLSSH)
	if err != nil {
		t.Fatalf("ImportGitLabProjects() second error = %v", err)
	}
	if second.Imported != 0 || second.Skipped != 1 {
		t.Fatalf("ImportGitLabProjects() second = %#v", second)
	}
}

func TestRepositoryRejectsInvalidGitLabBatchWithoutWritingAnyProject(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	if _, err := repository.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	before, err := os.ReadFile(repository.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	_, err = repository.ImportGitLabProjects([]gitlab.Project{
		gitLabProject(1, "api", "team/api", "git@gitlab.example.com:team/api.git", "https://gitlab.example.com/team/api.git"),
		gitLabProject(2, "web", "team/web", "git@gitlab.example.com:other/web.git", "https://gitlab.example.com/other/web.git"),
	}, gitlab.CloneURLSSH)
	if err == nil {
		t.Fatal("ImportGitLabProjects() error = nil")
	}
	after, readErr := os.ReadFile(repository.path)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(after) != string(before) {
		t.Fatal("无效批次修改了持久化数据")
	}
	loaded, loadErr := repository.Load()
	if loadErr != nil {
		t.Fatalf("Load() error = %v", loadErr)
	}
	if len(loaded.ExtraInfos) != 0 {
		t.Fatalf("ExtraInfos = %#v", loaded.ExtraInfos)
	}
}

func TestRepositoryPreservesDataWhenGitLabBatchSaveFails(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	if _, err := repository.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	before, err := os.ReadFile(repository.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	repository.persistData = func(Data) error { return errors.New("受控保存失败") }

	result, err := repository.ImportGitLabProjects([]gitlab.Project{
		gitLabProject(1, "api", "team/api", "git@gitlab.example.com:team/api.git", "https://gitlab.example.com/team/api.git"),
	}, gitlab.CloneURLSSH)
	if err == nil || result.Imported != 0 || result.Skipped != 0 || result.Infos != nil {
		t.Fatalf("ImportGitLabProjects() result/error = %#v / %v", result, err)
	}
	after, readErr := os.ReadFile(repository.path)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(after) != string(before) {
		t.Fatal("保存失败修改了持久化数据")
	}
}

func TestRepositoryRejectsUnsupportedGitLabCloneURLMode(t *testing.T) {
	repository := New(filepath.Join(t.TempDir(), "state.json"), settings.Default(t.TempDir()))
	if _, err := repository.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	_, err := repository.ImportGitLabProjects([]gitlab.Project{
		gitLabProject(1, "api", "team/api", "git@gitlab.example.com:team/api.git", "https://gitlab.example.com/team/api.git"),
	}, gitlab.CloneURLMode("token-in-url"))
	if err == nil {
		t.Fatal("ImportGitLabProjects() error = nil")
	}
}

func gitLabProject(id int64, name, path, sshURL, httpURL string) gitlab.Project {
	return gitlab.Project{ID: id, Name: name, PathWithNamespace: path, SSHURL: sshURL, HTTPURL: httpURL, Visibility: "private"}
}
