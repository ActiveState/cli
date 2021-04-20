// +build windows

package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
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
		return locale.WrapError(err, "err_clean_script", "Could not create new scriptfile")
	}

	exe, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_clean_executable", "Could not get executable name")
	}

	args := []string{"/C", sf.Filename(), fmt.Sprintf("%d", os.Getpid()), filepath.Base(exe)}
	args = append(args, dirs...)
	cmd := exec.Command("cmd.exe", args...)
	cmd.SysProcAttr = osutils.SysProcAttrForBackgroundProcess()
	err = cmd.Start()
	if err != nil {
		return locale.WrapError(err, "err_clean_start", "Could not start remove direcotry script")
	}

	return nil
}

func autostartFilePath() (string, error) {
	return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "activestate-desktop.lnk")
}

func removeTrayApp() error {
	// On Windows there is currently no separate app installation dir
	return nil
}
