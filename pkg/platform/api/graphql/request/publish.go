package request

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"gopkg.in/yaml.v2"
)

func Publish(path string, version, filepath, checksum string) (*PublishRequest, error) {
	f, err := os.Open(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, locale.WrapInputError(err, "err_upload_file_not_found", "Could not find file at {{.V0}}", filepath)
		}
		return nil, errs.Wrap(err, "Could not open file %s", filepath)
	}
	return &PublishRequest{
		vars: map[string]interface{}{
			"input": map[string]interface{}{
				"path":     path,
				"version":  version,
				"checksum": checksum,
			},
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
       mutation ($input: PublishInput!) {
            publish(input: $input) {
                ingredientID
				ingredientVersionID
				revision
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
