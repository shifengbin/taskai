package main

import (
	"context"
	"errors"
	"testing"

	"taskai/internal/updater"
)

func TestAppUpdateMethodsDelegateToServiceAndLauncher(t *testing.T) {
	state := updater.State{
		Status:      updater.StatusDownloaded,
		Version:     "v1.2.3",
		ReleaseURL:  updater.OfficialReleasePrefix + "v1.2.3",
		InstallPath: "/tmp/taskai.deb",
	}
	service := &fakeUpdateService{state: state}
	launcher := &fakeUpdateLauncher{}
	app := newApp(t.TempDir())
	app.updaterService = service
	app.updateLauncher = launcher
	app.ctx = context.Background()

	if got := app.GetUpdateState(); got != state {
		t.Fatalf("GetUpdateState() = %#v, want %#v", got, state)
	}
	if err := app.StartUpdateDownload(); err != nil {
		t.Fatal(err)
	}
	if service.downloadCalls != 1 {
		t.Fatalf("Download() calls = %d, want 1", service.downloadCalls)
	}
	if err := app.OpenUpdateReleasePage(); err != nil {
		t.Fatal(err)
	}
	if launcher.openedURL != state.ReleaseURL {
		t.Fatalf("opened URL = %q", launcher.openedURL)
	}
	if err := app.LaunchDownloadedUpdate(); err != nil {
		t.Fatal(err)
	}
	if launcher.installerPath != state.InstallPath {
		t.Fatalf("installer path = %q", launcher.installerPath)
	}
	if app.allowClose {
		t.Fatal("LaunchDownloadedUpdate() called PrepareQuit implicitly")
	}
}

func TestAppUpdateLaunchFailureKeepsApplicationRunning(t *testing.T) {
	launchErr := errors.New("cannot start installer")
	app := newApp(t.TempDir())
	app.updaterService = &fakeUpdateService{state: updater.State{
		Status:      updater.StatusDownloaded,
		Version:     "v1.2.3",
		ReleaseURL:  updater.OfficialReleasePrefix + "v1.2.3",
		InstallPath: "/tmp/taskai.deb",
	}}
	app.updateLauncher = &fakeUpdateLauncher{launchErr: launchErr}

	if err := app.LaunchDownloadedUpdate(); !errors.Is(err, launchErr) {
		t.Fatalf("LaunchDownloadedUpdate() error = %v", err)
	}
	if app.allowClose {
		t.Fatal("failed installer launch allowed the application to close")
	}
}

func TestAppUpdaterLifecycleAndStateEvent(t *testing.T) {
	service := &fakeUpdateService{state: updater.State{Status: updater.StatusIdle}}
	app := newApp(t.TempDir())
	app.updaterService = service
	var published updater.State
	app.updateStatePublisher = func(state updater.State) { published = state }

	ctx := context.Background()
	app.startup(ctx)
	if service.startCalls != 1 || service.startedContext != ctx {
		t.Fatalf("Start() calls = %d, context = %v", service.startCalls, service.startedContext)
	}
	want := updater.State{Status: updater.StatusAvailable, Version: "v1.2.3"}
	app.publishUpdateStateEvent(want)
	if published != want {
		t.Fatalf("published state = %#v, want %#v", published, want)
	}
	app.shutdown(ctx)
	if service.stopCalls != 1 {
		t.Fatalf("Stop() calls = %d, want 1", service.stopCalls)
	}
}

type fakeUpdateService struct {
	state          updater.State
	startCalls     int
	stopCalls      int
	downloadCalls  int
	startedContext context.Context
	downloadErr    error
}

func (service *fakeUpdateService) Start(ctx context.Context) {
	service.startCalls++
	service.startedContext = ctx
}

func (service *fakeUpdateService) Stop() {
	service.stopCalls++
}

func (service *fakeUpdateService) State() updater.State {
	return service.state
}

func (service *fakeUpdateService) Download(context.Context) error {
	service.downloadCalls++
	return service.downloadErr
}

type fakeUpdateLauncher struct {
	installerPath string
	openedURL     string
	launchErr     error
	openErr       error
}

func (launcher *fakeUpdateLauncher) LaunchInstaller(path string) error {
	launcher.installerPath = path
	return launcher.launchErr
}

func (launcher *fakeUpdateLauncher) OpenReleasePage(url string) error {
	launcher.openedURL = url
	return launcher.openErr
}
