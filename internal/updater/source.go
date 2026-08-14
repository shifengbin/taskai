package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	officialAPIBase          = "https://api.github.com"
	officialAssetURLPrefix   = "https://github.com/shifengbin/taskai/releases/download/"
	releasesResponseMaxBytes = 4 << 20
	manifestMaxBytes         = 1 << 20
)

type GitHubSource struct {
	client             *http.Client
	apiBase            *url.URL
	allowedAssetPrefix *url.URL
}

func NewOfficialGitHubSource(client *http.Client) (*GitHubSource, error) {
	return NewGitHubSource(client, officialAPIBase, officialAssetURLPrefix)
}

func NewGitHubSource(client *http.Client, apiBase, allowedAssetPrefix string) (*GitHubSource, error) {
	baseURL, err := parseAbsoluteHTTPURL(apiBase)
	if err != nil {
		return nil, fmt.Errorf("无效的 GitHub API 地址: %w", err)
	}
	assetPrefixURL, err := parseAbsoluteHTTPURL(allowedAssetPrefix)
	if err != nil {
		return nil, fmt.Errorf("无效的 Release 资源地址前缀: %w", err)
	}
	if !strings.HasSuffix(assetPrefixURL.Path, "/") {
		return nil, fmt.Errorf("Release 资源地址前缀必须以 / 结尾")
	}
	if client == nil {
		client = http.DefaultClient
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/")
	return &GitHubSource{client: client, apiBase: baseURL, allowedAssetPrefix: assetPrefixURL}, nil
}

func (source *GitHubSource) ListReleases(ctx context.Context) ([]Release, error) {
	endpoint := *source.apiBase
	endpoint.Path += "/repos/shifengbin/taskai/releases"
	query := endpoint.Query()
	query.Set("per_page", "100")
	endpoint.RawQuery = query.Encode()

	response, err := source.get(ctx, endpoint.String())
	if err != nil {
		return nil, fmt.Errorf("获取 GitHub Releases: %w", err)
	}
	defer response.Body.Close()

	var releases []Release
	if err := json.NewDecoder(io.LimitReader(response.Body, releasesResponseMaxBytes)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("解析 GitHub Releases: %w", err)
	}
	return releases, nil
}

func (source *GitHubSource) FetchManifest(ctx context.Context, release Release) (Manifest, error) {
	var manifestURL string
	for _, asset := range release.Assets {
		if asset.Name == "taskai-update.json" {
			manifestURL = asset.DownloadURL
			break
		}
	}
	if manifestURL == "" {
		return Manifest{}, fmt.Errorf("Release %s 缺少 taskai-update.json", release.TagName)
	}
	if err := source.ValidateAssetURL(manifestURL); err != nil {
		return Manifest{}, err
	}

	response, err := source.get(ctx, manifestURL)
	if err != nil {
		return Manifest{}, fmt.Errorf("下载更新清单: %w", err)
	}
	defer response.Body.Close()

	var manifest Manifest
	if err := json.NewDecoder(io.LimitReader(response.Body, manifestMaxBytes)).Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("解析更新清单: %w", err)
	}
	return manifest, nil
}

func (source *GitHubSource) OpenAsset(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	if err := source.ValidateAssetURL(rawURL); err != nil {
		return nil, err
	}
	response, err := source.get(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("下载安装包: %w", err)
	}
	return response.Body, nil
}

func (source *GitHubSource) ValidateAssetURL(rawURL string) error {
	assetURL, err := parseAbsoluteHTTPURL(rawURL)
	if err != nil {
		return fmt.Errorf("无效的 Release 资源地址: %w", err)
	}
	prefix := source.allowedAssetPrefix
	if assetURL.Scheme != prefix.Scheme || assetURL.Host != prefix.Host || !strings.HasPrefix(assetURL.Path, prefix.Path) {
		return fmt.Errorf("拒绝非官方 Release 资源地址: %s", rawURL)
	}
	return nil
}

func (source *GitHubSource) get(ctx context.Context, endpoint string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "taskai-updater")
	response, err := source.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("HTTP %d %s", response.StatusCode, response.Status)
	}
	return response, nil
}

func parseAbsoluteHTTPURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return nil, fmt.Errorf("必须是绝对 HTTP(S) 地址")
	}
	return parsed, nil
}
