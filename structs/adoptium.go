package structs

import "time"

type Adoptium []struct {
	Binaries    []AdoptiumBinaries `json:"binaries"`
	ReleaseName string             `json:"release_name"`
}
type AdoptiumPackage struct {
	Checksum      string `json:"checksum"`
	ChecksumLink  string `json:"checksum_link"`
	DownloadCount int    `json:"download_count"`
	Link          string `json:"link"`
	MetadataLink  string `json:"metadata_link"`
	Name          string `json:"name"`
	SignatureLink string `json:"signature_link"`
	Size          int    `json:"size"`
}
type AdoptiumBinaries struct {
	Architecture  string          `json:"architecture"`
	DownloadCount int             `json:"download_count"`
	HeapSize      string          `json:"heap_size"`
	ImageType     string          `json:"image_type"`
	JvmImpl       string          `json:"jvm_impl"`
	Os            string          `json:"os"`
	Package       AdoptiumPackage `json:"package"`
	Project       string          `json:"project"`
	ScmRef        string          `json:"scm_ref"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Installer     Installer       `json:"installer,omitempty"`
}
