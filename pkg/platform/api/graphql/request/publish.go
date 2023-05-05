package request

import (
	"bytes"
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"gopkg.in/yaml.v2"
)

func Publish(name, description, path, version, filepath, checksum string) (*PublishRequest, error) {
	f, err := os.Open(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, locale.WrapInputError(err, "err_upload_file_not_found", "Could not find file at {{.V0}}", filepath)
		}
		return nil, errs.Wrap(err, "Could not open file %s", filepath)
	}
	return &PublishRequest{
		Variables: publishVariables{
			Name:         name,
			Version:      version,
			Description:  description,
			path:         path,
			fileChecksum: checksum,
		},
		file: f,
	}, nil
}

type publishAuthor struct {
	Name     string   `yaml:"name"`
	Email    string   `yaml:"email"`
	Websites []string `yaml:"websites"`
}

type publishDep struct {
	Dependency
	Conditions []Dependency `yaml:"conditions"`
}

type Dependency struct {
	Name                string `yaml:"name"`
	Namespace           string `yaml:"namespace"`
	VersionRequirements string `yaml:"versionRequirements"`
}

type AuthorVariables struct {
	Authors []publishAuthor `yaml:"authors,omitempty"`
}

type DepVariables struct {
	Dependencies []publishDep `yaml:"dependencies,omitempty"`
}

type publishVariables struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`

	path         string  `yaml:"path"`
	fileChecksum string  `yaml:"file_checksum"`
	file         *string `yaml:"file"`

	// Optional
	Authors      []publishAuthor `yaml:"authors,omitempty"`
	Dependencies []publishDep    `yaml:"dependencies,omitempty"`
}

var exampleAuthor = AuthorVariables{[]publishAuthor{{
	Name:     "John Doe",
	Email:    "johndoe@domain.tld",
	Websites: []string{"https://example.com"},
}}}

var exampleDep = DepVariables{[]publishDep{{
	Dependency{
		Name:                "example-linux-specific-ingredient",
		Namespace:           "shared",
		VersionRequirements: ">= 1.0.0",
	},
	[]Dependency{
		{
			Name:                "linux",
			Namespace:           "kernel",
			VersionRequirements: ">= 0",
		},
	},
}}}

type PublishRequest struct {
	file      *os.File
	coreVars  map[string]interface{}
	Variables publishVariables
}

func (p *PublishRequest) Close() error {
	return p.file.Close()
}

func (p *PublishRequest) Files() []gqlclient.File {
	return []gqlclient.File{
		{
			Field: "file",
			Name:  p.Variables.Name,
			R:     p.file,
		},
	}
}

func (p *PublishRequest) Query() string {
	return `
		mutation ($description: String!, $path: String!, $file_checksum: String!, $version: String!, $file: FileUpload) {
			publish(input: {
				path: $path,
				file: $file,
				file_checksum: $file_checksum,
				version: $version,
				description: $description,
			}) {
				... on CreatedIngredientVersionRevision {
					ingredientID
					ingredientVersionID
					revision
				}
			}
		}
`
}

func (p *PublishRequest) Vars() map[string]interface{} {
	// Todo: remove redundancy
	return map[string]interface{}{
		"version":       p.Variables.Version,
		"description":   p.Variables.Description,
		"path":          p.Variables.path,
		"file_checksum": p.Variables.fileChecksum,
		"file":          p.Variables.file,
	}
}

func (p *PublishRequest) MarshalYaml() ([]byte, error) {
	v, err := yaml.Marshal(p.Variables)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal publish request")
	}

	if len(p.Variables.Authors) == 0 {
		exampleAuthorYaml, err := yaml.Marshal(exampleAuthor)
		if err != nil {
			return nil, errs.Wrap(err, "Could not marshal example author")
		}
		exampleAuthorYaml = append([]byte("# "), bytes.ReplaceAll(exampleAuthorYaml, []byte("\n"), []byte("\n# "))...)
		exampleAuthorYaml = append([]byte("\n## Optional -- Example Author:\n"), exampleAuthorYaml...)
		v = append(v, exampleAuthorYaml...)
	}

	if len(p.Variables.Dependencies) == 0 {
		exampleDepYaml, err := yaml.Marshal(exampleDep)
		if err != nil {
			return nil, errs.Wrap(err, "Could not marshal example deps")
		}
		exampleDepYaml = append([]byte("# "), bytes.ReplaceAll(exampleDepYaml, []byte("\n"), []byte("\n# "))...)
		exampleDepYaml = append([]byte("\n## Optional -- Example Dependencies:\n"), exampleDepYaml...)
		v = append(v, exampleDepYaml...)
	}

	return v, nil
}

func (p *PublishRequest) UnmarshalYaml(b []byte) error {
	return yaml.Unmarshal(b, &p.Variables)
}
