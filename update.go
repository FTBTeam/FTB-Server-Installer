package main

import (
	"encoding/json"
	"fmt"
	"ftb-server-downloader/util"
	semver "github.com/hashicorp/go-version"
	"github.com/pterm/pterm"
	"io"
	"net/http"
	"runtime"
	"strings"
)

const (
	org  = "FTBTeam"
	repo = "FTB-Server-Installer"
)

type GHRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

type VersionInfo struct {
	UpdateAvailable     bool
	CurrentVersion      string
	LatestVersion       string
	Name                string
	isPreReleaseOrDraft bool
}

func checkForUpdate() (VersionInfo, error) {
	var versionInfo VersionInfo = VersionInfo{
		UpdateAvailable:     false,
		CurrentVersion:      util.ReleaseVersion,
		LatestVersion:       "",
		Name:                "",
		isPreReleaseOrDraft: false,
	}
	releaseApi := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", org, repo)
	resp, err := http.Get(releaseApi)
	if err != nil {
		return versionInfo, fmt.Errorf("error checking for update: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return versionInfo, fmt.Errorf("bad status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return versionInfo, fmt.Errorf("error reading response body: %s", err.Error())
	}

	var release GHRelease
	err = json.Unmarshal(data, &release)
	if err != nil {
		return versionInfo, fmt.Errorf("error unmarshalling response: %s", err.Error())
	}

	if release.Prerelease || release.Draft {
		versionInfo.isPreReleaseOrDraft = true
		return versionInfo, nil
	}

	versionInfo.LatestVersion = release.TagName
	versionInfo.Name = release.Name

	currentVersion, err := semver.NewVersion(strings.ReplaceAll(util.ReleaseVersion, "v", ""))
	if err != nil {
		return versionInfo, fmt.Errorf("error parsing current version: %s", err.Error())
	}
	latestVersion, err := semver.NewVersion(strings.ReplaceAll(versionInfo.LatestVersion, "v", ""))
	if err != nil {
		return versionInfo, fmt.Errorf("error parsing latest version: %s", err.Error())
	}

	if latestVersion.GreaterThan(currentVersion) {
		versionInfo.UpdateAvailable = true
	}

	return versionInfo, nil
}

func doUpdate(versionInfo VersionInfo) {
	filename := fmt.Sprintf("ftb-server-%s-%s", strings.ToLower(runtime.GOOS), strings.ToLower(runtime.GOARCH))
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	downloadUrl := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", org, repo, versionInfo.LatestVersion, filename)
	hashResp, err := http.Get(fmt.Sprintf("%s.sha256", downloadUrl))
	if err != nil {
		pterm.Error.Println("Error downloading hash: ", err.Error())
		return
	}
	defer hashResp.Body.Close()
	if hashResp.StatusCode != http.StatusOK {
		pterm.Error.Println("Error downloading hash: ", hashResp.Status)
		return
	}
	hashData, err := io.ReadAll(hashResp.Body)
	if err != nil {
		pterm.Error.Println("Error reading hash response: ", err.Error())
		return
	}
	pterm.Debug.Println("Update Hash: ", string(hashData))

	resp, err := http.Get(downloadUrl)
	if err != nil {
		pterm.Error.Println("Error downloading update: ", err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		pterm.Error.Println("Error downloading update: ", resp.Status)
		return
	}
}
