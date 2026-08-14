//go:build updater_integration

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"taskai/internal/updater"
)

func newApplicationUpdater(dataDirectory string, publish func(updater.State)) (updateService, updater.Launcher) {
	baseURL := strings.TrimSuffix(os.Getenv("TASKAI_UPDATE_TEST_URL"), "/")
	platform := updater.PlatformKey(runtime.GOOS, runtime.GOARCH)
	if baseURL == "" || platform == "" {
		return nil, nil
	}
	source, err := newIntegrationUpdateSource(baseURL)
	if err != nil {
		return nil, nil
	}
	service, err := updater.NewService(updater.Options{
		CurrentVersion: appVersion,
		Platform:       platform,
		Source:         source,
		CacheDirectory: filepath.Join(dataDirectory, "updates"),
		Publish:        publish,
	})
	if err != nil {
		return nil, nil
	}
	return service, integrationUpdateLauncher{browser: updater.DefaultSystemLauncher()}
}

func newIntegrationUpdateSource(baseURL string) (*updater.GitHubSource, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return updater.NewGitHubSource(nil, baseURL, baseURL+"/")
}

type integrationUpdateLauncher struct {
	browser updater.Launcher
}

func (launcher integrationUpdateLauncher) LaunchInstaller(string) error {
	return nil
}

func (launcher integrationUpdateLauncher) OpenReleasePage(releaseURL string) error {
	return launcher.browser.OpenReleasePage(releaseURL)
}
