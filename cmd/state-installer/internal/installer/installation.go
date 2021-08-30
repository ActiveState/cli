package installer

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
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

// PrepareBinTargets will move aside any targets in the bin dir that we would otherwise overwrite.
// This guards us from file in use errors as well as false positives by security software
func (i *Installation) PrepareBinTargets() error {
	files, err := ioutil.ReadDir(filepath.Join(i.fromDir, "bin"))
	if err != nil {
		return errs.Wrap(err, "Could not read target dir")
	}

	temp, err := ioutil.TempDir("", "update-from-state-"+constants.Version)
	if err != nil {
		return errs.Wrap(err, "Could not access temp dir")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		targetFile := filepath.Join(i.binaryDir, file.Name())
		if fileutils.TargetExists(targetFile) {
			if err := os.Rename(targetFile, filepath.Join(temp, file.Name())); err != nil {
				return errs.Wrap(err, "Could not move executable aside prior to install: %s", targetFile)
			}
		}
	}

	return nil
}