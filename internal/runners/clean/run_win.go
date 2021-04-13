// +build windows

package clean

import (
	"fmt"
	"os/exec"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/scriptfile"
)

func removeConfig(cfg configurable) error {
	return runScript("removeConfig", cfg.ConfigPath())
}

// TODO: Update this to removeInstallDir
func removeInstall(installPath string) error {
	return runScript("removeInstall", installPath)
}

func runScript(scriptName, path string) error {
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, err := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if err != nil {
		return err
	}

	cmd := exec.Command("cmd.exe", "/C", sf.Filename(), path)
	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func removeTrayApp() error {
	trayAppPath := filepath.Join(autostart.StartupPath)
	if !fileutils.DirExists(trayAppPath) {
		return nil
	}

	err := os.RemoveAll(trayAppPath)
	if err != nil {
		return locale.WrapError(err, "err_remove_tray", "Could not remove state tray")
	}

	return nil
}
