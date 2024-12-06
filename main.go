package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"ftb-server-downloader/modloaders"
	"ftb-server-downloader/repos"
	"ftb-server-downloader/structs"
	"ftb-server-downloader/util"
	"github.com/cavaliergopher/grab/v3"
	"github.com/codeclysm/extract/v4"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	packId        int
	versionId     int
	installDir    string
	threads       int
	provider      string
	auto          bool
	force         bool
	latest        bool
	apiKey        string
	validate      bool
	skipModloader bool
	noJava        bool
	noColours     bool
	verbose       bool

	logFile *os.File
)

func init() {

	if util.ReleaseVersion == "" || util.ReleaseVersion == "main" {
		util.ReleaseVersion = "v0.0.0-beta.0"
	}

	if util.GitCommit == "" {
		util.GitCommit = "Dev"
	}

	userAgentVersion := util.ReleaseVersion
	if strings.HasPrefix(util.ReleaseVersion, "v") {
		userAgentVersion = strings.TrimPrefix(util.ReleaseVersion, "v")
	}

	util.UserAgent = fmt.Sprintf("ftb-server-installer/%s", userAgentVersion)
}

func main() {
	flag.StringVar(&provider, "provider", "ftb", "Modpack provider (Currently only 'ftb' is supported)")
	flag.IntVar(&packId, "pack", 0, "Modpack ID")
	flag.IntVar(&versionId, "version", 0, "Modpack version ID, if not provided, the latest version will be used")
	flag.StringVar(&installDir, "dir", "", "Installation directory")
	flag.BoolVar(&auto, "auto", false, "Dont ask questions, just install the server")
	flag.BoolVar(&latest, "latest", false, "Gets the latest (alpha/beta/release) version of the modpack")
	flag.BoolVar(&force, "force", false, "Force the modpack install, dont ask questions just continue (only works with -auto)")
	flag.IntVar(&threads, "threads", runtime.NumCPU()*2, "Number of threads to use (Default: CPU Cores * 2)")
	flag.StringVar(&apiKey, "apikey", "public", "FTB API key (Only for private FTB modpacks)")
	flag.BoolVar(&validate, "validate", false, "Validate the modpack after install")
	flag.BoolVar(&skipModloader, "skip-modloader", false, "Skip installing the modloader")
	flag.BoolVar(&noJava, "no-java", false, "Do not install Java")
	flag.BoolVar(&noColours, "no-colours", false, "Do not display console/terminal colours")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.Parse()

	var err error
	logFile, err = os.OpenFile("ftb-server-installer.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}

	util.LogMw = io.MultiWriter(os.Stdout, util.NewCustomWriter(logFile))
	pterm.SetDefaultOutput(util.LogMw)

	pterm.Debug.Prefix = pterm.Prefix{
		Text:  "DEBUG",
		Style: pterm.NewStyle(pterm.BgLightMagenta, pterm.FgBlack),
	}
	pterm.Debug.MessageStyle = pterm.NewStyle(98)

	if noColours {
		pterm.DisableStyling()
	}

	logo, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("F", pterm.NewStyle(pterm.FgCyan)),
		putils.LettersFromStringWithStyle("T", pterm.NewStyle(pterm.FgGreen)),
		putils.LettersFromStringWithStyle("B", pterm.NewStyle(pterm.FgRed))).Srender()
	pterm.DefaultCenter.Println(logo)
	pterm.DefaultCenter.WithCenterEachLineSeparately().Printfln("Server installer version: %s(%s)\n%s", util.ReleaseVersion, util.GitCommit, time.Now().UTC().Format(time.RFC1123))
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(pterm.Bold.Sprintf("Installer Issue tracker\nhttps://github.com/FTBTeam/FTB-Server-Installer/issues"))

	updateAvailable, err := util.CheckForUpdate()
	if err != nil {
		pterm.Warning.Printfln("Error checking for update: %v", err)
	}
	if updateAvailable {
		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}
		installerName := fmt.Sprintf("ftb-server-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)
		updateUrl := fmt.Sprintf("https://cdn.feed-the-beast.com/bin/server-installer/latest/%s", installerName)

		updateStyle := pterm.NewStyle(pterm.FgLightMagenta, pterm.Bold)
		updateURLStyle := pterm.NewStyle(pterm.FgLightCyan, pterm.Bold, pterm.Underscore)
		pterm.Info.Printfln("%s%s", updateStyle.Sprint("Installer update available download from: "), updateURLStyle.Sprint(updateUrl))
		pterm.Println()
	}
	if verbose {
		pterm.EnableDebugMessages()
		pterm.Debug.Println("Verbose output enabled")
	}

	abs, err := filepath.Abs(installDir)
	if err != nil {
		pterm.Fatal.Println("Error getting absolute path:", err.Error())
	}
	installDir = abs

	defer logFile.Close()
	// Get the pack ID and version ID from the installer name if not provided as flags
	if packId == 0 {
		pId, vId, err := util.ParseInstallerName(filepath.Base(os.Args[0]))
		if err != nil {
			pterm.Warning.Println("Unable to parse installer name for modpack and version id:", err)
			pId, vId, err = modpackQuestion()
			if err != nil {
				pterm.Fatal.Println(err)
			}
		}
		packId = pId
		versionId = vId
	}

	// Get the provider
	selectedProvider, err := getProvider()
	if err != nil {
		pterm.Fatal.Printfln("Error getting provider: %s\nValid providers are 'ftb'", err.Error())
	}
	pterm.Debug.Printfln("Got provider '%s'", provider)

	var filesToDownload []structs.File

	// Get modpack details from the provider
	modpack, err := selectedProvider.GetModpack()
	if err != nil {
		selectedProvider.FailedInstall()
		pterm.Error.Println("Error getting modpack:", err.Error())
		os.Exit(1)
	}
	pterm.Debug.Printfln("Modpack: %+v", modpack)

	// Get the latest version id if not provided
	if versionId == 0 {
		latestVersion, err := getLatestRelease(modpack.Versions, latest)
		if err != nil {
			pterm.Error.Println("Error getting latest release:", err.Error())
			os.Exit(1)
		}
		selectedProvider.SetVersionId(latestVersion.Id)
		pterm.Info.Printfln("No version provided, using latest version: %d", latestVersion.Id)
	}

	// Get the version information for the modpack from the provider
	modpackVersion, err := selectedProvider.GetVersion()
	if err != nil {
		selectedProvider.FailedInstall()
		pterm.Error.Println("Error getting modpack version:", err.Error())
		os.Exit(1)
	}
	filesToDownload = append(filesToDownload, modpackVersion.Files...)

	// build the version manifest
	manifest := structs.Manifest{
		Id:             modpack.Id,
		Name:           modpack.Name,
		VersionName:    modpackVersion.Name,
		VersionId:      modpackVersion.Id,
		ModpackTargets: modpackVersion.Targets,
		Files:          modpackVersion.Files,
	}

	// Check if the install location exists, if it doesnt, ask if they want to create the folder(s)
	exists, err := util.PathExists(installDir)
	if err != nil {
		selectedProvider.FailedInstall()
		pterm.Fatal.Println("Unable to check if path exists:", err.Error())
	}
	mkdir := true
	if !exists {
		if !auto {
			mkdir = util.ConfirmYN("Install folder does not exists, do you want to create it?", true, pterm.Info.MessageStyle)
			if !mkdir {
				pterm.Error.Println("Installation path does not exist...")
				os.Exit(1)
			}
		}
	}

	var updatedFiles, removedFiles, unchangedFiles []structs.File
	updateMsg := ""
	isUpdate := false
	if exists {
		manifestExists, err := util.PathExists(filepath.Join(installDir, util.ManifestName))
		if err != nil {
			return
		}

		if !manifestExists {
			installDirEmpty, err := util.IsEmptyDir(installDir)
			if err != nil {
				selectedProvider.FailedInstall()
				pterm.Fatal.Println("Error checking if directory is empty:", err.Error())
			}

			if !installDirEmpty {
				if !auto {
					pterm.Warning.Printfln("Install directory is not empty, installing the modpack may cause issues")
					cont := util.ConfirmYN("Would you like to continue?", false, pterm.Warning.MessageStyle)
					if !cont {
						pterm.Error.Println("Installation path is not empty, exiting...")
						os.Exit(1)
					}
				}
				if auto && !force {
					pterm.Warning.Printfln("Install directory is not empty, installing the modpack may cause issues")
					pterm.Warning.Printfln("To force install use the -force flag")
					os.Exit(1)
				}
			}
		}

		if manifestExists {
			existingManifest, err := util.ReadManifest(installDir)
			if err != nil {
				selectedProvider.FailedInstall()
				pterm.Fatal.Println("Error reading manifest:", err.Error())
			}

			isSamePack := isSameModpack(existingManifest, manifest)

			if !isSamePack {
				if !auto && !force {
					pterm.Warning.Printfln("You currently have a different modpack installed, installing this modpack may cause issues")
					cont := util.ConfirmYN("Would you like to continue?", false, pterm.Warning.MessageStyle)
					if !cont {
						os.Exit(1)
					}
				}
				if auto && !force {
					pterm.Warning.Printfln("You currently have a different modpack installed, installing this modpack may cause issues")
					pterm.Warning.Printfln("To force install use the -force flag")
					os.Exit(1)
				}
			}

			sameVersion := isSameModpackVersion(existingManifest, manifest)

			if !sameVersion && isSamePack {
				isUpdate, err = checkUpdate(existingManifest, manifest)
				if err != nil {
					selectedProvider.FailedInstall()
					pterm.Fatal.Println("Check Update error:", err.Error())
				}

				if isUpdate {
					existingManifest, err := util.ReadManifest(installDir)
					if err != nil {
						selectedProvider.FailedInstall()
						pterm.Fatal.Println("Error reading manifest:", err.Error())
						return
					}
					updatedFiles, removedFiles, unchangedFiles, err = computeUpdatedFiles(existingManifest.Files, manifest.Files)
					if err != nil {
						return
					}
					filesToDownload = removeUnchangedFiles(filesToDownload, unchangedFiles)
				}
			}
		}
	}

	// set up the modloader getter and installer
	modLoader, err := getModLoader(modpackVersion.Targets, modpackVersion.Memory)
	if err != nil {
		selectedProvider.FailedInstall()
		pterm.Error.Println("Error getting modloader:", err.Error())
		os.Exit(1)
	}

	// Add the modloader downloads to the files list
	mlDownloads, err := modLoader.GetDownload()
	if err != nil {
		selectedProvider.FailedInstall()
		pterm.Fatal.Println("Error getting mod loader downloads:", err.Error())
	}
	filesToDownload = append(filesToDownload, mlDownloads...)

	if isUpdate {
		updateMsg = fmt.Sprintf("\nUnchanged Files: %d\nFiles changed: %d\nFiles removed: %d", len(unchangedFiles), len(updatedFiles), len(removedFiles))
	}

	pterm.Debug.Printfln("Files to download: %d", len(filesToDownload))

	// Show a quick overview of the pack they are installing then ask if they want to continue with downloading the pack
	pterm.Info.Printfln("Fetched modpack:\nName: %s\nVersion: %s\nModLoader: %s (%s)\nIs Update: %t%s\nInstall Path: %s", modpack.Name, modpackVersion.Name, modpackVersion.Targets.ModLoader.Name, modpackVersion.Targets.ModLoader.Version, isUpdate, updateMsg, installDir)
	if !auto {
		cont := util.ConfirmYN("Do you want to continue?", true, pterm.Info.MessageStyle)
		if !cont {
			os.Exit(1)
		}
	}
	// Ask the user if they want to download java then set the noJava flag depending on their answer
	var java structs.File
	jreAlreadyExists := false
	jrePath, _ := util.GetJavaPath(modpackVersion.Targets.JavaVersion)
	if _, err = os.Stat(filepath.Join(installDir, jrePath)); err == nil {
		jreAlreadyExists = true
	}

	// If noJava is set or we already have java downloaded, we skip the java download
	if !noJava && !auto && !jreAlreadyExists {
		noJava = !util.ConfirmYN("Do you want to download java?", true, pterm.Info.MessageStyle)
	}
	if !noJava && !jreAlreadyExists {
		java, err = util.GetJava(modpackVersion.Targets.JavaVersion)
		if err != nil {
			selectedProvider.FailedInstall()
			pterm.Fatal.Println("Error getting java:", err.Error())
		}
		filesToDownload = append(filesToDownload, java)
	}

	if mkdir {
		err = os.MkdirAll(installDir, 0777)
		if err != nil {
			selectedProvider.FailedInstall()
			pterm.Fatal.Println("Unable to create install directory:", err.Error())
		}
	} else {
		pterm.Error.Println("Installation path does not exist...")
		os.Exit(1)
	}

	if isUpdate {
		for _, f := range removedFiles {
			err := os.Remove(filepath.Join(installDir, f.Path, f.Name))
			if err != nil {
				pterm.Error.Println(err.Error())
				continue
			}
		}

		// For now, we remove the files that have been updated so they can be freshly downloaded.
		for _, f := range updatedFiles {
			err := os.Remove(filepath.Join(installDir, f.Path, f.Name))
			if err != nil {
				pterm.Error.Println(err.Error())
				continue
			}
		}

		// Remove unchanged files from filesToDownload
		for _, f := range unchangedFiles {
			for i, v := range filesToDownload {
				if v.Name == f.Name && v.Path == f.Path {
					filesToDownload = append(filesToDownload[:i], filesToDownload[i+1:]...)
				}
			}
		}
	}

	// download the modpack files
	pterm.Info.Printfln("Starting mod pack download...")
	err = doDownload(filesToDownload...)
	if err != nil {
		selectedProvider.FailedInstall()
		pterm.Fatal.Printfln(err.Error())
	}

	pterm.Info.Printfln("Modpack files downloaded")

	// If we downloaded java, extract the files to a jre folder
	if !noJava && !jreAlreadyExists {

		javaFile, err := os.Open(filepath.Join(installDir, java.Name))
		if err != nil {
			selectedProvider.FailedInstall()
			pterm.Fatal.Println("Error opening java archive", err.Error())
		}
		javaPkg := bufio.NewReader(javaFile)

		var shift = func(path string) string {
			// Apparently zips in windows can use / instead of \
			// So we need to check if the path is using / or \
			sep := filepath.Separator
			if len(strings.Split(path, "\\")) > 1 {
				sep = '\\'
			} else if len(strings.Split(path, "/")) > 1 {
				sep = '/'
			}

			parts := strings.Split(path, string(sep))
			parts = parts[1:]
			join := strings.Join(parts, string(sep))
			return join
		}
		err = extract.Archive(context.TODO(), javaPkg, filepath.Join(installDir, "jre", modpackVersion.Targets.JavaVersion), shift)
		if err != nil {
			selectedProvider.FailedInstall()
			pterm.Fatal.Println("Error extracting java archive:", err.Error())
		}
		javaFile.Close()
		err = os.Remove(filepath.Join(installDir, java.Name))
		if err != nil {
			pterm.Warning.Println("Error removing java archive:", err.Error())
		}
	}

	// Ask if the user would like to run the modloader installer
	// todo: if the modloader is already installed check if its the same and ignore the update
	if !auto && !skipModloader {
		skipModloader = !util.ConfirmYN(
			fmt.Sprintf("Would you like to run the %s installer?", modpackVersion.Targets.ModLoader.Name),
			true,
			pterm.Info.MessageStyle,
		)
	}
	if noJava && !util.OsJavaExists() {
		// Revisit this, and possibly ask if they want to download java
		pterm.Warning.Printfln("Java is not installed, skipping modloader installer")
		skipModloader = true
	}
	if !skipModloader {
		err = modLoader.Install(!noJava)
		if err != nil {
			selectedProvider.FailedInstall()
			pterm.Fatal.Println("ModLoader installer error:", err.Error())
		}
	}

	// if the validate flag has been enabled, validate the files we downloaded and check if they match what they should be
	if validate {
		err = runValidation(manifest)
		if err != nil {
			selectedProvider.FailedInstall()
			pterm.Fatal.Println("Error running validation:", err.Error())
		}
	}

	// write the version manifest
	err = util.WriteManifest(installDir, manifest)
	if err != nil {
		selectedProvider.FailedInstall()
		pterm.Fatal.Println("Error creating manifest:", err.Error())
	}

	/*// Ask if the user would like to copy the overrides
	overridesExist, err := util.PathExists(filepath.Join(installDir, "overrides"))
	if err != nil {
		pterm.Fatal.Println("Error checking if overrides exists:", err.Error())
	}
	if overridesExist {
		copyOverriddenFiles()
	}*/

	selectedProvider.SuccessfulInstall()
	pterm.Success.Println("Modpack installed successfully")
}

// getProvider Gets and sets up the repo provider
func getProvider() (repos.ModpackRepo, error) {
	switch provider {
	case "ftb":
		return repos.GetFTB(packId, versionId, apiKey), nil
	// case "curseforge":
	//	return repos.GetCurseForge(packId, versionId), nil
	default:
		return nil, errors.New(fmt.Sprintf("'%s' not recognised", provider))
	}
}

// getModLoader function to get the correct modloader for the pack
func getModLoader(targets structs.ModpackTargets, memory structs.Memory) (modloaders.ModLoader, error) {
	switch targets.ModLoader.Name {
	case "neoforge":
		return modloaders.GetNeoForge(targets, memory, installDir), nil
	case "fabric":
		return modloaders.GetFabric(targets, memory, installDir)
	case "forge":
		return modloaders.GetForge(targets, memory, installDir), nil
	default:
		return nil, errors.New(fmt.Sprintf("'%s' not recognised", targets.ModLoader.Name))
	}
}

func doDownload(files ...structs.File) error {
	// failedDownloads = []structs.File{}
	p, _ := pterm.DefaultProgressbar.WithTitle("Downloading...").WithTotal(len(files)).Start()
	var wg sync.WaitGroup

	threadLimit := make(chan int, threads)
	var pCount atomic.Uint64
	for _, file := range files {
		wg.Add(1)
		threadLimit <- 1
		go func() {
			defer func() { wg.Done(); <-threadLimit; pCount.Add(1); p.Current = int(pCount.Load()) }()
			grab.DefaultClient.UserAgent = util.UserAgent
			destPath := filepath.Join(installDir, file.Path, file.Name)
			urls := append([]string{file.Url}, file.Mirrors...)
			var attempts = 0
			for _, u := range urls {
				attempts++
				pterm.Debug.Printfln("Downloading file: %s from %s | attempt: %d | Urls %d", file.Name, u, attempts, len(urls))
				reqUrl := u

				req, err := grab.NewRequest(destPath, reqUrl)
				if err != nil {
					pterm.Error.Println(err.Error())
					return
				}
				req.NoResume = true

				if file.Hash != "" {
					hexHash, _ := hex.DecodeString(file.Hash)
					switch file.HashType {
					case "sha1":
						req.SetChecksum(sha1.New(), hexHash, false)
					case "sha256":
						req.SetChecksum(sha256.New(), hexHash, false)
					default:
						pterm.Warning.Printfln("Unsupported hash type: %s", file.HashType)
					}
				}

				resp := grab.DefaultClient.Do(req)
				if resp.Err() == nil {
					pterm.Debug.Printfln("Downloaded file: %s", file.Name)
					break
				} else if resp.Err() != nil {
					_ = os.Remove(filepath.Join(installDir, file.Path, file.Name))
					pterm.Warning.Printfln("Failed to download:\nFile: %s (%s)\nResp Status: %s(%d)\nError:%s", file.Name, reqUrl, resp.HTTPResponse.Status, resp.HTTPResponse.StatusCode, resp.Err().Error())
					if attempts == len(urls) {
						pterm.Error.Printfln("Failed to download file: %s\nAll mirrors failed", file.Name)
						os.Exit(1)
					}
				}
			}
		}()
	}
	wg.Wait()
	p.UpdateTitle("Download complete")
	p.Current = int(pCount.Load()) // Hack to fix race condition in progress bar printer
	p.Stop()
	return nil
}

func runValidation(manifest structs.Manifest) error {
	var invalidFiles []structs.File
	for _, f := range manifest.Files {
		if f.HashType != "" && f.Hash != "" {
			fileHash, err := util.FileHash(filepath.Join(installDir, f.Path, f.Name), f.HashType)
			if err != nil {
				pterm.Error.Println("Error getting file hash:", err.Error())
				continue
			}
			if fileHash != f.Hash {
				pterm.Warning.Printfln(fmt.Sprintf("Unexpected file hash from %s\nExpected: %s\nGot: %s", f.Name, f.Hash, fileHash))
				invalidFiles = append(invalidFiles, f)
			}
		}
	}

	if len(invalidFiles) > 0 {
		if !auto {
			show := util.ConfirmYN(
				fmt.Sprintf("%d files failed validation, would you like to repair them?", len(invalidFiles)),
				true,
				pterm.Info.MessageStyle,
			)
			if !show {
				return nil
			}
		}

		err := doDownload(invalidFiles...)
		if err != nil {
			return err
		}
	}

	return nil
}

func isSameModpack(currentManifest, newManifest structs.Manifest) bool {
	if currentManifest.Id != newManifest.Id {
		return false
	}

	return true
}

func isSameModpackVersion(currentManifest, newManifest structs.Manifest) bool {
	if currentManifest.Id != newManifest.Id {
		return false
	}
	if currentManifest.VersionId != newManifest.VersionId {
		return false
	}

	return true
}

func checkUpdate(currentManifest, newManifest structs.Manifest) (isUpdate bool, err error) {
	if currentManifest.Id != newManifest.Id {
		return false, errors.New("mismatched modpack")
	}

	if currentManifest.VersionId != newManifest.VersionId {
		if newManifest.VersionId > currentManifest.VersionId {
			return true, nil
		}
		if newManifest.VersionId < currentManifest.VersionId {
			if !auto {
				show := util.ConfirmYN(
					fmt.Sprintf("%s will be downgraded from %s to version %s, are you sure you want to downgrade?", newManifest.Name, currentManifest.VersionName, newManifest.VersionName),
					false,
					pterm.Warning.MessageStyle,
				)
				if !show {
					pterm.Info.Println("Cancelling update...")
					os.Exit(0)
				}
			}
			if auto && !force {
				pterm.Warning.Printfln("Cancelling update... %s would be downgraded from %s to %s. To force this downgrade use the -force flag", newManifest.Name, currentManifest.VersionName, newManifest.VersionName)
				os.Exit(1)
			} else if auto && force {
				pterm.Warning.Printfln("Forcing downgrade")
			}
			return true, nil
		}
	} else {
		return false, nil
	}

	return currentManifest.VersionId != newManifest.VersionId, nil
}

func computeUpdatedFiles(currentFiles, newFiles []structs.File) (updatedFiles, removedFiles, unchangedFiles []structs.File, err error) {
	for _, v1 := range currentFiles {
		fileFound := false
		fileChanged := false
		for _, v2 := range newFiles {
			if v1.Name == v2.Name && v1.Path == v2.Path {
				// file still exists, so check if it has changed
				if v1.Hash != v2.Hash {
					fileChanged = true
				} else {
					unchangedFiles = append(unchangedFiles, v1)
				}
				fileFound = true
			}
		}

		if !fileFound {
			removedFiles = append(removedFiles, v1)
		} else if fileChanged {
			updatedFiles = append(updatedFiles, v1)
		}
	}

	return
}

func removeUnchangedFiles(files []structs.File, unchangedFiles []structs.File) []structs.File {
	// removed unchanged files from files
	for _, f := range unchangedFiles {
		for i, v := range files {
			if v.Name == f.Name && v.Path == f.Path {
				files = append(files[:i], files[i+1:]...)
			}
		}
	}
	return files
}

func getLatestRelease(versions []structs.ModpackV, latest bool) (structs.ModpackV, error) {
	pterm.Debug.Printfln("versions: %+v", versions)
	for _, v := range versions {
		if !latest {
			if v.Type == "release" {
				return v, nil
			}
		} else {
			return v, nil
		}
	}
	if !latest {
		return structs.ModpackV{}, errors.New("no stable release found, please rerun the installer with the -latest flag or specify a version using the -version flag")
	}
	return structs.ModpackV{}, errors.New("no release found, please rerun the installer with the -version flag")
}

func modpackQuestion() (int, int, error) {
	sPId, _ := pterm.DefaultInteractiveTextInput.
		WithDefaultText("Please enter the modpack ID").
		Show()

	pId, err := strconv.Atoi(sPId)
	if err != nil {
		return 0, 0, err
	}

	getLatest := util.ConfirmYN("Would you like to get the latest version?", true, pterm.Info.MessageStyle)

	vId := 0
	if !getLatest {
		sVId, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultText("Please enter the version id").
			Show()

		vId, err = strconv.Atoi(sVId)
		if err != nil {
			return 0, 0, err
		}
	}

	return pId, vId, nil
}

/*func copyOverriddenFiles() {
	pterm.Info.Printfln("Overrides folder found")
	doCopy := true
	if !auto {
		doCopy = util.ConfirmYN("Would you like to copy the overrides folder contents?", true, pterm.Info.MessageStyle)
	} else {
		pterm.Info.Printfln("Copying overrides folder contents")
	}
	if doCopy {
		err := util.CopyDir(filepath.Join(installDir, "overrides"), filepath.Join(installDir))
		if err != nil {
			pterm.Fatal.Println("Error copying overrides folder:", err.Error())
		}
	}
}*/
