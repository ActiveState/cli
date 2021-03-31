package updater2

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

type AvailableUpdate struct {
	version string
	channel string
	path    string
	url     string
	sha256  string
}

// InstallDeferred will fetch the update and run its installer in a deferred process
// This deferred process
func (u *AvailableUpdate) InstallDeferred() error {
	tmpDir, err := ioutil.TempDir("", "state-update")
	if err != nil {
		return errs.Wrap(err, "Could not create temp dir")
	}

	if err := NewFetcher().Fetch(u, tmpDir); err != nil {
		return errs.Wrap(err, "Could not download and unpack update")
	}

	if !fileutils.FileExists(filepath.Join(tmpDir, InstallerName)) {
		return errs.Wrap(err, "Downloaded update does not have installer")
	}

	cmd := exec.Command(filepath.Join(tmpDir, InstallerName), filepath.Dir(os.Args[0]))
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	if err := cmd.Start(); err != nil {
		return errs.Wrap(err, "Could not start installer")
	}

	return nil
}
