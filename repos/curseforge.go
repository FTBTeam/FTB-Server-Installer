package repos

import (
	"ftb-server-downloader/structs"
	"github.com/pterm/pterm"
)

const (
	cfAPIUrl = "https://api.curseforge.com"
)

type CurseForge struct {
	PackId    int
	VersionId int
}

func GetCurseForge(packId, versionId int) *CurseForge {
	return &CurseForge{
		PackId:    packId,
		VersionId: versionId,
	}
}

func (v *CurseForge) GetModpack() (structs.Modpack, error) {
	pterm.Info.Printfln("Getting modpack with id %d from CurseForge", v.PackId)
	return structs.Modpack{}, nil
}

func (v *CurseForge) GetVersion() (structs.ModpackVersion, error) {
	pterm.Info.Printfln("Getting modpack version with id %d from CurseForge", v.PackId)
	return structs.ModpackVersion{}, nil
}

func (v *CurseForge) SetVersionId(versionId int) {
	v.VersionId = versionId
}

func (m *CurseForge) SuccessfulInstall() {
	return
}

func (m *CurseForge) FailedInstall() {
	return
}
