package request

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"gopkg.in/yaml.v2"
)

func Publish(description, path, version, filepath, checksum string) (*PublishRequest, error) {
	f, err := os.Open(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, locale.WrapInputError(err, "err_upload_file_not_found", "Could not find file at {{.V0}}", filepath)
		}
		return nil, errs.Wrap(err, "Could not open file %s", filepath)
	}
	return &PublishRequest{
		Variables: publishVariables{
			Version:      version,
			Description:  description,
			path:         path,
			fileChecksum: checksum,
		},
		file: f,
	}, nil
}

type publishVariables struct {
	Version     string `json:"version"`
	Description string `json:"description"`

	path         string  `json:"path"`
	fileChecksum string  `json:"file_checksum"`
	file         *string `json:"file"`
}

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
			Name:  p.file.Name(),
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
	return yaml.Marshal(p.Variables)
}

func (p *PublishRequest) UnmarshalYaml(b []byte) error {
	return yaml.Unmarshal(b, &p.Variables)
}
