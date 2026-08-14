//go:build updater_integration

package main

import (
	"os"
	"testing"
)

func TestIntegrationUpdaterAcceptsConfiguredLocalSource(t *testing.T) {
	baseURL := "http://127.0.0.1:34567"
	source, err := newIntegrationUpdateSource(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.ValidateAssetURL(baseURL + "/taskai.deb"); err != nil {
		t.Fatalf("integration updater rejected configured local asset: %v", err)
	}
}

func TestIntegrationLauncherDoesNotStartRealInstaller(t *testing.T) {
	launcher := integrationUpdateLauncher{}
	if err := launcher.LaunchInstaller("/tmp/taskai.deb"); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationApplicationUsesDisposableDataDirectory(t *testing.T) {
	directory, cleanup, err := integrationApplicationDataDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(directory); err != nil {
		t.Fatalf("integration data directory does not exist: %v", err)
	}
	cleanup()
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("integration data directory remains after cleanup: %v", err)
	}
}
