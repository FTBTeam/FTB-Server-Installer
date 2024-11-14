package structs

type FTBModpack struct {
	Versions     []Versions `json:"versions"`
	Notification string     `json:"notification"`
	Status       string     `json:"status"`
	Message      string     `json:"message,omitempty"`
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
}
type FTBTargets struct {
	Version string `json:"version"`
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Updated int    `json:"updated"`
}
type Versions struct {
	Targets []FTBTargets `json:"targets"`
	ID      int          `json:"id"`
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	Updated int          `json:"updated"`
	Private bool         `json:"private"`
}

///////////////////////////////////////////

type FTBVersion struct {
	Files        []FTBFiles   `json:"files"`
	Targets      []FTBTargets `json:"targets"`
	Specs        FTBSpecs     `json:"specs"`
	Parent       int          `json:"parent"`
	Notification string       `json:"notification"`
	Status       string       `json:"status"`
	Message      string       `json:"message,omitempty"`
	ID           int          `json:"id"`
	Name         string       `json:"name"`
	Type         string       `json:"type"`
}

type FTBSpecs struct {
	ID          int `json:"id"`
	Minimum     int `json:"minimum"`
	Recommended int `json:"recommended"`
}

type FTBFiles struct {
	Version    string   `json:"version"`
	Path       string   `json:"path"`
	URL        string   `json:"url"`
	Sha1       string   `json:"sha1"`
	Size       int      `json:"size"`
	ClientOnly bool     `json:"clientonly"`
	ServerOnly bool     `json:"serveronly"`
	Optional   bool     `json:"optional"`
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Mirrors    []string `json:"mirrors"`
}
