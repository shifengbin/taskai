package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskai/internal/updater"
)

func TestUpdateTestServerFailsFirstInstallerRequestThenServesVerifiedPayload(t *testing.T) {
	server := httptest.NewServer(newUpdateTestHandlerWithDelay(0))
	defer server.Close()

	source, err := updater.NewGitHubSource(server.Client(), server.URL, server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	releases, err := source.ListReleases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].TagName != integrationTargetTag {
		t.Fatalf("releases = %#v", releases)
	}
	manifest, err := source.FetchManifest(context.Background(), releases[0])
	if err != nil {
		t.Fatal(err)
	}
	candidate, ok := updater.SelectCandidate("v0.0.0-rc5", integrationPlatform, releases, map[string]updater.Manifest{integrationTargetTag: manifest})
	if !ok {
		t.Fatal("expected rc6 candidate for rc5 current version")
	}

	if _, err := source.OpenAsset(context.Background(), candidate.DownloadURL); err == nil {
		t.Fatal("first installer request succeeded, want HTTP failure")
	}
	body, err := source.OpenAsset(context.Background(), candidate.DownloadURL)
	if err != nil {
		t.Fatalf("second installer request failed: %v", err)
	}
	payload, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	if int64(len(payload)) != candidate.Asset.Size || hex.EncodeToString(digest[:]) != candidate.Asset.SHA256 {
		t.Fatalf("payload does not match manifest: size=%d sha256=%s asset=%#v", len(payload), hex.EncodeToString(digest[:]), candidate.Asset)
	}

	response, err := server.Client().Get(server.URL + requestCountPath)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var counts requestCounts
	if err := json.NewDecoder(response.Body).Decode(&counts); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || counts.Installer != 2 || counts.Releases != 1 || counts.Manifest != 1 {
		t.Fatalf("request counts = %#v, status = %d", counts, response.StatusCode)
	}
}
