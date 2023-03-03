package request

import (
	"errors"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
)

func Publish(path string, version, filepath, checksum string) (*publish, error) {
	f, err := os.Open(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, locale.WrapInputError(err, "err_upload_file_not_found", "Could not find file at {{.V0}}", filepath)
		}
		return nil, errs.Wrap(err, "Could not open file %s", filepath)
	}
	return &publish{
		vars: map[string]interface{}{
			"path":     path,
			"version":  version,
			"checksum": checksum,
		},
		file: f,
	}, nil
}

type publish struct {
	file *os.File
	vars map[string]interface{}
}

func (p *publish) Close() error {
	return p.file.Close()
}

func (p *publish) Files() []gqlclient.File {
	return []gqlclient.File{
		{
			Field: "file",
			Name:  p.file.Name(),
			R:     p.file,
		},
	}
}

func (p *publish) Query() string {
	return `
	mutate ($path: string!, $version: string!, $checksum: string!) {
		publish(path: $path, version: $version, checksum: $checksum) {
			fileChecksum
		}
	}
`
}

func (p *publish) Vars() map[string]interface{} {
	return p.vars
}
