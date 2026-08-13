package appdata

import (
	"errors"
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
	ensureDirectory      func(string) error
	copyFile             func(string, string) error
	rewriteWorkspaceRoot func(string, string, string) error
	syncFile             func(string) error
	syncDirectory        func(string) error
	rename               func(string, string) error
}

func DefaultDirectory() string {
	return resolveDefaultDirectory(runtime.GOOS, directoryDependencies{
		userHomeDir:          os.UserHomeDir,
		userConfigDir:        os.UserConfigDir,
		tempDir:              os.TempDir,
		ensureDirectory:      ensureWritableDirectory,
		copyFile:             copyFile,
		rewriteWorkspaceRoot: rewriteWorkspaceRoot,
		syncFile:             syncFilePath,
		syncDirectory:        syncDirectoryPath,
		rename:               publishDirectory,
	})
}

// CoordinationDirectory is stable across the macOS data-directory migration.
// Holding its instance lock serializes migration before the final directory is chosen.
func CoordinationDirectory() string {
	configurationDirectory, err := os.UserConfigDir()
	if err == nil && configurationDirectory != "" {
		return filepath.Join(configurationDirectory, "taskai")
	}
	return filepath.Join(os.TempDir(), "taskai")
}

func resolveDefaultDirectory(operatingSystem string, dependencies directoryDependencies) string {
	if dependencies.userConfigDir == nil {
		dependencies.userConfigDir = os.UserConfigDir
	}
	if dependencies.tempDir == nil {
		dependencies.tempDir = os.TempDir
	}
	if dependencies.ensureDirectory == nil {
		dependencies.ensureDirectory = ensureWritableDirectory
	}
	if dependencies.copyFile == nil {
		dependencies.copyFile = copyFile
	}
	if dependencies.rewriteWorkspaceRoot == nil {
		dependencies.rewriteWorkspaceRoot = rewriteWorkspaceRoot
	}
	if dependencies.syncFile == nil {
		dependencies.syncFile = syncFilePath
	}
	if dependencies.syncDirectory == nil {
		dependencies.syncDirectory = syncDirectoryPath
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
	newDirectoryState := directoryState(newDirectory)
	oldDirectory := ""
	oldDataExists := false
	if configurationErr == nil {
		oldDirectory = filepath.Join(configurationDirectory, "taskai")
		oldDataExists = regularFileExists(filepath.Join(oldDirectory, "tasks.json"))
	}
	if newDirectoryState == directoryUsable {
		if err := dependencies.ensureDirectory(newDirectory); err == nil {
			return newDirectory
		}
		if oldDataExists {
			return oldDirectory
		}
		return fallbackDirectory(configurationDirectory, configurationErr, dependencies)
	}
	if newDirectoryState == directoryOccupied {
		if oldDataExists {
			return oldDirectory
		}
		return fallbackDirectory(configurationDirectory, configurationErr, dependencies)
	}
	if configurationErr != nil {
		if err := dependencies.ensureDirectory(newDirectory); err == nil {
			return newDirectory
		}
		return fallbackDirectory(configurationDirectory, configurationErr, dependencies)
	}
	if !oldDataExists {
		if err := dependencies.ensureDirectory(newDirectory); err == nil {
			return newDirectory
		}
		return fallbackDirectory(configurationDirectory, configurationErr, dependencies)
	}
	if err := migrateDirectory(oldDirectory, newDirectory, dependencies); err != nil {
		if directoryState(newDirectory) == directoryUsable && dependencies.ensureDirectory(newDirectory) == nil {
			return newDirectory
		}
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
	if err := dependencies.syncFile(temporaryDataPath); err != nil {
		return fmt.Errorf("sync migrated configuration: %w", err)
	}
	if err := dependencies.syncDirectory(temporaryDirectory); err != nil {
		return fmt.Errorf("sync migration directory: %w", err)
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
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

type dataDirectoryState int

const (
	directoryMissing dataDirectoryState = iota
	directoryUsable
	directoryOccupied
)

func directoryState(path string) dataDirectoryState {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return directoryUsable
		}
		return directoryOccupied
	}
	if _, linkErr := os.Lstat(path); linkErr == nil {
		return directoryOccupied
	}
	return directoryMissing
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func ensureWritableDirectory(path string) error {
	created := false
	if err := os.Mkdir(path, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) || directoryState(path) != directoryUsable {
			return err
		}
	} else {
		created = true
	}
	probe, err := os.CreateTemp(path, ".taskai-write-check-*")
	if err != nil {
		if created {
			os.Remove(path)
		}
		return err
	}
	probePath := probe.Name()
	if err := probe.Close(); err != nil {
		os.Remove(probePath)
		if created {
			os.Remove(path)
		}
		return err
	}
	if err := os.Remove(probePath); err != nil {
		if created {
			os.Remove(path)
		}
		return err
	}
	return nil
}

func fallbackDirectory(configurationDirectory string, configurationErr error, dependencies directoryDependencies) string {
	if configurationErr == nil {
		fallback := filepath.Join(configurationDirectory, "taskai")
		if dependencies.ensureDirectory(fallback) == nil {
			return fallback
		}
	}
	fallback := filepath.Join(dependencies.tempDir(), "taskai")
	_ = dependencies.ensureDirectory(fallback)
	return fallback
}

func syncFilePath(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func syncDirectoryPath(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
