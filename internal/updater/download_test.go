package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGitHubSourceOpenAssetRejectsHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	source, err := NewGitHubSource(server.Client(), server.URL, server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.OpenAsset(context.Background(), server.URL+"/installer.deb"); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("OpenAsset() error = %v, want HTTP 503", err)
	}
}

func TestCacheStoreRejectsInterruptedWrongSizeAndWrongHash(t *testing.T) {
	content := []byte("verified installer")
	asset := testAsset("taskai.deb", content)
	tests := []struct {
		name   string
		reader io.Reader
		asset  ManifestAsset
	}{
		{
			name:   "interrupted",
			reader: io.MultiReader(strings.NewReader("partial"), failingReader{}),
			asset:  asset,
		},
		{
			name:   "wrong size",
			reader: strings.NewReader(string(content)),
			asset:  ManifestAsset{Name: asset.Name, Size: asset.Size + 1, SHA256: asset.SHA256},
		},
		{
			name:   "wrong hash",
			reader: strings.NewReader(string(content)),
			asset:  ManifestAsset{Name: asset.Name, Size: asset.Size, SHA256: strings.Repeat("0", 64)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := NewCache(t.TempDir())
			path, err := cache.Store(context.Background(), "v1.2.3", test.asset, test.reader)
			if err == nil {
				t.Fatalf("Store() path = %q, want error", path)
			}
			assertNoInstallOrPartFile(t, cache.VersionDirectory("v1.2.3"), test.asset.Name)
		})
	}
}

func TestCacheStoreAtomicallyCommitsVerifiedInstaller(t *testing.T) {
	content := []byte("verified installer")
	asset := testAsset("taskai-amd64-installer.exe", content)
	cache := NewCache(t.TempDir())

	path, err := cache.Store(context.Background(), "v1.2.3-rc.1", asset, strings.NewReader(string(content)))
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("cached content = %q", got)
	}
	if _, err := os.Stat(path + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary file still exists: %v", err)
	}
}

func TestServiceRejectsConcurrentDownload(t *testing.T) {
	content := []byte("installer")
	asset := testAsset("taskai.deb", content)
	started := make(chan struct{})
	releaseRead := make(chan struct{})
	var opens atomic.Int32
	source := &scriptedReleaseSource{
		list: func(context.Context) ([]Release, error) { return nil, nil },
		open: func(context.Context, string) (io.ReadCloser, error) {
			opens.Add(1)
			return io.NopCloser(&blockingReader{started: started, release: releaseRead, content: content}), nil
		},
	}
	service, err := NewService(Options{
		CurrentVersion: "v1.0.0",
		Platform:       "linux-amd64",
		Source:         source,
		CacheDirectory: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	service.setCandidate(Candidate{
		Version:     "v1.1.0",
		Tag:         "v1.1.0",
		ReleaseURL:  OfficialReleasePrefix + "v1.1.0",
		Asset:       asset,
		DownloadURL: "https://github.com/shifengbin/taskai/releases/download/v1.1.0/taskai.deb",
	})

	result := make(chan error, 1)
	go func() { result <- service.Download(context.Background()) }()
	<-started
	if err := service.Download(context.Background()); !errors.Is(err, ErrDownloadInProgress) {
		t.Fatalf("second Download() error = %v, want ErrDownloadInProgress", err)
	}
	close(releaseRead)
	if err := <-result; err != nil {
		t.Fatalf("first Download() error = %v", err)
	}
	if opens.Load() != 1 {
		t.Fatalf("OpenAsset() calls = %d, want 1", opens.Load())
	}
	if state := service.State(); state.Status != StatusDownloaded || state.InstallPath == "" {
		t.Fatalf("state = %#v", state)
	}
}

func testAsset(name string, content []byte) ManifestAsset {
	hash := sha256.Sum256(content)
	return ManifestAsset{Name: name, Size: int64(len(content)), SHA256: hex.EncodeToString(hash[:])}
}

func assertNoInstallOrPartFile(t *testing.T, directory, name string) {
	t.Helper()
	for _, path := range []string{filepath.Join(directory, name), filepath.Join(directory, name+".part")} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("unexpected cache file %s: %v", path, err)
		}
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("connection interrupted")
}

type blockingReader struct {
	started chan struct{}
	release chan struct{}
	content []byte
	offset  int
}

func (reader *blockingReader) Read(buffer []byte) (int, error) {
	if reader.offset == 0 {
		close(reader.started)
		<-reader.release
	}
	if reader.offset >= len(reader.content) {
		return 0, io.EOF
	}
	count := copy(buffer, reader.content[reader.offset:])
	reader.offset += count
	return count, nil
}
