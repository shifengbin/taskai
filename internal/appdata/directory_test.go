package appdata

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"taskai/internal/settings"
	"taskai/internal/storage"
	"taskai/internal/task"
)

func TestResolveDefaultDirectoryUsesHomeTaskAIDirectoryOnMacOS(t *testing.T) {
	homeDirectory := t.TempDir()
	configurationDirectory := filepath.Join(t.TempDir(), "Library", "Application Support")

	got := resolveDefaultDirectory("darwin", directoryDependencies{
		userHomeDir:   func() (string, error) { return homeDirectory, nil },
		userConfigDir: func() (string, error) { return configurationDirectory, nil },
		tempDir:       t.TempDir,
	})
	want := filepath.Join(homeDirectory, ".taskai")
	if got != want {
		t.Fatalf("macOS 默认数据目录 = %q，期望 %q", got, want)
	}
	if workspaceRoot := settings.Default(got).WorkspaceRoot; workspaceRoot != filepath.Join(want, "workspaces") {
		t.Fatalf("macOS 默认工作区根目录 = %q，期望 %q", workspaceRoot, filepath.Join(want, "workspaces"))
	}
}

func TestResolveDefaultDirectoryKeepsUserConfigDirectoryOnOtherPlatforms(t *testing.T) {
	for _, operatingSystem := range []string{"linux", "windows"} {
		t.Run(operatingSystem, func(t *testing.T) {
			configurationDirectory := t.TempDir()
			got := resolveDefaultDirectory(operatingSystem, directoryDependencies{
				userHomeDir:   func() (string, error) { return t.TempDir(), nil },
				userConfigDir: func() (string, error) { return configurationDirectory, nil },
				tempDir:       t.TempDir,
			})
			want := filepath.Join(configurationDirectory, "taskai")
			if got != want {
				t.Fatalf("%s 默认数据目录 = %q，期望 %q", operatingSystem, got, want)
			}
		})
	}
}

func TestResolveDefaultDirectoryFallsBackWhenMacOSHomeIsUnavailable(t *testing.T) {
	configurationDirectory := t.TempDir()
	got := resolveDefaultDirectory("darwin", directoryDependencies{
		userHomeDir:   func() (string, error) { return "", errors.New("home unavailable") },
		userConfigDir: func() (string, error) { return configurationDirectory, nil },
		tempDir:       t.TempDir,
	})
	want := filepath.Join(configurationDirectory, "taskai")
	if got != want {
		t.Fatalf("Home 不可用时数据目录 = %q，期望 %q", got, want)
	}
}

func TestResolveDefaultDirectoryFallsBackWhenMacOSHomeDirectoryIsNotWritable(t *testing.T) {
	homeDirectory := t.TempDir()
	configurationDirectory := filepath.Join(t.TempDir(), "Library", "Application Support")
	newDirectory := filepath.Join(homeDirectory, ".taskai")
	got := resolveDefaultDirectory("darwin", directoryDependencies{
		userHomeDir:   func() (string, error) { return homeDirectory, nil },
		userConfigDir: func() (string, error) { return configurationDirectory, nil },
		tempDir:       t.TempDir,
		ensureDirectory: func(path string) error {
			if path == newDirectory {
				return errors.New("home is read only")
			}
			return os.MkdirAll(path, 0o700)
		},
	})
	want := filepath.Join(configurationDirectory, "taskai")
	if got != want {
		t.Fatalf("Home 不可写时数据目录 = %q，期望 %q", got, want)
	}
}

func TestResolveDefaultDirectoryFallsBackWhenExistingMacOSDirectoryIsNotWritable(t *testing.T) {
	homeDirectory := t.TempDir()
	configurationDirectory := filepath.Join(t.TempDir(), "Library", "Application Support")
	newDirectory := filepath.Join(homeDirectory, ".taskai")
	if err := os.Mkdir(newDirectory, 0o700); err != nil {
		t.Fatalf("创建新配置目录: %v", err)
	}
	got := resolveDefaultDirectory("darwin", directoryDependencies{
		userHomeDir:   func() (string, error) { return homeDirectory, nil },
		userConfigDir: func() (string, error) { return configurationDirectory, nil },
		tempDir:       t.TempDir,
		ensureDirectory: func(path string) error {
			if path == newDirectory {
				return errors.New("existing directory is read only")
			}
			return os.MkdirAll(path, 0o700)
		},
	})
	want := filepath.Join(configurationDirectory, "taskai")
	if got != want {
		t.Fatalf("现有新目录不可写时数据目录 = %q，期望 %q", got, want)
	}
}

func TestResolveDefaultDirectoryMigratesMacOSDataAndOnlyUpdatesOldDefaultWorkspace(t *testing.T) {
	homeDirectory := t.TempDir()
	oldDirectory := filepath.Join(homeDirectory, "Library", "Application Support", "taskai")
	newDirectory := filepath.Join(homeDirectory, ".taskai")
	oldWorkspaceRoot := filepath.Join(oldDirectory, "workspaces")
	existingTask := newTestTask(t, "保留旧任务目录")
	existingTask.WorkspaceRoot = oldWorkspaceRoot
	existingTask.WorkspacePath = filepath.Join(oldWorkspaceRoot, existingTask.ID)
	writeData(t, oldDirectory, oldWorkspaceRoot, existingTask)
	if err := os.MkdirAll(existingTask.WorkspacePath, 0o700); err != nil {
		t.Fatalf("创建旧任务工作目录: %v", err)
	}
	workspaceMarker := filepath.Join(existingTask.WorkspacePath, "preserved.txt")
	if err := os.WriteFile(workspaceMarker, []byte("保留"), 0o600); err != nil {
		t.Fatalf("创建旧工作区标记: %v", err)
	}
	originalContents := readFile(t, filepath.Join(oldDirectory, "tasks.json"))

	got := resolveDefaultDirectory("darwin", realDirectoryDependencies(homeDirectory))
	if got != newDirectory {
		t.Fatalf("迁移后的数据目录 = %q，期望 %q", got, newDirectory)
	}

	migrated := loadData(t, newDirectory)
	if migrated.Settings.WorkspaceRoot != filepath.Join(newDirectory, "workspaces") {
		t.Fatalf("迁移后的工作区根目录 = %q，期望 %q", migrated.Settings.WorkspaceRoot, filepath.Join(newDirectory, "workspaces"))
	}
	if len(migrated.Tasks) != 1 || migrated.Tasks[0].WorkspaceRoot != existingTask.WorkspaceRoot || migrated.Tasks[0].WorkspacePath != existingTask.WorkspacePath {
		t.Fatalf("迁移后的任务快照 = %#v，期望保留 %#v", migrated.Tasks, existingTask)
	}
	if gotContents := readFile(t, filepath.Join(oldDirectory, "tasks.json")); string(gotContents) != string(originalContents) {
		t.Fatal("迁移不得修改旧配置文件")
	}
	if gotMarker := readFile(t, workspaceMarker); string(gotMarker) != "保留" {
		t.Fatalf("迁移后的旧工作区标记 = %q，期望保留", gotMarker)
	}
}

func TestResolveDefaultDirectoryPreservesCustomWorkspaceDuringMigration(t *testing.T) {
	homeDirectory := t.TempDir()
	oldDirectory := filepath.Join(homeDirectory, "Library", "Application Support", "taskai")
	customWorkspaceRoot := filepath.Join(t.TempDir(), "custom-workspaces")
	writeData(t, oldDirectory, customWorkspaceRoot)

	newDirectory := resolveDefaultDirectory("darwin", realDirectoryDependencies(homeDirectory))
	migrated := loadData(t, newDirectory)
	if migrated.Settings.WorkspaceRoot != customWorkspaceRoot {
		t.Fatalf("迁移后的自定义工作区 = %q，期望 %q", migrated.Settings.WorkspaceRoot, customWorkspaceRoot)
	}
}

func TestResolveDefaultDirectoryPrefersExistingMacOSDirectoryWithoutMerging(t *testing.T) {
	homeDirectory := t.TempDir()
	oldDirectory := filepath.Join(homeDirectory, "Library", "Application Support", "taskai")
	newDirectory := filepath.Join(homeDirectory, ".taskai")
	writeData(t, oldDirectory, filepath.Join(oldDirectory, "workspaces"), newTestTask(t, "旧配置任务"))
	writeData(t, newDirectory, filepath.Join(newDirectory, "workspaces"), newTestTask(t, "新配置任务"))
	oldContents := readFile(t, filepath.Join(oldDirectory, "tasks.json"))
	newContents := readFile(t, filepath.Join(newDirectory, "tasks.json"))

	got := resolveDefaultDirectory("darwin", realDirectoryDependencies(homeDirectory))
	if got != newDirectory {
		t.Fatalf("冲突时数据目录 = %q，期望 %q", got, newDirectory)
	}
	if string(readFile(t, filepath.Join(oldDirectory, "tasks.json"))) != string(oldContents) {
		t.Fatal("冲突时不得修改旧配置")
	}
	if string(readFile(t, filepath.Join(newDirectory, "tasks.json"))) != string(newContents) {
		t.Fatal("冲突时不得修改新配置")
	}
}

func TestResolveDefaultDirectoryFallsBackToOldDataWhenNewMacOSPathIsNotDirectory(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		create func(*testing.T, string)
	}{
		{name: "普通文件", create: func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("occupied"), 0o600); err != nil {
				t.Fatalf("创建占位文件: %v", err)
			}
		}},
		{name: "失效符号链接", create: func(t *testing.T, path string) {
			if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), path); err != nil {
				t.Fatalf("创建失效符号链接: %v", err)
			}
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			homeDirectory := t.TempDir()
			oldDirectory := filepath.Join(homeDirectory, "Library", "Application Support", "taskai")
			newDirectory := filepath.Join(homeDirectory, ".taskai")
			writeData(t, oldDirectory, filepath.Join(oldDirectory, "workspaces"))
			testCase.create(t, newDirectory)

			got := resolveDefaultDirectory("darwin", realDirectoryDependencies(homeDirectory))
			if got != oldDirectory {
				t.Fatalf("新路径不是目录时数据目录 = %q，期望 %q", got, oldDirectory)
			}
			if _, err := os.Lstat(newDirectory); err != nil {
				t.Fatalf("不得修改新路径占位节点: %v", err)
			}
		})
	}
}

func TestResolveDefaultDirectoryAcceptsSymlinkToExistingMacOSDirectory(t *testing.T) {
	homeDirectory := t.TempDir()
	newDirectory := filepath.Join(homeDirectory, ".taskai")
	targetDirectory := t.TempDir()
	if err := os.Symlink(targetDirectory, newDirectory); err != nil {
		t.Fatalf("创建目录符号链接: %v", err)
	}

	got := resolveDefaultDirectory("darwin", realDirectoryDependencies(homeDirectory))
	if got != newDirectory {
		t.Fatalf("目录符号链接的数据目录 = %q，期望 %q", got, newDirectory)
	}
}

func TestResolveDefaultDirectoryUsesNewDataWhenConcurrentMigrationPublishesFirst(t *testing.T) {
	homeDirectory := t.TempDir()
	oldDirectory := filepath.Join(homeDirectory, "Library", "Application Support", "taskai")
	newDirectory := filepath.Join(homeDirectory, ".taskai")
	writeData(t, oldDirectory, filepath.Join(oldDirectory, "workspaces"))

	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	dependencies := realDirectoryDependencies(homeDirectory)
	dependencies.rename = func(source, destination string) error {
		ready <- struct{}{}
		<-release
		return publishDirectory(source, destination)
	}
	results := make(chan string, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			results <- resolveDefaultDirectory("darwin", dependencies)
		}()
	}
	<-ready
	<-ready
	close(release)
	waitGroup.Wait()
	close(results)

	for got := range results {
		if got != newDirectory {
			t.Fatalf("并发迁移后的数据目录 = %q，期望 %q", got, newDirectory)
		}
	}
}

func TestResolveDefaultDirectoryFallsBackToOldMacOSDataWhenMigrationFails(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		configure func(*directoryDependencies)
	}{
		{name: "复制失败", configure: func(dependencies *directoryDependencies) {
			dependencies.copyFile = func(string, string) error { return errors.New("copy failed") }
		}},
		{name: "保存失败", configure: func(dependencies *directoryDependencies) {
			dependencies.rewriteWorkspaceRoot = func(string, string, string) error { return errors.New("save failed") }
		}},
		{name: "同步文件失败", configure: func(dependencies *directoryDependencies) {
			dependencies.syncFile = func(string) error { return errors.New("file sync failed") }
		}},
		{name: "同步目录失败", configure: func(dependencies *directoryDependencies) {
			dependencies.syncDirectory = func(string) error { return errors.New("directory sync failed") }
		}},
		{name: "发布失败", configure: func(dependencies *directoryDependencies) {
			dependencies.rename = func(string, string) error { return errors.New("rename failed") }
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			homeDirectory := t.TempDir()
			oldDirectory := filepath.Join(homeDirectory, "Library", "Application Support", "taskai")
			newDirectory := filepath.Join(homeDirectory, ".taskai")
			writeData(t, oldDirectory, filepath.Join(oldDirectory, "workspaces"))
			originalContents := readFile(t, filepath.Join(oldDirectory, "tasks.json"))
			dependencies := realDirectoryDependencies(homeDirectory)
			testCase.configure(&dependencies)

			got := resolveDefaultDirectory("darwin", dependencies)
			if got != oldDirectory {
				t.Fatalf("迁移失败后的数据目录 = %q，期望回退到 %q", got, oldDirectory)
			}
			if _, err := os.Stat(newDirectory); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("迁移失败后的新目录状态 = %v，期望不存在", err)
			}
			matches, err := filepath.Glob(filepath.Join(homeDirectory, ".taskai-migration-*"))
			if err != nil || len(matches) != 0 {
				t.Fatalf("迁移失败后残留临时目录 = %v，错误 = %v", matches, err)
			}
			if string(readFile(t, filepath.Join(oldDirectory, "tasks.json"))) != string(originalContents) {
				t.Fatal("迁移失败不得修改旧配置")
			}
		})
	}
}

func TestPublishDirectoryDoesNotReplaceExistingDestination(t *testing.T) {
	parentDirectory := t.TempDir()
	sourceDirectory := filepath.Join(parentDirectory, "source")
	destinationDirectory := filepath.Join(parentDirectory, "destination")
	if err := os.Mkdir(sourceDirectory, 0o700); err != nil {
		t.Fatalf("创建源目录: %v", err)
	}
	if err := os.Mkdir(destinationDirectory, 0o700); err != nil {
		t.Fatalf("创建目标目录: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDirectory, "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatalf("写入源目录: %v", err)
	}
	if err := publishDirectory(sourceDirectory, destinationDirectory); err == nil {
		t.Fatal("目标目录已存在时发布应失败")
	}
	if got := readFile(t, filepath.Join(sourceDirectory, "source.txt")); string(got) != "source" {
		t.Fatalf("失败后的源文件 = %q，期望保留", got)
	}
	entries, err := os.ReadDir(destinationDirectory)
	if err != nil {
		t.Fatalf("读取目标目录: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("失败后的目标目录内容 = %v，期望为空", entries)
	}
}

func realDirectoryDependencies(homeDirectory string) directoryDependencies {
	return directoryDependencies{
		userHomeDir:   func() (string, error) { return homeDirectory, nil },
		userConfigDir: func() (string, error) { return filepath.Join(homeDirectory, "Library", "Application Support"), nil },
		tempDir:       os.TempDir,
	}
}

func writeData(t *testing.T, dataDirectory, workspaceRoot string, tasks ...task.Task) {
	t.Helper()
	repository := storage.New(filepath.Join(dataDirectory, "tasks.json"), settings.Default(dataDirectory))
	data, err := repository.Load()
	if err != nil {
		t.Fatalf("创建测试配置: %v", err)
	}
	data.Settings.WorkspaceRoot = workspaceRoot
	data.Tasks = append([]task.Task(nil), tasks...)
	if err := repository.Save(data); err != nil {
		t.Fatalf("保存测试配置: %v", err)
	}
}

func newTestTask(t *testing.T, title string) task.Task {
	t.Helper()
	created, err := task.NewTask(title, "", task.DefaultColor, time.Date(2026, time.August, 12, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("创建测试任务: %v", err)
	}
	return created
}

func loadData(t *testing.T, dataDirectory string) storage.Data {
	t.Helper()
	data, err := storage.New(filepath.Join(dataDirectory, "tasks.json"), settings.Default(dataDirectory)).Load()
	if err != nil {
		t.Fatalf("加载测试配置: %v", err)
	}
	return data
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	return contents
}
