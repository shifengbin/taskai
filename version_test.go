package main

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultAppVersionIsExplicitDevelopmentVersion(t *testing.T) {
	if appVersion != "v0.0.0-dev" {
		t.Fatalf("appVersion = %q, want v0.0.0-dev", appVersion)
	}
}

func TestPlatformBuildScriptsInjectAppVersion(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "scripts/build-linux.sh", want: `-ldflags "-X main.appVersion=$app_version"`},
		{path: "scripts/build-macos.sh", want: `-ldflags "-X main.appVersion=$app_version"`},
		{path: "scripts/build-windows.ps1", want: `$Arguments += @('-ldflags', "-X main.appVersion=$AppVersion")`},
	}
	for _, test := range tests {
		content, err := os.ReadFile(test.path)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", test.path, err)
		}
		if !strings.Contains(string(content), test.want) {
			t.Errorf("%s 未向 Wails 注入 appVersion", test.path)
		}
	}
}

func TestPlatformBuildScriptsDefineDevelopmentVersionFallback(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "scripts/build-linux.sh", want: "0.0.0+git."},
		{path: "scripts/build-macos.sh", want: "0.0.0+git."},
		{path: "scripts/build-windows.ps1", want: "v0.0.0-dev"},
	}
	for _, test := range tests {
		content, err := os.ReadFile(test.path)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", test.path, err)
		}
		if !strings.Contains(string(content), test.want) {
			t.Errorf("%s 缺少明确的开发版本回退", test.path)
		}
	}
}
