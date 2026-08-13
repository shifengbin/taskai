//go:build darwin || linux

package workspace

import (
	"errors"
	"fmt"
	"os"
	"syscall"

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

func secureAndValidatePrivateDirectory(path string, info os.FileInfo) error {
	if info.Mode().Perm() != 0o700 {
		if err := os.Chmod(path, 0o700); err != nil {
			return fmt.Errorf("设置权限失败: %w", err)
		}
		updated, err := os.Lstat(path)
		if err != nil {
			return err
		}
		info = updated
	}
	if err := securePrivateDirectoryACL(path); err != nil {
		return err
	}
	return validatePrivateDirectory(path, info)
}

func validatePrivateDirectory(path string, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("无法验证目录所有者")
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("目录所有者不是当前用户")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("目录权限允许其他用户访问")
	}
	return validateExtendedACL(path)
}

func validateOwnershipRoot(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("路径不是普通目录")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("无法验证目录所有者")
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("目录所有者不是当前用户")
	}
	if info.Mode().Perm()&0o020 != 0 && stat.Gid != uint32(os.Getegid()) {
		return fmt.Errorf("目录允许其他用户组写入")
	}
	if info.Mode().Perm()&0o002 != 0 && info.Mode()&os.ModeSticky == 0 {
		return fmt.Errorf("目录允许其他用户写入")
	}
	return validateExtendedACL(path)
}

func createPrivateDirectory(path string) error {
	return os.Mkdir(path, 0o700)
}
