package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

const (
	DefaultCheckInterval = time.Hour
	DefaultCheckTimeout  = 30 * time.Second
)

type Status string

const (
	StatusIdle           Status = "idle"
	StatusAvailable      Status = "available"
	StatusDownloading    Status = "downloading"
	StatusDownloaded     Status = "downloaded"
	StatusDownloadFailed Status = "download_failed"
)

type State struct {
	Status      Status `json:"status"`
	Version     string `json:"version,omitempty"`
	ReleaseURL  string `json:"releaseUrl,omitempty"`
	AssetName   string `json:"assetName,omitempty"`
	Error       string `json:"error,omitempty"`
	InstallPath string `json:"installPath,omitempty"`
}

type ReleaseSource interface {
	ListReleases(context.Context) ([]Release, error)
	FetchManifest(context.Context, Release) (Manifest, error)
	ValidateAssetURL(string) error
	OpenAsset(context.Context, string) (io.ReadCloser, error)
}

type Options struct {
	CurrentVersion string
	Platform       string
	Source         ReleaseSource
	CheckInterval  time.Duration
	CheckTimeout   time.Duration
	CacheDirectory string
	Publish        func(State)
}

type Service struct {
	currentVersion string
	platform       string
	source         ReleaseSource
	checkInterval  time.Duration
	checkTimeout   time.Duration
	publish        func(State)
	cache          *Cache

	checkMu     sync.Mutex
	mu          sync.RWMutex
	state       State
	target      Candidate
	downloading bool
	cancel      context.CancelFunc
	done        chan struct{}
}

var ErrDownloadInProgress = errors.New("安装包正在下载")

func NewService(options Options) (*Service, error) {
	currentVersion := canonicalVersion(options.CurrentVersion)
	if !semver.IsValid(currentVersion) {
		return nil, fmt.Errorf("无效的当前版本: %s", options.CurrentVersion)
	}
	if options.Platform == "" {
		return nil, fmt.Errorf("当前平台不支持自动更新")
	}
	if options.Source == nil {
		return nil, fmt.Errorf("更新源不能为空")
	}
	if options.CheckInterval <= 0 {
		options.CheckInterval = DefaultCheckInterval
	}
	if options.CheckTimeout <= 0 {
		options.CheckTimeout = DefaultCheckTimeout
	}
	return &Service{
		currentVersion: currentVersion,
		platform:       options.Platform,
		source:         options.Source,
		checkInterval:  options.CheckInterval,
		checkTimeout:   options.CheckTimeout,
		publish:        options.Publish,
		cache:          NewCache(options.CacheDirectory),
		state:          State{Status: StatusIdle},
	}, nil
}

func (service *Service) Download(ctx context.Context) error {
	service.mu.Lock()
	if service.downloading {
		service.mu.Unlock()
		return ErrDownloadInProgress
	}
	if service.target.Version == "" {
		service.mu.Unlock()
		return fmt.Errorf("没有可下载的更新")
	}
	service.downloading = true
	candidate := service.target
	state := State{
		Status:     StatusDownloading,
		Version:    candidate.Version,
		ReleaseURL: candidate.ReleaseURL,
		AssetName:  candidate.Asset.Name,
	}
	service.state = state
	service.mu.Unlock()
	service.publishState(state)

	defer func() {
		service.mu.Lock()
		service.downloading = false
		service.mu.Unlock()
	}()

	reader, err := service.source.OpenAsset(ctx, candidate.DownloadURL)
	if err != nil {
		service.setDownloadFailed(candidate, err)
		return err
	}
	defer reader.Close()

	path, err := service.cache.Store(ctx, candidate.Version, candidate.Asset, reader)
	if err != nil {
		service.setDownloadFailed(candidate, err)
		return err
	}
	downloaded := State{
		Status:      StatusDownloaded,
		Version:     candidate.Version,
		ReleaseURL:  candidate.ReleaseURL,
		AssetName:   candidate.Asset.Name,
		InstallPath: path,
	}
	service.mu.Lock()
	service.state = downloaded
	service.mu.Unlock()
	service.publishState(downloaded)
	return nil
}

func (service *Service) Start(parent context.Context) {
	service.mu.Lock()
	if service.cancel != nil {
		service.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	service.cancel = cancel
	service.done = done
	service.mu.Unlock()

	go service.run(ctx, done)
}

func (service *Service) Stop() {
	service.mu.RLock()
	cancel := service.cancel
	done := service.done
	service.mu.RUnlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done

	service.mu.Lock()
	if service.done == done {
		service.cancel = nil
		service.done = nil
	}
	service.mu.Unlock()
}

func (service *Service) State() State {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return service.state
}

func (service *Service) Check(ctx context.Context) error {
	service.checkMu.Lock()
	defer service.checkMu.Unlock()

	checkCtx, cancel := context.WithTimeout(ctx, service.checkTimeout)
	defer cancel()

	releases, err := service.source.ListReleases(checkCtx)
	if err != nil {
		return err
	}
	manifests := make(map[string]Manifest, len(releases))
	var fetchErr error
	for _, release := range releases {
		if release.Draft || !semver.IsValid(release.TagName) || semver.Compare(release.TagName, service.currentVersion) <= 0 {
			continue
		}
		manifest, err := service.source.FetchManifest(checkCtx, release)
		if err != nil {
			fetchErr = err
			if checkCtx.Err() != nil {
				return checkCtx.Err()
			}
			continue
		}
		manifests[release.TagName] = manifest
	}

	for len(manifests) > 0 {
		candidate, ok := SelectCandidate(service.currentVersion, service.platform, releases, manifests)
		if !ok {
			break
		}
		if err := service.source.ValidateAssetURL(candidate.DownloadURL); err != nil {
			delete(manifests, candidate.Tag)
			continue
		}
		service.setCandidate(candidate)
		return nil
	}
	if fetchErr != nil {
		return fetchErr
	}
	service.setIdle()
	return nil
}

func (service *Service) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	service.checkSilently(ctx)
	ticker := time.NewTicker(service.checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			service.checkSilently(ctx)
		}
	}
}

func (service *Service) checkSilently(ctx context.Context) {
	_ = service.Check(ctx)
}

func (service *Service) setCandidate(candidate Candidate) {
	service.mu.RLock()
	preserve := service.target.Tag == candidate.Tag && (service.downloading || service.state.Status == StatusDownloaded || service.state.Status == StatusDownloadFailed)
	service.mu.RUnlock()
	if preserve {
		return
	}

	state := State{
		Status:     StatusAvailable,
		Version:    candidate.Version,
		ReleaseURL: candidate.ReleaseURL,
		AssetName:  candidate.Asset.Name,
	}
	if path, ok, _ := service.cache.Restore(service.currentVersion, candidate); ok {
		state.Status = StatusDownloaded
		state.InstallPath = path
	}
	service.mu.Lock()
	changed := service.state != state
	service.target = candidate
	service.state = state
	service.mu.Unlock()
	if changed && service.publish != nil {
		service.publishState(state)
	}
}

func (service *Service) setIdle() {
	state := State{Status: StatusIdle}
	service.mu.Lock()
	changed := service.state != state
	service.target = Candidate{}
	service.state = state
	service.mu.Unlock()
	if changed && service.publish != nil {
		service.publishState(state)
	}
}

func (service *Service) setDownloadFailed(candidate Candidate, downloadErr error) {
	state := State{
		Status:     StatusDownloadFailed,
		Version:    candidate.Version,
		ReleaseURL: candidate.ReleaseURL,
		AssetName:  candidate.Asset.Name,
		Error:      downloadErr.Error(),
	}
	service.mu.Lock()
	service.state = state
	service.mu.Unlock()
	service.publishState(state)
}

func (service *Service) publishState(state State) {
	if service.publish != nil {
		service.publish(state)
	}
}
