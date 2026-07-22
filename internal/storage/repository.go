package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"taskai/internal/settings"
	"taskai/internal/task"
)

type Data struct {
	Tasks    []task.Task       `json:"tasks"`
	Settings settings.Settings `json:"settings"`
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
		return Data{Tasks: []task.Task{}, Settings: repository.defaults}, nil
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
