package updater

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ActiveState/ActiveState-CLI/internal/config" // MUST be first!
	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
)

// Runs the given updater function on a timeout.
func timeout(f func() (*Info, error), t time.Duration) (*Info, error) {
	timeoutCh := make(chan bool, 1)
	infoCh := make(chan *Info, 1)
	errCh := make(chan error, 1)
	// Run the timeout function.
	go func() {
		time.Sleep(t)
		timeoutCh <- true
		close(timeoutCh)
	}()
	// Run the updater function.
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
	// Return the appropriate value, taking timeout into consideration.
	select {
	case <-timeoutCh:
		return nil, errors.New("timeout")
	case info := <-infoCh:
		return info, nil
	case err := <-errCh:
		return nil, err
	}
}

// TimedCheck checks for updates once per day and, if one was found within a
// timeout period of one second, applies the update and returns `true`.
// Otherwise, returns `false`.
func TimedCheck() bool {
	// Determine whether or not an update check has been performed today.
	updateCheckMarker := filepath.Join(config.GetDataDir(), "update-check")
	marker, err := os.Stat(updateCheckMarker)
	if err != nil {
		// Marker does not exist. Create it.
		err = ioutil.WriteFile(updateCheckMarker, []byte(""), 0666)
		if err != nil {
			logging.Error("Unable to automatically check for updates: %s", err)
			return false
		}
	} else {
		// Check to see if it has been 24 hours since the last update check. If not,
		// skip another check.
		nextCheckTime := marker.ModTime().Add(24 * time.Hour)
		if time.Now().Before(nextCheckTime) {
			logging.Debug("Not checking for updates until %s", nextCheckTime)
			return false
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
	info, err := timeout(update.Info, time.Second)
	if err != nil {
		if err.Error() != "timeout" {
			logging.Error("Unable to automatically check for updates: %s", err)
		} else {
			logging.Debug("Automatically checking for updates timed out")
		}
		return false
	} else if info == nil {
		logging.Debug("No update available.")
		return false
	}

	// Self-update.
	logging.Debug("Self-updating.")
	err = update.Run()
	if err != nil {
		logging.Error("Unable to automatically check for updates: %s", err)
		return false
	}

	// Touch the update check marker so the next check will not happen for another
	// day.
	err = os.Chtimes(updateCheckMarker, time.Now(), time.Now())
	if err != nil {
		logging.Error("Unable to automatically check for updates: %s", err)
		return false
	}

	return true
}
