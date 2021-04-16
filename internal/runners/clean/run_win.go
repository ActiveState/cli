// +build windows

package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/scriptfile"
)

func removeConfig(configPath string) error {
	return removeDirs(configPath)
}

func removeInstallDir(installationPath string) error {
	return removeDirs(installationPath)
}

func removeDirs(dirs ...string) error {
	scriptName := "removeDirs"
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, err := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not get executable name")
	}

	args := []string{"/C", sf.Filename(), fmt.Sprintf("%d", os.Getpid()), filepath.Base(exe)}
	args = append(args, dirs...)
	cmd := exec.Command("cmd.exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000}
	err = cmd.Start()
	if err != nil {
		return errs.Wrap(err, "Could not start script")
	}

	return nil
}

func removeTrayApp() error {
	// On Windows there is currently no separate app installation dir
	return nil
}
