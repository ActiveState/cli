package legacytray

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/app"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/shirou/gopsutil/v3/process"
)

const stateTrayCmd = "state-tray"
const trayAppName = "ActiveState Desktop (Preview)"
const stateUpdateDialogCmd = "state-update-dialog"

func DetectAndRemove(path string, cfg *config.Instance) error {
	binDir := filepath.Join(path, installation.BinDirName)
	trayExec := filepath.Join(binDir, stateTrayCmd+osutils.ExeExtension)
	if !fileutils.FileExists(trayExec) {
		return nil // nothing to do
	}

	// Attempt to stop the tray app before removing it.
	err := stopTrayApp(cfg)
	if err != nil {
		return errs.Wrap(err, "Unable to stop try app")
	}

	appDir, err := installation.ApplicationInstallPath()
	if err != nil {
		return errs.Wrap(err, "Unable to get application install path")
	}

	// Disable autostart of state-tray.
	if app, err := app.New(trayAppName, trayExec, nil, appDir, app.Options{}); err == nil {
		disableErr := autostart.Disable(app.Path(), autostart.Options{
			LaunchFileName: trayLaunchFileName, // only used for Linux; ignored on macOS, Windows
		})
		if disableErr != nil {
			return errs.Wrap(err, "Unable to disable tray autostart")
		}
	} else {
		return errs.Wrap(err, "Unable to disable tray autostart")
	}

	// Remove Linux .desktop files, macOS .app bundles, Windows shortcuts, etc.
	err = osSpecificRemoveTray(path, trayExec)
	if err != nil {
		return err
	}

	// Finally, remove state-tray and state-update-dialog executables.
	for _, name := range []string{stateTrayCmd, stateUpdateDialogCmd} {
		if exec := filepath.Join(binDir, name+osutils.ExeExtension); fileutils.FileExists(exec) {
			err = os.Remove(exec)
			if err != nil {
				return errs.Wrap(err, "Unable to remove %s", name)
			}
		}
	}

	return nil
}

const configKeyTrayPid = "tray-pid"

func stopTrayApp(cfg *config.Instance) error {
	proc, err := getTrayProcess(cfg)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil
		}
		return errs.Wrap(err, "Could not detect if state-tray pid exists")
	}

	if err := proc.Kill(); err != nil {
		return errs.Wrap(err, "Could not kill state-tray")
	}

	return nil
}

func getTrayProcess(cfg *config.Instance) (*process.Process, error) {
	trayPid := cfg.GetInt(configKeyTrayPid)
	if trayPid <= 0 {
		return nil, errs.Wrap(process.ErrorProcessNotRunning, "state-tray pid not set in config")
	}

	proc, err := process.NewProcess(int32(trayPid))
	if err != nil {
		return nil, errs.Wrap(err, "Could not detect if state-tray pid exists")
	}

	return proc, nil
}
