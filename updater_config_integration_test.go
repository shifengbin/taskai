//go:build updater_integration

package main

import "testing"

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
