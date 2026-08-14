package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"taskai/internal/updater"
)

const (
	integrationTargetVersion = "0.0.0-rc6"
	integrationTargetTag     = "v" + integrationTargetVersion
	manifestPath             = "/releases/download/" + integrationTargetTag + "/taskai-update.json"
	installerPath            = "/releases/download/" + integrationTargetTag + "/taskai_" + integrationTargetVersion + "_amd64.deb"
	requestCountPath         = "/requests"
)

var integrationInstallerPayload = []byte("TaskAI automatic update integration payload\n")

type requestCounts struct {
	Releases  int32 `json:"releases"`
	Manifest  int32 `json:"manifest"`
	Installer int32 `json:"installer"`
}

type updateTestHandler struct {
	releases     atomic.Int32
	manifest     atomic.Int32
	installer    atomic.Int32
	requestDelay time.Duration
}

func main() {
	address := flag.String("addr", "127.0.0.1:0", "listening address")
	flag.Parse()

	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Handler: newUpdateTestHandler()}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("TASKAI_UPDATE_TEST_URL=http://%s\n", listener.Addr().String())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newUpdateTestHandler() http.Handler {
	return newUpdateTestHandlerWithDelay(time.Second)
}

func newUpdateTestHandlerWithDelay(requestDelay time.Duration) http.Handler {
	return &updateTestHandler{requestDelay: requestDelay}
}

func (handler *updateTestHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/repos/shifengbin/taskai/releases":
		handler.releases.Add(1)
		handler.writeReleases(writer, request)
	case manifestPath:
		handler.manifest.Add(1)
		handler.writeManifest(writer)
	case installerPath:
		requestNumber := handler.installer.Add(1)
		time.Sleep(handler.requestDelay)
		if requestNumber == 1 {
			http.Error(writer, "intentional first download failure", http.StatusServiceUnavailable)
			return
		}
		writer.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(integrationInstallerPayload)
	case requestCountPath:
		writeJSON(writer, requestCounts{
			Releases:  handler.releases.Load(),
			Manifest:  handler.manifest.Load(),
			Installer: handler.installer.Load(),
		})
	default:
		http.NotFound(writer, request)
	}
}

func (handler *updateTestHandler) writeReleases(writer http.ResponseWriter, request *http.Request) {
	baseURL := "http://" + request.Host
	writeJSON(writer, []updater.Release{{
		TagName:    integrationTargetTag,
		HTMLURL:    updater.OfficialReleasePrefix + integrationTargetTag,
		Prerelease: true,
		Assets: []updater.ReleaseAsset{
			{Name: "taskai-update.json", DownloadURL: baseURL + manifestPath},
			{Name: installerName(), DownloadURL: baseURL + installerPath},
		},
	}})
}

func (handler *updateTestHandler) writeManifest(writer http.ResponseWriter) {
	digest := sha256.Sum256(integrationInstallerPayload)
	writeJSON(writer, updater.Manifest{
		SchemaVersion: updater.ManifestSchemaVersion,
		Version:       integrationTargetVersion,
		Tag:           integrationTargetTag,
		ReleaseURL:    updater.OfficialReleasePrefix + integrationTargetTag,
		Assets: map[string]updater.ManifestAsset{
			"linux-amd64": {
				Name:   installerName(),
				Size:   int64(len(integrationInstallerPayload)),
				SHA256: hex.EncodeToString(digest[:]),
			},
		},
	})
}

func installerName() string {
	return installerPath[len("/releases/download/"+integrationTargetTag+"/"):]
}

func writeJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}
