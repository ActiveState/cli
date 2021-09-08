package installer

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/gobuffalo/packr"
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
	if err := i.ensureOriginalPath(); err != nil {
		return errs.Wrap(err, "Could not ensure original State Tool installation will remain valid")
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

func (i *Installation) ensureOriginalPath() error {
	if condition.InUnitTest() {
		return nil
	}

	stateExe := filepath.Base(appinfo.StateApp().Exec())
	statePath, err := exec.LookPath(stateExe)
	if err != nil {
		logging.Debug("State tool not already installed, error: %v", err)
		return nil
	}

	if i.binaryDir == filepath.Dir(statePath) {
		logging.Debug("State tool installation paths do not differ")
		return nil
	}

	tplParams := map[string]interface{}{
		"path": filepath.Join(i.binaryDir, stateExe),
	}
	box := packr.NewBox("../../../../assets/state")
	boxFile := "state.sh"
	if runtime.GOOS == "windows" {
		boxFile = "state.bat"
	}
	fileBytes := box.Bytes(boxFile)
	fileStr, err := strutils.ParseTemplate(string(fileBytes), tplParams)
	if err != nil {
		return errs.Wrap(err, "Could not parse %s template", boxFile)
	}

	if err = ioutil.WriteFile(statePath, []byte(fileStr), 0755); err != nil {
		return locale.WrapError(err, "Could not create State Tool script at {{.V0}}.", statePath)
	}

	return nil
}
