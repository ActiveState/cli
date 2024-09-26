package buildscript

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

func (b *BuildScript) Write(dir string) error {
	scriptBytes, err := b.Marshal()
	if err != nil {
		return errs.Wrap(err, "Unable to marshal build script")
	}

	scriptPath := filepath.Join(dir, constants.BuildScriptFileName)
	logging.Debug("Writing build script at %s", scriptPath)
	err = fileutils.WriteFile(scriptPath, scriptBytes)
	if err != nil {
		return errs.Wrap(err, "Unable to write build script")
	}

	return nil
}
