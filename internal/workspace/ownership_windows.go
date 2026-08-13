//go:build windows

package workspace

import (
	"fmt"
	"io"
	"os"
	"unsafe"

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

func secureAndValidatePrivateDirectory(path string, _ os.FileInfo) error {
	currentToken, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return err
	}
	defer currentToken.Close()
	currentUser, err := currentToken.GetTokenUser()
	if err != nil {
		return err
	}
	access := []windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.SET_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(currentUser.User.Sid),
		},
	}}
	acl, err := windows.ACLFromEntries(access, nil)
	if err != nil {
		return err
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		return err
	}
	updated, err := os.Lstat(path)
	if err != nil {
		return err
	}
	return validatePrivateDirectory(path, updated)
}

func createPrivateDirectory(path string) error {
	currentToken, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return err
	}
	defer currentToken.Close()
	currentUser, err := currentToken.GetTokenUser()
	if err != nil {
		return err
	}
	descriptor, err := windows.SecurityDescriptorFromString(fmt.Sprintf("O:%sD:P(A;OICI;GA;;;%s)", currentUser.User.Sid.String(), currentUser.User.Sid.String()))
	if err != nil {
		return err
	}
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attributes := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}
	return windows.CreateDirectory(pathPointer, &attributes)
}

func validatePrivateDirectory(path string, _ os.FileInfo) error {
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		return err
	}
	currentToken, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return err
	}
	defer currentToken.Close()
	currentUser, err := currentToken.GetTokenUser()
	if err != nil {
		return err
	}
	if !owner.Equals(currentUser.User.Sid) {
		return fmt.Errorf("目录所有者不是当前用户")
	}
	dacl, _, err := descriptor.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		return fmt.Errorf("目录访问控制列表不安全")
	}
	var entry *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &entry); err != nil {
		return err
	}
	entrySID := (*windows.SID)(unsafe.Pointer(&entry.SidStart))
	if entry.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || !entrySID.Equals(currentUser.User.Sid) {
		return fmt.Errorf("目录访问控制列表允许其他用户访问")
	}
	return nil
}

func validateOwnershipRoot(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("路径不是普通目录")
	}
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		return err
	}
	currentToken, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return err
	}
	defer currentToken.Close()
	currentUser, err := currentToken.GetTokenUser()
	if err != nil {
		return err
	}
	if !owner.Equals(currentUser.User.Sid) {
		return fmt.Errorf("目录所有者不是当前用户")
	}
	dacl, _, err := descriptor.DACL()
	if err != nil || dacl == nil {
		return fmt.Errorf("目录访问控制列表不安全")
	}
	for index := uint32(0); index < uint32(dacl.AceCount); index++ {
		var entry *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, index, &entry); err != nil {
			return err
		}
		if entry.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			continue
		}
		entrySID := (*windows.SID)(unsafe.Pointer(&entry.SidStart))
		if entrySID.Equals(currentUser.User.Sid) || entrySID.IsWellKnown(windows.WinBuiltinAdministratorsSid) || entrySID.IsWellKnown(windows.WinLocalSystemSid) {
			continue
		}
		if entry.Mask&(windows.GENERIC_ALL|windows.GENERIC_WRITE|windows.WRITE_DAC|windows.WRITE_OWNER|windows.DELETE|windows.ACCESS_MASK(windows.FILE_WRITE_DATA)|windows.ACCESS_MASK(windows.FILE_APPEND_DATA)) != 0 {
			return fmt.Errorf("目录允许其他用户写入")
		}
	}
	return nil
}
