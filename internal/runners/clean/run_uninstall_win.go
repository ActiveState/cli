// +build windows

package clean

import (
	"os/exec"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/gobuffalo/packr"
)

func runUninstall(params *UninstallParams, confirm confirmAble, outputer output.Outputer) error {
	logging.Debug("Removing cache path: %s", params.CachePath)
	logging.Debug("Removing State Tool binary: %s", params.InstallPath)
	logging.Debug("Removing config directory: %s", params.ConfigPath)

	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String("clean.bat")
	sf, fail := scriptfile.New(language.Batch, "clean", scriptBlock)
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
