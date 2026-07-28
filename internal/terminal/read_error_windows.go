//go:build windows

package terminal

func isExpectedTerminalReadError(err error) bool {
	return isExpectedWindowsTerminalReadError(err)
}
