package structures

type Project struct {
	Name        string        `json:"name"`
	Owner       string        `json:"owner"`
	Runtimes    []Runtime     `json:"runtimes"`
	Environment []Environment `json:"environment"`
}

type Runtime struct {
	Name     string    `json:"name"`
	Version  string    `json:"version"`
	Packages []Package `json:"packages"`
}

type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Environment struct {
	Platform  string     `json:"platform"`
	Variables []Variable `json:"variables"`
	Hooks     []Hook     `json:"hooks"`
	Commands  []Command  `json:"commands"`
}

type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Hook struct {
	Hook    string `json:"hook"`
	Command string `json:"command"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}
