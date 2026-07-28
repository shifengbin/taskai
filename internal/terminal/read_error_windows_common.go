package terminal

import (
	"errors"
	"syscall"
)

const windowsErrorOperationAborted = syscall.Errno(995)

func isExpectedWindowsTerminalReadError(err error) bool {
	return errors.Is(err, windowsErrorOperationAborted)
}
