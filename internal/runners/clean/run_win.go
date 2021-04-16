// +build windows

package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/scriptfile"
)

func removeConfig(configPath string) error {
	return removeDir(configPath)
}

func removeInstallDir(installationPath string) error {
	return removeDir(installationPath)
}

func removeDir(path string) error {
	// TODO: Batch scripts seem to be interferring with one another when
	// run in an integration test. Update script to accept multiple dirs
	time.Sleep(500 * time.Millisecond)
	scriptName := "removeDir"
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

	cmd := exec.Command("cmd.exe", "/C", sf.Filename(), path, fmt.Sprintf("%d", os.Getpid()), filepath.Base(exe))
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
