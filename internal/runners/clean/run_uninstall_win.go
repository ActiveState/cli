// +build windows

package clean

import (
	"os/exec"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/gobuffalo/packr"
)

func runUninstall(params *UninstallParams) error {
	logging.Debug("Removing cache path: %s", params.CachePath)
	logging.Debug("Removing State Tool binary: %s", params.InstallPath)
	logging.Debug("Removing config directory: %s", params.ConfigPath)

	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String("uninstall.bat")
	sf, fail := scriptfile.New(language.Batch, "uninstall", scriptBlock)
	if fail != nil {
		return fail.ToError()
	}

	cmd := exec.Command("cmd.exe", "/C", sf.Filename(), params.CachePath, params.ConfigPath, params.InstallPath)
	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func removeConfig(configPath string) error {
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String("removeConfig.bat")
	sf, fail := scriptfile.New(language.Batch, "removeConfig", scriptBlock)
	if fail != nil {
		return fail.ToError()
	}

	cmd := exec.Command("cmd.exe", "/C", sf.Filename(), configPath)
	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}
