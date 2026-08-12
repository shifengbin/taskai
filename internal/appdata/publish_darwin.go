//go:build darwin

package appdata

import (
	"path/filepath"

	"golang.org/x/sys/unix"
)

func publishDirectory(source, destination string) error {
	if err := unix.RenameatxNp(unix.AT_FDCWD, source, unix.AT_FDCWD, destination, unix.RENAME_EXCL); err != nil {
		return err
	}
	return syncDirectoryPath(filepath.Dir(destination))
}
