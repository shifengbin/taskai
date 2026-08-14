package updater

import (
	"encoding/hex"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

type Release struct {
	TagName    string         `json:"tag_name"`
	HTMLURL    string         `json:"html_url"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []ReleaseAsset `json:"assets"`
}

type ReleaseAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

type Candidate struct {
	Version     string        `json:"version"`
	Tag         string        `json:"tag"`
	ReleaseURL  string        `json:"releaseUrl"`
	Asset       ManifestAsset `json:"asset"`
	DownloadURL string        `json:"downloadUrl"`
}

func PlatformKey(goos, goarch string) string {
	switch {
	case goos == "linux" && goarch == "amd64":
		return "linux-amd64"
	case goos == "windows" && goarch == "amd64":
		return "windows-amd64"
	case goos == "darwin" && (goarch == "amd64" || goarch == "arm64"):
		return "darwin-universal"
	default:
		return ""
	}
}

func SelectCandidate(currentVersion, platform string, releases []Release, manifests map[string]Manifest) (Candidate, bool) {
	current := canonicalVersion(currentVersion)
	if !semver.IsValid(current) || platform == "" {
		return Candidate{}, false
	}

	ordered := append([]Release(nil), releases...)
	sort.SliceStable(ordered, func(left, right int) bool {
		return semver.Compare(ordered[left].TagName, ordered[right].TagName) > 0
	})

	for _, release := range ordered {
		if release.Draft || !semver.IsValid(release.TagName) || semver.Compare(release.TagName, current) <= 0 {
			continue
		}
		manifest, ok := manifests[release.TagName]
		if !ok || !validManifestForRelease(manifest, release) {
			continue
		}
		asset, ok := manifest.Assets[platform]
		if !ok || !validManifestAsset(asset) {
			continue
		}
		for _, releaseAsset := range release.Assets {
			if releaseAsset.Name == asset.Name && releaseAsset.DownloadURL != "" {
				return Candidate{
					Version:     canonicalVersion(manifest.Version),
					Tag:         release.TagName,
					ReleaseURL:  release.HTMLURL,
					Asset:       asset,
					DownloadURL: releaseAsset.DownloadURL,
				}, true
			}
		}
	}
	return Candidate{}, false
}

func validManifestForRelease(manifest Manifest, release Release) bool {
	version := canonicalVersion(manifest.Version)
	return manifest.SchemaVersion == ManifestSchemaVersion &&
		semver.IsValid(version) &&
		version == release.TagName &&
		manifest.Tag == release.TagName &&
		release.HTMLURL == OfficialReleasePrefix+release.TagName &&
		manifest.ReleaseURL == release.HTMLURL
}

func validManifestAsset(asset ManifestAsset) bool {
	if asset.Name == "" || asset.Size <= 0 || len(asset.SHA256) != 64 || asset.SHA256 != strings.ToLower(asset.SHA256) {
		return false
	}
	decoded, err := hex.DecodeString(asset.SHA256)
	return err == nil && len(decoded) == 32
}
