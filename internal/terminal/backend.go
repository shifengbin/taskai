package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
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
