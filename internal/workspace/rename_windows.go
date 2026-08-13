//go:build windows

package workspace

import "golang.org/x/sys/windows"

func renameNoReplace(source, destination string) error {
	sourcePointer, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPointer, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	return windows.MoveFile(sourcePointer, destinationPointer)
}
