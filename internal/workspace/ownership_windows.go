//go:build windows

package workspace

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/windows"
)

const directoryOwnershipStream = ":taskai.workspace-token"

func setDirectoryOwnershipToken(path, token string) error {
	pointer, err := windows.UTF16PtrFromString(path + directoryOwnershipStream)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(pointer, windows.GENERIC_WRITE, 0, nil, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_HIDDEN, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	written, err := windows.Write(handle, []byte(token))
	if err != nil {
		return err
	}
	if written != len(token) {
		return io.ErrShortWrite
	}
	return windows.FlushFileBuffers(handle)
}

func directoryOwnershipToken(path string) (string, bool, error) {
	pointer, err := windows.UTF16PtrFromString(path + directoryOwnershipStream)
	if err != nil {
		return "", false, err
	}
	handle, err := windows.CreateFile(pointer, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if err == windows.ERROR_FILE_NOT_FOUND || err == windows.ERROR_PATH_NOT_FOUND {
			return "", false, nil
		}
		return "", false, err
	}
	defer windows.CloseHandle(handle)
	contents := make([]byte, 65)
	read, err := windows.Read(handle, contents)
	if err != nil {
		return "", false, err
	}
	if read != 64 {
		return "", false, fmt.Errorf("工作目录所有权标记无效")
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
