// +build windows

package clean

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/scriptfile"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return err
	}

	logFile, err := ioutil.TempFile("", "state-clean-uninstall")
	if err != nil {
		return locale.WrapError(err, "err_clean_logfile", "Could not create temporary log file")
	}

	err = removeInstall(u.cfg, logFile.Name(), u.installDir, u.cfg.ConfigPath())
	if err != nil {
		return err
	}

	u.out.Print(locale.Tr("clean_message_windows", logFile.Name()))
	return nil
}

func removeConfig(configPath string, out output.Outputer) error {
	logFile, err := ioutil.TempFile("", "state-clean-config")
	if err != nil {
		return locale.WrapError(err, "err_clean_logfile", "Could not create temporary log file")
	}

	out.Print(locale.Tr("clean_config_message_windows", logFile.Name()))
	return removePaths(logFile.Name(), configPath)
}

func removeInstall(cfg configurable, logFile, installPath, configPath string) error {
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

	return removePaths(logFile, filepath.Join(installPath, "state"+osutils.ExeExt), configPath)
}

func removePaths(logFile string, dirs ...string) error {
	logging.Debug("Removing paths: %v", dirs)
	scriptName := "removePaths"
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, err := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if err != nil {
		return locale.WrapError(err, "err_clean_script", "Could not create new scriptfile")
	}

	exe, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_clean_executable", "Could not get executable name")
	}

	args := []string{"/C", sf.Filename(), logFile, fmt.Sprintf("%d", os.Getpid()), filepath.Base(exe)}
	args = append(args, dirs...)
	_, err = exeutils.ExecuteAndForget("cmd.exe", args)
	if err != nil {
		return locale.WrapError(err, "err_clean_start", "Could not start remove direcotry script")
	}

	return nil
}

func runScript(scriptName, path string) error {
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, err := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if err != nil {
		return err
	}

	_, err = exeutils.ExecuteAndForget("cmd.exe", []string{"/C", sf.Filename(), path})
	if err != nil {
		return err
	}

	return nil
}
