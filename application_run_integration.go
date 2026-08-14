//go:build updater_integration && !bindings

package main

import (
	"os"

	"github.com/wailsapp/wails/v2"
)

func runApplication() error {
	dataDirectory, cleanup, err := integrationApplicationDataDirectory()
	if err != nil {
		return err
	}
	defer cleanup()
	return wails.Run(applicationOptions(newApp(dataDirectory), dataDirectory))
}

func integrationApplicationDataDirectory() (string, func(), error) {
	directory, err := os.MkdirTemp("", "taskai-updater-integration-")
	if err != nil {
		return "", nil, err
	}
	return directory, func() { _ = os.RemoveAll(directory) }, nil
}
