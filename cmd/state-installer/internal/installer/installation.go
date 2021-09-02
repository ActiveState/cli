package installer

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

type Installation struct {
	fromDir   string
	binaryDir string
	appDir    string
}

func New(fromDir, binaryDir, appDir string) *Installation {
	return &Installation{
		fromDir, binaryDir, appDir,
	}
}

func (i *Installation) Install() error {
	if err := i.PrepareBinTargets(); err != nil {
		return errs.Wrap(err, "Could not prepare for installation")
	}
	if err := fileutils.MkdirUnlessExists(i.binaryDir); err != nil {
		return errs.Wrap(err, "Could not create target directory: %s", i.binaryDir)
	}
	if err := fileutils.CopyAndRenameFiles(filepath.Join(i.fromDir, "bin"), i.binaryDir); err != nil {
		return errs.Wrap(err, "Failed to copy installation files to dir %s", i.binaryDir)
	}
	if err := InstallSystemFiles(filepath.Join(i.fromDir, "system"), i.binaryDir, i.appDir); err != nil {
		return errs.Wrap(err, "Installation of system files failed.")
	}

	return nil
}
