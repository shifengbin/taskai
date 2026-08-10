package main

import (
	"context"
	"testing"
)

func TestBeforeClosePersistsWindowMaximized(t *testing.T) {
	app := newApp(t.TempDir())
	app.windowIsMaximised = func(context.Context) bool { return true }

	if prevented := app.beforeClose(context.Background()); prevented {
		t.Fatal("beforeClose() prevented closing without running tasks")
	}
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if !current.WindowMaximized {
		t.Fatal("WindowMaximized = false, want true")
	}
}

func TestStartupRestoresSavedWindowMaximized(t *testing.T) {
	app := newApp(t.TempDir())
	current, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	current.WindowMaximized = true
	if _, err := app.repository.SaveSettings(current); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	maximised := false
	app.windowMaximise = func(context.Context) { maximised = true }

	app.startup(context.Background())

	if !maximised {
		t.Fatal("startup() did not maximise a previously maximised window")
	}
}
