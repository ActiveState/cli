package updater

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/kardianos/osext"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var (
	// FailNoUpdate identifies the failure as a no update available failure
	FailNoUpdate = failures.Type("updater.fail.noupdate")

	// FailUpdate identifies the failure as a failure in the update process
	FailUpdate = failures.Type("updater.fail.update")
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
	Requester      Requester
}

func New(currentVersion string) *Updater {
	return &Updater{
		CurrentVersion: currentVersion,
		APIURL:         constants.APIUpdateURL,
		CmdName:        constants.CommandName,
	}
}

// Info reports updater.info, but only if we have an actual update
func (u *Updater) Info() (*Info, error) {
	if u.info.Version != "" && u.info.Version != u.CurrentVersion {
		return &u.info, nil
	}

	err := u.fetchInfo()
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
	info, err := u.Info()
	if err != nil {
		logging.Error(err.Error())
		return false
	}

	return info != nil
}

// PrintUpdateMessage will print a message to stdout when an update is available.
// This will only print the message if the current project has a version lock AND if an update is available
func PrintUpdateMessage(pjPath string) {
	if versionInfo, _ := projectfile.ParseVersionInfo(pjPath); versionInfo == nil {
		return
	}

	up := Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
		CmdName:        constants.CommandName,
	}

	info, err := up.Info()
	if err != nil {
		logging.Error("Could not check for updates: %v", err)
		return
	}

	if info != nil && info.Version != constants.Version {
		print.Warning(locale.Tr("update_available", constants.Version, info.Version))
	}
}

// Download acts as Run except that it unpacks it to the specified path rather than replace the current binary
func (u *Updater) Download(path string) error {
	if !u.CanUpdate() {
		return failures.FailNotFound.New("No update available")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}
	return u.download(path)
}

// Run starts the update check and apply cycle.
func (u *Updater) Run(out output.Outputer) error {
	if !u.CanUpdate() {
		return failures.FailNotFound.New("No update available")
	}

	return u.update(out)
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
	err := u.fetchInfo()
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

func AcquireUpdateLock(updateDir string) (ok bool, cleanup func()) {
	fn := filepath.Join(updateDir, fmt.Sprintf(".%s.update-lock", "state"))
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("could not open file: %s: %v\n", fn, err)
		return false, nil
	}

	// attempting to obtain read lock on update file
	ft := &syscall.Flock_t{
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
		Type:   syscall.F_RDLCK,
	}

	err = syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, ft)
	if err != nil {
		fmt.Printf("could not lock")
		// if lock cannot be acquired -> a different process is updating, return nil
		f.Close()
		return false, nil
	}

	// check if update lock is not expired yet (this is just a back-up solution, in case the file locking does not work)
	info, err := f.Stat()
	expirationTime := info.ModTime().Add(5 * time.Minute)
	if info.Size() > 0 && time.Now().Before(expirationTime) {
		logging.Debug("not updating due to fallback-stat check of update file")
		return false, nil
	}

	// update expiration time stamp
	f.Write([]byte("1"))

	// defer release of lock
	return true, func() {
		ft.Type = syscall.F_UNLCK
		syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, ft)
		f.Close()
		os.Remove(fn)
	}
}

// update performs the actual update of the executable
func (u *Updater) update(out output.Outputer) error {
	path, err := osext.Executable()
	if err != nil {
		return err
	}

	logging.Debug("Attempting to open executable path at: %s", path)

	ok, cleanup := AcquireUpdateLock(filepath.Dir(path))
	if !ok {
		return nil
	}
	defer cleanup()

	old, err := os.Open(path)
	if err != nil {
		_ = fileutils.LogPath(path)
		return err
	}

	err = u.fetchInfo()
	if err != nil {
		return err
	}
	if u.info.Version == u.CurrentVersion {
		logging.Debug("Already at latest version :)")
		return nil
	}

	out.Notice(locale.T("auto_update_attempt"))
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
		return failures.FailVerify.New(fmt.Sprintf("update and recovery errors: %q %q", err, errRecover))
	}
	if err != nil {
		return err
	}
	return nil
}

func (u *Updater) fetchBranch() string {
	branchName := u.DesiredBranch
	if branchName == "" {
		branchName = constants.BranchName
	}
	if overrideBranch := os.Getenv(constants.UpdateBranchEnvVarName); overrideBranch != "" {
		branchName = overrideBranch
	}
	return branchName
}

// fetchInfo gets the `json` file containing update information
func (u *Updater) fetchInfo() error {
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

	r, err := u.fetch(fullURL)
	if err != nil {
		return err
	}

	err = json.NewDecoder(bytes.NewReader(r)).Decode(&u.info)
	if err != nil {
		logging.Error(err.Error())
		return err
	}
	if len(u.info.Sha256v2) != sha256.Size*2 {
		return failures.FailVerify.New("Bad cmd hash in JSON info")
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
		return nil, failures.FailVerify.New(locale.T("update_hash_mismatch"))
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

	r, err := u.fetch(fetchURL)
	if err != nil {
		logging.Error(err.Error())
		return nil, err
	}

	return r, nil
}

func (u *Updater) fetch(url string) ([]byte, error) {
	var requester Requester
	if u.Requester != nil {
		requester = u.Requester
	} else {
		requester = &HTTPRequester{}
	}

	readCloser, err := requester.Fetch(url)
	if err != nil {
		return nil, err
	}

	if readCloser == nil {
		return nil, failures.FailIO.New("fetch was expected to return non-nil ReadCloser")
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
func cleanOld() error {
	path, err := osext.Executable()
	if err != nil {
		return err
	}
	oldFile := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.old", "state"))

	if !fileutils.FileExists(oldFile) {
		return nil
	}

	err = os.Remove(oldFile)
	if err != nil {
		errHide := hideFile(oldFile)
		if errHide != nil {
			return errHide
		}
		return err
	}

	return nil
}
