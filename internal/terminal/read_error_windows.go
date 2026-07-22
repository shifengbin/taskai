//go:build windows

package terminal

func isExpectedTerminalReadError(error) bool {
	return false
}
