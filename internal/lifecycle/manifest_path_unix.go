//go:build !windows

package lifecycle

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const manifestTemporaryNameAttempts = 16

func writeManifestContents(workspacePath, directory, name string, contents []byte) error {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return fmt.Errorf("任务工作目录不可用")
	}
	absWorkspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return fmt.Errorf("解析任务工作目录失败: %w", err)
	}
	workspaceFD, err := unix.Open(filepath.Clean(absWorkspacePath), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("任务工作目录不可用: %w", err)
	}
	defer unix.Close(workspaceFD)

	directoryFD, err := openManifestDirectory(workspaceFD, directory)
	if err != nil {
		return err
	}
	defer unix.Close(directoryFD)

	return writeManifestContentsAt(directoryFD, name, contents)
}

func openManifestDirectory(workspaceFD int, directory string) (int, error) {
	directory = filepath.Clean(strings.TrimSpace(directory))
	if directory == "." {
		directoryFD, err := unix.Dup(workspaceFD)
		if err != nil {
			return -1, fmt.Errorf("打开清单文件目录失败: %w", err)
		}
		return directoryFD, nil
	}
	if filepath.IsAbs(directory) || directory == ".." || strings.HasPrefix(directory, ".."+string(filepath.Separator)) {
		return -1, fmt.Errorf("清单文件目录不安全")
	}

	currentFD, err := unix.Dup(workspaceFD)
	if err != nil {
		return -1, fmt.Errorf("打开清单文件目录失败: %w", err)
	}
	for _, component := range strings.Split(directory, string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			unix.Close(currentFD)
			return -1, fmt.Errorf("清单文件目录不安全")
		}
		nextFD, err := openOrCreateManifestDirectory(currentFD, component)
		unix.Close(currentFD)
		if err != nil {
			return -1, err
		}
		currentFD = nextFD
	}
	return currentFD, nil
}

func openOrCreateManifestDirectory(parentFD int, name string) (int, error) {
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW
	childFD, err := unix.Openat(parentFD, name, flags, 0)
	if err == nil {
		return childFD, nil
	}
	if !errors.Is(err, unix.ENOENT) {
		return -1, fmt.Errorf("清单文件目录不安全: %w", err)
	}
	if err := unix.Mkdirat(parentFD, name, 0o700); err != nil && !errors.Is(err, unix.EEXIST) {
		return -1, fmt.Errorf("创建清单文件目录失败: %w", err)
	}
	childFD, err = unix.Openat(parentFD, name, flags, 0)
	if err != nil {
		return -1, fmt.Errorf("清单文件目录不安全: %w", err)
	}
	return childFD, nil
}

func writeManifestContentsAt(directoryFD int, name string, contents []byte) error {
	if err := validateManifestTarget(directoryFD, name); err != nil {
		return err
	}
	temporary, temporaryName, err := createManifestTemporaryFile(directoryFD, name)
	if err != nil {
		return err
	}
	defer func() {
		if temporary != nil {
			_ = temporary.Close()
		}
		if temporaryName != "" {
			_ = unix.Unlinkat(directoryFD, temporaryName, 0)
		}
	}()

	if written, err := temporary.Write(contents); err != nil || written != len(contents) {
		if err == nil {
			err = fmt.Errorf("仅写入 %d/%d 字节", written, len(contents))
		}
		return fmt.Errorf("写入清单文件失败: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("同步清单文件失败: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("关闭清单文件失败: %w", err)
	}
	temporary = nil
	if err := unix.Renameat(directoryFD, temporaryName, directoryFD, name); err != nil {
		return fmt.Errorf("替换清单文件失败: %w", err)
	}
	temporaryName = ""
	if err := unix.Fsync(directoryFD); err != nil {
		return fmt.Errorf("同步清单文件目录失败: %w", err)
	}
	return nil
}

func validateManifestTarget(directoryFD int, name string) error {
	var information unix.Stat_t
	err := unix.Fstatat(directoryFD, name, &information, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("检查清单文件目标失败: %w", err)
	}
	if information.Mode&unix.S_IFMT != unix.S_IFREG {
		return fmt.Errorf("清单文件目标不可用: 目标不是普通文件")
	}
	return nil
}

func createManifestTemporaryFile(directoryFD int, name string) (*os.File, string, error) {
	for range manifestTemporaryNameAttempts {
		suffix := make([]byte, 12)
		if _, err := rand.Read(suffix); err != nil {
			return nil, "", fmt.Errorf("生成清单文件临时名称失败: %w", err)
		}
		temporaryName := "." + name + "-" + hex.EncodeToString(suffix)
		fileDescriptor, err := unix.Openat(directoryFD, temporaryName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
		if errors.Is(err, unix.EEXIST) {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("创建清单文件临时文件失败: %w", err)
		}
		return os.NewFile(uintptr(fileDescriptor), temporaryName), temporaryName, nil
	}
	return nil, "", fmt.Errorf("创建清单文件临时文件失败: 名称冲突")
}
