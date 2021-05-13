// +build windows

package clean

import (
	"errors"
	"fmt"
	"os"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/scriptfile"
)

func removeConfig(cfg configurable) error {
	return runScript("removeConfig", cfg.ConfigPath())
}

func removeInstall(cfg configurable, installPath string) error {
	// On Windows we need to halt the state tray and the state service before we can remove them
	svcInfo := appinfo.SvcApp(installPath)
	trayInfo := appinfo.TrayApp(installPath)

	// Todo: https://www.pivotaltracker.com/story/show/177585085
	// Yes this is awkward right now
	if err := installation.StopTrayApp(cfg); err != nil {
		return errs.Wrap(err, "Failed to stop %s", trayInfo.Name())
	}

	// Stop state-svc before accessing its files
	if fileutils.FileExists(svcInfo.Exec()) {
		exitCode, _, err := exeutils.Execute(svcInfo.Exec(), []string{"stop"}, nil)
		if err != nil {
			return errs.Wrap(err, "Stopping %s returned error", svcInfo.Name())
		}
		if exitCode != 0 {
			return errs.New("Stopping %s exited with code %d", svcInfo.Name(), exitCode)
		}
	}

	var aggErr error
	for _, info := range []*appinfo.AppInfo{svcInfo, trayInfo} {
		err := os.Remove(info.Exec())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			aggErr = errs.Wrap(aggErr, "Could not remove %s: %v", info.Exec(), err)
		}
	}

	return runScript("removeInstall", installPath)
}

func runScript(scriptName, path string) error {
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, err := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if err != nil {
		return err
	}

	_, err = exeutils.ExecuteAndForget("cmd.exe", []string{"/C", sf.Filename(), filepath.Join(path, "state"+osutils.ExeExt)})
	if err != nil {
		return err
	}

	return nil
}
