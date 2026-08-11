//go:build windows

package lifecycle

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const manifestTemporaryNameAttempts = 16

type manifestWindowsRenameInformation struct {
	ReplaceIfExists uint32
	RootDirectory   windows.Handle
	FileNameLength  uint32
	FileName        [1]uint16
}

func writeManifestContents(workspacePath, directory, name string, contents []byte) error {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return fmt.Errorf("任务工作目录不可用")
	}
	absWorkspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return fmt.Errorf("解析任务工作目录失败: %w", err)
	}
	workspaceFD, err := openManifestWindowsWorkspaceDirectory("\\??\\" + filepath.Clean(absWorkspacePath))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(workspaceFD)

	directoryFD, err := openManifestWindowsDirectoryPath(workspaceFD, directory)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(directoryFD)

	return writeManifestWindowsContentsAt(directoryFD, name, contents)
}

func openManifestWindowsWorkspaceDirectory(path string) (windows.Handle, error) {
	workspaceFD, err := createManifestWindowsObject(0, path, windows.FILE_GENERIC_READ|windows.FILE_GENERIC_WRITE|windows.DELETE, windows.FILE_OPEN, windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, false)
	if err != nil {
		return 0, fmt.Errorf("任务工作目录不可用: %w", err)
	}
	hasReparsePoint, err := manifestWindowsHandleHasReparsePoint(workspaceFD)
	if err != nil {
		windows.CloseHandle(workspaceFD)
		return 0, fmt.Errorf("检查任务工作目录失败: %w", err)
	}
	if hasReparsePoint {
		windows.CloseHandle(workspaceFD)
		return 0, fmt.Errorf("任务工作目录不安全: 不能使用重解析点")
	}
	return workspaceFD, nil
}

func manifestWindowsHandleHasReparsePoint(handle windows.Handle) (bool, error) {
	var information windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &information); err != nil {
		return false, err
	}
	return information.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}

func openManifestWindowsDirectoryPath(workspaceFD windows.Handle, directory string) (windows.Handle, error) {
	directory = filepath.Clean(strings.TrimSpace(directory))
	if directory == "." {
		return duplicateManifestWindowsHandle(workspaceFD)
	}
	if filepath.IsAbs(directory) || directory == ".." || strings.HasPrefix(directory, ".."+string(filepath.Separator)) {
		return 0, fmt.Errorf("清单文件目录不安全")
	}

	currentFD, err := duplicateManifestWindowsHandle(workspaceFD)
	if err != nil {
		return 0, fmt.Errorf("打开清单文件目录失败: %w", err)
	}
	for _, component := range strings.Split(directory, string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			windows.CloseHandle(currentFD)
			return 0, fmt.Errorf("清单文件目录不安全")
		}
		nextFD, err := openOrCreateManifestWindowsDirectory(currentFD, component)
		windows.CloseHandle(currentFD)
		if err != nil {
			return 0, err
		}
		currentFD = nextFD
	}
	return currentFD, nil
}

func duplicateManifestWindowsHandle(handle windows.Handle) (windows.Handle, error) {
	var copy windows.Handle
	if err := windows.DuplicateHandle(windows.CurrentProcess(), handle, windows.CurrentProcess(), &copy, 0, false, windows.DUPLICATE_SAME_ACCESS); err != nil {
		return 0, err
	}
	return copy, nil
}

func openOrCreateManifestWindowsDirectory(parentFD windows.Handle, name string) (windows.Handle, error) {
	childFD, err := openManifestWindowsDirectory(parentFD, name)
	if err == nil {
		return childFD, nil
	}
	if !isWindowsObjectNotFound(err) {
		return 0, fmt.Errorf("清单文件目录不安全: %w", err)
	}
	if _, err := createManifestWindowsDirectory(parentFD, name); err != nil && !isWindowsObjectNameCollision(err) {
		return 0, fmt.Errorf("创建清单文件目录失败: %w", err)
	}
	childFD, err = openManifestWindowsDirectory(parentFD, name)
	if err != nil {
		return 0, fmt.Errorf("清单文件目录不安全: %w", err)
	}
	return childFD, nil
}

func openManifestWindowsDirectory(parentFD windows.Handle, name string) (windows.Handle, error) {
	directoryFD, err := createManifestWindowsObject(parentFD, name, windows.FILE_GENERIC_READ|windows.FILE_GENERIC_WRITE|windows.DELETE, windows.FILE_OPEN, windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, false)
	if err != nil {
		return 0, err
	}
	hasReparsePoint, err := manifestWindowsHandleHasReparsePoint(directoryFD)
	if err != nil {
		windows.CloseHandle(directoryFD)
		return 0, err
	}
	if hasReparsePoint {
		windows.CloseHandle(directoryFD)
		return 0, fmt.Errorf("不能使用重解析点")
	}
	return directoryFD, nil
}

func createManifestWindowsDirectory(parentFD windows.Handle, name string) (windows.Handle, error) {
	return createManifestWindowsObject(parentFD, name, windows.FILE_GENERIC_READ|windows.FILE_GENERIC_WRITE|windows.DELETE, windows.FILE_CREATE, windows.FILE_DIRECTORY_FILE|windows.FILE_SYNCHRONOUS_IO_NONALERT, true)
}

func writeManifestWindowsContentsAt(directoryFD windows.Handle, name string, contents []byte) error {
	if err := validateManifestWindowsTarget(directoryFD, name); err != nil {
		return err
	}
	temporaryFD, temporaryName, err := createManifestWindowsTemporaryFile(directoryFD, name)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(temporaryFD)
	completed := false
	defer func() {
		if !completed {
			deleteManifestWindowsTemporaryFile(temporaryFD)
		}
	}()

	for remaining := contents; len(remaining) > 0; {
		var written uint32
		if err := windows.WriteFile(temporaryFD, remaining, &written, nil); err != nil {
			return fmt.Errorf("写入清单文件失败: %w", err)
		}
		if written == 0 {
			return fmt.Errorf("写入清单文件失败: 未写入内容")
		}
		remaining = remaining[written:]
	}
	if err := windows.FlushFileBuffers(temporaryFD); err != nil {
		return fmt.Errorf("同步清单文件失败: %w", err)
	}
	if err := renameManifestWindowsTemporaryFile(temporaryFD, directoryFD, name); err != nil {
		return fmt.Errorf("替换清单文件失败: %w", err)
	}
	completed = true
	_ = temporaryName
	return nil
}

func validateManifestWindowsTarget(directoryFD windows.Handle, name string) error {
	// FILE_SYNCHRONOUS_IO_NONALERT 要求 DesiredAccess 包含 SYNCHRONIZE，否则 NtCreateFile 在参数校验阶段即返回 STATUS_INVALID_PARAMETER。
	targetFD, err := createManifestWindowsObject(directoryFD, name, windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE, windows.FILE_OPEN, windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, false)
	if isWindowsObjectNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("检查清单文件目标失败: %w", err)
	}
	defer windows.CloseHandle(targetFD)
	var information windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(targetFD, &information); err != nil {
		return fmt.Errorf("检查清单文件目标失败: %w", err)
	}
	if information.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("清单文件目标不安全: 不能使用符号链接")
	}
	return nil
}

func createManifestWindowsTemporaryFile(directoryFD windows.Handle, name string) (windows.Handle, string, error) {
	for range manifestTemporaryNameAttempts {
		suffix := make([]byte, 12)
		if _, err := rand.Read(suffix); err != nil {
			return 0, "", fmt.Errorf("生成清单文件临时名称失败: %w", err)
		}
		temporaryName := "." + name + "-" + hex.EncodeToString(suffix)
		temporaryFD, err := createManifestWindowsObject(directoryFD, temporaryName, windows.FILE_GENERIC_WRITE|windows.DELETE, windows.FILE_CREATE, windows.FILE_NON_DIRECTORY_FILE|windows.FILE_SYNCHRONOUS_IO_NONALERT, true)
		if isWindowsObjectNameCollision(err) {
			continue
		}
		if err != nil {
			return 0, "", fmt.Errorf("创建清单文件临时文件失败: %w", err)
		}
		return temporaryFD, temporaryName, nil
	}
	return 0, "", fmt.Errorf("创建清单文件临时文件失败: 名称冲突")
}

func createManifestWindowsObject(parentFD windows.Handle, name string, access, disposition, options uint32, rejectReparse bool) (windows.Handle, error) {
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return 0, err
	}
	attributes := uint32(windows.OBJ_CASE_INSENSITIVE)
	if rejectReparse {
		attributes |= windows.OBJ_DONT_REPARSE
	}
	objectAttributes := &windows.OBJECT_ATTRIBUTES{
		Length:        uint32(unsafe.Sizeof(windows.OBJECT_ATTRIBUTES{})),
		RootDirectory: parentFD,
		ObjectName:    objectName,
		Attributes:    attributes,
	}
	var status windows.IO_STATUS_BLOCK
	var allocationSize int64
	var handle windows.Handle
	err = windows.NtCreateFile(&handle, access, objectAttributes, &status, &allocationSize, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, disposition, options, 0, 0)
	if err != nil {
		return 0, err
	}
	return handle, nil
}

func renameManifestWindowsTemporaryFile(temporaryFD, directoryFD windows.Handle, name string) error {
	encodedName, err := windows.UTF16FromString(name)
	if err != nil {
		return err
	}
	nameLength := (len(encodedName) - 1) * 2
	var information manifestWindowsRenameInformation
	buffer := make([]byte, int(unsafe.Offsetof(information.FileName))+nameLength)
	rename := (*manifestWindowsRenameInformation)(unsafe.Pointer(&buffer[0]))
	rename.ReplaceIfExists = windows.FILE_RENAME_REPLACE_IF_EXISTS
	rename.RootDirectory = directoryFD
	rename.FileNameLength = uint32(nameLength)
	copy((*[windows.MAX_LONG_PATH]uint16)(unsafe.Pointer(&rename.FileName[0]))[:nameLength/2:nameLength/2], encodedName)
	var status windows.IO_STATUS_BLOCK
	return windows.NtSetInformationFile(temporaryFD, &status, &buffer[0], uint32(len(buffer)), windows.FileRenameInformation)
}

func deleteManifestWindowsTemporaryFile(temporaryFD windows.Handle) {
	deleteFile := byte(1)
	var status windows.IO_STATUS_BLOCK
	_ = windows.NtSetInformationFile(temporaryFD, &status, &deleteFile, 1, windows.FileDispositionInformation)
}

func isWindowsObjectNotFound(err error) bool {
	return errors.Is(err, windows.STATUS_NO_SUCH_FILE) || errors.Is(err, windows.STATUS_OBJECT_NAME_NOT_FOUND) || errors.Is(err, windows.STATUS_OBJECT_PATH_NOT_FOUND)
}

func isWindowsObjectNameCollision(err error) bool {
	return errors.Is(err, windows.STATUS_OBJECT_NAME_COLLISION)
}
