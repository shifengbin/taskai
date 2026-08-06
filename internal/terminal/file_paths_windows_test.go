//go:build windows

package terminal

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCmdDroppedFilePathParsesAsOneLiteralArgument(t *testing.T) {
	path := `C:\Work Files\a&b^f%HOME%!name!.txt`
	formatted, err := formatDroppedFilePaths(`C:\Windows\System32\cmd.exe`, []string{path})
	if err != nil {
		t.Fatalf("格式化 cmd.exe 路径: %v", err)
	}

	process := exec.Command("cmd.exe", "/D", "/V:OFF", "/Q")
	process.Env = append(os.Environ(), "HOME=expanded", "name=expanded")
	input, err := process.StdinPipe()
	if err != nil {
		t.Fatalf("创建 cmd.exe 标准输入: %v", err)
	}
	output, err := process.StdoutPipe()
	if err != nil {
		t.Fatalf("创建 cmd.exe 标准输出: %v", err)
	}
	if err := process.Start(); err != nil {
		t.Fatalf("启动 cmd.exe: %v", err)
	}
	if _, err := io.WriteString(input, "for %A in ("+formatted+") do @echo [%~A]\r\nexit\r\n"); err != nil {
		t.Fatalf("写入 cmd.exe: %v", err)
	}
	if err := input.Close(); err != nil {
		t.Fatalf("关闭 cmd.exe 标准输入: %v", err)
	}
	data, readErr := io.ReadAll(output)
	waitErr := process.Wait()
	if readErr != nil {
		t.Fatalf("读取 cmd.exe 输出: %v", readErr)
	}
	if waitErr != nil {
		t.Fatalf("等待 cmd.exe: %v", waitErr)
	}
	if got, want := strings.TrimSpace(string(data)), "["+path+"]"; got != want {
		t.Fatalf("cmd.exe 参数 = %q，期望 %q", got, want)
	}
}
