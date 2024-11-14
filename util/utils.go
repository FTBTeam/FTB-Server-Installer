package util

import (
	"archive/zip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"ftb-server-downloader/structs"
	semVer "github.com/hashicorp/go-version"
	"github.com/pterm/pterm"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

const (
	ManifestName   = ".manifest.json"
	adoptiumApiUrl = "https://api.adoptium.net"
)

var (
	ReleaseVersion string
	GitCommit      string
	UserAgent      string
	LogMw          io.Writer
)

func ParseInstallerName(filename string) (int, int, error) {
	re := regexp.MustCompile(`^.*?_(\d+)(?:_(\d+))?`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) < 3 {
		return 0, 0, errors.New("no pack/version id in installer name")
	}
	pId, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}
	vId := 0
	if matches[2] != "" {
		vId, err = strconv.Atoi(matches[2])
		if err != nil {
			return 0, 0, err
		}
	}

	return pId, vId, nil
}

func makeRequest(method, url string, requestHeaders map[string][]string) (*http.Response, error) {
	headers := map[string][]string{}
	for k, v := range requestHeaders {
		headers[k] = v
	}
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = headers

	return client.Do(req)
}

func DoGet(url string) (*http.Response, error) {
	headers := map[string][]string{
		"User-Agent": {UserAgent},
	}
	resp, err := makeRequest("GET", url, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, errors.New(fmt.Sprintf("Error: %d\n%s", resp.StatusCode, b))
	}
	return resp, nil
}

func DoHead(url string) (*http.Response, error) {
	headers := map[string][]string{
		"User-Agent": {UserAgent},
	}
	resp, err := makeRequest("HEAD", url, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, errors.New(fmt.Sprintf("Error: %d\n%s", resp.StatusCode, b))
	}
	return resp, nil
}

func IsEmptyDir(path string) (bool, error) {
	dir, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	count := len(dir) == 0
	pterm.Debug.Printfln("Is %s is empty: %t", path, count)
	return count, nil
}

func IsEmptyDirRecursive(path string) (bool, error) {
	dir, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, f := range dir {
		path := filepath.Join(path, f.Name())
		if f.IsDir() {
			empty, err := IsEmptyDirRecursive(path)
			if err != nil {
				return false, err
			}

			if !empty {
				return false, nil
			}
		} else {
			return false, nil
		}
	}
	return true, nil
}

func ReadManifest(installDir string) (structs.Manifest, error) {
	pterm.Debug.Println("Reading manifest from", installDir)
	file, err := os.ReadFile(filepath.Join(installDir, ManifestName))
	if err != nil {
		return structs.Manifest{}, err
	}

	var manifest structs.Manifest
	err = json.Unmarshal(file, &manifest)
	if err != nil {
		return structs.Manifest{}, err
	}
	return manifest, nil
}

// WriteManifest handy function to write the version manifest
func WriteManifest(installDir string, manifest structs.Manifest) error {
	manifestJson, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal manifest: %s", err.Error())
	}
	versionFile := filepath.Join(installDir, ManifestName)
	vFile, err := os.Create(versionFile)
	if err != nil {
		return fmt.Errorf("unable to create manifest: %s", err.Error())
	}
	_, err = vFile.Write(manifestJson)
	if err != nil {
		return fmt.Errorf("unable to write manifest: %s", err.Error())
	}
	return nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetJava(version string) (structs.File, error) {
	adoptiumUrl, err := makeAdoptiumUrl(version)
	if err != nil {
		return structs.File{}, err
	}

	get, err := DoGet(adoptiumUrl)
	if err != nil {
		return structs.File{}, err
	}
	defer get.Body.Close()

	var adoptium structs.Adoptium

	err = json.NewDecoder(get.Body).Decode(&adoptium)
	if err != nil {
		return structs.File{}, err
	}

	var fileExt string
	if strings.HasSuffix(adoptium[0].Binaries[0].Package.Name, ".zip") {
		fileExt = ".zip"
	} else if strings.HasSuffix(adoptium[0].Binaries[0].Package.Name, ".tar.gz") {
		fileExt = ".tar.gz"
	} else {
		fileExt = "" // shrug
	}

	return structs.File{
		Name:     "jre" + fileExt,
		Path:     "",
		Url:      adoptium[0].Binaries[0].Package.Link,
		Hash:     adoptium[0].Binaries[0].Package.Checksum,
		HashType: "sha256",
	}, nil
}

func GetJavaPath(installDir string, version string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(installDir, "jre", version, "bin", "java.exe"), nil
	case "darwin":
		return filepath.Join(installDir, "jre", version, "Contents", "Home", "bin", "java"), nil
	case "linux":
		return filepath.Join(installDir, "jre", version, "bin", "java"), nil
	default:
		return "", errors.New("unsupported platform")
	}
}

func makeAdoptiumUrl(version string) (string, error) {
	parsedUrl, err := url.Parse(adoptiumApiUrl + "/v3/assets/version/" + version)
	if err != nil {
		return "", err
	}

	q := parsedUrl.Query()
	q.Add("heap_size", "normal")
	q.Add("image_type", "jre")
	q.Add("page", "0")
	q.Add("page_size", "10")
	q.Add("project", "jdk")
	q.Add("release_type", "ga")
	q.Add("semver", "false")
	q.Add("sort_method", "DEFAULT")
	q.Add("sort_order", "DESC")
	q.Add("vendor", "eclipse")
	if runtime.GOOS == "windows" {
		q.Add("os", "windows")
	}
	if runtime.GOOS == "darwin" {
		q.Add("os", "mac")
	}
	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/etc/alpine-release"); !os.IsNotExist(err) {
			q.Add("os", "alpine-linux")
		} else {
			q.Add("os", "linux")
		}
	}

	arch, err := validJavaArch(version)
	if err != nil {
		return "", err
	}
	q.Add("architecture", arch)

	parsedUrl.RawQuery = q.Encode()

	return parsedUrl.String(), nil
}

func validJavaArch(version string) (string, error) {
	targetVersion, err := semVer.NewVersion(version)
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			limit, err := semVer.NewVersion("11.0.0")
			if err != nil {
				return "", err
			}
			if targetVersion.LessThan(limit) {
				return "x64", nil
			}
			return "aarch64", nil
		}
		if runtime.GOARCH == "amd64" {
			return "x64", nil
		}
		if runtime.GOARCH == "386" {
			return "x86", nil
		}
	case "windows":
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			return "x64", nil
		}
		if runtime.GOARCH == "386" || runtime.GOARCH == "arm" {
			return "x86", nil
		}
	case "linux":
		if runtime.GOARCH == "amd64" {
			return "x64", nil
		}
		if runtime.GOARCH == "386" {
			return "x86", nil
		}
		if runtime.GOARCH == "arm64" {
			return "aarch64", nil
		}
		if runtime.GOARCH == "arm" {
			return "arm", nil
		}
	}
	return "", errors.New("unsupported architecture, please contact FTB support")
}

func FileHash(path string, hash string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	switch hash {
	case "sha1":
		h := sha1.New()
		if _, err = io.Copy(h, f); err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", h.Sum(nil)), nil
	case "sha256":
		h := sha256.New()
		if _, err = io.Copy(h, f); err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", h.Sum(nil)), nil
	default:
		return "", errors.New("unsupported hash type")
	}
}

func CombineZip(inZip string, destZip string) error {
	_ = os.Rename(destZip, destZip+".tmp")
	defer os.Remove(destZip + ".tmp")

	newZipFile, err := os.Create(destZip)
	if err != nil {
		log.Fatal(err)
	}
	defer newZipFile.Close()

	writer := zip.NewWriter(newZipFile)
	defer writer.Close()

	zips := []string{destZip + ".tmp", inZip}

	for _, filename := range zips {
		zipReader, err := zip.OpenReader(filename)
		if err != nil {
			return err
		}

		for _, file := range zipReader.File {
			zipFileReader, err := file.Open()
			if err != nil {
				return err
			}
			defer zipFileReader.Close()

			header, err := zip.FileInfoHeader(file.FileInfo())
			if err != nil {
				return err
			}
			header.Name = file.Name

			zipWriter, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}

			_, err = io.Copy(zipWriter, zipFileReader)
			if err != nil {
				return err
			}
		}
		zipReader.Close()
	}
	return nil
}

func ConfirmYN(text string, value bool, style *pterm.Style) bool {
	if style == nil {
		style = pterm.Info.MessageStyle
	}
	show, err := pterm.DefaultInteractiveConfirm.
		WithDefaultText(text).
		WithDefaultValue(value).
		WithTextStyle(style).
		Show()
	if err != nil {
		pterm.Fatal.Printfln("Interactive confirm error: %s", err.Error())
	}
	return show
}

func CopyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				if err := os.Mkdir(dstPath, d.Type().Perm()); err != nil {
					return err
				}
			}
		} else {
			if err := CopyFile(path, dstPath); err != nil {
				return err
			}
		}
		return nil
	})
}

func CopyFile(src string, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, file); err != nil {
		return err
	}

	return nil
}

// CustomWriter to strip ascii characters
type CustomWriter struct {
	writer io.Writer
}

// NewCustomWriter creates a new CustomWriter.
func NewCustomWriter(writer io.Writer) *CustomWriter {
	return &CustomWriter{writer: writer}
}

// Write implements the io.Writer interface.
func (cw *CustomWriter) Write(p []byte) (n int, err error) {

	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	stripped := re.ReplaceAll(p, []byte{})

	filtered := make([]byte, 0, len(stripped))
	for _, b := range stripped {
		if b == '\n' || (unicode.IsPrint(rune(b)) || b < 0x20 || b > 0x7E) {
			filtered = append(filtered, b)
		}
	}
	return cw.writer.Write(filtered)
}
