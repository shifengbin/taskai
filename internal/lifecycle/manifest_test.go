package lifecycle

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"taskai/internal/settings"
	"taskai/internal/task"
)

func TestCommandChainRunnerWritesManifestFile(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	input := []byte(`{"taskId":"task-1"}`)
	runner := NewCommandChainRunner(nil)

	output, err := runner.Run(CommandChainRequest{
		Task: task.Task{
			ID:          "task-1",
			Title:       "Android 2.45",
			Description: "发布准备",
			ExtraInfo: []task.TaskExtraInfo{
				builtInGitTaskInfo("istudy-v2", "git@gitlab.jiandan100.cn:webdev/istudy-v2.git", ""),
				builtInGitTaskInfo("ucv2", "git@gitlab.jiandan100.cn:webdev/ucv2.git", "dev-cj-1.2"),
			},
		},
		TemplateFields: map[string]any{"branch": "android2.45-0727"},
		WorkspacePath:  workspacePath,
		Input:          input,
		Commands: []settings.LifecycleCommand{
			{ID: settings.LifecycleCommandUpdateDefaultBranchID, Kind: settings.LifecycleCommandKindUpdateDefaultBranch, Name: "更新默认分支"},
			{ID: settings.LifecycleCommandManifestFileID, Kind: settings.LifecycleCommandKindManifestFile, Name: "生成清单文件", Arguments: []string{"dir=configs/task", "name=iteration.yaml"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(output) != string(input) {
		t.Fatalf("清单文件命令输出 = %q，期望原样透传 %q", output, input)
	}
	contents, err := os.ReadFile(filepath.Join(workspacePath, "configs", "task", "iteration.yaml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	const want = "iteration: Android 2.45\ndesc: 发布准备\nrepos:\n    - name: istudy-v2\n      url: git@gitlab.jiandan100.cn:webdev/istudy-v2.git\n      branch: android2.45-0727\n    - name: ucv2\n      url: git@gitlab.jiandan100.cn:webdev/ucv2.git\n      branch: dev-cj-1.2\n"
	if got := string(contents); got != want {
		t.Fatalf("清单文件内容 = %q，期望 %q", got, want)
	}
}

func TestCommandChainRunnerWritesEmptyManifestFileByDefault(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	runner := NewCommandChainRunner(nil)

	_, err := runner.Run(CommandChainRequest{
		Task:          task.Task{ID: "task-1", Title: "无仓库任务", Description: ""},
		WorkspacePath: workspacePath,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandManifestFileID, Kind: settings.LifecycleCommandKindManifestFile, Name: "生成清单文件",
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	manifest := decodeManifestFile(t, filepath.Join(workspacePath, "manifest.yaml"))
	if manifest.Iteration != "无仓库任务" || manifest.Description != "" || len(manifest.Repositories) != 0 {
		t.Fatalf("默认清单 = %#v", manifest)
	}
}

func TestCommandChainRunnerWritesEmptyGitBranch(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	runner := NewCommandChainRunner(nil)

	_, err := runner.Run(CommandChainRequest{
		Task: task.Task{
			ID: "task-1", Title: "空分支任务", ExtraInfo: []task.TaskExtraInfo{
				builtInGitTaskInfo("istudy-v2", "git@gitlab.jiandan100.cn:webdev/istudy-v2.git", ""),
			},
		},
		WorkspacePath: workspacePath,
		Commands: []settings.LifecycleCommand{{
			ID: settings.LifecycleCommandManifestFileID, Kind: settings.LifecycleCommandKindManifestFile, Name: "生成清单文件",
		}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	manifest := decodeManifestFile(t, filepath.Join(workspacePath, "manifest.yaml"))
	if len(manifest.Repositories) != 1 || manifest.Repositories[0].Branch != "" {
		t.Fatalf("空分支清单 = %#v", manifest.Repositories)
	}
	contents, err := os.ReadFile(filepath.Join(workspacePath, "manifest.yaml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(contents), "branch: \"\"") {
		t.Fatalf("空分支清单缺少 branch 键: %q", contents)
	}
}

func TestCommandChainRunnerSerializesManifestSpecialCharacters(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	runner := NewCommandChainRunner(nil)
	current := task.Task{
		ID:          "task-1",
		Title:       "Android: 2.45 # 发布",
		Description: "第一行\n第二行: # 保留",
		ExtraInfo:   []task.TaskExtraInfo{builtInGitTaskInfo("API: 服务", "git@example.com:team/api.git", "release:2")},
	}

	_, err := runner.Run(CommandChainRequest{
		Task: current, WorkspacePath: workspacePath,
		Commands: []settings.LifecycleCommand{{ID: settings.LifecycleCommandManifestFileID, Kind: settings.LifecycleCommandKindManifestFile, Name: "生成清单文件"}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	manifest := decodeManifestFile(t, filepath.Join(workspacePath, "manifest.yaml"))
	if manifest.Iteration != current.Title || manifest.Description != current.Description || len(manifest.Repositories) != 1 || manifest.Repositories[0] != (decodedManifestRepository{Name: "API: 服务", URL: "git@example.com:team/api.git", Branch: "release:2"}) {
		t.Fatalf("特殊字符清单 = %#v", manifest)
	}
}

func TestCommandChainRunnerReplacesExistingManifestFile(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "task-1")
	if err := os.MkdirAll(workspacePath, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	path := filepath.Join(workspacePath, "manifest.yaml")
	if err := os.WriteFile(path, []byte("iteration: 旧任务\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runner := NewCommandChainRunner(nil)

	_, err := runner.Run(CommandChainRequest{
		Task:          task.Task{ID: "task-1", Title: "新任务", Description: "已更新"},
		WorkspacePath: workspacePath,
		Commands:      []settings.LifecycleCommand{{ID: settings.LifecycleCommandManifestFileID, Kind: settings.LifecycleCommandKindManifestFile, Name: "生成清单文件"}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	manifest := decodeManifestFile(t, path)
	if manifest.Iteration != "新任务" || manifest.Description != "已更新" {
		t.Fatalf("覆盖后的清单 = %#v", manifest)
	}
}

func TestCommandChainRunnerRejectsUnsafeManifestFileTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 创建符号链接需要额外权限")
	}
	for _, test := range []struct {
		name      string
		arguments []string
		setup     func(workspacePath string) error
	}{
		{name: "任务工作目录不存在"},
		{name: "目录越界", arguments: []string{"dir=../outside"}, setup: func(string) error { return nil }},
		{name: "目录是符号链接", arguments: []string{"dir=config"}, setup: func(workspacePath string) error {
			return os.Symlink(t.TempDir(), filepath.Join(workspacePath, "config"))
		}},
		{name: "目标是符号链接", setup: func(workspacePath string) error {
			outsidePath := filepath.Join(t.TempDir(), "manifest.yaml")
			if err := os.WriteFile(outsidePath, []byte("保留"), 0o600); err != nil {
				return err
			}
			return os.Symlink(outsidePath, filepath.Join(workspacePath, "manifest.yaml"))
		}},
		{name: "目标是目录", setup: func(workspacePath string) error {
			return os.Mkdir(filepath.Join(workspacePath, "manifest.yaml"), 0o700)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspacePath := filepath.Join(t.TempDir(), "task-1")
			if test.name != "任务工作目录不存在" {
				if err := os.MkdirAll(workspacePath, 0o700); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			}
			if test.setup != nil {
				if err := test.setup(workspacePath); err != nil {
					t.Fatalf("setup() error = %v", err)
				}
			}
			runner := NewCommandChainRunner(nil)
			_, err := runner.Run(CommandChainRequest{
				Task:          task.Task{ID: "task-1", Title: "任务"},
				WorkspacePath: workspacePath,
				Commands: []settings.LifecycleCommand{{
					ID: settings.LifecycleCommandManifestFileID, Kind: settings.LifecycleCommandKindManifestFile, Name: "生成清单文件", Arguments: test.arguments,
				}},
			})
			if err == nil {
				t.Fatal("Run() error = nil，期望拒绝不安全清单文件目标")
			}
		})
	}
}

type decodedManifest struct {
	Iteration    string                      `yaml:"iteration"`
	Description  string                      `yaml:"desc"`
	Repositories []decodedManifestRepository `yaml:"repos"`
}

type decodedManifestRepository struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Branch string `yaml:"branch"`
}

func decodeManifestFile(t *testing.T, path string) decodedManifest {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var manifest decodedManifest
	if err := yaml.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", contents, err)
	}
	return manifest
}
