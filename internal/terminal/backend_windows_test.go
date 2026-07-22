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
		output  []byte
		readErr error
		waitErr error
	}
	completed := make(chan result, 1)
	go func() {
		if _, err := session.Write([]byte("cd\r\nexit\r\n")); err != nil {
			completed <- result{readErr: err}
			return
		}
		output, readErr := io.ReadAll(session)
		completed <- result{output: output, readErr: readErr, waitErr: session.Wait()}
	}()
	var completedResult result
	select {
	case completedResult = <-completed:
	case <-time.After(10 * time.Second):
		_ = session.Close()
		t.Fatal("ConPTY 未在预期时间内退出")
	}
	if completedResult.readErr != nil && !strings.Contains(strings.ToLower(completedResult.readErr.Error()), "pipe") {
		t.Fatalf("读取 ConPTY 输出: %v", completedResult.readErr)
	}
	if completedResult.waitErr != nil {
		t.Fatalf("等待 cmd 退出: %v", completedResult.waitErr)
	}
	if !strings.Contains(strings.ToLower(string(completedResult.output)), strings.ToLower(filepath.Base(directory))) {
		t.Fatalf("ConPTY 启动目录错误，输出: %q", completedResult.output)
	}
}
