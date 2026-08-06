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
		return `"` + strings.NewReplacer(
			"^", "^^",
			"&", "^&",
			"|", "^|",
			"<", "^<",
			">", "^>",
			"%", "^%",
			"!", "^!",
		).Replace(path) + `"`, nil
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
