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
		vars: map[string]interface{}{
			"path":          path, // namespace
			"version":       version,
			"file_checksum": checksum,
			"file":          nil,
			"description":   description,
		},
		file: f,
	}, nil
}

type PublishRequest struct {
	file *os.File
	vars map[string]interface{}
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
	return p.vars
}

func (p *PublishRequest) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(p.vars)
}

func (p *PublishRequest) UnmarshalYaml(b []byte) error {
	return yaml.Unmarshal(b, &p.vars)
}
