package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheRestoreRevalidatesInstallerOnEveryStartup(t *testing.T) {
	content := []byte("valid installer")
	candidate := testCandidate("v1.2.0", "taskai.deb", content)
	cache := NewCache(t.TempDir())
	path, err := cache.StoreCandidate(context.Background(), candidate, "linux-amd64", strings.NewReader(string(content)))
	if err != nil {
		t.Fatal(err)
	}

	restored, restoredPath, ok, err := cache.RestoreLatest("v1.0.0", "linux-amd64")
	if err != nil || !ok || restoredPath != path {
		t.Fatalf("RestoreLatest() = %#v, %q, %v, %v", restored, restoredPath, ok, err)
	}
	if restored != candidate {
		t.Fatalf("restored candidate = %#v, want %#v", restored, candidate)
	}
	corrupted := append([]byte(nil), content...)
	corrupted[0] ^= 0xff
	if err := os.WriteFile(path, corrupted, 0o600); err != nil {
		t.Fatal(err)
	}
	if restored, restoredPath, ok, err := cache.RestoreLatest("v1.0.0", "linux-amd64"); err != nil || ok || restoredPath != "" || restored != (Candidate{}) {
		t.Fatalf("RestoreLatest(corrupted) = %#v, %q, %v, %v", restored, restoredPath, ok, err)
	}
	if _, err := os.Stat(cache.VersionDirectory(candidate.Version)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("corrupted cache directory still exists: %v", err)
	}
}

func TestCacheRestoreRemovesPartsAndNonCandidateVersions(t *testing.T) {
	root := t.TempDir()
	cache := NewCache(root)
	content := []byte("target installer")
	candidate := testCandidate("v1.2.0", "taskai-amd64-installer.exe", content)
	if _, err := cache.StoreCandidate(context.Background(), candidate, "windows-amd64", strings.NewReader(string(content))); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v0.9.0", "v1.0.0", "v1.3.0", "invalid"} {
		directory := cache.VersionDirectory(version)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "stale.part"), []byte("partial"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	targetPart := filepath.Join(cache.VersionDirectory("v1.2.0"), "old.part")
	if err := os.WriteFile(targetPart, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}

	restored, path, ok, err := cache.RestoreLatest("v1.0.0", "windows-amd64")
	if err != nil || !ok || path == "" {
		t.Fatalf("RestoreLatest() = %#v, %q, %v, %v", restored, path, ok, err)
	}
	if restored != candidate {
		t.Fatalf("restored candidate = %#v, want %#v", restored, candidate)
	}
	for _, removed := range []string{"v0.9.0", "v1.0.0", "v1.3.0", "invalid"} {
		if _, err := os.Stat(cache.VersionDirectory(removed)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("stale cache %s still exists: %v", removed, err)
		}
	}
	if _, err := os.Stat(targetPart); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("target .part still exists: %v", err)
	}
}

func TestCacheRestoreRejectsCurrentOrOlderCandidate(t *testing.T) {
	content := []byte("old installer")
	candidate := testCandidate("v1.0.0", "taskai.deb", content)
	cache := NewCache(t.TempDir())
	if _, err := cache.StoreCandidate(context.Background(), candidate, "linux-amd64", strings.NewReader(string(content))); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, err := cache.RestoreLatest("v1.0.0", "linux-amd64"); err != nil || ok {
		t.Fatalf("RestoreLatest(current) ok = %v, error = %v", ok, err)
	}
	if _, err := os.Stat(cache.VersionDirectory("v1.0.0")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("current-version cache still exists: %v", err)
	}
}

func TestCacheRestoreKeepsOnlyHighestValidPlatformCandidate(t *testing.T) {
	cache := NewCache(t.TempDir())
	for _, candidate := range []Candidate{
		testCandidate("v1.1.0", "taskai.deb", []byte("older")),
		testCandidate("v1.3.0", "taskai.deb", []byte("latest")),
	} {
		if _, err := cache.StoreCandidate(context.Background(), candidate, "linux-amd64", strings.NewReader(map[string]string{
			"v1.1.0": "older",
			"v1.3.0": "latest",
		}[candidate.Version])); err != nil {
			t.Fatal(err)
		}
	}
	wrongPlatform := testCandidate("v1.4.0", "taskai-amd64-installer.exe", []byte("windows"))
	if _, err := cache.StoreCandidate(context.Background(), wrongPlatform, "windows-amd64", strings.NewReader("windows")); err != nil {
		t.Fatal(err)
	}

	restored, _, ok, err := cache.RestoreLatest("v1.0.0", "linux-amd64")
	if err != nil || !ok || restored.Version != "v1.3.0" {
		t.Fatalf("RestoreLatest() = %#v, %v, %v", restored, ok, err)
	}
	for _, removed := range []string{"v1.1.0", "v1.4.0"} {
		if _, err := os.Stat(cache.VersionDirectory(removed)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("stale cache %s still exists: %v", removed, err)
		}
	}
}

func testCandidate(version, assetName string, content []byte) Candidate {
	return Candidate{
		Version:     version,
		Tag:         version,
		ReleaseURL:  OfficialReleasePrefix + version,
		Asset:       testAsset(assetName, content),
		DownloadURL: "https://github.com/shifengbin/taskai/releases/download/" + version + "/" + assetName,
	}
}
