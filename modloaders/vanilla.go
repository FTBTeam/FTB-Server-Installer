package modloaders

import (
	"errors"
	"fmt"
	"ftb-server-downloader/structs"
	"ftb-server-downloader/util"
)

const launcherMeta = "https://launchermeta.mojang.com/mc/game/version_manifest.json"

type Vanilla struct {
	InstallDir string
	Version    string
	Meta       LauncherMeta
}

func GetVanilla(target structs.ModpackTargets, installDir string) (*Vanilla, error) {
	var meta LauncherMeta
	rawMeta, err := util.ReqClient.R().
		SetSuccessResult(&meta).
		Get(launcherMeta)
	if err != nil {
		return nil, err
	}

	if !rawMeta.IsSuccessState() {
		return nil, fmt.Errorf("failed to fetch launcher meta: %s", rawMeta.Status)
	}

	return &Vanilla{
		InstallDir: installDir,
		Version:    target.McVersion,
		Meta:       meta,
	}, nil
}

func (v Vanilla) GetDownload() ([]structs.File, error) {

	var servDlUrl string
	for _, version := range v.Meta.Versions {
		if version.ID == v.Version {
			servDlUrl = version.URL
			break
		}
	}

	if servDlUrl == "" {
		return nil, errors.New("version not found")
	}

	var version VanillaVersion
	rawVer, err := util.ReqClient.R().
		SetSuccessResult(&version).
		Get(servDlUrl)

	if err != nil {
		return nil, err
	}

	if !rawVer.IsSuccessState() {
		return nil, fmt.Errorf("failed to fetch vanilla version: %s", rawVer.Status)
	}

	var mlFiles []structs.File
	mlFiles = append(mlFiles, structs.File{
		Name:     fmt.Sprintf("minecraft_server.%s.jar", v.Version),
		Url:      version.Downloads.Server.URL,
		Hash:     version.Downloads.Server.Sha1,
		HashType: "sha1",
	})

	return mlFiles, nil
}

func (v Vanilla) Install(bool) error {
	return nil
}

type LauncherMeta struct {
	Latest   VanillaLatest     `json:"latest"`
	Versions []VanillaVersions `json:"versions"`
}
type VanillaLatest struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}
type VanillaVersions struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type VanillaVersion struct {
	Downloads struct {
		Server struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"server"`
	} `json:"downloads"`
	ID   string `json:"id"`
	Type string `json:"type"`
}
