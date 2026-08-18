//go:build !windows

package directorylinks

import (
	"fmt"
	"os"
	"path/filepath"
)

type nativeDirectoryLinkFS struct{}

func newNativeDirectoryLinkFS() DirectoryLinkFS {
	return nativeDirectoryLinkFS{}
}

func (nativeDirectoryLinkFS) Create(linkPath, targetPath string) error {
	if err := validateDirectoryLinkTarget(targetPath); err != nil {
		return err
	}
	if err := os.Symlink(targetPath, linkPath); err != nil {
		return fmt.Errorf("创建目录链接失败 %q -> %q: %w", linkPath, targetPath, err)
	}
	return nil
}

func (nativeDirectoryLinkFS) Read(linkPath string) (string, bool, error) {
	info, err := os.Lstat(linkPath)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("检查目录链接失败 %q: %w", linkPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "", false, fmt.Errorf("路径不是受支持的目录链接: %q", linkPath)
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		return "", false, fmt.Errorf("读取目录链接失败 %q: %w", linkPath, err)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(linkPath), target)
	}
	return filepath.Clean(target), true, nil
}

func (nativeDirectoryLinkFS) Remove(linkPath string) error {
	if _, exists, err := (nativeDirectoryLinkFS{}).Read(linkPath); err != nil {
		return err
	} else if !exists {
		return nil
	}
	if err := os.Remove(linkPath); err != nil {
		return fmt.Errorf("删除目录链接失败 %q: %w", linkPath, err)
	}
	return nil
}

func validateDirectoryLinkTarget(targetPath string) error {
	if !filepath.IsAbs(targetPath) {
		return fmt.Errorf("目录链接来源必须是绝对路径: %q", targetPath)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("目录链接来源不可访问 %q: %w", targetPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("目录链接来源不是目录: %q", targetPath)
	}
	return nil
}
