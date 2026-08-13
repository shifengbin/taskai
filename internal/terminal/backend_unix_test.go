//go:build !windows

package terminal

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnixBackendStartsShellInRequestedDirectory(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	backend := &unixBackend{shell: "/bin/sh"}
	session, err := backend.Start(StartRequest{TaskID: "task-a", Directory: directory, Columns: 80, Rows: 24})
	if err != nil {
		t.Fatalf("启动 PTY: %v", err)
	}
	defer session.Close()
	if _, err := session.Write([]byte("pwd\nexit\n")); err != nil {
		t.Fatalf("写入 shell: %v", err)
	}

	output, readErr := io.ReadAll(session)
	if readErr != nil && !strings.Contains(readErr.Error(), "input/output error") {
		t.Fatalf("读取 PTY 输出: %v", readErr)
	}
	if _, err := session.Wait(); err != nil {
		t.Fatalf("等待 shell 退出: %v", err)
	}
	if !strings.Contains(string(output), directory) {
		t.Fatalf("PTY 启动目录错误，输出: %q", output)
	}
}

func TestUnixBackendReportsProcessExitCode(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		command  string
		exitCode int
	}{
		{name: "正常退出", command: "exit 0", exitCode: 0},
		{name: "非零退出", command: "exit 7", exitCode: 7},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			session, err := (&unixBackend{}).Start(StartRequest{
				TaskID: "task-a", Directory: t.TempDir(), Command: "/bin/sh", Arguments: []string{"-c", testCase.command}, Columns: 80, Rows: 24,
			})
			if err != nil {
				t.Fatalf("启动终端: %v", err)
			}
			defer session.Close()

			_, _ = io.ReadAll(session)
			exitResult, err := session.Wait()
			if err != nil {
				t.Fatalf("等待终端退出: %v", err)
			}
			if exitResult.ExitCode == nil || *exitResult.ExitCode != testCase.exitCode {
				t.Fatalf("终端退出码 = %v，期望 %d", exitResult.ExitCode, testCase.exitCode)
			}
		})
	}
}

func TestUnixBackendSetsTerminalTypeForEmbeddedXterm(t *testing.T) {
	t.Setenv("TERM", "dumb")

	backend := &unixBackend{shell: "/bin/sh"}
	session, err := backend.Start(StartRequest{TaskID: "task-a", Directory: t.TempDir(), Columns: 80, Rows: 24})
	if err != nil {
		t.Fatalf("启动 PTY: %v", err)
	}
	defer session.Close()
	if _, err := session.Write([]byte("printf '__TERM__%s__\\n' \"$TERM\"\nexit\n")); err != nil {
		t.Fatalf("写入 shell: %v", err)
	}

	output, readErr := io.ReadAll(session)
	if readErr != nil && !strings.Contains(readErr.Error(), "input/output error") {
		t.Fatalf("读取 PTY 输出: %v", readErr)
	}
	if _, err := session.Wait(); err != nil {
		t.Fatalf("等待 shell 退出: %v", err)
	}
	if !strings.Contains(string(output), "__TERM__xterm-256color__") {
		t.Fatalf("嵌入式终端类型错误，输出: %q", output)
	}
}

func TestUnixBackendPassesRequestedEnvironmentToTerminalProcess(t *testing.T) {
	backend := &unixBackend{shell: "/bin/sh"}
	session, err := backend.Start(StartRequest{
		ID:          "terminal-environment",
		TaskID:      "task-a",
		Directory:   t.TempDir(),
		Environment: []string{"TASKAI_STATUS_API=http://127.0.0.1:18765/api/v1", "TASKAI_TERMINAL_ID=terminal-environment"},
		Columns:     80,
		Rows:        24,
	})
	if err != nil {
		t.Fatalf("启动 PTY: %v", err)
	}
	defer session.Close()
	if _, err := session.Write([]byte("printf '__STATUS_API__%s__ __TERMINAL_ID__%s__\\n' \"$TASKAI_STATUS_API\" \"$TASKAI_TERMINAL_ID\"\nexit\n")); err != nil {
		t.Fatalf("写入 shell: %v", err)
	}

	output, readErr := io.ReadAll(session)
	if readErr != nil && !strings.Contains(readErr.Error(), "input/output error") {
		t.Fatalf("读取 PTY 输出: %v", readErr)
	}
	if _, err := session.Wait(); err != nil {
		t.Fatalf("等待 shell 退出: %v", err)
	}
	if !strings.Contains(string(output), "__STATUS_API__http://127.0.0.1:18765/api/v1__ __TERMINAL_ID__terminal-environment__") {
		t.Fatalf("终端环境变量错误，输出: %q", output)
	}
}

func TestUnixBackendUsesShellFromStartRequest(t *testing.T) {
	shellPath := filepath.Join(t.TempDir(), "configured-shell")
	if err := os.WriteFile(shellPath, []byte("#!/bin/sh\nprintf '__CONFIGURED_SHELL__\\n'\nread ignored\n"), 0o700); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	contents, err := json.Marshal(map[string]any{
		"TaskID":    "task-a",
		"Directory": t.TempDir(),
		"Columns":   80,
		"Rows":      24,
		"ShellPath": shellPath,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var request StartRequest
	if err := json.Unmarshal(contents, &request); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	session, err := (&unixBackend{shell: "/bin/sh"}).Start(request)
	if err != nil {
		t.Fatalf("启动 PTY: %v", err)
	}
	defer session.Close()
	if _, err := session.Write([]byte("exit\n")); err != nil {
		t.Fatalf("写入 Shell: %v", err)
	}

	output, readErr := io.ReadAll(session)
	if readErr != nil && !strings.Contains(readErr.Error(), "input/output error") {
		t.Fatalf("读取 PTY 输出: %v", readErr)
	}
	if _, err := session.Wait(); err != nil {
		t.Fatalf("等待 Shell 退出: %v", err)
	}
	if !strings.Contains(string(output), "__CONFIGURED_SHELL__") {
		t.Fatalf("未使用请求指定的 Shell，输出: %q", output)
	}
}

func TestUnixBackendStartsConfiguredCommandWithArguments(t *testing.T) {
	commandPath := filepath.Join(t.TempDir(), "task-command")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nprintf '__TASK_COMMAND__%s__\\n' \"$1\"\n"), 0o700); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	session, err := (&unixBackend{}).Start(StartRequest{
		TaskID: "task-a", Directory: t.TempDir(), Command: commandPath, Arguments: []string{"--full-auto"}, Columns: 80, Rows: 24,
	})
	if err != nil {
		t.Fatalf("启动配置命令: %v", err)
	}
	defer session.Close()

	output, readErr := io.ReadAll(session)
	if readErr != nil && !strings.Contains(readErr.Error(), "input/output error") {
		t.Fatalf("读取配置命令输出: %v", readErr)
	}
	if _, err := session.Wait(); err != nil {
		t.Fatalf("等待配置命令: %v", err)
	}
	if !strings.Contains(string(output), "__TASK_COMMAND__--full-auto__") {
		t.Fatalf("配置命令输出 = %q", output)
	}
}

func TestUnixBackendStartsDetectedAgentCommandsWithExactArgumentsAndDirectory(t *testing.T) {
	for _, current := range []struct {
		name     string
		argument string
	}{
		{name: "codex", argument: "--yolo"},
		{name: "claude", argument: "--dangerously-skip-permissions"},
	} {
		t.Run(current.name, func(t *testing.T) {
			directory := t.TempDir()
			commandPath := filepath.Join(directory, current.name)
			if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nprintf '__AGENT__%s__ARG__%s__PWD__%s__\\n' \"$0\" \"$1\" \"$PWD\"\n"), 0o700); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			session, err := (&unixBackend{}).Start(StartRequest{
				TaskID: "task-a", Directory: directory, ShellPath: "/bin/sh", Command: commandPath,
				Arguments: []string{current.argument}, Columns: 80, Rows: 24,
			})
			if err != nil {
				t.Fatalf("启动 %s: %v", current.name, err)
			}
			defer session.Close()

			output, readErr := io.ReadAll(session)
			if readErr != nil && !strings.Contains(readErr.Error(), "input/output error") {
				t.Fatalf("读取 %s 输出: %v", current.name, readErr)
			}
			if _, err := session.Wait(); err != nil {
				t.Fatalf("等待 %s: %v", current.name, err)
			}
			want := "__AGENT__" + commandPath + "__ARG__" + current.argument + "__PWD__" + directory + "__"
			if !strings.Contains(string(output), want) {
				t.Fatalf("%s 输出 = %q，期望包含 %q", current.name, output, want)
			}
		})
	}
}

func TestUnixBackendStartsConfiguredCommandThroughRequestedShell(t *testing.T) {
	directory := t.TempDir()
	shellPath := filepath.Join(directory, "configured-shell")
	commandPath := filepath.Join(directory, "task-command")
	if err := os.WriteFile(shellPath, []byte("#!/bin/sh\nexport TASKAI_SHELL_INITIALIZED=1\nshift 3\nexec \"$@\"\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(shell) error = %v", err)
	}
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nprintf '__SHELL_PATH__%s__\\n' \"$TASKAI_SHELL_INITIALIZED\"\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(command) error = %v", err)
	}

	session, err := (&unixBackend{}).Start(StartRequest{
		TaskID: "task-a", Directory: directory, ShellPath: shellPath, Command: commandPath, Columns: 80, Rows: 24,
	})
	if err != nil {
		t.Fatalf("启动配置命令: %v", err)
	}
	defer session.Close()

	output, readErr := io.ReadAll(session)
	if readErr != nil && !strings.Contains(readErr.Error(), "input/output error") {
		t.Fatalf("读取配置命令输出: %v", readErr)
	}
	if _, err := session.Wait(); err != nil {
		t.Fatalf("等待配置命令: %v", err)
	}
	if !strings.Contains(string(output), "__SHELL_PATH__1__") {
		t.Fatalf("配置命令未通过请求指定的 Shell 启动，输出: %q", output)
	}
}

func TestUnixBackendUsesFishInvocationForConfiguredCommand(t *testing.T) {
	directory := t.TempDir()
	shellPath := filepath.Join(directory, "fish")
	if err := os.WriteFile(shellPath, []byte("#!/bin/sh\nprintf '__FISH_ARGS__%s__%s__%s__%s__%s__\\n' \"$1\" \"$2\" \"$3\" \"$4\" \"$5\"\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(fish) error = %v", err)
	}

	session, err := (&unixBackend{}).Start(StartRequest{
		TaskID: "task-a", Directory: directory, ShellPath: shellPath, Command: "codex", Arguments: []string{"--yolo"}, Columns: 80, Rows: 24,
	})
	if err != nil {
		t.Fatalf("启动 fish 配置命令: %v", err)
	}
	defer session.Close()

	output, readErr := io.ReadAll(session)
	if readErr != nil && !strings.Contains(readErr.Error(), "input/output error") {
		t.Fatalf("读取 fish 配置命令输出: %v", readErr)
	}
	if _, err := session.Wait(); err != nil {
		t.Fatalf("等待 fish 配置命令: %v", err)
	}
	want := "__FISH_ARGS__-ic__exec $argv[2..-1]__" + shellPath + "__codex__--yolo__"
	if !strings.Contains(string(output), want) {
		t.Fatalf("fish 启动参数 = %q，期望包含 %q", output, want)
	}
}
