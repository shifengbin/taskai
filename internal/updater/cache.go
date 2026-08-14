package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

func (cache *Cache) Restore(currentVersion string, candidate Candidate) (string, bool, error) {
	if cache.root == "" {
		return "", false, nil
	}
	entries, err := os.ReadDir(cache.root)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("读取更新缓存目录: %w", err)
	}

	current := canonicalVersion(currentVersion)
	candidateValid := semver.IsValid(current) && semver.IsValid(candidate.Version) && semver.Compare(candidate.Version, current) > 0
	for _, entry := range entries {
		if candidateValid && entry.IsDir() && entry.Name() == candidate.Version {
			continue
		}
		if err := os.RemoveAll(filepath.Join(cache.root, entry.Name())); err != nil {
			return "", false, fmt.Errorf("清理过期更新缓存 %s: %w", entry.Name(), err)
		}
	}
	if !candidateValid || !validManifestAsset(candidate.Asset) || filepath.Base(candidate.Asset.Name) != candidate.Asset.Name {
		return "", false, nil
	}

	directory := cache.VersionDirectory(candidate.Version)
	entries, err = os.ReadDir(directory)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("读取目标版本缓存: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == candidate.Asset.Name && !entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".part") || entry.Name() != candidate.Asset.Name {
			if err := os.RemoveAll(filepath.Join(directory, entry.Name())); err != nil {
				return "", false, fmt.Errorf("清理目标版本临时缓存 %s: %w", entry.Name(), err)
			}
		}
	}

	path := filepath.Join(directory, candidate.Asset.Name)
	actual, err := describeAsset(path)
	if err == nil && actual.Size == candidate.Asset.Size && actual.SHA256 == candidate.Asset.SHA256 {
		return path, true, nil
	}
	if err := os.RemoveAll(directory); err != nil {
		return "", false, fmt.Errorf("删除损坏的更新缓存: %w", err)
	}
	return "", false, nil
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
