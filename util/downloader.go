package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/pterm/pterm"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Download struct {
	destPath           string
	reqURL             string
	hash               hash.Hash
	checksum           []byte
	deleteOnError      bool
	checkContentLength bool
	Progress           float64
	CancelFunc         context.CancelFunc
}

func NewDownload(destPath string, reqUrl string) (*Download, error) {
	if reqUrl == "" {
		return nil, fmt.Errorf("required URL is empty")
	}
	return &Download{
		reqURL:             reqUrl,
		destPath:           destPath,
		checkContentLength: false,
	}, nil
}

// Do performs the file download with the configured parameters.
// It handles directory creation, checksum verification, and cleanup on error if configured.
// Returns an error if the download or verification fails.
func (dl *Download) Do() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	dl.CancelFunc = cancel
	defer dl.Cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", dl.reqURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", UserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//if resp.Header.Get("Cf-Cache-Status") != "HIT" && resp.Header.Get("Cf-Cache-Status") != "" {
	//	pterm.Debug.Printfln("Cf-Cache-Status for %s: %s", dl.reqURL, resp.Header.Get("Cf-Cache-Status"))
	//}
	if resp.StatusCode != http.StatusOK {
		pterm.Debug.Printfln("Headers: %+v", resp.Header)
		return fmt.Errorf("failed to download file from %s: bad status %s", dl.reqURL, resp.Status)
	}
	if dl.checkContentLength && resp.ContentLength < 1 {
		pterm.Debug.Printfln("Headers: %+v", resp.Header)
		return fmt.Errorf("invalid content length: %d", resp.ContentLength)
	}

	b := resp.Body
	err = dl.write(b)
	if err != nil {
		return err
	}

	return nil
}

func (dl *Download) write(b io.ReadCloser) error {
	// Check if the destination directory exists
	destDir := filepath.Dir(dl.destPath)
	if _, err := os.Stat(destDir); errors.Is(err, os.ErrNotExist) {
		// Create the destination directory if it doesn't exist
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %s", err.Error())
		}
	}

	f, err := os.OpenFile(dl.destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var writer io.Writer = f

	if dl.hash != nil {
		writer = io.MultiWriter(f, dl.hash)
	}

	if _, err = io.Copy(writer, b); err != nil {
		return fmt.Errorf("failed to write file: %s", err.Error())
	}

	if dl.hash != nil && dl.checksum != nil {
		sum := dl.hash.Sum(nil)
		if !bytes.Equal(dl.checksum, sum) {
			if dl.deleteOnError {
				if err := os.Remove(dl.destPath); err != nil {
					return fmt.Errorf("checksum mismatch, failed to remove file: %s", err.Error())
				}
			}
			return fmt.Errorf("checksum mismatch")
		}
	}
	return nil
}

func (dl *Download) CheckContentLength(check bool) {
	dl.checkContentLength = check
}

func (dl *Download) SetChecksum(hash hash.Hash, sum []byte, deleteOnError bool) {
	dl.hash = hash
	dl.checksum = sum
	dl.deleteOnError = deleteOnError
}

func (dl *Download) Cancel() {
	if dl.CancelFunc != nil {
		dl.CancelFunc()
	}
}
