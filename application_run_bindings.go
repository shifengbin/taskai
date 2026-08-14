//go:build bindings

package main

import "github.com/wailsapp/wails/v2"

func runApplication() error {
	dataDirectory := defaultDataDirectory()
	return wails.Run(applicationOptions(newApp(dataDirectory), dataDirectory))
}
