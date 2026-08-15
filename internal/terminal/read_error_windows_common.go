package terminal

import (
	"errors"
	"syscall"
)

const (
	windowsErrorOperationAborted = syscall.Errno(995)
	windowsErrorBrokenPipe       = syscall.Errno(109)
)

func isExpectedWindowsTerminalReadError(err error) bool {
	return errors.Is(err, windowsErrorOperationAborted) || errors.Is(err, windowsErrorBrokenPipe)
}
