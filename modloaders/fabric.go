package modloaders

import (
	"encoding/json"
	"fmt"
	"ftb-server-downloader/structs"
	"ftb-server-downloader/util"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/pterm/pterm"
)

const fabricMeta = "https://meta.fabricmc.net"

type Fabric struct {
	InstallDir      string
	Targets         structs.ModpackTargets
	Memory          structs.Memory
	IsAutoVersion   bool
	FabricInstaller FabricInstaller
}

type FabricInstaller struct {
	URL     string `json:"url"`
	Maven   string `json:"maven"`
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func GetFabric(target structs.ModpackTargets, memory structs.Memory, installDir string) (Fabric, error) {
	fabricInstaller, err := getInstaller()
	if err != nil {
		return Fabric{}, err
	}

	return Fabric{
		InstallDir:      installDir,
		Targets:         target,
		Memory:          memory,
		FabricInstaller: fabricInstaller[0],
	}, nil
}

func (s Fabric) GetDownload() ([]structs.File, error) {
	var mlFiles []structs.File

	mlFiles = append(mlFiles, structs.File{
		Name: fmt.Sprintf("fabric-installer-%s.jar", s.FabricInstaller.Version),
		Url:  s.FabricInstaller.URL,
	})

	return mlFiles, nil
}

func (s Fabric) Install(useOwnJava bool) error {
	installerName := fmt.Sprintf("fabric-installer-%s.jar", s.FabricInstaller.Version)
	exists, err := util.PathExists(filepath.Join(s.InstallDir, installerName))
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("installer %s does not exist", installerName)
	}

	jrePath := "java"
	if useOwnJava {
		jrePath, err = util.GetJavaPath(s.Targets.JavaVersion)
		if err != nil {
			jrePath = "java"
		} else {
			jrePath = filepath.Join(s.InstallDir, jrePath)
		}
	}

	pterm.Debug.Printfln("JRE Path: %s", jrePath)
	cmd := exec.Command(jrePath, "-jar", installerName, "server", "-mcversion", s.Targets.McVersion, "-loader", s.Targets.ModLoader.Version, "-downloadMinecraft")
	cmd.Dir = s.InstallDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	pterm.Info.Println("Running Fabric installer")
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("error running fabric installer: %s", err.Error())
	}
	if err = cmd.Wait(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if err.ExitCode() != 0 {
				return fmt.Errorf("fabric installer failed with exit code %d", err.ExitCode())
			}
		} else {
			return fmt.Errorf("error waiting for command: %s", err.Error())
		}
	}
	pterm.Success.Println("Fabric installed successfully")
	_ = os.Remove(filepath.Join(s.InstallDir, installerName))

	err = s.startScript(useOwnJava)

	return nil
}

func getInstaller() ([]FabricInstaller, error) {
	url := fmt.Sprintf("%s/v2/versions/installer", fabricMeta)
	resp, err := util.DoGet(url)
	if err != nil {
		return []FabricInstaller{}, err
	}
	defer resp.Body.Close()
	var fabricInstaller []FabricInstaller

	err = json.NewDecoder(resp.Body).Decode(&fabricInstaller)
	if err != nil {
		return []FabricInstaller{}, err
	}

	return fabricInstaller, nil
}

func (s Fabric) startScript(ownJava bool) error {
	pterm.Debug.Println("Use own java:", ownJava)
	var runScriptPath string
	if runtime.GOOS == "windows" {
		runScriptPath = filepath.Join(s.InstallDir, "start.bat")
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		runScriptPath = filepath.Join(s.InstallDir, "start.sh")
	}
	pterm.Debug.Println("runScriptPath:", runScriptPath)

	log4jFix, err := Log4JFixer(s.InstallDir, s.Targets.McVersion)
	if err != nil {
		pterm.Warning.Printfln("Failed to apply log4j fix: %s", err.Error())
	}

	runFile, err := os.OpenFile(runScriptPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer runFile.Close()
	javaPath := "java"
	if ownJava {
		javaPath, err = util.GetJavaPath(s.Targets.JavaVersion)
		if err != nil {
			javaPath = "java"
		}
	}
	runJarName := "fabric-server-launch.jar"

	if runtime.GOOS == "windows" {
		_, err = runFile.WriteString(fmt.Sprintf("\"%s\" -jar %s -Xmx%dM %s nogui", javaPath, log4jFix, s.Memory.Recommended, runJarName))
		if err != nil {
			return err
		}
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		_, err = runFile.WriteString(fmt.Sprintf("#!/usr/bin/env sh\n\"%s\" -jar %s -Xmx%dM %s nogui", javaPath, log4jFix, s.Memory.Recommended, runJarName))
		if err != nil {
			return err
		}
	}

	return nil
}
