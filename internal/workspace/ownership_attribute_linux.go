//go:build linux

package workspace

func isMissingOwnershipAttribute(_ error) bool {
	return false
}
