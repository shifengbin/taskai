package storage

import (
	"fmt"
	"strings"

	"taskai/internal/gitlab"
	"taskai/internal/task"
)

type GitLabImportResult struct {
	Imported int              `json:"imported"`
	Skipped  int              `json:"skipped"`
	Infos    []task.ExtraInfo `json:"infos"`
}

func (repository *Repository) GetGitLabImportDefaults() (gitlab.ConnectionDefaults, error) {
	data, err := repository.Load()
	if err != nil {
		return gitlab.ConnectionDefaults{}, err
	}
	if data.GitLabImportDefaults == nil {
		return gitlab.ConnectionDefaults{}, nil
	}
	return *data.GitLabImportDefaults, nil
}

func (repository *Repository) SaveGitLabImportDefaults(next gitlab.ConnectionDefaults) (gitlab.ConnectionDefaults, error) {
	normalized, err := gitlab.NormalizeConnectionDefaults(next)
	if err != nil {
		return gitlab.ConnectionDefaults{}, err
	}
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return gitlab.ConnectionDefaults{}, err
	}
	data.GitLabImportDefaults = &normalized
	if err := repository.persistData(data); err != nil {
		return gitlab.ConnectionDefaults{}, err
	}
	return normalized, nil
}

func (repository *Repository) ImportGitLabProjects(projects []gitlab.Project, mode gitlab.CloneURLMode) (GitLabImportResult, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	if mode != gitlab.CloneURLSSH && mode != gitlab.CloneURLHTTPS {
		return GitLabImportResult{}, fmt.Errorf("不支持的 Git 仓库地址格式 %q", mode)
	}
	if len(projects) == 0 {
		return GitLabImportResult{}, fmt.Errorf("请至少选择一个 GitLab 项目")
	}
	data, err := repository.Load()
	if err != nil {
		return GitLabImportResult{}, err
	}
	template, err := builtInGitTemplate(data.ExtraInfoTemplates)
	if err != nil {
		return GitLabImportResult{}, err
	}
	existing := existingGitRepositoryIdentities(data.ExtraInfos)
	created := make([]task.ExtraInfo, 0, len(projects))
	skipped := 0
	for _, project := range projects {
		repositoryURL, identities, err := validateGitLabImportProject(project, mode)
		if err != nil {
			return GitLabImportResult{}, err
		}
		if containsGitRepositoryIdentity(existing, identities) {
			skipped++
			continue
		}
		information, err := task.NewExtraInfo(template, map[string]string{
			"name":       strings.TrimSpace(project.Name),
			"repository": repositoryURL,
		})
		if err != nil {
			return GitLabImportResult{}, fmt.Errorf("构造 Git 信息 %q: %w", project.PathWithNamespace, err)
		}
		created = append(created, information)
		for _, identity := range identities {
			existing[identity] = struct{}{}
		}
	}
	if len(created) == 0 {
		return GitLabImportResult{Skipped: skipped, Infos: []task.ExtraInfo{}}, nil
	}
	data.ExtraInfos = append(data.ExtraInfos, created...)
	if err := repository.persistData(data); err != nil {
		return GitLabImportResult{}, fmt.Errorf("批量保存 GitLab 项目: %w", err)
	}
	return GitLabImportResult{Imported: len(created), Skipped: skipped, Infos: created}, nil
}

func builtInGitTemplate(templates []task.ExtraInfoTemplate) (task.ExtraInfoTemplate, error) {
	for _, template := range templates {
		if template.BuiltIn && template.Catalogue == "git" {
			return template, nil
		}
	}
	return task.ExtraInfoTemplate{}, fmt.Errorf("内置 Git 信息模板不存在")
}

func existingGitRepositoryIdentities(infos []task.ExtraInfo) map[gitlab.RepositoryIdentity]struct{} {
	identities := make(map[gitlab.RepositoryIdentity]struct{}, len(infos))
	for _, information := range infos {
		if information.Catalogue != "git" {
			continue
		}
		identity, err := gitlab.ParseRepositoryIdentity(extraInfoField(information, "repository"))
		if err == nil {
			identities[identity] = struct{}{}
		}
	}
	return identities
}

func validateGitLabImportProject(project gitlab.Project, mode gitlab.CloneURLMode) (string, []gitlab.RepositoryIdentity, error) {
	project.Name = strings.TrimSpace(project.Name)
	project.PathWithNamespace = strings.Trim(strings.TrimSpace(project.PathWithNamespace), "/")
	if project.ID <= 0 || project.Name == "" || project.PathWithNamespace == "" {
		return "", nil, fmt.Errorf("GitLab 项目信息不完整")
	}
	identities, err := gitlab.ProjectRepositoryIdentities(project)
	if err != nil {
		return "", nil, err
	}
	repositoryURL, err := project.CloneURL(mode)
	if err != nil {
		return "", nil, err
	}
	return repositoryURL, identities, nil
}

func containsGitRepositoryIdentity(existing map[gitlab.RepositoryIdentity]struct{}, identities []gitlab.RepositoryIdentity) bool {
	for _, identity := range identities {
		if _, duplicate := existing[identity]; duplicate {
			return true
		}
	}
	return false
}

func extraInfoField(information task.ExtraInfo, key string) string {
	for _, field := range information.Fields {
		if field.Key == key {
			return strings.TrimSpace(field.Value)
		}
	}
	return ""
}
