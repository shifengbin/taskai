package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	ManifestSchemaVersion = 1
	OfficialReleasePrefix = "https://github.com/shifengbin/taskai/releases/tag/"
)

var SupportedPlatformKeys = []string{
	"linux-amd64",
	"windows-amd64",
	"darwin-universal",
}

type Manifest struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Version       string                   `json:"version"`
	Tag           string                   `json:"tag"`
	ReleaseURL    string                   `json:"releaseUrl"`
	Assets        map[string]ManifestAsset `json:"assets"`
}

type ManifestAsset struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func BuildManifest(version, tag, releaseURL string, assetPaths map[string]string) (Manifest, error) {
	canonicalVersion := canonicalVersion(version)
	if !semver.IsValid(canonicalVersion) {
		return Manifest{}, fmt.Errorf("无效的语义版本: %s", version)
	}
	if tag != canonicalVersion {
		return Manifest{}, fmt.Errorf("tag %q 与版本 %q 不一致", tag, version)
	}
	if releaseURL != OfficialReleasePrefix+tag {
		return Manifest{}, fmt.Errorf("Release 页面不是官方版本页面: %s", releaseURL)
	}

	assets := make(map[string]ManifestAsset, len(SupportedPlatformKeys))
	for _, platform := range SupportedPlatformKeys {
		path := assetPaths[platform]
		if path == "" {
			return Manifest{}, fmt.Errorf("缺少 %s 安装包", platform)
		}
		asset, err := describeAsset(path)
		if err != nil {
			return Manifest{}, fmt.Errorf("读取 %s 安装包: %w", platform, err)
		}
		assets[platform] = asset
	}

	return Manifest{
		SchemaVersion: ManifestSchemaVersion,
		Version:       strings.TrimPrefix(canonicalVersion, "v"),
		Tag:           canonicalVersion,
		ReleaseURL:    releaseURL,
		Assets:        assets,
	}, nil
}

func WriteManifest(path string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("编码更新清单: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("写入更新清单: %w", err)
	}
	return nil
}

func canonicalVersion(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func describeAsset(path string) (ManifestAsset, error) {
	file, err := os.Open(path)
	if err != nil {
		return ManifestAsset{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return ManifestAsset{}, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 {
		return ManifestAsset{}, fmt.Errorf("%s 不是非空普通文件", path)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ManifestAsset{}, err
	}
	return ManifestAsset{
		Name:   filepath.Base(path),
		Size:   info.Size(),
		SHA256: hex.EncodeToString(hash.Sum(nil)),
	}, nil
}
