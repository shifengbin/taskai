package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubSourceListsReleasesAndFetchesManifest(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/shifengbin/taskai/releases":
			writer.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(writer, `[{
				"tag_name":"v1.2.3-rc.1",
				"html_url":"%sv1.2.3-rc.1",
				"draft":false,
				"prerelease":true,
				"assets":[{"name":"taskai-update.json","browser_download_url":"%s/manifest"}]
			}]`, OfficialReleasePrefix, server.URL)
		case "/manifest":
			writer.Header().Set("Content-Type", "application/json")
			fmt.Fprint(writer, `{
				"schemaVersion":1,
				"version":"1.2.3-rc.1",
				"tag":"v1.2.3-rc.1",
				"releaseUrl":"https://github.com/shifengbin/taskai/releases/tag/v1.2.3-rc.1",
				"assets":{}
			}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	source, err := NewGitHubSource(server.Client(), server.URL, server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	releases, err := source.ListReleases(context.Background())
	if err != nil {
		t.Fatalf("ListReleases() error = %v", err)
	}
	if len(releases) != 1 || releases[0].TagName != "v1.2.3-rc.1" || !releases[0].Prerelease {
		t.Fatalf("releases = %#v", releases)
	}
	manifest, err := source.FetchManifest(context.Background(), releases[0])
	if err != nil {
		t.Fatalf("FetchManifest() error = %v", err)
	}
	if manifest.Tag != "v1.2.3-rc.1" {
		t.Fatalf("manifest = %#v", manifest)
	}
}

func TestGitHubSourceRejectsHTTPErrorAndExternalAssetURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	source, err := NewGitHubSource(server.Client(), server.URL, server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.ListReleases(context.Background()); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("ListReleases() error = %v, want 403", err)
	}

	release := Release{Assets: []ReleaseAsset{{Name: "taskai-update.json", DownloadURL: "https://example.com/manifest"}}}
	if _, err := source.FetchManifest(context.Background(), release); err == nil {
		t.Fatal("FetchManifest() accepted external asset URL")
	}
}
