package modloaders

import (
	"fmt"
	"ftb-server-downloader/util"
	"os"
	"path/filepath"
	"runtime"

	semVer "github.com/hashicorp/go-version"
	"github.com/pterm/pterm"
)

func Log4JFixer(installDir string, mcVersion string) (string, error) {
	patchesPath := filepath.Join(".patches")
	mcSemVer, err := semVer.NewVersion(mcVersion)
	if err != nil {
		return "", err
	}
	if mcSemVer.GreaterThanOrEqual(semVer.Must(semVer.NewVersion("1.18.1"))) {
		return " -Dlog4j2.formatMsgNoLookups=true ", err
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

		const log4JUrl = "https://launcher.mojang.com/v1/objects/4bb89a97a66f350bc9f73b3ca8509632682aea2e/log4j2_17-111.xml"
		log4jPath := filepath.Join(installDir, patchesPath, "log4j2_17-111.xml")

		resp, err := util.ReqClient.R().
			SetOutputFile(log4jPath).
			Get(log4JUrl)

		if err != nil {
			return "", err
		}

		if !resp.IsSuccessState() {
			_ = os.Remove(log4jPath)
			return "", fmt.Errorf("failed to download log4j fix: %s", resp.Status)
		}

		return fmt.Sprintf(" -Dlog4j.configurationFile=%s ", filepath.Join(patchesPath, "log4j2_17-111.xml")), nil

	}

	if mcSemVer.GreaterThanOrEqual(semVer.Must(semVer.NewVersion("1.12"))) && mcSemVer.LessThanOrEqual(semVer.Must(semVer.NewVersion("1.16.5"))) {
		pterm.Info.Printfln("Downloading log4j fix log4j2_112-116.xml")
		const log4jUrl = "https://launcher.mojang.com/v1/objects/02937d122c86ce73319ef9975b58896fc1b491d1/log4j2_112-116.xml"
		log4jPath := filepath.Join(installDir, patchesPath, "log4j2_112-116.xml")

		resp, err := util.ReqClient.R().
			SetOutputFile(log4jPath).
			Get(log4jUrl)

		if err != nil {
			return "", err
		}

		if !resp.IsSuccessState() {
			_ = os.Remove(log4jPath)
			return "", fmt.Errorf("failed to download log4j fix: %s", resp.Status)
		}
		return fmt.Sprintf(" -Dlog4j.configurationFile=%s ", filepath.Join(patchesPath, "log4j2_112-116.xml")), nil
	}

	if mcSemVer.GreaterThanOrEqual(semVer.Must(semVer.NewVersion("1.17"))) && mcSemVer.LessThanOrEqual(semVer.Must(semVer.NewVersion("1.18"))) {
		return " -Dlog4j2.formatMsgNoLookups=true ", nil
	}

	return "", nil
}

func writeRunFile(runFile *os.File, javaPath string, log4jFix string, recMem int, runJarName string) error {
	var err error
	if runtime.GOOS == "windows" {
		_, err = runFile.WriteString(fmt.Sprintf("\"%s\"%s-Xmx%dM -jar %s nogui", javaPath, log4jFix, recMem, runJarName))
		if err != nil {
			return err
		}
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		_, err = runFile.WriteString(fmt.Sprintf("#!/usr/bin/env sh\n\"%s\"%s-Xmx%dM -jar %s nogui", javaPath, log4jFix, recMem, runJarName))
		if err != nil {
			return err
		}
	}

	return nil
}
