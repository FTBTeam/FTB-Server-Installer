package modloaders

import "ftb-server-downloader/structs"

type ModLoader interface {
	GetDownload() ([]structs.File, error)
	Install(useOwnJava bool) error
}
