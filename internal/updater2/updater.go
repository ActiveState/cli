package updater2

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
)

type AvailableUpdate struct {
	version string
	channel string
	path    string
	url     string
	sha256  string
}

const InstallerName = "installer" + osutils.ExeExt

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

	installTargetPath := filepath.Dir(os.Args[0])
	err := exeutils.ExecuteAndForget(filepath.Join(tmpDir, InstallerName), installTargetPath)
	if err != nil {
		return errs.Wrap(err, "Could not start installer")
	}

	return nil
}
