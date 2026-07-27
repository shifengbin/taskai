package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

func validateDirectory(directory string) error {
	if directory == "" {
		return fmt.Errorf("终端工作目录不能为空")
	}
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("读取终端工作目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("终端工作目录不是文件夹")
	}
	return nil
}

func newSessionID() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err == nil {
		return hex.EncodeToString(bytes)
	}
	return fmt.Sprintf("terminal-%d", time.Now().UnixNano())
}

func sessionID(requestID string) string {
	if requestID != "" {
		return requestID
	}
	return newSessionID()
}

func embeddedTerminalEnvironment(extra []string) []string {
	environment := append([]string(nil), os.Environ()...)
	for _, entry := range append(append([]string(nil), extra...), "TERM=xterm-256color") {
		key, _, found := strings.Cut(entry, "=")
		if !found || key == "" {
			continue
		}
		prefix := key + "="
		replaced := false
		for index, current := range environment {
			if strings.HasPrefix(current, prefix) {
				environment[index] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			environment = append(environment, entry)
		}
	}
	return environment
}
