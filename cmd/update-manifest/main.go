package main

import (
	"flag"
	"fmt"
	"os"

	"taskai/internal/updater"
)

func main() {
	version := flag.String("version", "", "发布语义版本（不含 v 前缀）")
	tag := flag.String("tag", "", "Git tag")
	releaseURL := flag.String("release-url", "", "GitHub Release 页面")
	linuxAMD64 := flag.String("linux-amd64", "", "Linux amd64 DEB 路径")
	windowsAMD64 := flag.String("windows-amd64", "", "Windows amd64 NSIS 路径")
	darwinUniversal := flag.String("darwin-universal", "", "macOS universal DMG 路径")
	output := flag.String("output", "taskai-update.json", "输出清单路径")
	flag.Parse()

	manifest, err := updater.BuildManifest(*version, *tag, *releaseURL, map[string]string{
		"linux-amd64":      *linuxAMD64,
		"windows-amd64":    *windowsAMD64,
		"darwin-universal": *darwinUniversal,
	})
	if err == nil {
		err = updater.WriteManifest(*output, manifest)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
