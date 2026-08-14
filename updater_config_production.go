//go:build !updater_integration

package main

import (
	"path/filepath"
	"runtime"

	"taskai/internal/updater"
)

func newApplicationUpdater(dataDirectory string, publish func(updater.State)) (updateService, updater.Launcher) {
	platform := updater.PlatformKey(runtime.GOOS, runtime.GOARCH)
	if platform == "" {
		return nil, nil
	}
	source, err := newProductionUpdateSource()
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
	return service, updater.DefaultSystemLauncher()
}

func newProductionUpdateSource() (*updater.GitHubSource, error) {
	return updater.NewOfficialGitHubSource(nil)
}
