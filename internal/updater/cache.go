package updater

import (
	"context"
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

type Cache struct {
	root string
}

const cacheMetadataName = "candidate.json"

type cacheMetadata struct {
	Platform  string    `json:"platform"`
	Candidate Candidate `json:"candidate"`
}

func (cache *Cache) RestoreLatest(currentVersion, platform string) (Candidate, string, bool, error) {
	if cache.root == "" {
		return Candidate{}, "", false, nil
	}
	current := canonicalVersion(currentVersion)
	if !semver.IsValid(current) || platform == "" {
		return Candidate{}, "", false, nil
	}
	entries, err := os.ReadDir(cache.root)
	if os.IsNotExist(err) {
		return Candidate{}, "", false, nil
	}
	if err != nil {
		return Candidate{}, "", false, fmt.Errorf("读取更新缓存目录: %w", err)
	}

	var latest Candidate
	var latestPath string
	for _, entry := range entries {
		entryPath := filepath.Join(cache.root, entry.Name())
		if !entry.IsDir() {
			if err := os.RemoveAll(entryPath); err != nil {
				return Candidate{}, "", false, fmt.Errorf("清理无效更新缓存 %s: %w", entry.Name(), err)
			}
			continue
		}
		candidate, path, ok, err := cache.restoreVersion(current, platform, entry.Name())
		if err != nil {
			return Candidate{}, "", false, err
		}
		if !ok {
			if err := os.RemoveAll(entryPath); err != nil {
				return Candidate{}, "", false, fmt.Errorf("清理无效更新缓存 %s: %w", entry.Name(), err)
			}
			continue
		}
		if latest.Version == "" || semver.Compare(candidate.Version, latest.Version) > 0 {
			latest = candidate
			latestPath = path
		}
	}

	for _, entry := range entries {
		if latest.Version != "" && entry.IsDir() && entry.Name() == latest.Version {
			continue
		}
		entryPath := filepath.Join(cache.root, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return Candidate{}, "", false, fmt.Errorf("清理过期更新缓存 %s: %w", entry.Name(), err)
		}
	}
	if latest.Version == "" {
		return Candidate{}, "", false, nil
	}
	return latest, latestPath, true, nil
}

func (cache *Cache) restoreVersion(currentVersion, platform, versionDirectory string) (Candidate, string, bool, error) {
	directory := cache.VersionDirectory(versionDirectory)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return Candidate{}, "", false, fmt.Errorf("读取版本缓存 %s: %w", versionDirectory, err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".part") {
			continue
		}
		if err := os.RemoveAll(filepath.Join(directory, entry.Name())); err != nil {
			return Candidate{}, "", false, fmt.Errorf("清理下载临时缓存 %s: %w", entry.Name(), err)
		}
	}

	data, err := os.ReadFile(filepath.Join(directory, cacheMetadataName))
	if os.IsNotExist(err) {
		return Candidate{}, "", false, nil
	}
	if err != nil {
		return Candidate{}, "", false, fmt.Errorf("读取更新缓存元数据 %s: %w", versionDirectory, err)
	}
	var metadata cacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return Candidate{}, "", false, nil
	}
	candidate := metadata.Candidate
	if metadata.Platform != platform || candidate.Version != versionDirectory || !validCachedCandidate(candidate) || semver.Compare(candidate.Version, currentVersion) <= 0 {
		return Candidate{}, "", false, nil
	}

	for _, entry := range entries {
		if (entry.Name() == candidate.Asset.Name || entry.Name() == cacheMetadataName) && !entry.IsDir() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(directory, entry.Name())); err != nil {
			return Candidate{}, "", false, fmt.Errorf("清理版本缓存文件 %s: %w", entry.Name(), err)
		}
	}

	path := filepath.Join(directory, candidate.Asset.Name)
	actual, err := describeAsset(path)
	if err == nil && actual.Size == candidate.Asset.Size && actual.SHA256 == candidate.Asset.SHA256 {
		return candidate, path, true, nil
	}
	return Candidate{}, "", false, nil
}

func NewCache(root string) *Cache {
	return &Cache{root: root}
}

func (cache *Cache) VersionDirectory(version string) string {
	return filepath.Join(cache.root, version)
}

func (cache *Cache) Store(ctx context.Context, version string, asset ManifestAsset, reader io.Reader) (string, error) {
	if cache.root == "" {
		return "", fmt.Errorf("更新缓存目录不能为空")
	}
	if !semver.IsValid(version) {
		return "", fmt.Errorf("无效的缓存版本: %s", version)
	}
	if !validManifestAsset(asset) || filepath.Base(asset.Name) != asset.Name {
		return "", fmt.Errorf("无效的安装包描述: %s", asset.Name)
	}

	directory := cache.VersionDirectory(version)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("创建更新缓存目录: %w", err)
	}
	finalPath := filepath.Join(directory, asset.Name)
	partPath := finalPath + ".part"
	if err := os.Remove(partPath); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("清理旧下载临时文件: %w", err)
	}

	file, err := os.OpenFile(partPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("创建下载临时文件: %w", err)
	}
	committed := false
	defer func() {
		file.Close()
		if !committed {
			_ = os.Remove(partPath)
		}
	}()

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), contextReader{ctx: ctx, reader: reader})
	if err != nil {
		return "", fmt.Errorf("写入安装包: %w", err)
	}
	if written != asset.Size {
		return "", fmt.Errorf("安装包大小不匹配: 实际 %d，预期 %d", written, asset.Size)
	}
	actualHash := hex.EncodeToString(hash.Sum(nil))
	if actualHash != asset.SHA256 {
		return "", fmt.Errorf("安装包 SHA-256 不匹配")
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("同步安装包缓存: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("关闭安装包缓存: %w", err)
	}
	if err := os.Remove(finalPath); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("替换旧安装包: %w", err)
	}
	if err := os.Rename(partPath, finalPath); err != nil {
		return "", fmt.Errorf("提交安装包缓存: %w", err)
	}
	committed = true
	return finalPath, nil
}

func (cache *Cache) StoreCandidate(ctx context.Context, candidate Candidate, platform string, reader io.Reader) (string, error) {
	if platform == "" || !validCachedCandidate(candidate) {
		return "", fmt.Errorf("无效的更新缓存候选: %s", candidate.Version)
	}
	path, err := cache.Store(ctx, candidate.Version, candidate.Asset, reader)
	if err != nil {
		return "", err
	}
	metadata := cacheMetadata{Platform: platform, Candidate: candidate}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		_ = os.RemoveAll(cache.VersionDirectory(candidate.Version))
		return "", fmt.Errorf("编码更新缓存元数据: %w", err)
	}
	data = append(data, '\n')
	metadataPath := filepath.Join(cache.VersionDirectory(candidate.Version), cacheMetadataName)
	if err := writeCacheMetadata(metadataPath, data); err != nil {
		_ = os.RemoveAll(cache.VersionDirectory(candidate.Version))
		return "", err
	}
	return path, nil
}

func validCachedCandidate(candidate Candidate) bool {
	return semver.IsValid(candidate.Version) &&
		candidate.Version == candidate.Tag &&
		candidate.ReleaseURL == OfficialReleasePrefix+candidate.Tag &&
		candidate.DownloadURL != "" &&
		validManifestAsset(candidate.Asset) &&
		candidate.Asset.Name != cacheMetadataName &&
		filepath.Base(candidate.Asset.Name) == candidate.Asset.Name
}

func writeCacheMetadata(path string, data []byte) error {
	partPath := path + ".part"
	if err := os.Remove(partPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理旧缓存元数据临时文件: %w", err)
	}
	file, err := os.OpenFile(partPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("创建缓存元数据临时文件: %w", err)
	}
	committed := false
	defer func() {
		_ = file.Close()
		if !committed {
			_ = os.Remove(partPath)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("写入更新缓存元数据: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("同步更新缓存元数据: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭更新缓存元数据: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("替换旧更新缓存元数据: %w", err)
	}
	if err := os.Rename(partPath, path); err != nil {
		return fmt.Errorf("提交更新缓存元数据: %w", err)
	}
	committed = true
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextReader) Read(buffer []byte) (int, error) {
	select {
	case <-reader.ctx.Done():
		return 0, reader.ctx.Err()
	default:
		return reader.reader.Read(buffer)
	}
}
