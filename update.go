package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/blang/semver"
	"github.com/charmbracelet/glamour"
	charmlog "github.com/charmbracelet/log"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

var updateLogger = charmlog.NewWithOptions(os.Stderr, charmlog.Options{
	ReportTimestamp: false,
	Prefix:          "updating",
})

func performUpdate() {
	printBanner()

	latest, found, err := selfupdate.DetectLatest("coffinxp/crtmon")
	if err != nil {
		printErr("failed to check for updates: " + err.Error())
		os.Exit(1)
	}
	if !found {
		printErr("no releases found")
		os.Exit(1)
	}

	current, err := semver.ParseTolerant(Version)
	if err != nil {
		printErr("invalid version: " + err.Error())
		os.Exit(1)
	}

	if !latest.Version.GT(current) {
		printInfo("crtmon already up to date v" + Version)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		printErr("could not locate binary: " + err.Error())
		os.Exit(1)
	}

	printInfo(fmt.Sprintf("downloading v%s", latest.Version.String()))

	updater, _ := selfupdate.NewUpdater(selfupdate.Config{})
	if err := updater.UpdateTo(latest, exe); err != nil {
		printErr("update failed: " + err.Error())
		os.Exit(1)
	}

	printInfo(fmt.Sprintf("crtmon successfully updated v%s -> v%s (latest)", Version, latest.Version.String()))
	fmt.Println()

	showChangelog("coffinxp/crtmon", latest.Version.String())
}

func printInfo(msg string) {
	updateLogger.Info(msg)
}

func printErr(msg string) {
	updateLogger.Error(msg)
}

func showChangelog(repo, version string) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/v%s", repo, version)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		updateLogger.Error("failed to fetch changelog", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		updateLogger.Warn("changelog not available", "status", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		updateLogger.Error("failed to read changelog", "error", err)
		return
	}

	var release struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		updateLogger.Error("failed to parse changelog", "error", err)
		return
	}

	if release.Body == "" {
		updateLogger.Warn("no changelog available for this release")
		return
	}

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		updateLogger.Error("failed to initialize markdown renderer", "error", err)
		fmt.Println(release.Body)
		return
	}

	rendered, err := r.Render(release.Body)
	if err != nil {
		updateLogger.Error("failed to render changelog", "error", err)
		fmt.Println(release.Body)
		return
	}

	fmt.Println()
	fmt.Print(rendered)
}
