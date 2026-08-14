package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

const taskAISingleInstanceID = "com.taskai.desktop"

func main() {
	if err := runApplication(); err != nil {
		println("Error:", err.Error())
	}
}

func resolveApplicationDataDirectory(coordinationDirectory string, resolve func() string) (string, *applicationInstanceLock, error) {
	instanceLock, err := acquireApplicationInstanceLock(coordinationDirectory)
	if err != nil {
		return "", nil, err
	}
	return resolve(), instanceLock, nil
}

func applicationOptions(app *App, dataDirectory string) *options.App {
	return &options.App{
		Title:  "taskai",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 248, G: 250, B: 252, A: 1},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: applicationSingleInstanceID(dataDirectory),
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		OnStartup:     app.startup,
		OnShutdown:    app.shutdown,
		OnBeforeClose: app.beforeClose,
		Bind: []interface{}{
			app,
		},
	}
}

func applicationSingleInstanceID(dataDirectory string) string {
	digest := sha256.Sum256([]byte(dataDirectory))
	return taskAISingleInstanceID + "." + hex.EncodeToString(digest[:8])
}
