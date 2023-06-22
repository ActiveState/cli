package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"gopkg.in/yaml.v3"
)

func Publish(vars PublishVariables, filepath string) (*PublishInput, error) {
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

	vars.FileChecksum = checksum

	return &PublishInput{
		Variables: vars,
		file:      f,
	}, nil
}

// PublishVariables holds the input variables
// It is ultimately used as the input for the graphql query, but before that we may want to present the data to the user
// which is done with yaml. As such the yaml tags are used for representing data to the user, and the json is used for
// inputs to graphql.
type PublishVariables struct {
	Name        string `yaml:"name" json:"-"`      // User representation only
	Namespace   string `yaml:"namespace" json:"-"` // User representation only
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description" json:"description"`

	// Optional
	Authors      []PublishVariableAuthor `yaml:"authors,omitempty" json:"authors,omitempty"`
	Dependencies []PublishVariableDep    `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`

	// GraphQL input only
	Path         string  `yaml:"-" json:"path"`
	File         *string `yaml:"-" json:"file"` // Intentionally a pointer that never gets set as the server expects this to always be nil
	FileChecksum string  `yaml:"-" json:"file_checksum"`
}

type PublishVariableAuthor struct {
	Name     string   `yaml:"name,omitempty" json:"name,omitempty"`
	Email    string   `yaml:"email,omitempty" json:"email,omitempty"`
	Websites []string `yaml:"websites,omitempty" json:"websites,omitempty"`
}

type PublishVariableDep struct {
	Dependency
	Conditions []Dependency `yaml:"conditions,omitempty" json:"conditions,omitempty"`
}

type Dependency struct {
	Name                string `yaml:"name" json:"name"`
	Namespace           string `yaml:"namespace" json:"namespace"`
	VersionRequirements string `yaml:"versionRequirements,omitempty" json:"versionRequirements,omitempty"`
}

// ExampleAuthorVariables is used for presenting sample data to the user, it's not used for graphql input
type ExampleAuthorVariables struct {
	Authors []PublishVariableAuthor `yaml:"authors,omitempty"`
}

// ExampleDepVariables is used for presenting sample data to the user, it's not used for graphql input
type ExampleDepVariables struct {
	Dependencies []PublishVariableDep `yaml:"dependencies,omitempty"`
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

func (p *PublishVariables) UnmarshalYaml(b []byte) error {
	err := yaml.Unmarshal(b, p)
	return err
}

var exampleAuthor = ExampleAuthorVariables{[]PublishVariableAuthor{{
	Name:     "John Doe",
	Email:    "johndoe@domain.tld",
	Websites: []string{"https://example.com"},
}}}

var exampleDep = ExampleDepVariables{[]PublishVariableDep{{
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

type PublishInput struct {
	file      *os.File
	Variables PublishVariables
}

func (p *PublishInput) Close() error {
	return p.file.Close()
}

func (p *PublishInput) Files() []gqlclient.File {
	return []gqlclient.File{
		{
			Field: "variables.input.file", // this needs to map to the graphql input, eg. variables.input.file
			Name:  p.Variables.Name,
			R:     p.file,
		},
	}
}

func (p *PublishInput) Query() string {
	return `
		mutation ($input: PublishInput!) {
			publish(input: $input) {
				... on CreatedIngredientVersionRevision {
					ingredientID
					ingredientVersionID
					revision
				}
			}
		}
`
}

func (p *PublishInput) Vars() (map[string]interface{}, error) {
	// Path is only used when sending data to graphql, so rather than updating it multiple times as source vars
	// are changed we just set it here once prior to its use.
	p.Variables.Path = p.Variables.Namespace + "/" + p.Variables.Name

	// Convert our json data to a map
	vars, err := json.Marshal(p.Variables)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal publish input vars")
	}
	varMap := make(map[string]interface{})
	if err := json.Unmarshal(vars, &varMap); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal publish input vars")
	}

	return map[string]interface{}{
		"input": varMap,
	}, nil
}
