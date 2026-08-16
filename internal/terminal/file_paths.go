package terminal

import (
	"fmt"
	"strings"
)

func formatDroppedFilePaths(shellPath string, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("拖放路径不能为空")
	}

	quoted := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" || containsTerminalControlCharacter(path) {
			return "", fmt.Errorf("拖放路径无效")
		}
		value, err := quoteDroppedFilePath(shellPath, path)
		if err != nil {
			return "", err
		}
		quoted = append(quoted, value)
	}
	return strings.Join(quoted, " "), nil
}

func containsTerminalControlCharacter(path string) bool {
	return strings.IndexFunc(path, func(character rune) bool {
		return character < 0x20 || character == 0x7f
	}) >= 0
}

func quoteDroppedFilePath(shellPath, path string) (string, error) {
	switch terminalShellName(shellPath) {
	case "sh", "bash", "zsh", "dash", "ksh", "fish":
		return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'", nil
	case "powershell", "pwsh":
		return "'" + strings.ReplaceAll(path, "'", "''") + "'", nil
	case "cmd":
		if strings.Contains(path, `"`) {
			return "", fmt.Errorf("cmd.exe 路径不能包含双引号")
		}
		// Windows 10.0.26200 起 cmd.exe 不再处理双引号内的 caret 转义（caret 一律
		// 字面保留），而引号内的 & | < > ^ ! 与空格本身就是字面安全的，因此整体
		// 加引号、内容保持原样。成对的 %VAR% 仍会被 cmd 展开，这与 Windows Terminal
		// 等终端的拖放行为一致，属于 cmd.exe 的已知限制。
		return `"` + path + `"`, nil
	default:
		return "", fmt.Errorf("不支持终端 Shell: %s", shellPath)
	}
}

func terminalShellName(shellPath string) string {
	if index := strings.LastIndexAny(shellPath, `/\\`); index >= 0 {
		shellPath = shellPath[index+1:]
	}
	return strings.TrimSuffix(strings.ToLower(shellPath), ".exe")
}
