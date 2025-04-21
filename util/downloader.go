package util

import (
	"bytes"
	"context"
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
	CancelFunc         *context.CancelFunc
}

func NewDownload(destPath string, reqUrl string) *Download {
	return &Download{
		reqURL:             reqUrl,
		destPath:           destPath,
		checkContentLength: false,
	}
}

func (dl *Download) Do() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	dl.CancelFunc = &cancel
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}
	if dl.checkContentLength && resp.ContentLength < 1 {
		return fmt.Errorf("invalid content length: %d", resp.ContentLength)
	}

	b := resp.Body
	err = dl.write(b)
	if err != nil {
		return err
	}

	if dl.hash != nil && dl.checksum != nil {
		f, err := os.OpenFile(dl.destPath, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(dl.hash, f); err != nil {
			return err
		}
		sum := dl.hash.Sum(nil)
		//fmt.Println("Checksum: ", fmt.Sprintf("%x", dl.checksum))
		//fmt.Println("CalSum: ", fmt.Sprintf("%x", sum))
		if !bytes.Equal(dl.checksum, sum) {
			if dl.deleteOnError {
				if err := os.Remove(dl.destPath); err != nil {
					return fmt.Errorf("checksum mismatch, failed to remove file: %s", err.Error())
				}
				return fmt.Errorf("checksum mismatch, file deleted")
			}
			return fmt.Errorf("checksum mismatch")
		}
	}

	return nil
}

func (dl *Download) write(b io.Reader) error {
	// Check if the destination directory exists
	destDir := filepath.Dir(dl.destPath)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		// Create the destination directory if it doesn't exist
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %s", err.Error())
		}
	}

	f, err := os.OpenFile(dl.destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			pterm.Error.Printfln("Error closing file: %s", err.Error())
		}
	}()
	_, err = io.Copy(f, b)
	if err != nil {
		return fmt.Errorf("failed to write file: %s", err.Error())
	}
	//fmt.Println(size)
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
	if dl.Cancel != nil {
		(*dl.CancelFunc)()
	}
}
