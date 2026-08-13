//go:build darwin || linux

package workspace

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func directoryIdentity(path string) (string, error) {
	var information unix.Stat_t
	if err := unix.Lstat(path, &information); err != nil {
		return "", err
	}
	if information.Mode&unix.S_IFMT != unix.S_IFDIR {
		return "", fmt.Errorf("路径不是普通目录")
	}
	return fmt.Sprintf("%d:%d", uint64(information.Dev), uint64(information.Ino)), nil
}
