//go:build windows

package terminal

import (
	"io"
	"os/exec"
	"strings"
	"testing"
)

// 26200 构建起 cmd.exe 不再处理双引号内的 caret 转义，拖放路径改为整体加引号、
// 内容保持原样。路径含空格、& | ^ ! 与单个 %（不成对，cmd 不会展开）。
// 成对的 %VAR% 会被 cmd 展开，与 Windows Terminal 行为一致，属于已知限制。
func TestCmdDroppedFilePathParsesAsOneLiteralArgument(t *testing.T) {
	path := `C:\Work Files\a&b^f 50% x!y.txt`
	formatted, err := formatDroppedFilePaths(`C:\Windows\System32\cmd.exe`, []string{path})
	if err != nil {
		t.Fatalf("格式化 cmd.exe 路径: %v", err)
	}

	process := exec.Command("cmd.exe", "/D", "/V:OFF", "/Q")
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
	// /Q 模式下 cmd 的输出会和提示符同行，因此用子串断言整段输出。
	if !strings.Contains(string(data), "["+path+"]") {
		t.Fatalf("cmd.exe 输出未包含字面参数 [%s]，完整输出: %q", path, string(data))
	}
}
