package updater

import (
	"bufio"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
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

	return installerPath, nil
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


// Install will fetch the update and run its installer
func (u *AvailableUpdate) Install() (*os.Process, chan string, error) {
	installerPath, err := u.download()
	if err != nil {
		return nil, nil, errs.Wrap(err, "Could not download update")
	}

	installTargetPath, err := installation.InstallPath()
	if err != nil {
		return nil, nil, errs.Wrap(err, "Could not detect install path")
	}

	outputChannel := make(chan string, 999)
	proc, err := exeutils.ExecuteAndForget(installerPath, []string{installTargetPath}, func(cmd *exec.Cmd) error {
		var stdout io.ReadCloser
		var stderr io.ReadCloser
		if stderr, err = cmd.StderrPipe(); err != nil {
			return errs.Wrap(err, "Could not obtain stderr pipe")
		}
		if stdout, err = cmd.StdoutPipe(); err != nil {
			return errs.Wrap(err, "Could not obtain stderr pipe")
		}
		scanner := bufio.NewScanner(io.MultiReader(stderr, stdout))
		for scanner.Scan() {
			outputChannel <- scanner.Text()
		}
		close(outputChannel)
		return nil
	})
	if err != nil {
		return nil, nil, errs.Wrap(err, "Could not start installer")
	}

	if proc == nil {
		return nil, nil, errs.Wrap(err, "Could not obtain process information for installer")
	}

	return proc, outputChannel, nil
}