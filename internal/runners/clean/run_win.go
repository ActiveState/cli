// +build windows

package clean

import (
	"fmt"
	"os/exec"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/gobuffalo/packr"
)

func removeConfig(configPath string) error {
	return runScript("removeConfig", configPath)
}

func removeInstall(installPath string) error {
	return runScript("removeInstall", installPath)
}

func runScript(scriptName, path string) error {
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, fail := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if fail != nil {
		return fail.ToError()
	}

	cmd := exec.Command("cmd.exe", "/C", sf.Filename(), path)
	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}
