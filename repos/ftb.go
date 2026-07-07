package repos

import (
	"errors"
	"fmt"
	"ftb-server-downloader/structs"
	"ftb-server-downloader/util"
	"sort"
	"strings"

	"github.com/pterm/pterm"
)

const (
	ftbApiUrl = "https://api.feed-the-beast.com/v1/modpacks/modpack"
)

type FTB struct {
	PackId    int
	VersionId int
	IsPrivate bool
}

func GetFTB(packId, versionId int) *FTB {
	return &FTB{
		PackId:    packId,
		VersionId: versionId,
	}
}

func (m *FTB) GetModpack() (structs.Modpack, error) {
	url := fmt.Sprintf("%s/%d", ftbApiUrl, m.PackId)
	pterm.Debug.Printfln("Getting modpack from ftb using %s", url)

	var ftbModpack structs.FTBModpack
	var ftbModpackErr structs.FTBModpackErr
	resp, err := util.ReqClient.R().
		SetSuccessResult(&ftbModpack).
		SetErrorResult(&ftbModpackErr).
		Get(url)
	if err != nil {
		return structs.Modpack{}, err
	}

	if !resp.IsSuccessState() {
		errMsg := fmt.Sprintf("unsuccessful response: %s", resp.Status)
		if ftbModpackErr.Message != "" {
			errMsg = fmt.Sprintf("%s, %s", errMsg, ftbModpackErr.Message)
		}
		return structs.Modpack{}, errors.New(errMsg)
	}

	m.IsPrivate = ftbModpack.Private
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
	url := fmt.Sprintf("%s/%d/%d", ftbApiUrl, m.PackId, m.VersionId)
	pterm.Debug.Printfln("Getting modpack version from ftb using %s", url)

	var ftbModpackVer structs.FTBVersion
	var ftbModpackErr structs.FTBModpackErr
	resp, err := util.ReqClient.R().
		SetSuccessResult(&ftbModpackVer).
		SetErrorResult(&ftbModpackErr).
		Get(url)
	if err != nil {
		return structs.ModpackVersion{}, err
	}
	if !resp.IsSuccessState() {
		errMsg := fmt.Sprintf("unsuccessful response: %s", resp.Status)
		if ftbModpackErr.Message != "" {
			errMsg = fmt.Sprintf("%s, %s", errMsg, ftbModpackErr.Message)
		}

		return structs.ModpackVersion{}, errors.New(errMsg)
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

func (m *FTB) SuccessfulInstall() {
	if m.IsPrivate {
		// If pack is private don't send success request
		return
	}

	url := fmt.Sprintf("%s/%d/%d/serverInstall/success", ftbApiUrl, m.PackId, m.VersionId)
	resp, err := util.ReqClient.R().Get(url)
	if err != nil {
		pterm.Debug.WithMessageStyle(pterm.Error.MessageStyle).Printfln("Error while sending successful install request to ftb: %s", err.Error())
		return
	}
	if !resp.IsSuccessState() {
		pterm.Debug.WithMessageStyle(pterm.Error.MessageStyle).Printfln("Error while sending successful install request to ftb: %s", resp.Status)
		return
	}
}

func (m *FTB) FailedInstall() {
	if m.IsPrivate {
		// If pack is private don't send failure request
		return
	}

	url := fmt.Sprintf("%s/%d/%d/serverInstall/failure", ftbApiUrl, m.PackId, m.VersionId)

	resp, err := util.ReqClient.R().Get(url)
	if err != nil {
		pterm.Debug.WithMessageStyle(pterm.Error.MessageStyle).Printfln("Error while sending failed install request to ftb: %s", err.Error())
		return
	}
	if !resp.IsSuccessState() {
		pterm.Debug.WithMessageStyle(pterm.Error.MessageStyle).Printfln("Error while sending failed install request to ftb: %s", resp.Status)
		return
	}
}

func (m *FTB) SetVersionId(versionId int) {
	m.VersionId = versionId
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
