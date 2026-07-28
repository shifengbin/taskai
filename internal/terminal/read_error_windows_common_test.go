package terminal

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestExpectedWindowsTerminalReadError(t *testing.T) {
	if !isExpectedWindowsTerminalReadError(fmt.Errorf("ConPTY 读取失败: %w", syscall.Errno(995))) {
		t.Fatal("ERROR_OPERATION_ABORTED 应被识别为预期的 ConPTY 读取结束")
	}
	if isExpectedWindowsTerminalReadError(errors.New("意外的 ConPTY 读取错误")) {
		t.Fatal("普通读取错误不应被识别为预期结束")
	}
}
