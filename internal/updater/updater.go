package updater

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
)

type AvailableUpdate struct {
	Version  string  `json:"version"`
	Channel  string  `json:"channel"`
	Platform string  `json:"platform"`
	Path     string  `json:"path"`
	Sha256   string  `json:"sha256"`
	Tag      *string `json:"tag,omitempty"`
	url      string
}

func NewAvailableUpdate(version, channel, platform, path, sha256, tag string) *AvailableUpdate {
	var t *string
	if tag != "" {
		t = &tag
	}
	return &AvailableUpdate{
		Version:  version,
		Channel:  channel,
		Platform: platform,
		Path:     path,
		Sha256:   sha256,
		Tag:      t,
	}
}

const InstallerName = "state-installer" + osutils.ExeExt

func (u *AvailableUpdate) prepare() (string, error) {
	tmpDir, err := ioutil.TempDir("", "state-update")
	if err != nil {
		return "", errs.Wrap(err, "Could not create temp dir")
	}

	if err := NewFetcher().Fetch(u, tmpDir); err != nil {
		return "", errs.Wrap(err, "Could not download and unpack update")
	}

	installerPath := filepath.Join(tmpDir, constants.ToplevelInstallArchiveDir, InstallerName)
	if !fileutils.FileExists(installerPath) {
		return "", errs.Wrap(err, "Downloaded update does not have installer")
	}

	return installerPath, nil
}

// InstallDeferred will fetch the update and run its installer in a deferred process
func (u *AvailableUpdate) InstallDeferred(installTargetPath string) (*os.Process, error) {
	installerPath, err := u.prepare()
	if err != nil {
		return nil, err
	}

	var args []string
	if installTargetPath != "" {
		args = append(args, installTargetPath)
	}
	proc, err := exeutils.ExecuteAndForget(installerPath, args, func(cmd *exec.Cmd) error {
		if u.Tag != nil {
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", constants.UpdateTagEnvVarName, *u.Tag))
		}
		return nil
	})
	if err != nil {
		return nil, errs.Wrap(err, "Could not start installer")
	}

	return proc, nil
}

func (u *AvailableUpdate) InstallBlocking(installTargetPath string) error {
	installerPath, err := u.prepare()
	if err != nil {
		return err
	}

	if err := prepareBinTargets(installTargetPath); err != nil {
		return errs.Wrap(err, "Could not prepare bin dir")
	}

	var args []string
	if installTargetPath != "" {
		args = append(args, installTargetPath)
	}
	var envs []string
	if u.Tag != nil {
		envs = append(envs, fmt.Sprintf("%s=%s", constants.UpdateTagEnvVarName, *u.Tag))
	}
	_, _, err = exeutils.ExecuteAndPipeStd(installerPath, args, envs)
	if err != nil {
		return errs.Wrap(err, "Could not run installer")
	}

	return nil
}

// InstallWithProgress will fetch the update and run its installer
func (u *AvailableUpdate) InstallWithProgress(installTargetPath string, progressCb func(string, bool)) (*os.Process, error) {
	installerPath, err := u.prepare()
	if err != nil {
		return nil, errs.Wrap(err, "Could not download update")
	}

	proc, err := exeutils.ExecuteAndForget(installerPath, []string{installTargetPath}, func(cmd *exec.Cmd) error {
		var stdout io.ReadCloser
		var stderr io.ReadCloser
		if stderr, err = cmd.StderrPipe(); err != nil {
			return errs.Wrap(err, "Could not obtain stderr pipe")
		}
		if stdout, err = cmd.StdoutPipe(); err != nil {
			return errs.Wrap(err, "Could not obtain stderr pipe")
		}
		if u.Tag != nil {
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", constants.UpdateTagEnvVarName, *u.Tag))
		}
		go func() {
			scanner := bufio.NewScanner(io.MultiReader(stderr, stdout))
			for scanner.Scan() {
				progressCb(scanner.Text(), false)
			}
			progressCb(scanner.Text(), true)
		}()
		return nil
	})
	if err != nil {
		return nil, errs.Wrap(err, "Could not start installer")
	}

	if proc == nil {
		return nil, errs.Wrap(err, "Could not obtain process information for installer")
	}

	return proc, nil
}

// prepareBinTargets moves state executables to a temp dir prior to running the installer to avoid conflicts and
// security software false-positives
func prepareBinTargets(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errs.Wrap(err, "Could not read target dir")
	}

	temp, err := ioutil.TempDir("", "update-state")
	if err != nil {
		return errs.Wrap(err, "Could not access temp dir")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		targetFile := filepath.Join(dir, file.Name())
		if err := os.Rename(targetFile, filepath.Join(temp, file.Name())); err != nil {
			return errs.Wrap(err, "Could not move executable aside prior to install: %s", targetFile)
		}
	}

	return nil
}