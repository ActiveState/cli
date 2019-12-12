package integration

import (
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func setupASY(dir, contents string) error {
	errScope := "cannot setup activestate.yaml file"

	contents = strings.TrimSpace(contents)
	projectFile := &projectfile.Project{}

	if err := yaml.Unmarshal([]byte(contents), projectFile); err != nil {
		return errors.Wrap(err, errScope)
	}

	projectFile.SetPath(filepath.Join(dir, "activestate.yaml"))
	if fail := projectFile.Save(); fail != nil {
		return errors.Wrap(fail, errScope)
	}

	return nil
}
