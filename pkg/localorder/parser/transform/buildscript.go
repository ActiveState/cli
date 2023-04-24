package transform

import "encoding/json"

type BuildScript struct {
	Let map[string]Binding `json:"let"`
	In  InStatement        `json:"in"`
}

type Binding interface {
	MarshalJSON() ([]byte, error)
}

type SolveBinding struct {
	Platforms    []string      `json:"platforms"`
	Requirements []Requirement `json:"requirements"`
}

func (s SolveBinding) MarshalJSON() ([]byte, error) {
	type Alias SolveBinding         // Create an alias for SolveBinding
	return json.Marshal((Alias)(s)) // Use the alias to perform marshalling
}

type Requirement struct {
	Name    string            `json:"name"`
	Version map[string]string `json:"version"`
}

type ListBinding []string

func (l ListBinding) MarshalJSON() ([]byte, error) {
	type Alias ListBinding
	return json.Marshal((Alias)(l))
}

type InStatement interface {
	MarshalJSON() ([]byte, error)
}

type InIdentifier string

func (i InIdentifier) MarshalJSON() ([]byte, error) {
	type Alias InIdentifier
	return json.Marshal((Alias)(i))
}

type MergeApplication map[string]string

// Example:
// {
// 	"merge": {
// 		"runtime": "win_runtime",
// 		"installer": "lin_runtime"
// 	}

func (m MergeApplication) MarshalJSON() ([]byte, error) {
	type Alias MergeApplication
	return json.Marshal((Alias)(m))
}

type MergeArgument interface {
	MarshalJSON() ([]byte, error)
}

type WinInstallerApplication struct {
	Runtime string `json:"runtime"`
}

func (w WinInstallerApplication) MarshalJSON() ([]byte, error) {
	type Alias WinInstallerApplication
	return json.Marshal((Alias)(w))
}

type TarInstallerApplication struct {
	Runtime string `json:"runtime"`
}

func (t TarInstallerApplication) MarshalJSON() ([]byte, error) {
	type Alias TarInstallerApplication
	return json.Marshal((Alias)(t))
}
