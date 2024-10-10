package types

type Requirement struct {
	Name               string               `json:"name"`
	Namespace          string               `json:"namespace"`
	VersionRequirement []VersionRequirement `json:"-"`
	Revision           *int                 `json:"revision,omitempty"`
}

type VersionRequirement map[string]string
