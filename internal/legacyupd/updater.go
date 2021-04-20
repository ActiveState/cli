package legacyupd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kardianos/osext"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/lockfile"
	"github.com/ActiveState/cli/internal/output"
)

const plat = runtime.GOOS + "-" + runtime.GOARCH

// Info holds the version and sha info
type Info struct {
	Version  string
	Sha256v2 string
}

// Updater holds all the information about our update
type Updater struct {
	CurrentVersion string // Currently running version.
	APIURL         string // Base URL for API requests (json files).
	CmdName        string // Command name is appended to the APIURL like http://apiurl/CmdName/. This represents one binary.
	ForceCheck     bool   // Check for update regardless of cktime timestamp
	DesiredBranch  string
	DesiredVersion string
	info           Info
}

func New(currentVersion string) *Updater {
	return &Updater{
		CurrentVersion: currentVersion,
		APIURL:         constants.APIUpdateURL,
		CmdName:        constants.CommandName,
	}
}

// Info reports updater.info, but only if we have an actual update
func (u *Updater) Info(ctx context.Context) (*Info, error) {
	if u.info.Version != "" && u.info.Version != u.CurrentVersion {
		return &u.info, nil
	}

	err := u.fetchInfo(ctx)
	if err != nil {
		return nil, err
	}

	if u.info.Version != "" && u.info.Version != u.CurrentVersion {
		return &u.info, nil
	}

	return nil, nil
}

// CanUpdate returns a bool conveying whether there is an update available
func (u *Updater) CanUpdate() bool {
	info, err := u.Info(context.Background())
	if err != nil {
		logging.Error(err.Error())
		return false
	}

	return info != nil
}

// Download acts as Run except that it unpacks it to the specified path rather than replace the current binary
func (u *Updater) Download(path string) error {
	if !u.CanUpdate() {
		return errs.New("No update available")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}
	return u.download(path)
}

// Run starts the update check and apply cycle.
func (u *Updater) Run(out output.Outputer, autoUpdate bool) error {
	if !u.CanUpdate() {
		return errs.New("No update available")
	}

	if err := u.update(out, autoUpdate); err != nil {
		return errs.Wrap(err, "update failed")
	}

	// Run _prepare after updates to facilitate anything the new version of the state tool might need to set up
	// Yes this is awkward, followup story here: https://www.pivotaltracker.com/story/show/176507898
	if stdout, stderr, err := exeutils.ExecSimple(os.Args[0], "_prepare"); err != nil {
		logging.Error("_prepare failed after update: %v\n\nstdout: %s\n\nstderr: %s", err, stdout, stderr)
	}

	return nil
}

// getExecRelativeDir relativizes the directory to store selfupdate state
// from the executable directory.
func (u *Updater) getExecRelativeDir(dir string) (string, error) {
	filename, err := osext.Executable()
	if err != nil {
		return "", err
	}

	path := filepath.Join(filepath.Dir(filename), dir)

	return path, nil
}

// update performs the actual update of the executable
func (u *Updater) download(path string) error {
	err := u.fetchInfo(context.Background())
	if err != nil {
		return err
	}
	bin, err := u.fetchAndVerifyFullBin()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, bin, 0666)
	if err != nil {
		return err
	}

	return nil
}

// update performs the actual update of the executable
func (u *Updater) update(out output.Outputer, autoUpdate bool) error {
	path, err := osext.Executable()
	if err != nil {
		return err
	}

	// Synchronize the update process between state tool instances by acquiring a lock file
	lockFile := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.update-lock", "state"))
	logging.Debug("Attempting to open lock file at %s", lockFile)
	pl, err := lockfile.NewPidLock(lockFile)
	if err != nil {
		return errs.Wrap(err, "could not create pid lock file for update process")
	}
	defer pl.Close()

	// This will succeed for only one of several concurrently state tool
	// instances. By returning otherwise, we are preventing that we download the
	// same new state tool version several times.
	_, err = pl.TryLock()
	if err != nil {
		if inProgErr := new(*lockfile.AlreadyLockedError); errors.As(err, inProgErr) {
			logging.Debug("Already updating: %s", errs.Join(*inProgErr, ": "))
			return *inProgErr
		}
		return errs.Wrap(err, "failed to acquire lock for update process")
	}

	logging.Debug("Attempting to open executable path at: %s", path)
	old, err := os.Open(path)
	if err != nil {
		fileutils.LogPath(path)
		return err
	}

	err = u.fetchInfo(context.Background())
	if err != nil {
		return err
	}
	if u.info.Version == u.CurrentVersion {
		logging.Debug("Already at latest version :)")
		return nil
	}

	out.Notice(locale.T("update_attempt"))
	if autoUpdate {
		out.Notice(locale.Tl(
			"auto_update_disable_notice",
			fmt.Sprintf("To avoid auto updating run [ACTIONABLE]state update --lock[/RESET] (only recommended for production environments) or set environment variable [ACTIONABLE]%s=true[/RESET]", constants.DisableUpdates),
		))
	}
	bin, err := u.fetchAndVerifyFullBin()
	if err != nil {
		return err
	}

	// close the old binary before installing because on windows
	// it can't be renamed if a handle to the file is still open
	err = old.Close()
	if err != nil {
		return err
	}

	err, errRecover := u.fromStream(path, bytes.NewBuffer(bin))
	if errRecover != nil {
		return errs.New("update and recovery errors: %q %q", err, errRecover)
	}
	if err != nil {
		return err
	}
	return nil
}

func (u *Updater) fetchBranch() string {
	if u.DesiredBranch != "" {
		return u.DesiredBranch
	}
	if overrideBranch := os.Getenv(constants.UpdateBranchEnvVarName); overrideBranch != "" {
		return overrideBranch
	}
	return constants.BranchName
}

// fetchInfo gets the `json` file containing update information
func (u *Updater) fetchInfo(ctx context.Context) error {
	if u.info.Version != "" {
		// already called fetchInfo
		return nil
	}
	branchName := u.fetchBranch()
	var fullURL = u.APIURL + url.QueryEscape(u.CmdName) + "/" + branchName + "/"
	if u.DesiredVersion != "" {
		fullURL += u.DesiredVersion + "/"
	}
	fullURL += url.QueryEscape(plat) + ".json"

	logging.Debug("Fetching update URL: %s", fullURL)

	r, err := u.fetch(ctx, fullURL)
	if err != nil {
		return err
	}

	err = json.NewDecoder(bytes.NewReader(r)).Decode(&u.info)
	if err != nil {
		logging.Error(err.Error())
		return err
	}
	if len(u.info.Sha256v2) != sha256.Size*2 {
		return errs.New("Bad cmd hash in JSON info")
	}
	return nil
}

func (u *Updater) fetchAndVerifyFullBin() ([]byte, error) {
	archive, err := u.fetchArchive()
	if err != nil {
		return nil, err
	}

	archive, err = ioutil.ReadAll(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}

	verified := verifySha(archive, u.info.Sha256v2)
	if !verified {
		return nil, locale.NewError("update_hash_mismatch")
	}

	bin, err := u.fetchBin(archive)
	if err != nil {
		logging.Error(err.Error())
		return nil, err
	}
	return bin, nil
}

func (u *Updater) fetchArchive() ([]byte, error) {
	var argCmdName = url.QueryEscape(u.CmdName)
	var argInfoVersion = url.QueryEscape(u.info.Version)
	var argPlatform = url.QueryEscape(plat)
	var branchName = u.fetchBranch()
	var ext = ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	var fetchURL = u.APIURL + fmt.Sprintf("%s/%s/%s/%s%s",
		argCmdName, branchName, argInfoVersion, argPlatform, ext)

	logging.Debug("Starting to fetch full binary from: %s", fetchURL)

	r, err := u.fetch(context.Background(), fetchURL)
	if err != nil {
		logging.Error(err.Error())
		return nil, err
	}

	return r, nil
}

func (u *Updater) fetch(ctx context.Context, url string) ([]byte, error) {
	readCloser, err := Fetch(ctx, url)
	if err != nil {
		return nil, err
	}

	if readCloser == nil {
		return nil, errs.New("fetch was expected to return non-nil ReadCloser")
	}

	bytes, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}
	readCloser.Close()

	return bytes, nil
}

func verifySha(bin []byte, sha string) bool {
	h := sha256.New()
	h.Write(bin)

	var computed = h.Sum(nil)
	var computedSha = fmt.Sprintf("%x", computed)
	var bytesEqual = computedSha == sha
	if !bytesEqual {
		logging.Error("SHA mismatch, expected: %s, actual: %s", sha, computedSha)
	}

	return bytesEqual
}

// cleanOld will remove any leftover binary files from previous updates
func Cleanup() {
	path, err := osext.Executable()
	if err != nil {
		logging.Error("Could not get executable: %v", err)
	}
	oldFile := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.old", "state"))

	if !fileutils.FileExists(oldFile) {
		return
	}

	err = os.Remove(oldFile)
	if err != nil {
		logging.Debug("Could not remove old file: %v", err)
		errHide := hideFile(oldFile)
		if errHide != nil {
			logging.Error("Could not hide old file: %v (remove err: %v)", errHide, err)
			return
		}
		return
	}

	return
}
