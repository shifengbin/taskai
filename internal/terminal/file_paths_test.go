package terminal

import (
	"strings"
	"testing"
)

func TestFormatDroppedFilePathsForSupportedShells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		shellPath string
		paths     []string
		want      string
	}{
		{
			name:      "POSIX Shell",
			shellPath: "/bin/zsh",
			paths:     []string{"/tmp/My Project/it's [ready];.txt", "/tmp/second file.txt"},
			want:      "'/tmp/My Project/it'\\''s [ready];.txt' '/tmp/second file.txt'",
		},
		{
			name:      "fish",
			shellPath: "/usr/bin/fish",
			paths:     []string{"/tmp/$HOME & *.txt"},
			want:      "'/tmp/$HOME & *.txt'",
		},
		{
			name:      "PowerShell",
			shellPath: `C:\\Program Files\\PowerShell\\7\\pwsh.exe`,
			paths:     []string{`C:\\Work Files\\O'Brien & $value.txt`},
			want:      `'C:\\Work Files\\O''Brien & $value.txt'`,
		},
		{
			// 26200 构建起 cmd.exe 引号内 caret 转义失效（caret 字面保留），
			// 因此引号内内容保持原样；% 不再转义（成对 %VAR% 展开是 cmd 已知限制）。
			name:      "cmd",
			shellPath: `C:\\Windows\\System32\\cmd.exe`,
			paths:     []string{`C:\\Work Files\\a&b|c<d>e^f 50% x!y.txt`},
			want:      `"C:\\Work Files\\a&b|c<d>e^f 50% x!y.txt"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := formatDroppedFilePaths(test.shellPath, test.paths)
			if err != nil {
				t.Fatalf("格式化拖放路径: %v", err)
			}
			if got != test.want {
				t.Errorf("格式化结果 = %q，期望 %q", got, test.want)
			}
			if strings.ContainsAny(got, "\r\n") {
				t.Errorf("格式化结果包含换行: %q", got)
			}
		})
	}
}

func TestFormatDroppedFilePathsRejectsUnsafeInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		shellPath string
		paths     []string
	}{
		{name: "空路径列表", shellPath: "/bin/sh"},
		{name: "空路径", shellPath: "/bin/sh", paths: []string{""}},
		{name: "换行路径", shellPath: "/bin/sh", paths: []string{"/tmp/first\nsecond"}},
		{name: "转义序列路径", shellPath: "/bin/sh", paths: []string{"/tmp/clear\x1b[2J.txt"}},
		{name: "中断字符路径", shellPath: "/bin/sh", paths: []string{"/tmp/interrupt\x03.txt"}},
		{name: "未知 Shell", shellPath: "/usr/local/bin/nu", paths: []string{"/tmp/file.txt"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := formatDroppedFilePaths(test.shellPath, test.paths); err == nil {
				t.Fatal("期望拒绝拖放路径")
			}
		})
	}
}

func TestManagerWritesDroppedFilePathsUsingTheSessionShell(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})
	created, err := manager.Create("task-a", t.TempDir(), "/bin/zsh", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	if err := manager.WriteFilePaths("task-a", created.ID, []string{"/tmp/My Project/it's ready.txt", "/tmp/next file.txt"}); err != nil {
		t.Fatalf("写入拖放路径: %v", err)
	}

	got := backend.session(created.ID).input()
	want := "'/tmp/My Project/it'\\''s ready.txt' '/tmp/next file.txt'"
	if got != want {
		t.Errorf("PTY 输入 = %q，期望 %q", got, want)
	}
	if got := backend.session(created.ID).inputWriteCount(); got != 1 {
		t.Errorf("PTY 写入次数 = %d，期望 1", got)
	}
	if strings.ContainsAny(got, "\r\n") {
		t.Errorf("PTY 输入包含换行: %q", got)
	}
}

func TestManagerDoesNotWriteDroppedFilePathsWhenFormattingFails(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})
	created, err := manager.Create("task-a", t.TempDir(), "/usr/local/bin/nu", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	if err := manager.WriteFilePaths("task-a", created.ID, []string{"/tmp/file.txt"}); err == nil {
		t.Fatal("期望未知 Shell 返回错误")
	}
	if got := backend.session(created.ID).input(); got != "" {
		t.Errorf("格式化失败仍写入 PTY: %q", got)
	}
}

func TestManagerDoesNotWriteDroppedFilePathsToClosedTerminal(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})
	created, err := manager.Create("task-a", t.TempDir(), "/bin/sh", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	if err := manager.Close("task-a", created.ID); err != nil {
		t.Fatalf("关闭终端: %v", err)
	}

	if err := manager.WriteFilePaths("task-a", created.ID, []string{"/tmp/file.txt"}); err == nil {
		t.Fatal("期望关闭终端拒绝拖放路径")
	}
	if got := backend.session(created.ID).input(); got != "" {
		t.Errorf("关闭终端仍写入 PTY: %q", got)
	}
}
