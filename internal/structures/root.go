package structures

type Project struct {
	Name     string    `json:"name"`
	Owner    string    `json:"owner"`
	Runtimes []Runtime `json:"runtimes"`
}

type Runtime struct {
	Name     string    `json:"name"`
	Version  int       `json:"version"`
	Packages []Package `json:"packages"`
}

type Package struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}
