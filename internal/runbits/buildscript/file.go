package buildscript_runbit

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/buildscript"
)

var ErrBuildscriptNotExist = errors.New("Build script does not exist")

func ScriptFromProject(projectDir string) (*buildscript.BuildScript, error) {
	path := filepath.Join(projectDir, constants.BuildScriptFileName)

	data, err := fileutils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errs.Pack(err, ErrBuildscriptNotExist)
		}
		return nil, errs.Wrap(err, "Could not read build script from file")
	}

	script, err := buildscript.Unmarshal(data)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build script")
	}

	return script, nil
}
