package repos

import "ftb-server-downloader/structs"

type ModpackRepo interface {
	GetModpack() (structs.Modpack, error)
	GetVersion() (structs.ModpackVersion, error)
	SetVersionId(versionId int)
}
