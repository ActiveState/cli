package updater

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Runs the given updater function on a timeout.
func timeout(f func() (*Info, error), t time.Duration) (*Info, error) {
	timeoutCh := make(chan bool, 1)
	infoCh := make(chan *Info, 1)
	errCh := make(chan error, 1)
	// Run the timeout function in a separate thread.
	go func() {
		time.Sleep(t)
		timeoutCh <- true
		close(timeoutCh)
	}()
	// Run the updater function in a separate thread.
	go func() {
		info, err := f()
		if err == nil {
			infoCh <- info
		} else {
			errCh <- err
		}
		close(infoCh)
		close(errCh)
	}()
	// Wait until one of the threads produces data in one of the channels being
	// monitored. If the timeout comes first, report the timeout. If the update
	// info comes first, return that. If there was some other error, return that.
	select {
	case <-timeoutCh:
		return nil, errors.New("timeout")
	case info := <-infoCh:
		return info, nil
	case err := <-errCh:
		return nil, err
	}
}

type UpdateResult struct {
	Updated     bool
	FromVersion string
	ToVersion   string
}

// AutoUpdate checks for updates once per day and, if one was found within a
// timeout period of one second, applies the update and returns `true`.
// Otherwise, returns `false`.
// AutoUpdate is skipped altogether if the current project has a locked version.
func AutoUpdate(pjPath string, out outputer.Output) (updated bool, resultVersion string) {
	if versionInfo, _ := projectfile.ParseVersionInfo(pjPath); versionInfo != nil {
		return false, ""
	}

	// Determine whether or not an update check has been performed today.
	updateCheckMarker := filepath.Join(config.ConfigPath(), "update-check")
	marker, err := os.Stat(updateCheckMarker)
	if err != nil {
		// Marker does not exist. Create it.
		err = ioutil.WriteFile(updateCheckMarker, []byte(""), 0666)
		if err != nil {
			logging.Error("Unable to create/write update marker: %s", err)
			_ = fileutils.LogPath(config.ConfigPath())
			return false, ""
		}
	} else {
		// Check to see if it has been 24 hours since the last update check. If not,
		// skip another check.
		nextCheckTime := marker.ModTime().Add(24 * time.Hour)
		if time.Now().Before(nextCheckTime) {
			logging.Debug("Not checking for updates until %s", nextCheckTime)
			return false, ""
		}
	}

	// Check for an update, but timeout after one second.
	logging.Debug("Checking for updates.")
	update := Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
		Dir:            constants.UpdateStorageDir,
		CmdName:        constants.CommandName,
	}
	seconds := 1
	if secondsOverride := os.Getenv(constants.AutoUpdateTimeoutEnvVarName); secondsOverride != "" {
		override, err := strconv.Atoi(secondsOverride)
		if err == nil {
			seconds = override
		}
	}
	info, err := timeout(update.Info, time.Duration(seconds)*time.Second)
	if err != nil {
		if err.Error() != "timeout" {
			logging.Error("Unable to automatically check for updates: %s", err)
		} else {
			logging.Debug("Automatically checking for updates timed out")
		}
		return false, ""
	} else if info == nil {
		logging.Debug("No update available.")
		return false, ""
	}

	// Self-update.
	logging.Debug("Self-updating.")
	err = update.Run(out)
	if err != nil {
		logging.Error("Unable to self update: %s", err)
		return false, ""
	}

	// Touch the update check marker so the next check will not happen for another
	// day.
	err = os.Chtimes(updateCheckMarker, time.Now(), time.Now())
	if err != nil {
		logging.Error("Unable to update modification times of check marker: %s", err)
		return false, ""
	}

	cleanOld()

	return true, info.Version
}
