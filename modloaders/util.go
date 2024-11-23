package modloaders

import (
	"fmt"
	"ftb-server-downloader/util"
	semVer "github.com/hashicorp/go-version"
	"github.com/pterm/pterm"
	"io"
	"os"
	"path/filepath"
)

func Log4JFixer(installDir string, mcVersion string) (string, error) {
	patchesPath := filepath.Join(".patches")
	mcSemVer, err := semVer.NewVersion(mcVersion)
	if err != nil {
		return "", err
	}
	if mcSemVer.GreaterThanOrEqual(semVer.Must(semVer.NewVersion("1.18.1"))) {
		return "-Dlog4j2.formatMsgNoLookups=true", err
	}
	exists, err := util.PathExists(patchesPath)
	if err != nil {
		return "", err
	}
	if !exists {
		err := os.MkdirAll(filepath.Join(installDir, patchesPath), os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	if mcSemVer.GreaterThanOrEqual(semVer.Must(semVer.NewVersion("1.7"))) && mcSemVer.LessThanOrEqual(semVer.Must(semVer.NewVersion("1.11.2"))) {
		pterm.Info.Printfln("Downloading log4j fix log4j2_17-111.xml")
		get, err := util.DoGet("https://launcher.mojang.com/v1/objects/4bb89a97a66f350bc9f73b3ca8509632682aea2e/log4j2_17-111.xml")
		if err != nil {
			return "", err
		}
		defer get.Body.Close()

		out, err := os.Create(filepath.Join(installDir, patchesPath, "log4j2_17-111.xml"))
		if err != nil {
			return "", err
		}
		defer out.Close()
		_, err = io.Copy(out, get.Body)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("-Dlog4j.configurationFile=%s", filepath.Join(patchesPath, "log4j2_17-111.xml")), nil

	}

	if mcSemVer.GreaterThanOrEqual(semVer.Must(semVer.NewVersion("1.12"))) && mcSemVer.LessThanOrEqual(semVer.Must(semVer.NewVersion("1.16.5"))) {
		pterm.Info.Printfln("Downloading log4j fix log4j2_112-116.xml")
		get, err := util.DoGet("https://launcher.mojang.com/v1/objects/02937d122c86ce73319ef9975b58896fc1b491d1/log4j2_112-116.xml")
		if err != nil {
			return "", err
		}
		defer get.Body.Close()

		out, err := os.Create(filepath.Join(installDir, patchesPath, "log4j2_112-116.xml"))
		if err != nil {
			return "", err
		}
		defer out.Close()
		_, err = io.Copy(out, get.Body)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("-Dlog4j.configurationFile=%s", filepath.Join(patchesPath, "log4j2_112-116.xml")), nil
	}

	if mcSemVer.GreaterThanOrEqual(semVer.Must(semVer.NewVersion("1.17"))) && mcSemVer.LessThanOrEqual(semVer.Must(semVer.NewVersion("1.18"))) {
		return "-Dlog4j2.formatMsgNoLookups=true", nil
	}

	return "", nil
}
