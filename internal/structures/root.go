package structures

type Project struct {
	Name      string     `json:"name"`
	Languages []Language `json:"languages"`
}

type Language struct {
	Name     string    `json:"name"`
	Packages []Package `json:"packages"`
}

type Package struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}
