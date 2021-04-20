package legacyupd

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/lockfile"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type UpdateResult struct {
	Updated     bool
	FromVersion string
	ToVersion   string
}

// AutoUpdate checks for updates once per day and, if one was found within a
// timeout period of one second, applies the update and returns `true`.
// Otherwise, returns `false`.
// AutoUpdate is skipped altogether if the current project has a locked version.
func AutoUpdate(pjPath string, out output.Outputer) (updated bool, resultVersion string) {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()
	info, err := update.Info(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logging.Debug("Automatically checking for updates timed out")
		} else {
			logging.Error("Unable to automatically check for updates: %s", err)
		}
		return false, ""
	} else if info == nil {
		logging.Debug("No update available.")
		return false, ""
	}

	// Self-update.
	logging.Debug("Self-updating.")
	err = update.Run(out, true)
	if err != nil {
		log := logging.Error
		if os.IsPermission(errs.InnerError(err)) {
			out.Error(locale.T("auto_update_permission_err"))
		}
		if errors.As(err, new(*lockfile.AlreadyLockedError)) {
			log = logging.Debug
		}
		log("Unable to self update: %s", err)
		return false, ""
	}

	return true, info.Version
}
