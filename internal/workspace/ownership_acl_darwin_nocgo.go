//go:build darwin && !cgo

package workspace

import "fmt"

func securePrivateDirectoryACL(string) error {
	return fmt.Errorf("当前构建无法安全配置 macOS 扩展 ACL")
}

func validateExtendedACL(string) error {
	return fmt.Errorf("当前构建无法安全验证 macOS 扩展 ACL")
}
