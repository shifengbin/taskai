//go:build !updater_integration

package main

import "testing"

func TestProductionUpdaterIgnoresTestURLOverride(t *testing.T) {
	t.Setenv("TASKAI_UPDATE_TEST_URL", "http://127.0.0.1:1")
	source, err := newProductionUpdateSource()
	if err != nil {
		t.Fatal(err)
	}
	if err := source.ValidateAssetURL("http://127.0.0.1:1/taskai.deb"); err == nil {
		t.Fatal("production updater accepted TASKAI_UPDATE_TEST_URL asset")
	}
}
