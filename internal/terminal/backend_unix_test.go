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
	if err := session.Wait(); err != nil {
		t.Fatalf("等待 shell 退出: %v", err)
	}
	if !strings.Contains(string(output), directory) {
		t.Fatalf("PTY 启动目录错误，输出: %q", output)
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
	if err := session.Wait(); err != nil {
		t.Fatalf("等待 shell 退出: %v", err)
	}
	if !strings.Contains(string(output), "__TERM__xterm-256color__") {
		t.Fatalf("嵌入式终端类型错误，输出: %q", output)
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
	if err := session.Wait(); err != nil {
		t.Fatalf("等待 Shell 退出: %v", err)
	}
	if !strings.Contains(string(output), "__CONFIGURED_SHELL__") {
		t.Fatalf("未使用请求指定的 Shell，输出: %q", output)
	}
}
