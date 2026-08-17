//go:build darwin || linux

package workspace

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const directoryOwnershipAttribute = "user.taskai.workspace-token"

func setDirectoryOwnershipToken(path, token string) error {
	if err := unix.Setxattr(path, directoryOwnershipAttribute, []byte(token), unix.XATTR_CREATE); err != nil {
		return err
	}
	return nil
}

func directoryOwnershipToken(path string) (string, bool, error) {
	size, err := unix.Getxattr(path, directoryOwnershipAttribute, nil)
	if err != nil {
		if errors.Is(err, unix.ENODATA) || isMissingOwnershipAttribute(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if size <= 0 || size > 256 {
		return "", false, fmt.Errorf("工作目录所有权标记无效")
	}
	contents := make([]byte, size)
	read, err := unix.Getxattr(path, directoryOwnershipAttribute, contents)
	if err != nil {
		return "", false, err
	}
	return string(contents[:read]), true, nil
}

func validateOwnershipRoot(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("路径不是普通目录")
	}
	return nil
}

func createPrivateDirectory(path string) error {
	return os.Mkdir(path, 0o755)
}
