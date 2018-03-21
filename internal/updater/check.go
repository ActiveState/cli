package updater

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ActiveState/ActiveState-CLI/internal/config" // MUST be first!
	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
)

// CheckForAndApplyUpdates checks for updates once per day and, if one was
// found, applies it and returns `true`. Otherwise, returns `false`.
func CheckForAndApplyUpdates() bool {
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
	// Will check for updates. Touch the update check marker so the next check
	// will not happen for another day.
	err = os.Chtimes(updateCheckMarker, time.Now(), time.Now())
	if err != nil {
		logging.Error("Unable to automatically check for updates: %s", err)
		return false
	}

	// Check for an update.
	logging.Debug("Checking for updates.")
	update := Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
		Dir:            constants.UpdateStorageDir,
		CmdName:        constants.CommandName,
	}
	info, err := update.Info()
	if err != nil {
		logging.Error("Unable to automatically check for updates: %s", err)
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

	return true
}
