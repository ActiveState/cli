// +build windows

package clean

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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

	err = removePaths(logFile.Name(), u.installPath, u.cfg.ConfigPath())
	if err != nil {
		return err
	}

	u.out.Print(locale.Tr("clean_message_windows", u.installPath, u.cfg.ConfigPath(), logFile.Name()))
	return nil
}

func removeConfig(configPath string, out output.Outputer) error {
	logFile, err := ioutil.TempFile("", "state-clean-config")
	if err != nil {
		return locale.WrapError(err, "err_clean_logfile", "Could not create temporary log file")
	}

	out.Print(locale.Tr("clean_config_message_windows", configPath, logFile.Name()))
	return removePaths(logFile.Name(), configPath)
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
