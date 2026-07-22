//go:build !windows

package terminal

import (
	"errors"
	"syscall"
)

func isExpectedTerminalReadError(err error) bool {
	return errors.Is(err, syscall.EIO)
}
