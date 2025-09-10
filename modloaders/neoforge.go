package modloaders

import (
	"bufio"
	"fmt"
	"ftb-server-downloader/structs"
	"ftb-server-downloader/util"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	semVer "github.com/hashicorp/go-version"
	"github.com/pterm/pterm"
)

type NeoForge struct {
	InstallDir   string
	Targets      structs.ModpackTargets
	Memory       structs.Memory
	IsAfterSplit bool
}

const neoForgeMaven = "https://maven.neoforged.net"

func GetNeoForge(target structs.ModpackTargets, memory structs.Memory, installDir string) NeoForge {
	// After 1.20.2 NeoForge changed their package names
	isAfterSplit := false
	mcVersion, _ := semVer.NewVersion(target.McVersion)
	breakingMcVersion, _ := semVer.NewVersion("1.20.2")
	if mcVersion.GreaterThanOrEqual(breakingMcVersion) {
		isAfterSplit = true
	}

	return NeoForge{
		Targets:      target,
		Memory:       memory,
		IsAfterSplit: isAfterSplit,
		InstallDir:   installDir,
	}
}

func (s NeoForge) GetDownload() ([]structs.File, error) {
	var mlFiles []structs.File

	installerUrl := fmt.Sprintf("%s/releases/net/neoforged/neoforge/%s/neoforge-%s-installer.jar", neoForgeMaven, s.Targets.ModLoader.Version, s.Targets.ModLoader.Version)
	jarName := fmt.Sprintf("neoforge-%s-installer.jar", s.Targets.ModLoader.Version)
	if !s.IsAfterSplit {
		jarName = fmt.Sprintf("forge-%s-%s-installer.jar", s.Targets.McVersion, s.Targets.ModLoader.Version)
		installerUrl = fmt.Sprintf("%s/releases/net/neoforged/forge/%s-%s/forge-%s-%s-installer.jar", neoForgeMaven, s.Targets.McVersion, s.Targets.ModLoader.Version, s.Targets.McVersion, s.Targets.ModLoader.Version)
	}

	mlFiles = append(mlFiles, structs.File{
		Name: jarName,
		Url:  installerUrl,
	})
	return mlFiles, nil
}

func (s NeoForge) Install(useOwnJava bool) error {
	installerName := fmt.Sprintf("neoforge-%s-installer.jar", s.Targets.ModLoader.Version)
	if !s.IsAfterSplit {
		installerName = fmt.Sprintf("forge-%s-%s-installer.jar", s.Targets.McVersion, s.Targets.ModLoader.Version)
	}
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
	cmd := exec.Command(jrePath, "-jar", installerName, "--installServer")
	cmd.Dir = s.InstallDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	pterm.Info.Println("Running NeoForge installer")
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("error running neoforge installer: %s", err.Error())
	}
	if err = cmd.Wait(); err != nil {
		// Todo test this with errors.As
		if err, ok := err.(*exec.ExitError); ok {
			if err.ExitCode() != 0 {
				return fmt.Errorf("neoforge installer failed with exit code %d", err.ExitCode())
			}
		} else {
			return fmt.Errorf("error waiting for command: %s", err.Error())
		}
	}
	pterm.Success.Println("NeoForge installed successfully")
	// _ = os.Remove(filepath.Join(s.InstallDir, installerName) + ".log")
	_ = os.Remove(filepath.Join(s.InstallDir, installerName))

	err = s.startScript(useOwnJava)
	if err != nil {
		return err
	}
	return nil
}

func (s NeoForge) startScript(ownJava bool) error {
	pterm.Debug.Println("Use own java:", ownJava)
	argsFilePath := filepath.Join(s.InstallDir, "user_jvm_args.txt")
	var runScriptPath string
	if runtime.GOOS == "windows" {
		runScriptPath = filepath.Join(s.InstallDir, "run.bat")
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		runScriptPath = filepath.Join(s.InstallDir, "run.sh")
	}
	pterm.Debug.Println("argFilePath:", argsFilePath)
	pterm.Debug.Println("runScriptPath:", runScriptPath)

	if argsExist, _ := util.PathExists(argsFilePath); argsExist {
		argsFile, err := os.Open(argsFilePath)
		if err != nil {
			return err
		}
		defer argsFile.Close()

		scanner := bufio.NewScanner(argsFile)
		hasXmx := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "-Xmx") {
				hasXmx = true
				break
			}
		}

		if !hasXmx {
			argsFile, err = os.OpenFile(argsFilePath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer argsFile.Close()

			_, err = argsFile.WriteString(fmt.Sprintf("\n-Xmx%dM", s.Memory.Recommended))
			if err != nil {
				return err
			}
		}
	}

	if runExists, _ := util.PathExists(runScriptPath); runExists {
		pterm.Debug.Println("Parsing run script")
		file, _ := os.Open(runScriptPath)
		defer file.Close()

		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			match, _ := regexp.MatchString("^(java).+$", line)
			if match {
				if ownJava {
					pterm.Debug.Println("Replacing java path in run script")
					javaPath, err := util.GetJavaPath(s.Targets.JavaVersion)
					if err != nil {
						return err
					}
					line = regexp.MustCompile("^java").
						ReplaceAllString(line, fmt.Sprintf("\"%s\"", javaPath))
				}

				if runtime.GOOS == "windows" {
					line = regexp.MustCompile(`%\*`).
						ReplaceAllString(line, fmt.Sprintf("%s", "nogui %*"))
				} else if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
					line = regexp.MustCompile(`"\$@"`).
						ReplaceAllString(line, fmt.Sprintf("%s", "nogui \"$@\""))
				}
			}
			lines = append(lines, line)
		}

		file.Close()

		if ownJava {
			// Rewrite the file with our own java path
			file, _ = os.Create(runScriptPath)
			defer file.Close()

			writer := bufio.NewWriter(file)
			for _, line := range lines {
				_, _ = writer.WriteString(line + "\n")
			}
			_ = writer.Flush()
		}
	}

	return nil
}
