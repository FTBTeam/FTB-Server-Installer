package modloaders

import (
	"bufio"
	"bytes"
	"fmt"
	"ftb-server-downloader/util"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

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

		return fmt.Sprintf("-Dlog4j.configurationFile=%s", filepath.Join(patchesPath, "log4j2_17-111.xml")), nil

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
		return fmt.Sprintf("-Dlog4j.configurationFile=%s", filepath.Join(patchesPath, "log4j2_112-116.xml")), nil
	}

	if mcSemVer.GreaterThanOrEqual(semVer.Must(semVer.NewVersion("1.17"))) && mcSemVer.LessThanOrEqual(semVer.Must(semVer.NewVersion("1.18"))) {
		return "-Dlog4j2.formatMsgNoLookups=true", nil
	}

	return "", nil
}

func Java4ShellFix(javaVersion string) (string, error) {
	splitVersion := strings.Split(javaVersion, ".")
	if len(splitVersion) < 1 {
		return "", fmt.Errorf("invalid java version: %s", javaVersion)
	}
	javaVer, err := strconv.Atoi(splitVersion[0])
	if err != nil {
		return "", err
	}
	if javaVer >= 17 {
		return "-XX:+UnlockDiagnosticVMOptions -XX:-UseBASE64Intrinsics", nil
	}
	return "", nil
}

func appendArsFile(path string, args []string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	existing := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		existing[scanner.Text()] = true
	}
	if err = scanner.Err(); err != nil {
		return err
	}

	var missing []string
	for _, arg := range args {
		if !existing[arg] {
			missing = append(missing, arg)
			existing[arg] = true
		}
	}

	if len(missing) == 0 {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if len(data) > 0 && data[len(data)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	_, err = f.WriteString(strings.Join(missing, "\n") + "\n")
	return err
}

func writeRunFile(runFile *os.File, javaPath string, extraArgsSlice []string, recMem int, runJarName string) error {
	var err error
	extraArgs := strings.Join(extraArgsSlice, " ")
	if len(extraArgsSlice) > 0 {
		extraArgs = " " + extraArgs
	}

	startCommand := fmt.Sprintf("\"%s\"%s -Xmx%dM -jar %s nogui\"", javaPath, extraArgs, recMem, runJarName)
	if runtime.GOOS == "windows" {
		_, err = runFile.WriteString(startCommand)
		if err != nil {
			return err
		}
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		startCommand = fmt.Sprintf("#!/usr/bin/env sh\n%s", startCommand)
		_, err = runFile.WriteString(startCommand)
		if err != nil {
			return err
		}
	}

	return nil
}
