package installation

type VersionData struct {
	Name       string `json:"name"`
	License    string `json:"license"`
	Version    string `json:"version"`
	Channel    string `json:"channel"`
	Revision   string `json:"revision"`
	Date       string `json:"date"`
	BuiltViaCI bool   `json:"builtViaCI"`
}
