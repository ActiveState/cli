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
	Version  string `json:"version"`
	Channel  string `json:"channel"`
	Platform string `json:"platform"`
	Path     string `json:"path"`
	Sha256   string `json:"sha256"`
	url      string
}

func NewAvailableUpdate(version, channel, platform, path, sha256 string) *AvailableUpdate {
	return &AvailableUpdate{
		Version:  version,
		Channel:  channel,
		Platform: platform,
		Path:     path,
		Sha256:   sha256,
	}
}

const InstallerName = "state-installer" + osutils.ExeExt

// InstallDeferred will fetch the update and run its installer in a deferred process
func (u *AvailableUpdate) InstallDeferred(configPath string) error {
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

	installLogFile := filepath.Join(configPath, "installer.log")

	installTargetPath := filepath.Dir(os.Args[0])
	err = exeutils.ExecuteAndForget(filepath.Join(tmpDir, InstallerName),
		installLogFile, installTargetPath)
	if err != nil {
		return errs.Wrap(err, "Could not start installer")
	}

	return nil
}
