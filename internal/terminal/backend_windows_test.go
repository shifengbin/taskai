//go:build windows

package terminal

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWindowsBackendStartsCommandProcessorInRequestedDirectory(t *testing.T) {
	if os.Getenv("WINELOADER") != "" {
		t.Skip("Wine 的 ConPTY 读写不完整；请在原生 Windows 执行该集成测试")
	}
	directory := t.TempDir()
	backend := &windowsBackend{}
	session, err := backend.Start(StartRequest{TaskID: "task-a", Directory: directory, Columns: 80, Rows: 24})
	if err != nil {
		t.Fatalf("启动 Windows ConPTY: %v", err)
	}
	defer session.Close()
	type result struct {
		output     []byte
		readErr    error
		exitResult ExitResult
		waitErr    error
	}
	completed := make(chan result, 1)
	go func() {
		if _, err := session.Write([]byte("cd\r\necho TERMINAL_EXIT_SENTINEL\r\nexit\r\n")); err != nil {
			completed <- result{readErr: err}
			return
		}
		output, readErr := io.ReadAll(session)
		exitResult, waitErr := session.Wait()
		completed <- result{output: output, readErr: readErr, exitResult: exitResult, waitErr: waitErr}
	}()
	var completedResult result
	select {
	case completedResult = <-completed:
	case <-time.After(10 * time.Second):
		_ = session.Close()
		t.Fatal("ConPTY 未在预期时间内退出")
	}
	if completedResult.readErr != nil && !isExpectedTerminalReadError(completedResult.readErr) {
		t.Fatalf("读取 ConPTY 输出: %v", completedResult.readErr)
	}
	if completedResult.waitErr != nil {
		t.Fatalf("等待 cmd 退出: %v", completedResult.waitErr)
	}
	if completedResult.exitResult.ExitCode == nil || *completedResult.exitResult.ExitCode != 0 {
		code := -99999
		if completedResult.exitResult.ExitCode != nil {
			code = *completedResult.exitResult.ExitCode
		}
		t.Fatalf("cmd 退出码 = %d (0x%x)，期望 0；readErr=%v output=%q", code, uint32(code), completedResult.readErr, completedResult.output)
	}
	if !strings.Contains(strings.ToLower(string(completedResult.output)), strings.ToLower(filepath.Base(directory))) {
		t.Fatalf("ConPTY 启动目录错误，输出: %q", completedResult.output)
	}
	if !strings.Contains(string(completedResult.output), "TERMINAL_EXIT_SENTINEL") {
		t.Fatalf("退出前的最终输出丢失，输出: %q", completedResult.output)
	}
}

func TestWindowsDetectedAgentCommandUsesConfiguredShellInvocation(t *testing.T) {
	invocation := CommandInvocationForPlatform("windows", `C:\Windows\System32\cmd.exe`, `C:\tools\codex.cmd`, []string{"--yolo"})
	if invocation.Command != `C:\Windows\System32\cmd.exe` {
		t.Fatalf("Windows 代理启动命令 = %q", invocation.Command)
	}
	wantArguments := []string{"/C", `C:\tools\codex.cmd`, "--yolo"}
	if len(invocation.Arguments) != len(wantArguments) {
		t.Fatalf("Windows 代理启动参数 = %#v，期望 %#v", invocation.Arguments, wantArguments)
	}
	for index := range wantArguments {
		if invocation.Arguments[index] != wantArguments[index] {
			t.Fatalf("Windows 代理启动参数 = %#v，期望 %#v", invocation.Arguments, wantArguments)
		}
	}
}

func TestWindowsBackendStartsCmdAgentWrapperThroughConPTY(t *testing.T) {
	if os.Getenv("WINELOADER") != "" {
		t.Skip("Wine 的 ConPTY 读写不完整；请在原生 Windows 执行该集成测试")
	}
	directory := t.TempDir()
	wrapper := filepath.Join(directory, "codex.cmd")
	contents := "@echo off\r\necho __AGENT_ARGS__%*__\r\necho __AGENT_PWD__%CD%__\r\n"
	if err := os.WriteFile(wrapper, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile(codex.cmd) error = %v", err)
	}
	session, err := (&windowsBackend{}).Start(StartRequest{
		TaskID: "task-a", Directory: directory, ShellPath: "cmd.exe", Command: wrapper,
		Arguments: []string{"--yolo"}, Columns: 80, Rows: 24,
	})
	if err != nil {
		t.Fatalf("启动 codex.cmd ConPTY: %v", err)
	}
	defer session.Close()

	type result struct {
		output     []byte
		readErr    error
		exitResult ExitResult
		waitErr    error
	}
	completed := make(chan result, 1)
	go func() {
		output, readErr := io.ReadAll(session)
		exitResult, waitErr := session.Wait()
		completed <- result{output: output, readErr: readErr, exitResult: exitResult, waitErr: waitErr}
	}()
	var completedResult result
	select {
	case completedResult = <-completed:
	case <-time.After(10 * time.Second):
		_ = session.Close()
		t.Fatal("codex.cmd ConPTY 未在预期时间内退出")
	}
	if completedResult.readErr != nil && !isExpectedTerminalReadError(completedResult.readErr) {
		t.Fatalf("读取 codex.cmd ConPTY 输出: %v", completedResult.readErr)
	}
	if completedResult.waitErr != nil || completedResult.exitResult.ExitCode == nil || *completedResult.exitResult.ExitCode != 0 {
		t.Fatalf("codex.cmd 等待结果 = (%v, %v)，输出: %q", completedResult.exitResult.ExitCode, completedResult.waitErr, completedResult.output)
	}
	output := string(completedResult.output)
	if !strings.Contains(output, "__AGENT_ARGS__--yolo__") {
		t.Fatalf("codex.cmd 参数错误，输出: %q", output)
	}
	if !strings.Contains(strings.ToLower(output), strings.ToLower("__AGENT_PWD__"+directory+"__")) {
		t.Fatalf("codex.cmd 工作目录错误，输出: %q", output)
	}
}
