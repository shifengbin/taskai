//go:build darwin && cgo

package workspace

/*
#include <sys/types.h>
#include <sys/acl.h>
#include <errno.h>
#include <stdlib.h>

static int taskai_acl_has_entries(const char *path) {
	acl_t acl = acl_get_file(path, ACL_TYPE_EXTENDED);
	if (acl == NULL) {
		return -1;
	}
	acl_entry_t entry;
	int result = acl_get_entry(acl, ACL_FIRST_ENTRY, &entry);
	int saved_errno = errno;
	acl_free(acl);
	errno = saved_errno;
	return result;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

func securePrivateDirectoryACL(path string) error {
	pathPointer := C.CString(path)
	defer C.free(unsafe.Pointer(pathPointer))
	hasEntries, err := C.taskai_acl_has_entries(pathPointer)
	if hasEntries < 0 {
		if errors.Is(err, syscall.ENOTSUP) {
			return nil
		}
		return fmt.Errorf("读取目录扩展 ACL 失败: %w", err)
	}
	if hasEntries == 0 {
		return nil
	}
	if result, err := C.acl_delete_file_np(pathPointer, C.ACL_TYPE_EXTENDED); result != 0 {
		return fmt.Errorf("清除目录扩展 ACL 失败: %w", err)
	}
	return validateExtendedACL(path)
}

func validateExtendedACL(path string) error {
	pathPointer := C.CString(path)
	defer C.free(unsafe.Pointer(pathPointer))
	result, err := C.taskai_acl_has_entries(pathPointer)
	if result < 0 {
		if errors.Is(err, syscall.ENOTSUP) {
			return nil
		}
		return fmt.Errorf("读取目录扩展 ACL 失败: %w", err)
	}
	if result > 0 {
		return fmt.Errorf("目录扩展 ACL 允许额外访问")
	}
	return nil
}
