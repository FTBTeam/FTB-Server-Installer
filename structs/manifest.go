package structs

type Manifest struct {
	Id             int            `json:"id"`
	Name           string         `json:"name"`
	VersionName    string         `json:"versionName"`
	VersionId      int            `json:"versionId"`
	ModpackTargets ModpackTargets `json:"modPackTargets"`
	Files          []File         `json:"files,omitempty"`
}
