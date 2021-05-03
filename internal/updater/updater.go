package updater

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
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

func (u *AvailableUpdate) prepare() (string, string, error) {
	tmpDir, err := ioutil.TempDir("", "state-update")
	if err != nil {
		return "", "", errs.Wrap(err, "Could not create temp dir")
	}

	if err := NewFetcher().Fetch(u, tmpDir); err != nil {
		return "", "", errs.Wrap(err, "Could not download and unpack update")
	}

	installerPath := filepath.Join(tmpDir, constants.ToplevelInstallArchiveDir, InstallerName)
	if !fileutils.FileExists(installerPath) {
		return "", "", errs.Wrap(err, "Downloaded update does not have installer")
	}
	installTargetPath := filepath.Dir(os.Args[0])

	return installerPath, installTargetPath, nil
}

// InstallDeferred will fetch the update and run its installer in a deferred process
func (u *AvailableUpdate) InstallDeferred() (int, error) {
	installerPath, installTargetPath, err := u.prepare()
	if err != nil {
		return 0, err
	}
	proc, err := exeutils.ExecuteAndForget(installerPath, installTargetPath)
	if err != nil {
		return 0, errs.Wrap(err, "Could not start installer")
	}

	if proc == nil {
		return 0, errs.Wrap(err, "Could not obtain process information for installer")
	}

	return proc.Pid, nil
}

func (u *AvailableUpdate) InstallBlocking() error {
	installerPath, installTargetPath, err := u.prepare()
	if err != nil {
		return err
	}

	_, _, err = exeutils.ExecuteAndPipeStd(installerPath, []string{installTargetPath}, []string{})
	if err != nil {
		return errs.Wrap(err, "Could not run installer")
	}

	return nil
}
