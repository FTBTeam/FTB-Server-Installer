package repos

import (
	"encoding/json"
	"fmt"
	"ftb-server-downloader/structs"
	"ftb-server-downloader/util"
	"github.com/pterm/pterm"
	"sort"
	"strings"
)

const (
	ftbApiUrl = "https://api.feed-the-beast.com/v1/modpacks"
)

type FTB struct {
	PackId    int
	VersionId int
	ApiKey    string
}

func GetFTB(packId, versionId int, apiKey string) *FTB {
	return &FTB{
		PackId:    packId,
		VersionId: versionId,
		ApiKey:    apiKey,
	}
}

func (m *FTB) GetModpack() (structs.Modpack, error) {
	url := fmt.Sprintf("%s/modpack/%d", makeFTBUrl(m), m.PackId)
	pterm.Debug.Printfln("Getting modpack from ftb using %s", url)
	resp, err := util.DoGet(url)
	if err != nil {
		return structs.Modpack{}, err
	}
	defer resp.Body.Close()

	var ftbModpack structs.FTBModpack

	err = json.NewDecoder(resp.Body).Decode(&ftbModpack)
	if err != nil {
		return structs.Modpack{}, err
	}

	if ftbModpack.Status != "success" {
		return structs.Modpack{}, fmt.Errorf("unsuccessful response: %s, %s", ftbModpack.Status, ftbModpack.Message)
	}

	var versionList []structs.ModpackV
	for _, v := range ftbModpack.Versions {
		ver := structs.ModpackV{
			Id:   v.ID,
			Type: strings.ToLower(v.Type),
		}
		versionList = append(versionList, ver)
	}

	sort.Slice(versionList, func(i, j int) bool {
		return versionList[i].Id > versionList[j].Id
	})

	return structs.Modpack{
		Name:     ftbModpack.Name,
		Id:       ftbModpack.ID,
		Versions: versionList,
	}, nil
}

func (m *FTB) GetVersion() (structs.ModpackVersion, error) {
	url := fmt.Sprintf("%s/modpack/%d/%d", makeFTBUrl(m), m.PackId, m.VersionId)
	pterm.Debug.Printfln("Getting modpack version from ftb using %s", url)
	resp, err := util.DoGet(url)
	if err != nil {
		return structs.ModpackVersion{}, err
	}
	defer resp.Body.Close()

	var ftbModpackVer structs.FTBVersion

	err = json.NewDecoder(resp.Body).Decode(&ftbModpackVer)
	if err != nil {
		return structs.ModpackVersion{}, err
	}

	if ftbModpackVer.Status != "success" {
		return structs.ModpackVersion{}, fmt.Errorf("unsuccessful response: %s, %s", ftbModpackVer.Status, ftbModpackVer.Message)
	}

	var mem structs.Memory
	mem.Minimum = ftbModpackVer.Specs.Minimum
	mem.Recommended = ftbModpackVer.Specs.Recommended

	return structs.ModpackVersion{
		Id:      ftbModpackVer.ID,
		Name:    ftbModpackVer.Name,
		Targets: parseFTBTargets(ftbModpackVer.Targets),
		Memory:  mem,
		Files:   parseFTBFiles(ftbModpackVer.Files),
	}, nil
}

func (m *FTB) SetVersionId(versionId int) {
	m.VersionId = versionId
}

func makeFTBUrl(m *FTB) string {
	return fmt.Sprintf("%s/%s", ftbApiUrl, m.ApiKey)
}

func parseFTBTargets(targets []structs.FTBTargets) structs.ModpackTargets {
	var modpackTargets structs.ModpackTargets
	for _, t := range targets {
		if t.Type == "modloader" {
			modpackTargets.ModLoader.Name = t.Name
			modpackTargets.ModLoader.Version = t.Version
		}
		if t.Type == "game" && t.Name == "minecraft" {
			modpackTargets.McVersion = t.Version
		}
		if t.Type == "runtime" && t.Name == "java" {
			modpackTargets.JavaVersion = t.Version
		}
	}
	return modpackTargets
}

func parseFTBFiles(files []structs.FTBFiles) []structs.File {
	var parsedFiles []structs.File
	for _, f := range files {
		if !f.ClientOnly {
			parsedFiles = append(parsedFiles, structs.File{
				Name:     f.Name,
				Path:     f.Path,
				Url:      f.URL,
				Hash:     f.Sha1,
				HashType: "sha1",
				Mirrors:  f.Mirrors,
			})
		}
	}
	return parsedFiles
}
