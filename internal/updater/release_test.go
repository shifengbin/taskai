package updater

import "testing"

func TestSelectCandidateUsesHighestSemanticVersionIncludingPrereleases(t *testing.T) {
	releases := []Release{
		testRelease("v1.2.3-beta.2", false, "beta.exe"),
		testRelease("v1.2.3", false, "stable.exe"),
		testRelease("v1.2.3-rc.1", false, "rc.exe"),
		testRelease("v1.3.0-rc.1", true, "draft.exe"),
	}
	manifests := map[string]Manifest{
		"v1.2.3-beta.2": testManifest("v1.2.3-beta.2", "windows-amd64", "beta.exe"),
		"v1.2.3-rc.1":   testManifest("v1.2.3-rc.1", "windows-amd64", "rc.exe"),
		"v1.2.3":        testManifest("v1.2.3", "windows-amd64", "stable.exe"),
		"v1.3.0-rc.1":   testManifest("v1.3.0-rc.1", "windows-amd64", "draft.exe"),
	}

	candidate, ok := SelectCandidate("v1.2.3-beta.1", "windows-amd64", releases, manifests)
	if !ok {
		t.Fatal("SelectCandidate() ok = false")
	}
	if candidate.Version != "v1.2.3" || candidate.Asset.Name != "stable.exe" {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestSelectCandidateSkipsInvalidOldAndMismatchedReleases(t *testing.T) {
	releases := []Release{
		testRelease("not-semver", false, "invalid.exe"),
		testRelease("v1.0.0", false, "old.exe"),
		testRelease("v1.3.0", false, "mismatch.exe"),
		testRelease("v1.2.0", false, "valid.exe"),
	}
	manifests := map[string]Manifest{
		"not-semver": testManifest("v9.0.0", "windows-amd64", "invalid.exe"),
		"v1.0.0":     testManifest("v1.0.0", "windows-amd64", "old.exe"),
		"v1.3.0":     testManifest("v1.3.1", "windows-amd64", "mismatch.exe"),
		"v1.2.0":     testManifest("v1.2.0", "windows-amd64", "valid.exe"),
	}

	candidate, ok := SelectCandidate("v1.1.0", "windows-amd64", releases, manifests)
	if !ok || candidate.Version != "v1.2.0" {
		t.Fatalf("candidate = %#v, ok = %v", candidate, ok)
	}
}

func TestSelectCandidateRequiresCurrentPlatformAssetInRelease(t *testing.T) {
	release := testRelease("v2.0.0", false, "taskai.deb")
	manifest := testManifest("v2.0.0", "linux-amd64", "taskai.deb")

	if _, ok := SelectCandidate("v1.0.0", "windows-amd64", []Release{release}, map[string]Manifest{release.TagName: manifest}); ok {
		t.Fatal("SelectCandidate() ok = true, want no Windows candidate")
	}

	manifest.Assets["linux-amd64"] = ManifestAsset{Name: "other.deb", Size: 5, SHA256: testSHA256}
	if _, ok := SelectCandidate("v1.0.0", "linux-amd64", []Release{release}, map[string]Manifest{release.TagName: manifest}); ok {
		t.Fatal("SelectCandidate() accepted asset name missing from Release")
	}
}

func TestPlatformKeySupportsPublishedTargets(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{goos: "linux", goarch: "amd64", want: "linux-amd64"},
		{goos: "windows", goarch: "amd64", want: "windows-amd64"},
		{goos: "darwin", goarch: "amd64", want: "darwin-universal"},
		{goos: "darwin", goarch: "arm64", want: "darwin-universal"},
		{goos: "linux", goarch: "arm64", want: ""},
	}
	for _, test := range tests {
		if got := PlatformKey(test.goos, test.goarch); got != test.want {
			t.Errorf("PlatformKey(%q, %q) = %q, want %q", test.goos, test.goarch, got, test.want)
		}
	}
}

const testSHA256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testRelease(tag string, draft bool, assetName string) Release {
	return Release{
		TagName: tag,
		HTMLURL: OfficialReleasePrefix + tag,
		Draft:   draft,
		Assets: []ReleaseAsset{
			{Name: "taskai-update.json", DownloadURL: "https://github.com/shifengbin/taskai/releases/download/" + tag + "/taskai-update.json"},
			{Name: assetName, DownloadURL: "https://github.com/shifengbin/taskai/releases/download/" + tag + "/" + assetName},
		},
	}
}

func testManifest(tag, platform, assetName string) Manifest {
	return Manifest{
		SchemaVersion: ManifestSchemaVersion,
		Version:       tag[1:],
		Tag:           tag,
		ReleaseURL:    OfficialReleasePrefix + tag,
		Assets: map[string]ManifestAsset{
			platform: {Name: assetName, Size: 5, SHA256: testSHA256},
		},
	}
}
