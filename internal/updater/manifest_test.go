package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManifestDescribesReleaseAssets(t *testing.T) {
	testDir := t.TempDir()
	assetPaths := map[string]string{
		"linux-amd64":      writeTestAsset(t, testDir, "taskai_1.2.3-rc.1_amd64.deb", "linux-package"),
		"windows-amd64":    writeTestAsset(t, testDir, "taskai-amd64-installer.exe", "windows-package"),
		"darwin-universal": writeTestAsset(t, testDir, "TaskAI-1.2.3-rc.1-universal.dmg", "macos-package"),
	}

	manifest, err := BuildManifest(
		"1.2.3-rc.1",
		"v1.2.3-rc.1",
		"https://github.com/shifengbin/taskai/releases/tag/v1.2.3-rc.1",
		assetPaths,
	)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	if manifest.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", manifest.SchemaVersion)
	}
	if manifest.Version != "1.2.3-rc.1" || manifest.Tag != "v1.2.3-rc.1" {
		t.Fatalf("manifest version identity = %#v", manifest)
	}
	if manifest.ReleaseURL != "https://github.com/shifengbin/taskai/releases/tag/v1.2.3-rc.1" {
		t.Fatalf("ReleaseURL = %q", manifest.ReleaseURL)
	}
	for platform, path := range assetPaths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(content)
		asset := manifest.Assets[platform]
		if asset.Name != filepath.Base(path) {
			t.Errorf("%s name = %q, want %q", platform, asset.Name, filepath.Base(path))
		}
		if asset.Size != int64(len(content)) {
			t.Errorf("%s size = %d, want %d", platform, asset.Size, len(content))
		}
		if asset.SHA256 != hex.EncodeToString(digest[:]) {
			t.Errorf("%s sha256 = %q", platform, asset.SHA256)
		}
	}
}

func TestBuildManifestRejectsInvalidReleaseIdentity(t *testing.T) {
	assetPaths := completeTestAssets(t)
	tests := []struct {
		name       string
		version    string
		tag        string
		releaseURL string
	}{
		{name: "invalid version", version: "release_1", tag: "vrelease_1", releaseURL: "https://github.com/shifengbin/taskai/releases/tag/vrelease_1"},
		{name: "tag mismatch", version: "1.2.3", tag: "v1.2.4", releaseURL: "https://github.com/shifengbin/taskai/releases/tag/v1.2.4"},
		{name: "external release", version: "1.2.3", tag: "v1.2.3", releaseURL: "https://example.com/releases/v1.2.3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := BuildManifest(test.version, test.tag, test.releaseURL, assetPaths); err == nil {
				t.Fatal("BuildManifest() error = nil, want validation error")
			}
		})
	}
}

func TestBuildManifestRejectsMissingPlatformAsset(t *testing.T) {
	assetPaths := completeTestAssets(t)
	delete(assetPaths, "windows-amd64")

	if _, err := BuildManifest(
		"1.2.3",
		"v1.2.3",
		"https://github.com/shifengbin/taskai/releases/tag/v1.2.3",
		assetPaths,
	); err == nil {
		t.Fatal("BuildManifest() error = nil, want missing asset error")
	}
}

func completeTestAssets(t *testing.T) map[string]string {
	t.Helper()
	testDir := t.TempDir()
	return map[string]string{
		"linux-amd64":      writeTestAsset(t, testDir, "taskai_1.2.3_amd64.deb", "linux"),
		"windows-amd64":    writeTestAsset(t, testDir, "taskai-amd64-installer.exe", "windows"),
		"darwin-universal": writeTestAsset(t, testDir, "TaskAI-1.2.3-universal.dmg", "macos"),
	}
}

func writeTestAsset(t *testing.T, directory, name, content string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
