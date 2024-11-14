package structs

type Modpack struct {
	Id       int
	Name     string
	Versions []ModpackV
}

type ModpackV struct {
	Id   int
	Type string
}

type ModpackVersion struct {
	Id      int
	Name    string
	Files   []File
	Targets ModpackTargets
	Memory  Memory
}

type File struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Url      string   `json:"url"`
	Mirrors  []string `json:"mirrors"`
	Hash     string   `json:"hash"`
	HashType string   `json:"hash_type"`
}

type ModLoaderTarget struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Memory struct {
	Minimum     int
	Recommended int
}

type ModpackTargets struct { // I want to rename this
	ModLoader   ModLoaderTarget `json:"modLoader"`
	JavaVersion string          `json:"javaVersion"`
	McVersion   string          `json:"mcVersion"`
}
