package updater

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

func TestServiceChecksImmediatelyAndRetriesAfterFailure(t *testing.T) {
	var calls atomic.Int32
	source := &scriptedReleaseSource{
		list: func(context.Context) ([]Release, error) {
			call := calls.Add(1)
			if call == 1 {
				return nil, errors.New("temporary failure")
			}
			return []Release{testRelease("v1.1.0-beta.1", false, "taskai.exe")}, nil
		},
		manifest: func(_ context.Context, release Release) (Manifest, error) {
			return testManifest(release.TagName, "windows-amd64", "taskai.exe"), nil
		},
	}
	states := make(chan State, 4)
	service, err := NewService(Options{
		CurrentVersion: "v1.0.0",
		Platform:       "windows-amd64",
		Source:         source,
		CheckInterval:  10 * time.Millisecond,
		CheckTimeout:   100 * time.Millisecond,
		Publish: func(state State) {
			states <- state
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	service.Start(context.Background())
	defer service.Stop()

	state := waitForState(t, states, StatusAvailable)
	if state.Version != "v1.1.0-beta.1" || state.ReleaseURL != OfficialReleasePrefix+"v1.1.0-beta.1" {
		t.Fatalf("state = %#v", state)
	}
	if calls.Load() < 2 {
		t.Fatalf("ListReleases() calls = %d, want retry after first failure", calls.Load())
	}
}

func TestServiceCheckTimeoutRetriesAndChecksRemainSerial(t *testing.T) {
	var calls atomic.Int32
	var active atomic.Int32
	var maxActive atomic.Int32
	source := &scriptedReleaseSource{
		list: func(ctx context.Context) ([]Release, error) {
			call := calls.Add(1)
			current := active.Add(1)
			defer active.Add(-1)
			for {
				maximum := maxActive.Load()
				if current <= maximum || maxActive.CompareAndSwap(maximum, current) {
					break
				}
			}
			if call == 1 {
				<-ctx.Done()
				return nil, ctx.Err()
			}
			return []Release{testRelease("v1.1.0", false, "taskai.deb")}, nil
		},
		manifest: func(_ context.Context, release Release) (Manifest, error) {
			return testManifest(release.TagName, "linux-amd64", "taskai.deb"), nil
		},
	}
	states := make(chan State, 4)
	service, err := NewService(Options{
		CurrentVersion: "v1.0.0",
		Platform:       "linux-amd64",
		Source:         source,
		CheckInterval:  5 * time.Millisecond,
		CheckTimeout:   15 * time.Millisecond,
		Publish:        func(state State) { states <- state },
	})
	if err != nil {
		t.Fatal(err)
	}
	service.Start(context.Background())
	defer service.Stop()

	waitForState(t, states, StatusAvailable)
	if maxActive.Load() != 1 {
		t.Fatalf("maximum concurrent checks = %d, want 1", maxActive.Load())
	}
}

func TestServiceStopCancelsActiveCheckAndCleansUpTimer(t *testing.T) {
	started := make(chan struct{}, 1)
	cancelled := make(chan struct{}, 1)
	var calls atomic.Int32
	source := &scriptedReleaseSource{
		list: func(ctx context.Context) ([]Release, error) {
			calls.Add(1)
			started <- struct{}{}
			<-ctx.Done()
			cancelled <- struct{}{}
			return nil, ctx.Err()
		},
	}
	service, err := NewService(Options{
		CurrentVersion: "v1.0.0",
		Platform:       "linux-amd64",
		Source:         source,
		CheckInterval:  time.Hour,
		CheckTimeout:   time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.Start(context.Background())
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("startup check did not begin")
	}

	service.Stop()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("Stop() did not cancel the active request")
	}
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("ListReleases() calls after Stop = %d, want 1", calls.Load())
	}
}

type scriptedReleaseSource struct {
	list     func(context.Context) ([]Release, error)
	manifest func(context.Context, Release) (Manifest, error)
	open     func(context.Context, string) (io.ReadCloser, error)
}

func (source *scriptedReleaseSource) ListReleases(ctx context.Context) ([]Release, error) {
	return source.list(ctx)
}

func (source *scriptedReleaseSource) FetchManifest(ctx context.Context, release Release) (Manifest, error) {
	if source.manifest == nil {
		return Manifest{}, errors.New("unexpected FetchManifest call")
	}
	return source.manifest(ctx, release)
}

func (source *scriptedReleaseSource) ValidateAssetURL(string) error {
	return nil
}

func (source *scriptedReleaseSource) OpenAsset(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	if source.open == nil {
		return nil, errors.New("unexpected OpenAsset call")
	}
	return source.open(ctx, rawURL)
}

func waitForState(t *testing.T, states <-chan State, status Status) State {
	t.Helper()
	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	for {
		select {
		case state := <-states:
			if state.Status == status {
				return state
			}
		case <-timeout.C:
			t.Fatalf("timed out waiting for state %q", status)
		}
	}
}
