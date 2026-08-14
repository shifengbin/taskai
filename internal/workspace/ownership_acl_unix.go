//go:build darwin || linux

package workspace

func securePrivateDirectoryACL(string) error {
	return nil
}

func validateExtendedACL(string) error {
	return nil
}
