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
	asset := testAsset("taskai.deb", content)
	cache := NewCache(t.TempDir())
	path, err := cache.Store(context.Background(), "v1.2.0", asset, strings.NewReader(string(content)))
	if err != nil {
		t.Fatal(err)
	}
	candidate := Candidate{Version: "v1.2.0", Asset: asset}

	restoredPath, ok, err := cache.Restore("v1.0.0", candidate)
	if err != nil || !ok || restoredPath != path {
		t.Fatalf("Restore() = %q, %v, %v", restoredPath, ok, err)
	}
	corrupted := append([]byte(nil), content...)
	corrupted[0] ^= 0xff
	if err := os.WriteFile(path, corrupted, 0o600); err != nil {
		t.Fatal(err)
	}
	if restoredPath, ok, err := cache.Restore("v1.0.0", candidate); err != nil || ok || restoredPath != "" {
		t.Fatalf("Restore(corrupted) = %q, %v, %v", restoredPath, ok, err)
	}
	if _, err := os.Stat(cache.VersionDirectory(candidate.Version)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("corrupted cache directory still exists: %v", err)
	}
}

func TestCacheRestoreRemovesPartsAndNonCandidateVersions(t *testing.T) {
	root := t.TempDir()
	cache := NewCache(root)
	content := []byte("target installer")
	asset := testAsset("taskai-amd64-installer.exe", content)
	if _, err := cache.Store(context.Background(), "v1.2.0", asset, strings.NewReader(string(content))); err != nil {
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

	path, ok, err := cache.Restore("v1.0.0", Candidate{Version: "v1.2.0", Asset: asset})
	if err != nil || !ok || path == "" {
		t.Fatalf("Restore() = %q, %v, %v", path, ok, err)
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
	asset := testAsset("taskai.deb", content)
	cache := NewCache(t.TempDir())
	if _, err := cache.Store(context.Background(), "v1.0.0", asset, strings.NewReader(string(content))); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := cache.Restore("v1.0.0", Candidate{Version: "v1.0.0", Asset: asset}); err != nil || ok {
		t.Fatalf("Restore(current) ok = %v, error = %v", ok, err)
	}
	if _, err := os.Stat(cache.VersionDirectory("v1.0.0")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("current-version cache still exists: %v", err)
	}
}
