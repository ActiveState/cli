// +build windows

package clean

import (
	"os/exec"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/gobuffalo/packr"
)

func runUninstall(params *UninstallParams, confirm confirmAble, outputer output.Outputer) error {
	err := removeCache(params.CachePath)
	if err != nil {
		return err
	}

	err = removeConfig(params.ConfigPath)
	if err != nil {
		return err
	}

	err = removeInstall(params.InstallPath)
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

func removeInstall(installPath string) error {
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String("removeInstall.bat")
	sf, fail := scriptfile.New(language.Batch, "removeInstall", scriptBlock)
	if fail != nil {
		return fail.ToError()
	}

	cmd := exec.Command("cmd.exe", "/C", sf.Filename(), installPath)
	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}
