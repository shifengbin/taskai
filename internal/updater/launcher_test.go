package updater

import (
	"reflect"
	"testing"
)

func TestInstallerInvocationUsesPlatformCommandWithoutShell(t *testing.T) {
	tests := []struct {
		platform string
		path     string
		want     Invocation
	}{
		{
			platform: "windows",
			path:     `C:\Users\tester\Task AI\taskai-amd64-installer.exe`,
			want:     Invocation{Command: `C:\Users\tester\Task AI\taskai-amd64-installer.exe`},
		},
		{
			platform: "darwin",
			path:     "/Users/tester/Task AI/TaskAI.dmg",
			want:     Invocation{Command: "open", Arguments: []string{"/Users/tester/Task AI/TaskAI.dmg"}},
		},
		{
			platform: "linux",
			path:     "/home/tester/Task AI/taskai.deb",
			want:     Invocation{Command: "xdg-open", Arguments: []string{"/home/tester/Task AI/taskai.deb"}},
		},
	}
	for _, test := range tests {
		t.Run(test.platform, func(t *testing.T) {
			got, err := InstallerInvocation(test.platform, test.path)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("InstallerInvocation() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestInstallerInvocationRejectsWrongPackageType(t *testing.T) {
	for _, test := range []struct {
		platform string
		path     string
	}{
		{platform: "windows", path: "taskai.zip"},
		{platform: "darwin", path: "TaskAI.pkg"},
		{platform: "linux", path: "taskai.AppImage"},
	} {
		if _, err := InstallerInvocation(test.platform, test.path); err == nil {
			t.Errorf("InstallerInvocation(%q, %q) accepted wrong package type", test.platform, test.path)
		}
	}
}

func TestReleasePageInvocationUsesExactOfficialURL(t *testing.T) {
	url := OfficialReleasePrefix + "v1.2.3-rc.1"
	tests := []struct {
		platform string
		want     Invocation
	}{
		{platform: "windows", want: Invocation{Command: "rundll32.exe", Arguments: []string{"url.dll,FileProtocolHandler", url}}},
		{platform: "darwin", want: Invocation{Command: "open", Arguments: []string{url}}},
		{platform: "linux", want: Invocation{Command: "xdg-open", Arguments: []string{url}}},
	}
	for _, test := range tests {
		got, err := ReleasePageInvocation(test.platform, url)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("ReleasePageInvocation(%q) = %#v, want %#v", test.platform, got, test.want)
		}
	}
	if _, err := ReleasePageInvocation("linux", "https://example.com/download"); err == nil {
		t.Fatal("ReleasePageInvocation() accepted non-official URL")
	}
}
