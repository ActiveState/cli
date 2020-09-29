package updater

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
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

// autoUpdate checks for updates once per day and, if one was found within a
// timeout period of one second, applies the update and returns `true`.
// Otherwise, returns `false`.
// autoUpdate is skipped altogether if the current project has a locked version.
func autoUpdate(pjPath string, out output.Outputer) (updated bool, resultVersion string) {
	if versionInfo, _ := projectfile.ParseVersionInfo(pjPath); versionInfo != nil {
		return false, ""
	}

	// Check for an update, but timeout after one second.
	logging.Debug("Checking for updates.")
	update := Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
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
		if os.IsPermission(errs.InnerError(err)) {
			out.Error(locale.T("auto_update_permission_err"))
		}
		logging.Error("Unable to self update: %s", err)
		return false, ""
	}

	return true, info.Version
}
