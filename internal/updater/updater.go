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
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/gofrs/flock"
)

type ErrorInProgress struct{ *locale.LocalizedError }

const CfgKeyInstallVersion = "state_tool_installer_version"

type AvailableUpdate struct {
	Version  string  `json:"version"`
	Channel  string  `json:"channel"`
	Platform string  `json:"platform"`
	Path     string  `json:"path"`
	Sha256   string  `json:"sha256"`
	Tag      *string `json:"tag,omitempty"`
	url      string
	tmpDir   string
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

func (u *AvailableUpdate) DownloadAndUnpack() (string, error) {
	if u.tmpDir != "" {
		// To facilitate callers explicitly calling this method we cache the tmp dir and just return it if it's set
		return u.tmpDir, nil
	}

	tmpDir, err := ioutil.TempDir("", "state-update")
	if err != nil {
		return "", errs.Wrap(err, "Could not create temp dir")
	}

	if err := NewFetcher().Fetch(u, tmpDir); err != nil {
		return "", errs.Wrap(err, "Could not download and unpack update")
	}

	u.tmpDir = filepath.Join(tmpDir, constants.ToplevelInstallArchiveDir)
	return u.tmpDir, nil
}

func (u *AvailableUpdate) prepareInstall(installTargetPath string, args []string) (string, []string, error) {
	sourcePath, err := u.DownloadAndUnpack()
	if err != nil {
		return "", nil, err
	}

	installerPath := filepath.Join(sourcePath, InstallerName)
	logging.Debug("Using installer: %s", installerPath)
	if !fileutils.FileExists(installerPath) {
		return "", nil, errs.Wrap(err, "Downloaded update does not have installer")
	}

	if installTargetPath == "" {
		installTargetPath, err = installation.InstallPath()
		if err != nil {
			return "", nil, errs.Wrap(err, "Could not detect install path")
		}
	}

	args = append(args, "--source-path", sourcePath)
	args = append([]string{installTargetPath}, args...)
	return installerPath, args, nil
}

func (u *AvailableUpdate) InstallBlocking(installTargetPath string, args ...string) error {
	logging.Debug("InstallBlocking path: %s, args: %v", installTargetPath, args)

	appdata, err := storage.AppDataPath()
	if err != nil {
		return errs.Wrap(err, "Could not detect appdata path")
	}

	// Protect against multiple updates happening simultaneously
	lockFile := filepath.Join(appdata, "install.lock")
	fileLock := flock.New(lockFile)
	lockSuccess, err := fileLock.TryLock()
	if err != nil {
		return errs.Wrap(err, "Could not create file lock required to install update")
	}
	if !lockSuccess {
		return &ErrorInProgress{locale.NewInputError("err_update_in_progress", "", lockFile)}
	}
	defer fileLock.Unlock()

	installTargetPath, args, err = u.prepareInstall(installTargetPath, args)
	if err != nil {
		return err
	}

	var envs []string
	if u.Tag != nil {
		envs = append(envs, fmt.Sprintf("%s=%s", constants.UpdateTagEnvVarName, *u.Tag))
	}
	_, _, err = exeutils.ExecuteAndPipeStd(installTargetPath, args, envs)
	if err != nil {
		return errs.Wrap(err, "Could not run installer")
	}

	return nil
}

// InstallWithProgress will fetch the update and run its installer
// Leave installTargetPath empty to use the default/existing installation path
func (u *AvailableUpdate) InstallWithProgress(installTargetPath string, progressCb func(string, bool)) (*os.Process, error) {
	installerPath, args, err := u.prepareInstall(installTargetPath, []string{})
	if err != nil {
		return nil, err
	}

	proc, err := exeutils.ExecuteAndForget(installerPath, args, func(cmd *exec.Cmd) error {
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
