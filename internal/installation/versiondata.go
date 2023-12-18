package installation

type VersionData struct {
	Name       string `json:"name"`
	License    string `json:"license"`
	Version    string `json:"version"`
	Branch     string `json:"branch"`
	Revision   string `json:"revision"`
	Date       string `json:"date"`
	BuiltViaCI bool   `json:"builtViaCI"`
}
