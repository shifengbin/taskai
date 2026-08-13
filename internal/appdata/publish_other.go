//go:build !darwin

package appdata

import (
	"errors"
	"os"
	"path/filepath"
)

func publishDirectory(source, destination string) error {
	if _, err := os.Lstat(destination); err == nil {
		return os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(source, destination); err != nil {
		return err
	}
	return syncDirectoryPath(filepath.Dir(destination))
}
