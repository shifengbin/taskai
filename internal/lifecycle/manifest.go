package lifecycle

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"taskai/internal/settings"
	"taskai/internal/task"
)

type manifestFile struct {
	Iteration    string               `yaml:"iteration"`
	Description  string               `yaml:"desc"`
	Repositories []manifestRepository `yaml:"repos"`
}

type manifestRepository struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Branch string `yaml:"branch"`
}

func (runner *CommandChainRunner) writeManifestFile(request CommandChainRequest, arguments []string) error {
	parameters, err := settings.ManifestFileArguments(arguments)
	if err != nil {
		return err
	}
	contents, err := manifestFileContents(request.Task, request.GitCloneRepositoryBranch)
	if err != nil {
		return err
	}
	return writeManifestContents(request.WorkspacePath, parameters.Directory, parameters.Name, contents)
}

func manifestFileContents(current task.Task, templateBranch string) ([]byte, error) {
	manifest := manifestFile{
		Iteration:    current.Title,
		Description:  current.Description,
		Repositories: make([]manifestRepository, 0),
	}
	templateBranch = strings.TrimSpace(templateBranch)
	for _, information := range builtInGitInfos(current.ExtraInfo) {
		name, repository, branch, err := gitInformationValues(information)
		if err != nil {
			return nil, fmt.Errorf("解析清单 Git 项目失败: %w", err)
		}
		if branch == "" {
			branch = templateBranch
		}
		manifest.Repositories = append(manifest.Repositories, manifestRepository{Name: name, URL: repository, Branch: branch})
	}
	contents, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("序列化清单文件失败: %w", err)
	}
	return contents, nil
}
