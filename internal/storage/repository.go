package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"taskai/internal/settings"
	"taskai/internal/task"
)

type Data struct {
	Tasks               []task.Task              `json:"tasks"`
	ExtraInfoCatalogues []string                 `json:"extraInfoCatalogues"`
	ExtraInfoTemplates  []task.ExtraInfoTemplate `json:"extraInfoTemplates"`
	Settings            settings.Settings        `json:"settings"`
}

type Repository struct {
	path     string
	defaults settings.Settings
}

func New(path string, defaults settings.Settings) Repository {
	return Repository{path: path, defaults: defaults}
}

func (repository Repository) Load() (Data, error) {
	contents, err := os.ReadFile(repository.path)
	if os.IsNotExist(err) {
		return Data{Tasks: []task.Task{}, ExtraInfoCatalogues: []string{}, ExtraInfoTemplates: []task.ExtraInfoTemplate{}, Settings: repository.defaults}, nil
	}
	if err != nil {
		return Data{}, fmt.Errorf("读取本地数据失败: %w", err)
	}

	var data Data
	if err := json.Unmarshal(contents, &data); err != nil {
		return Data{}, fmt.Errorf("解析本地数据失败: %w", err)
	}
	if data.Tasks == nil {
		data.Tasks = []task.Task{}
	}
	if data.ExtraInfoTemplates == nil {
		data.ExtraInfoTemplates = []task.ExtraInfoTemplate{}
	}
	for index, current := range data.ExtraInfoTemplates {
		normalized, normalizeErr := task.NormalizeExtraInfoTemplate(current)
		if normalizeErr != nil {
			return Data{}, fmt.Errorf("规范化额外信息模板失败: %w", normalizeErr)
		}
		data.ExtraInfoTemplates[index] = normalized
	}
	if data.ExtraInfoCatalogues == nil {
		data.ExtraInfoCatalogues = make([]string, 0, len(data.ExtraInfoTemplates))
		for _, template := range data.ExtraInfoTemplates {
			data.ExtraInfoCatalogues = append(data.ExtraInfoCatalogues, template.Catalogue)
		}
	}
	var catalogueErr error
	data.ExtraInfoCatalogues, catalogueErr = normalizeCatalogues(data.ExtraInfoCatalogues)
	if catalogueErr != nil {
		return Data{}, catalogueErr
	}
	for index := range data.Tasks {
		if data.Tasks[index].ExtraInfo == nil {
			data.Tasks[index].ExtraInfo = []task.ExtraInfo{}
			continue
		}
		normalized, normalizeErr := task.NormalizeExtraInfo(data.Tasks[index].ExtraInfo)
		if normalizeErr != nil {
			return Data{}, fmt.Errorf("规范化任务附加信息失败: %w", normalizeErr)
		}
		data.Tasks[index].ExtraInfo = normalized
	}

	return data, nil
}

func (repository Repository) Save(data Data) error {
	directory := filepath.Dir(repository.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	temporaryFile, err := os.CreateTemp(directory, ".taskai-*.json")
	if err != nil {
		return fmt.Errorf("创建临时数据文件失败: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)

	encoder := json.NewEncoder(temporaryFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("写入临时数据文件失败: %w", err)
	}
	if err := temporaryFile.Chmod(0o600); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("设置临时数据文件权限失败: %w", err)
	}
	if err := temporaryFile.Sync(); err != nil {
		temporaryFile.Close()
		return fmt.Errorf("同步临时数据文件失败: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("关闭临时数据文件失败: %w", err)
	}
	if err := os.Rename(temporaryPath, repository.path); err != nil {
		return fmt.Errorf("替换本地数据文件失败: %w", err)
	}

	return nil
}

func (repository Repository) SaveSettings(next settings.Settings) (settings.Settings, error) {
	data, err := repository.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	data.Settings = next
	if err := repository.Save(data); err != nil {
		return settings.Settings{}, err
	}

	return next, nil
}

func (repository Repository) ListExtraInfoTemplates() ([]task.ExtraInfoTemplate, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return append([]task.ExtraInfoTemplate{}, data.ExtraInfoTemplates...), nil
}

func (repository Repository) ListExtraInfoCatalogues() ([]string, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return append([]string{}, data.ExtraInfoCatalogues...), nil
}

func (repository Repository) SaveExtraInfoCatalogue(name string) (string, error) {
	data, err := repository.Load()
	if err != nil {
		return "", err
	}
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", fmt.Errorf("分类名称不能为空")
	}
	for _, current := range data.ExtraInfoCatalogues {
		if current == normalized {
			return "", fmt.Errorf("额外信息分类已存在: %q", normalized)
		}
	}
	data.ExtraInfoCatalogues = append(data.ExtraInfoCatalogues, normalized)
	if err := repository.Save(data); err != nil {
		return "", err
	}
	return normalized, nil
}

func (repository Repository) DeleteExtraInfoCatalogue(name string) error {
	data, err := repository.Load()
	if err != nil {
		return err
	}
	normalized := strings.TrimSpace(name)
	index := -1
	for candidateIndex, current := range data.ExtraInfoCatalogues {
		if current == normalized {
			index = candidateIndex
			break
		}
	}
	if index == -1 {
		return fmt.Errorf("额外信息分类不存在")
	}
	for _, template := range data.ExtraInfoTemplates {
		if template.Catalogue == normalized {
			return fmt.Errorf("分类 %q 仍包含额外信息，请先删除分类中的全部信息", normalized)
		}
	}
	data.ExtraInfoCatalogues = append(data.ExtraInfoCatalogues[:index], data.ExtraInfoCatalogues[index+1:]...)
	return repository.Save(data)
}

func (repository Repository) SaveExtraInfoTemplate(next task.ExtraInfoTemplate) (task.ExtraInfoTemplate, error) {
	data, err := repository.Load()
	if err != nil {
		return task.ExtraInfoTemplate{}, err
	}
	if next.ID == "" {
		if len(next.Fields) == 0 && (next.Key != "" || next.KeyDisplayName != "" || next.Value != "") {
			next.Fields = []task.ExtraInfoField{{Key: next.Key, DisplayName: next.KeyDisplayName, Value: next.Value}}
		}
		next, err = task.NewExtraInfoTemplate(next.Catalogue, next.DisplayName, next.Fields, next.Parameters)
	} else {
		next, err = task.NormalizeExtraInfoTemplate(next)
	}
	if err != nil {
		return task.ExtraInfoTemplate{}, err
	}
	if !containsCatalogue(data.ExtraInfoCatalogues, next.Catalogue) {
		return task.ExtraInfoTemplate{}, fmt.Errorf("额外信息模板必须选择已创建的分类: %q", next.Catalogue)
	}

	updated := false
	templates := append([]task.ExtraInfoTemplate(nil), data.ExtraInfoTemplates...)
	for index, current := range templates {
		if current.ID == next.ID {
			templates[index] = next
			updated = true
			break
		}
	}
	if !updated {
		templates = append(templates, next)
	}
	templates, err = task.ValidateExtraInfoTemplates(templates)
	if err != nil {
		return task.ExtraInfoTemplate{}, err
	}
	data.ExtraInfoTemplates = templates
	if err := repository.Save(data); err != nil {
		return task.ExtraInfoTemplate{}, err
	}
	return next, nil
}

func containsCatalogue(catalogues []string, target string) bool {
	for _, catalogue := range catalogues {
		if catalogue == target {
			return true
		}
	}
	return false
}

func normalizeCatalogues(catalogues []string) ([]string, error) {
	normalized := make([]string, 0, len(catalogues))
	seen := make(map[string]bool, len(catalogues))
	for _, catalogue := range catalogues {
		catalogue = strings.TrimSpace(catalogue)
		if catalogue == "" {
			return nil, fmt.Errorf("分类名称不能为空")
		}
		if seen[catalogue] {
			continue
		}
		seen[catalogue] = true
		normalized = append(normalized, catalogue)
	}
	return normalized, nil
}

func (repository Repository) DeleteExtraInfoTemplate(templateID string) error {
	data, err := repository.Load()
	if err != nil {
		return err
	}
	for index, current := range data.ExtraInfoTemplates {
		if current.ID != templateID {
			continue
		}
		data.ExtraInfoTemplates = append(data.ExtraInfoTemplates[:index], data.ExtraInfoTemplates[index+1:]...)
		return repository.Save(data)
	}
	return fmt.Errorf("额外信息模板不存在")
}
