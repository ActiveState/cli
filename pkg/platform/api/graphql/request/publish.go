package request

import (
	"bytes"
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"gopkg.in/yaml.v3"
)

func Publish(vars PublishVariables, filepath string) (*PublishRequest, error) {
	f, err := os.Open(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, locale.WrapInputError(err, "err_upload_file_not_found", "Could not find file at {{.V0}}", filepath)
		}
		return nil, errs.Wrap(err, "Could not open file %s", filepath)
	}

	checksum, err := fileutils.Sha256Hash(filepath)
	if err != nil {
		return nil, locale.WrapError(err, "err_upload_file_checksum", "Could not calculate checksum for file")
	}

	return &PublishRequest{
		Variables:    vars,
		fileChecksum: checksum,
		file:         f,
	}, nil
}

type PublishVariableAuthor struct {
	Name     string   `yaml:"name,omitempty"`
	Email    string   `yaml:"email,omitempty"`
	Websites []string `yaml:"websites,omitempty"`
}

type PublishVariableDep struct {
	Dependency
	Conditions []Dependency `yaml:"conditions,omitempty"`
}

type Dependency struct {
	Name                string `yaml:"name"`
	Namespace           string `yaml:"namespace"`
	VersionRequirements string `yaml:"versionRequirements,omitempty"`
}

type AuthorVariables struct {
	Authors []PublishVariableAuthor `yaml:"authors,omitempty"`
}

type DepVariables struct {
	Dependencies []PublishVariableDep `yaml:"dependencies,omitempty"`
}

type PublishVariables struct {
	Name        string `yaml:"name"`
	Namespace   string `yaml:"namespace"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`

	// Optional
	Authors      []PublishVariableAuthor `yaml:"authors,omitempty"`
	Dependencies []PublishVariableDep    `yaml:"dependencies,omitempty"`
}

func (p PublishVariables) MarshalYaml(includeExample bool) ([]byte, error) {
	v, err := yaml.Marshal(p)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal publish request")
	}

	if includeExample {
		if len(p.Authors) == 0 {
			exampleAuthorYaml, err := yaml.Marshal(exampleAuthor)
			if err != nil {
				return nil, errs.Wrap(err, "Could not marshal example author")
			}
			exampleAuthorYaml = append([]byte("# "), bytes.ReplaceAll(exampleAuthorYaml, []byte("\n"), []byte("\n# "))...)
			exampleAuthorYaml = append([]byte("\n## Optional -- Example Author:\n"), exampleAuthorYaml...)
			v = append(v, exampleAuthorYaml...)
		}

		if len(p.Dependencies) == 0 {
			exampleDepYaml, err := yaml.Marshal(exampleDep)
			if err != nil {
				return nil, errs.Wrap(err, "Could not marshal example deps")
			}
			exampleDepYaml = append([]byte("# "), bytes.ReplaceAll(exampleDepYaml, []byte("\n"), []byte("\n# "))...)
			exampleDepYaml = append([]byte("\n## Optional -- Example Dependencies:\n"), exampleDepYaml...)
			v = append(v, exampleDepYaml...)
		}
	}

	return v, nil
}

func (p PublishVariables) UnmarshalYaml(b []byte) error {
	return yaml.Unmarshal(b, &p)
}

var exampleAuthor = AuthorVariables{[]PublishVariableAuthor{{
	Name:     "John Doe",
	Email:    "johndoe@domain.tld",
	Websites: []string{"https://example.com"},
}}}

var exampleDep = DepVariables{[]PublishVariableDep{{
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
	fileChecksum string `yaml:"file_checksum"`
	file         *os.File
	Variables    PublishVariables
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
		"path":          p.Variables.Namespace + "/" + p.Variables.Name,
		"file_checksum": p.fileChecksum,
		"file":          nil, // This feels counter-intuitive, but it's what the API expects..
	}
}
