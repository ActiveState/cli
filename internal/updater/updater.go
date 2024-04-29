package updater

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gofrs/flock"

	"github.com/ActiveState/cli/internal/analytics"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

const (
	CfgKeyInstallVersion = "state_tool_installer_version"
	InstallerName        = "state-installer" + osutils.ExeExtension
)

type ErrorInProgress struct{ *locale.LocalizedError }

var errPrivilegeMistmatch = errs.New("Privilege mismatch")

type Origin struct {
	Channel string
	Version string
}

func NewOriginDefault() *Origin {
	return &Origin{
		Channel: constants.ChannelName,
		Version: constants.Version,
	}
}

type AvailableUpdate struct {
	Channel  string  `json:"channel"`
	Version  string  `json:"version"`
	Platform string  `json:"platform"`
	Path     string  `json:"path"`
	Sha256   string  `json:"sha256"`
	Tag      *string `json:"tag,omitempty"`
}

func NewAvailableUpdate(channel, version, platform, path, sha256, tag string) *AvailableUpdate {
	var t *string
	if tag != "" {
		t = &tag
	}

	return &AvailableUpdate{
		Channel:  channel,
		Version:  version,
		Platform: platform,
		Path:     path,
		Sha256:   sha256,
		Tag:      t,
	}
}

func NewAvailableUpdateFromGraph(au *graph.AvailableUpdate) *AvailableUpdate {
	if au == nil {
		return &AvailableUpdate{}
	}
	return NewAvailableUpdate(au.Channel, au.Version, au.Platform, au.Path, au.Sha256, "")
}

func (u *AvailableUpdate) IsValid() bool {
	return u != nil && u.Channel != "" && u.Version != "" && u.Platform != "" && u.Path != "" && u.Sha256 != ""
}

func (u *AvailableUpdate) Equals(origin *Origin) bool {
	return u.Channel == origin.Channel && u.Version == origin.Version
}

type UpdateInstaller struct {
	AvailableUpdate *AvailableUpdate
	Origin          *Origin

	url    string
	tmpDir string
	an     analytics.Dispatcher
}

// NewUpdateInstallerByOrigin returns an instance of Update. Allowing origin to
// be set is useful for testing.
func NewUpdateInstallerByOrigin(an analytics.Dispatcher, origin *Origin, avUpdate *AvailableUpdate) *UpdateInstaller {
	apiUpdateURL := constants.APIUpdateURL
	if url, ok := os.LookupEnv("_TEST_UPDATE_URL"); ok {
		apiUpdateURL = url
	}

	return &UpdateInstaller{
		AvailableUpdate: avUpdate,
		Origin:          origin,
		url:             apiUpdateURL + "/" + avUpdate.Path,
		an:              an,
	}
}

func NewUpdateInstaller(an analytics.Dispatcher, avUpdate *AvailableUpdate) *UpdateInstaller {
	return NewUpdateInstallerByOrigin(an, NewOriginDefault(), avUpdate)
}

func (u *UpdateInstaller) ShouldInstall() bool {
	return u.AvailableUpdate.IsValid() &&
		(os.Getenv(constants.ForceUpdateEnvVarName) == "true" ||
			!u.AvailableUpdate.Equals(u.Origin))
}

func (u *UpdateInstaller) DownloadAndUnpack() (string, error) {
	if u.tmpDir != "" {
		// To facilitate callers explicitly calling this method we cache the tmp dir and just return it if it's set
		return u.tmpDir, nil
	}

	tmpDir, err := os.MkdirTemp("", "state-update")
	if err != nil {
		msg := anaConst.UpdateErrorTempDir
		u.analyticsEvent(anaConst.ActUpdateDownload, anaConst.UpdateLabelFailed, msg)
		return "", errs.Wrap(err, msg)
	}

	if err := NewFetcher(u.an).Fetch(u, tmpDir); err != nil {
		return "", errs.Wrap(err, "Could not download and unpack update")
	}

	payloadDir := tmpDir
	if legacyDir := filepath.Join(tmpDir, constants.LegacyToplevelInstallArchiveDir); fileutils.DirExists(legacyDir) {
		payloadDir = legacyDir
	}
	u.tmpDir = payloadDir
	return u.tmpDir, nil
}

func (u *UpdateInstaller) prepareInstall(installTargetPath string, args []string) (string, []string, error) {
	sourcePath, err := u.DownloadAndUnpack()
	if err != nil {
		return "", nil, err
	}
	u.analyticsEvent(anaConst.ActUpdateDownload, anaConst.UpdateLabelSuccess, "")

	installerPath := filepath.Join(sourcePath, InstallerName)
	logging.Debug("Using installer: %s", installerPath)
	if !fileutils.FileExists(installerPath) {
		msg := anaConst.UpdateErrorNoInstaller
		u.analyticsEvent(anaConst.ActUpdateInstall, anaConst.UpdateLabelFailed, msg)
		return "", nil, errs.Wrap(err, msg)
	}

	if installTargetPath == "" {
		installTargetPath, err = installation.InstallPathFromExecPath()
		if err != nil {
			msg := anaConst.UpdateErrorInstallPath
			u.analyticsEvent(anaConst.ActUpdateInstall, anaConst.UpdateLabelFailed, msg)
			return "", nil, errs.Wrap(err, msg)
		}
	}

	args = append(args, "--update")
	args = append([]string{installTargetPath}, args...)
	return installerPath, args, nil
}

func (u *UpdateInstaller) InstallBlocking(installTargetPath string, args ...string) (rerr error) {
	logging.Debug("InstallBlocking path: %s, args: %v", installTargetPath, args)

	// Report any failure to analytics.
	defer func() {
		if rerr == nil {
			return
		}
		switch {
		case os.IsPermission(rerr):
			u.analyticsEvent(anaConst.ActUpdateInstall, anaConst.UpdateLabelFailed, "Could not update the state tool due to insufficient permissions.")
		case errs.Matches(rerr, &ErrorInProgress{}):
			u.analyticsEvent(anaConst.ActUpdateInstall, anaConst.UpdateLabelFailed, anaConst.UpdateErrorInProgress)
		default:
			u.analyticsEvent(anaConst.ActUpdateInstall, anaConst.UpdateLabelFailed, anaConst.UpdateErrorInstallFailed)
		}
	}()

	err := checkAdmin()
	if errors.Is(err, errPrivilegeMistmatch) {
		return locale.NewInputError("err_update_privilege_mismatch")
	} else if err != nil {
		return errs.Wrap(err, "Could not check if State Tool was installed as admin")
	}

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
	defer rtutils.Closer(fileLock.Unlock, &rerr)

	var installerPath string
	installerPath, args, err = u.prepareInstall(installTargetPath, args)
	if err != nil {
		return err
	}

	var envs []string
	if u.AvailableUpdate.Tag != nil {
		envs = append(envs, fmt.Sprintf("%s=%s", constants.UpdateTagEnvVarName, *u.AvailableUpdate.Tag))
	}

	_, _, err = osutils.ExecuteAndPipeStd(installerPath, args, envs)
	if err != nil {
		return errs.Wrap(err, "Could not run installer")
	}

	// installerPath looks like "<tempDir>/state-update\d{10}/state-install/state-installer".
	updateDir := filepath.Dir(filepath.Dir(installerPath))
	logging.Debug("Cleaning up temporary update directory: %s", updateDir)
	if strings.HasPrefix(filepath.Base(updateDir), "state-update") {
		err = os.RemoveAll(updateDir)
		if err != nil {
			multilog.Error("Unable to remove update directory '%s': %v", updateDir, errs.JoinMessage(err))
		}
	} else {
		// Do not report to rollbar, but log the error for our integration tests to catch.
		logging.Error("Did not remove temporary update directory. "+
			"installerPath: %s\nupdateDir: %s\nExpected a 'state-update' prefix for the latter", installerPath, updateDir)
	}

	u.analyticsEvent(anaConst.ActUpdateInstall, anaConst.UpdateLabelSuccess, "")

	return nil
}

// InstallWithProgress will fetch the update and run its installer
// Leave installTargetPath empty to use the default/existing installation path
func (u *UpdateInstaller) InstallWithProgress(installTargetPath string, progressCb func(string, bool)) (*os.Process, error) {
	installerPath, args, err := u.prepareInstall(installTargetPath, []string{})
	if err != nil {
		return nil, err
	}

	proc, err := osutils.ExecuteAndForget(installerPath, args, func(cmd *exec.Cmd) error {
		var stdout io.ReadCloser
		var stderr io.ReadCloser
		if stderr, err = cmd.StderrPipe(); err != nil {
			return errs.Wrap(err, "Could not obtain stderr pipe")
		}
		if stdout, err = cmd.StdoutPipe(); err != nil {
			return errs.Wrap(err, "Could not obtain stderr pipe")
		}
		if u.AvailableUpdate.Tag != nil {
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", constants.UpdateTagEnvVarName, *u.AvailableUpdate.Tag))
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

func (u *UpdateInstaller) analyticsEvent(action, label, msg string) {
	dims := &dimensions.Values{}
	if u.AvailableUpdate != nil {
		dims.TargetVersion = ptr.To(u.AvailableUpdate.Version)
	}

	if msg != "" {
		dims.Error = ptr.To(msg)
	}

	u.an.EventWithLabel(anaConst.CatUpdates, action, label, dims)
}
