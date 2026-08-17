package updater

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"taskai/internal/backgroundprocess"
)

type Invocation struct {
	Command   string
	Arguments []string
}

type Launcher interface {
	LaunchInstaller(string) error
	OpenReleasePage(string) error
}

type SystemLauncher struct {
	platform       string
	start          func(Invocation) error
	startInstaller func(Invocation) error
}

func NewSystemLauncher(platform string) *SystemLauncher {
	return &SystemLauncher{platform: platform, start: startDetached, startInstaller: startInstallerDetached}
}

func DefaultSystemLauncher() *SystemLauncher {
	return NewSystemLauncher(runtime.GOOS)
}

func (launcher *SystemLauncher) LaunchInstaller(path string) error {
	invocation, err := InstallerInvocation(launcher.platform, path)
	if err != nil {
		return err
	}
	if err := launcher.startInstaller(invocation); err != nil {
		return fmt.Errorf("启动安装程序: %w", err)
	}
	return nil
}

func (launcher *SystemLauncher) OpenReleasePage(releaseURL string) error {
	invocation, err := ReleasePageInvocation(launcher.platform, releaseURL)
	if err != nil {
		return err
	}
	if err := launcher.start(invocation); err != nil {
		return fmt.Errorf("打开 Release 页面: %w", err)
	}
	return nil
}

func InstallerInvocation(platform, path string) (Invocation, error) {
	if path == "" {
		return Invocation{}, fmt.Errorf("安装包路径不能为空")
	}
	wantExtension := map[string]string{
		"windows": ".exe",
		"darwin":  ".dmg",
		"linux":   ".deb",
	}[platform]
	if wantExtension == "" {
		return Invocation{}, fmt.Errorf("不支持在 %s 上自动安装", platform)
	}
	if !strings.EqualFold(filepath.Ext(path), wantExtension) {
		return Invocation{}, fmt.Errorf("%s 安装包必须使用 %s 格式", platform, wantExtension)
	}
	switch platform {
	case "windows":
		return Invocation{Command: path}, nil
	case "darwin":
		return Invocation{Command: "open", Arguments: []string{path}}, nil
	case "linux":
		return Invocation{Command: "xdg-open", Arguments: []string{path}}, nil
	default:
		panic("unreachable")
	}
}

func ReleasePageInvocation(platform, releaseURL string) (Invocation, error) {
	if !strings.HasPrefix(releaseURL, OfficialReleasePrefix) || releaseURL == OfficialReleasePrefix {
		return Invocation{}, fmt.Errorf("拒绝打开非官方 Release 页面: %s", releaseURL)
	}
	switch platform {
	case "windows":
		return Invocation{Command: "rundll32.exe", Arguments: []string{"url.dll,FileProtocolHandler", releaseURL}}, nil
	case "darwin":
		return Invocation{Command: "open", Arguments: []string{releaseURL}}, nil
	case "linux":
		return Invocation{Command: "xdg-open", Arguments: []string{releaseURL}}, nil
	default:
		return Invocation{}, fmt.Errorf("不支持在 %s 上打开 Release 页面", platform)
	}
}

func startDetached(invocation Invocation) error {
	command := detachedCommand(invocation)
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

func detachedCommand(invocation Invocation) *exec.Cmd {
	command := exec.Command(invocation.Command, invocation.Arguments...)
	backgroundprocess.Configure(command)
	return command
}
