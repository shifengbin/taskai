package directorylinks

type DirectoryLinkFS interface {
	Create(linkPath, targetPath string) error
	Read(linkPath string) (targetPath string, exists bool, err error)
	Remove(linkPath string) error
}

func NativeDirectoryLinkFS() DirectoryLinkFS {
	return newNativeDirectoryLinkFS()
}
