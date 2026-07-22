//go:build !windows

package terminal

import (
	"io"
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
