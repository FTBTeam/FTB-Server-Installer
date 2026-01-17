package modloaders

import (
	"encoding/json"
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

func GetVanilla(target structs.ModpackTargets, installDir string) (Vanilla, error) {

	rawMeta, err := util.DoGet(launcherMeta)
	if err != nil {
		return Vanilla{}, err
	}
	defer rawMeta.Body.Close()

	var meta LauncherMeta
	err = json.NewDecoder(rawMeta.Body).Decode(&meta)
	if err != nil {
		return Vanilla{}, err
	}

	return Vanilla{
		InstallDir: installDir,
		Version:    target.McVersion,
		Meta:       meta,
	}, nil
}

func (v Vanilla) GetDownload() ([]structs.File, error) {
	var mlFiles []structs.File

	var servDlUrl string
	for _, version := range v.Meta.Versions {
		if version.ID == v.Version {
			servDlUrl = version.URL
			break
		}
	}

	if servDlUrl == "" {
		return mlFiles, errors.New("version not found")
	}

	rawVer, err := util.DoGet(servDlUrl)
	if err != nil {
		return []structs.File{}, err
	}
	defer rawVer.Body.Close()

	var version VanillaVersion
	err = json.NewDecoder(rawVer.Body).Decode(&version)
	if err != nil {
		return []structs.File{}, err
	}

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
