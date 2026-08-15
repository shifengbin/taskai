//go:build !windows

package updater

func startInstallerDetached(invocation Invocation) error {
	return startDetached(invocation)
}
