package appdata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"taskai/internal/settings"
	"taskai/internal/storage"
)

type directoryDependencies struct {
	userHomeDir          func() (string, error)
	userConfigDir        func() (string, error)
	tempDir              func() string
	copyFile             func(string, string) error
	rewriteWorkspaceRoot func(string, string, string) error
	rename               func(string, string) error
}

func DefaultDirectory() string {
	return resolveDefaultDirectory(runtime.GOOS, directoryDependencies{
		userHomeDir:          os.UserHomeDir,
		userConfigDir:        os.UserConfigDir,
		tempDir:              os.TempDir,
		copyFile:             copyFile,
		rewriteWorkspaceRoot: rewriteWorkspaceRoot,
		rename:               publishDirectory,
	})
}

func resolveDefaultDirectory(operatingSystem string, dependencies directoryDependencies) string {
	if dependencies.userConfigDir == nil {
		dependencies.userConfigDir = os.UserConfigDir
	}
	if dependencies.tempDir == nil {
		dependencies.tempDir = os.TempDir
	}
	if dependencies.copyFile == nil {
		dependencies.copyFile = copyFile
	}
	if dependencies.rewriteWorkspaceRoot == nil {
		dependencies.rewriteWorkspaceRoot = rewriteWorkspaceRoot
	}
	if dependencies.rename == nil {
		dependencies.rename = publishDirectory
	}

	configurationDirectory, configurationErr := dependencies.userConfigDir()
	if operatingSystem != "darwin" {
		if configurationErr == nil {
			return filepath.Join(configurationDirectory, "taskai")
		}
		return filepath.Join(dependencies.tempDir(), "taskai")
	}

	if dependencies.userHomeDir == nil {
		dependencies.userHomeDir = os.UserHomeDir
	}
	homeDirectory, homeErr := dependencies.userHomeDir()
	if homeErr != nil || homeDirectory == "" {
		if configurationErr == nil {
			return filepath.Join(configurationDirectory, "taskai")
		}
		return filepath.Join(dependencies.tempDir(), "taskai")
	}

	newDirectory := filepath.Join(homeDirectory, ".taskai")
	if pathExists(newDirectory) {
		return newDirectory
	}
	if configurationErr != nil {
		return newDirectory
	}
	oldDirectory := filepath.Join(configurationDirectory, "taskai")
	if !pathExists(filepath.Join(oldDirectory, "tasks.json")) {
		return newDirectory
	}
	if err := migrateDirectory(oldDirectory, newDirectory, dependencies); err != nil {
		return oldDirectory
	}
	return newDirectory
}

func migrateDirectory(oldDirectory, newDirectory string, dependencies directoryDependencies) error {
	temporaryDirectory, err := os.MkdirTemp(filepath.Dir(newDirectory), ".taskai-migration-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporaryDirectory)

	temporaryDataPath := filepath.Join(temporaryDirectory, "tasks.json")
	if err := dependencies.copyFile(filepath.Join(oldDirectory, "tasks.json"), temporaryDataPath); err != nil {
		return fmt.Errorf("copy configuration: %w", err)
	}
	oldDefaultWorkspace := filepath.Join(oldDirectory, "workspaces")
	newDefaultWorkspace := filepath.Join(newDirectory, "workspaces")
	data, err := loadMigratedData(temporaryDataPath, temporaryDirectory)
	if err != nil {
		return err
	}
	if filepath.Clean(data.Settings.WorkspaceRoot) == filepath.Clean(oldDefaultWorkspace) {
		if err := dependencies.rewriteWorkspaceRoot(temporaryDataPath, oldDefaultWorkspace, newDefaultWorkspace); err != nil {
			return fmt.Errorf("update default workspace: %w", err)
		}
	}
	if err := dependencies.rename(temporaryDirectory, newDirectory); err != nil {
		return fmt.Errorf("publish migrated configuration: %w", err)
	}
	return nil
}

func loadMigratedData(path, dataDirectory string) (storage.Data, error) {
	data, err := storage.New(path, settings.Default(dataDirectory)).Load()
	if err != nil {
		return storage.Data{}, fmt.Errorf("load migrated configuration: %w", err)
	}
	return data, nil
}

func rewriteWorkspaceRoot(path, oldRoot, newRoot string) error {
	data, err := loadMigratedData(path, filepath.Dir(path))
	if err != nil {
		return err
	}
	if filepath.Clean(data.Settings.WorkspaceRoot) != filepath.Clean(oldRoot) {
		return nil
	}
	data.Settings.WorkspaceRoot = newRoot
	repository := storage.New(path, settings.Default(filepath.Dir(path)))
	return repository.Save(data)
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
