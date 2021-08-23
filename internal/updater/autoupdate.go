package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/lockfile"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/gofrs/flock"
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
func AutoUpdate(svcm *svcmanager.Manager, cfg *config.Instance, pjPath string, out output.Outputer) (updated bool, resultVersion string) {
	if versionInfo, _ := projectfile.ParseVersionInfo(pjPath); versionInfo != nil {
		return false, ""
	}

	model, err := model.NewSvcModel(context.Background(), cfg, svcm)
	if err != nil {
		logging.Error("Failed to initial state-svc model: %s", errs.JoinMessage(err))
		return false, ""
	}
	milliseconds := 100
	if milliSecondsOverride := os.Getenv(constants.AutoUpdateTimeoutEnvVarName); milliSecondsOverride != "" {
		override, err := strconv.Atoi(milliSecondsOverride)
		if err == nil {
			milliseconds = override
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(milliseconds)*time.Millisecond)
	defer cancel()
	gi, err := model.CheckUpdate(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logging.Debug("Automatically checking for updates timed out")
		} else {
			logging.Error("Unable to automatically check for updates: %s", err)
		}
		return false, ""
	} else if gi == nil {
		logging.Debug("No update available.")
		return false, ""
	}

	au := AvailableUpdate(*gi)
	info := &au

	// Self-update.
	logging.Debug("Self-updating.")

	fileLock := flock.New(filepath.Join(cfg.ConfigPath(), "install.lock"))
	locked, err := fileLock.TryLock()
	if err != nil {
		logging.Error("Failed to get lock for update: %s", errs.JoinMessage(err))
		return false, ""
	}
	if !locked {
		logging.Debug("Another update is already in progress")
		return false, ""
	}
	defer fileLock.Unlock()

	targetDir := filepath.Dir(appinfo.StateApp().Exec())
	err = info.InstallBlocking(targetDir)
	if err != nil {
		log := logging.Error
		if os.IsPermission(errs.InnerError(err)) {
			out.Error(locale.T("auto_update_permission_err"))
		}
		if errors.As(err, new(*lockfile.AlreadyLockedError)) {
			log = logging.Debug
		}
		log("Unable to self update: %s", errs.JoinMessage(err))
		return false, ""
	}

	return true, info.Version
}
