//go:build !bindings

package main

import (
	"github.com/wailsapp/wails/v2"

	"taskai/internal/appdata"
)

func runApplication() error {
	dataDirectory, instanceLock, err := resolveApplicationDataDirectory(appdata.CoordinationDirectory(), defaultDataDirectory)
	if err != nil {
		return err
	}
	defer instanceLock.Close()

	app := newApp(dataDirectory)
	return wails.Run(applicationOptions(app, dataDirectory))
}
