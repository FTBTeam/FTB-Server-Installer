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

const (
	forgeMaven = "https://maven.minecraftforge.net"
)

var (
	jarName string
)

type Forge struct {
	InstallDir string
	Targets    structs.ModpackTargets
	Memory     structs.Memory
}

func GetForge(target structs.ModpackTargets, memory structs.Memory, installDir string) Forge {

	return Forge{
		Targets:    target,
		Memory:     memory,
		InstallDir: installDir,
	}
}

func (s Forge) GetDownload() ([]structs.File, error) {
	var mlFiles []structs.File
	var installerUrl string

	installerUrl = fmt.Sprintf("%s/releases/net/minecraftforge/forge/%s-%s/forge-%s-%s-installer.jar", forgeMaven, s.Targets.McVersion, s.Targets.ModLoader.Version, s.Targets.McVersion, s.Targets.ModLoader.Version)
	jarName = fmt.Sprintf("forge-%s-%s-installer.jar", s.Targets.McVersion, s.Targets.ModLoader.Version)
	if !doesForgeExist(installerUrl) {
		installerUrl = fmt.Sprintf("%s/releases/net/minecraftforge/forge/%s-%s-%s/forge-%s-%s-%s-installer.jar", forgeMaven, s.Targets.McVersion, s.Targets.ModLoader.Version, s.Targets.McVersion, s.Targets.McVersion, s.Targets.ModLoader.Version, s.Targets.McVersion)
		jarName = fmt.Sprintf("forge-%s-%s-%s-installer.jar", s.Targets.McVersion, s.Targets.ModLoader.Version, s.Targets.McVersion)
		if !doesForgeExist(installerUrl) {
			installerUrl = fmt.Sprintf("%s/releases/net/minecraftforge/forge/%s-%s/forge-%s-%s-universal.zip", forgeMaven, s.Targets.McVersion, s.Targets.ModLoader.Version, s.Targets.McVersion, s.Targets.ModLoader.Version)
			jarName = fmt.Sprintf("forge-%s-%s-universal.zip", s.Targets.McVersion, s.Targets.ModLoader.Version)
			if !doesForgeExist(installerUrl) {
				return mlFiles, fmt.Errorf("cant find forge version %s", s.Targets.ModLoader.Version)
			}
		}
	}

	mlFiles = append(mlFiles, structs.File{
		Name: jarName,
		Url:  installerUrl,
	})
	return mlFiles, nil
}

func (s Forge) Install(useOwnJava bool) error {

	exists, err := util.PathExists(filepath.Join(s.InstallDir, jarName))
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("installer %s does not exist", jarName)
	}

	if filepath.Ext(jarName) == ".jar" {
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
		cmd := exec.Command(jrePath, "-jar", jarName, "--installServer")
		cmd.Dir = s.InstallDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		pterm.Info.Println("Running Forge installer")
		if err = cmd.Start(); err != nil {
			return fmt.Errorf("error running forge installer: %s", err.Error())
		}
		if err = cmd.Wait(); err != nil {
			if err, ok := err.(*exec.ExitError); ok {
				if err.ExitCode() != 0 {
					return fmt.Errorf("forge installer failed with exit code %d, error: %s", err.ExitCode(), err.Error())
				}
			} else {
				return fmt.Errorf("error waiting for command: %s", err.Error())
			}
		}
		pterm.Success.Println("Forge installed successfully")
		// _ = os.Remove(filepath.Join(s.InstallDir, jarName) + ".log")
		_ = os.Remove(filepath.Join(s.InstallDir, jarName))
	} else if filepath.Ext(jarName) == ".zip" {
		pathExists, err := util.PathExists(filepath.Join(s.InstallDir, fmt.Sprintf("minecraft_server.%s.jar", s.Targets.McVersion)))
		if err != nil {
			return err
		}
		if pathExists {
			_ = os.Remove(filepath.Join(s.InstallDir, fmt.Sprintf("minecraft_server.%s.jar", s.Targets.McVersion)))
		}
		vanilla, err := GetVanilla(s.Targets, s.InstallDir)
		if err != nil {
			return err
		}
		vanillaDl, err := vanilla.GetDownload()
		if err != nil {
			return err
		}
		dest := filepath.Join(s.InstallDir, vanillaDl[0].Path, vanillaDl[0].Name)
		fDl, err := util.NewDownload(dest, vanillaDl[0].Url)
		if err != nil {
			return err
		}
		err = fDl.Do()
		if err != nil {
			return err
		}

		err = util.CombineZip(filepath.Join(s.InstallDir, jarName), filepath.Join(s.InstallDir, fmt.Sprintf("minecraft_server.%s.jar", s.Targets.McVersion)))
		if err != nil {
			return err
		}
		_ = os.Remove(filepath.Join(s.InstallDir, jarName))
	}

	err = s.startScript(useOwnJava)
	if err != nil {
		return err
	}

	return nil
}

func doesForgeExist(url string) bool {
	resp, err := util.DoHead(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true
}

func (s Forge) startScript(ownJava bool) error {
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

	log4jFix, err := Log4JFixer(s.InstallDir, s.Targets.McVersion)
	if err != nil {
		pterm.Warning.Printfln("Failed to apply log4j fix: %s", err.Error())
	}

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

			customArgs := fmt.Sprintf("\n-Xmx%dM\n%s", s.Memory.Recommended, log4jFix)
			_, err = argsFile.WriteString(customArgs)
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

		_ = file.Close()

		// Rewrite the file with our changes
		file, _ = os.Create(runScriptPath)
		defer file.Close()

		writer := bufio.NewWriter(file)
		for _, line := range lines {
			_, _ = writer.WriteString(line + "\n")
		}
		_ = writer.Flush()
	} else {
		runFile, err := os.OpenFile(strings.ReplaceAll(runScriptPath, "run", "start"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
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
		dir, err := os.ReadDir(s.InstallDir)
		if err != nil {
			return err
		}
		var runJarName string
		preForgeJarVer, _ := semVer.NewVersion("1.5.1")
		mcVer, _ := semVer.NewVersion(s.Targets.McVersion)

		var re *regexp.Regexp
		if mcVer.GreaterThan(preForgeJarVer) {
			re = regexp.MustCompile(`^forge-(\d+.\d+.\d+)-(\d+.\d+.\d+(.\d+)?)(-\d+.\d+.\d+)?(-[a-zA-Z]+)?.jar$`)

		} else {
			re = regexp.MustCompile(`^minecraft_server.(\d+.\d+.\d+)?.jar$`)
		}

		var filesInDir []pterm.TreeNode
		for _, file := range dir {
			if !file.IsDir() {
				filesInDir = append(filesInDir, pterm.TreeNode{
					Text: file.Name(),
				})
				matches := re.MatchString(file.Name())
				if matches {
					runJarName = file.Name()
					break
				}
			}
		}

		if pterm.PrintDebugMessages {
			_ = pterm.DefaultTree.WithRoot(pterm.TreeNode{Text: "Files in dir:", Children: filesInDir}).Render()
		}
		pterm.Debug.Println("Runtime jar file:", runJarName)

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
	}

	return nil
}
