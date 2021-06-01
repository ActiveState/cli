// +build windows

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/gobuffalo/packr"
)

func removeSelf() error {
	scriptName := "removePaths"
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, err := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if err != nil {
		return locale.WrapError(err, "err_clean_script", "Could not create new scriptfile")
	}

	exe := appinfo.StateApp().Exec()

	var logFile string = "remove-old-state.log"
	logDir, err := ioutil.TempDir("", "")
	if err != nil {
		logging.Error("Failed to create temporary dir for old State Tool removal-log file: %v", err)
	} else {
		logFile = filepath.Join(logDir, "remove-old-state.log")
	}
	args := []string{"/C", sf.Filename(), logFile, fmt.Sprintf("%d", os.Getpid()), filepath.Base(exe), exe}
	_, err = exeutils.ExecuteAndForget("cmd.exe", args)
	if err != nil {
		return locale.WrapError(err, "err_clean_start", "Could not start remove the transitional State Tool")
	}

	return nil
}
