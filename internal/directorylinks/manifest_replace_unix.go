//go:build !windows

package directorylinks

import "os"

func replaceManifestFile(source, target string) error {
	return os.Rename(source, target)
}
